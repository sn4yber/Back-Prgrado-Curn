package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
	"github.com/sn4yber/curn-networking/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
)

// ─── Configuración interna ────────────────────────────────────────────────────

type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	keyLength   uint32
}

type jwtParams struct {
	secret        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// Service implementa input.AuthUseCase.
// Solo conoce los puertos — nunca detalles de HTTP ni de PostgreSQL.
type Service struct {
	userRepo         output.UserRepository
	refreshTokenRepo output.RefreshTokenRepository
	resetTokenRepo   output.PasswordResetTokenRepository
	argon2           argon2Params
	jwt              jwtParams
	log              logger.Logger
}

// jwtClaims define los claims del access token.
type jwtClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// New construye el servicio con todas sus dependencias inyectadas.
func New(
	userRepo output.UserRepository,
	refreshTokenRepo output.RefreshTokenRepository,
	resetTokenRepo output.PasswordResetTokenRepository,
	argon2Memory, argon2Iterations uint32,
	argon2Parallelism uint8,
	argon2KeyLength uint32,
	jwtSecret string,
	jwtAccessExpiry, jwtRefreshExpiry time.Duration,
	log logger.Logger,
) *Service {
	return &Service{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		resetTokenRepo:   resetTokenRepo,
		argon2: argon2Params{
			memory:      argon2Memory,
			iterations:  argon2Iterations,
			parallelism: argon2Parallelism,
			keyLength:   argon2KeyLength,
		},
		jwt: jwtParams{
			secret:        jwtSecret,
			accessExpiry:  jwtAccessExpiry,
			refreshExpiry: jwtRefreshExpiry,
		},
		log: log,
	}
}

// ─── Register ─────────────────────────────────────────────────────────────────

// emailDomainInstitucional es el único dominio permitido para registrarse.
const emailDomainInstitucional = "@campusuninunez.edu.co"

// Solo se permiten correos con dominio @campusuninunez.edu.co
func (s *Service) Register(ctx context.Context, req input.RegisterRequest) (*input.AuthResponse, error) {
	if !strings.HasSuffix(req.Email, emailDomainInstitucional) {
		s.log.Warn("intento de registro con correo no institucional",
			zap.String("email", req.Email),
		)
		return nil, apperrors.New(
			400,
			"solo se permiten correos institucionales (@campusuninunez.edu.co)",
			nil,
		)
	}

	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.ErrInternal
	}
	if exists {
		return nil, apperrors.ErrEmailAlreadyExists
	}

	passwordHash, err := s.hashPassword(req.Password)
	if err != nil {
		s.log.Error("error al hashear contraseña", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	programID, err := uuid.Parse(req.ProgramID)
	if err != nil {
		return nil, apperrors.ErrValidation
	}

	user := &domain.User{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: passwordHash,
		ProgramID:    programID,
		Status:       domain.UserStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Save(ctx, user); err != nil {
		s.log.Error("error al guardar usuario", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	s.log.Audit("usuario registrado",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)

	return s.buildAuthResponse(ctx, user)
}

// ─── Login ────────────────────────────────────────────────────────────────────

// Login verifica credenciales y emite access + refresh token.
func (s *Service) Login(ctx context.Context, req input.LoginRequest) (*input.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// No revelamos si el email existe o no — siempre el mismo error
		return nil, apperrors.ErrInvalidCredentials
	}

	if err := s.verifyPassword(req.Password, user.PasswordHash); err != nil {
		s.log.Audit("intento de login fallido",
			zap.String("email", req.Email),
		)
		return nil, apperrors.ErrInvalidCredentials
	}

	if !user.IsActive() {
		if user.Status == domain.UserStatusBanned {
			return nil, apperrors.ErrUserBanned
		}
		return nil, apperrors.ErrUserInactive
	}

	s.log.Audit("login exitoso",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)

	return s.buildAuthResponse(ctx, user)
}

// ─── RefreshToken ─────────────────────────────────────────────────────────────

// RefreshToken valida el refresh token y emite un nuevo par de tokens.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*input.AuthResponse, error) {
	tokenHash := hashSHA256(refreshToken)

	stored, err := s.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, apperrors.ErrRefreshTokenNotFound
	}

	if time.Now().After(stored.ExpiresAt) {
		_ = s.refreshTokenRepo.DeleteByTokenHash(ctx, tokenHash)
		return nil, apperrors.ErrTokenExpired
	}

	user, err := s.userRepo.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	// Rotación de refresh token — invalidamos el anterior
	_ = s.refreshTokenRepo.DeleteByTokenHash(ctx, tokenHash)

	return s.buildAuthResponse(ctx, user)
}

// ─── ForgotPassword ───────────────────────────────────────────────────────────

// ForgotPassword genera un token de recuperación.
// Siempre responde OK aunque el email no exista — evita enumeración de usuarios.
func (s *Service) ForgotPassword(ctx context.Context, req input.ForgotPasswordRequest) error {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// Respuesta silenciosa — no revelamos si el email existe
		return nil
	}

	// Eliminamos tokens anteriores no usados del mismo usuario
	_ = s.resetTokenRepo.DeleteExpiredByUserID(ctx, user.ID)

	rawToken, err := generateSecureToken(32)
	if err != nil {
		s.log.Error("error generando token de recuperación", zap.Error(err))
		return apperrors.ErrInternal
	}

	resetToken := &domain.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashSHA256(rawToken),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Used:      false,
		CreatedAt: time.Now(),
	}

	if err := s.resetTokenRepo.Save(ctx, resetToken); err != nil {
		s.log.Error("error guardando token de recuperación", zap.Error(err))
		return apperrors.ErrInternal
	}

	s.log.Audit("token de recuperación generado",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)

	// TODO: enviar rawToken por correo electrónico (adaptador de email pendiente)
	fmt.Printf("[DEV] Token de recuperación para %s: %s\n", user.Email, rawToken)

	return nil
}

// ─── ResetPassword ────────────────────────────────────────────────────────────

// ResetPassword valida el token y actualiza la contraseña del usuario.
func (s *Service) ResetPassword(ctx context.Context, req input.ResetPasswordRequest) error {
	tokenHash := hashSHA256(req.Token)

	stored, err := s.resetTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return apperrors.ErrResetTokenInvalid
	}

	if stored.Used || stored.IsExpired() {
		return apperrors.ErrResetTokenInvalid
	}

	newHash, err := s.hashPassword(req.NewPassword)
	if err != nil {
		s.log.Error("error hasheando nueva contraseña", zap.Error(err))
		return apperrors.ErrInternal
	}

	user, err := s.userRepo.FindByID(ctx, stored.UserID)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	user.PasswordHash = newHash
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Save(ctx, user); err != nil {
		s.log.Error("error actualizando contraseña", zap.Error(err))
		return apperrors.ErrInternal
	}

	_ = s.resetTokenRepo.MarkAsUsed(ctx, stored.ID)

	// Invalidamos todos los refresh tokens activos del usuario por seguridad
	_ = s.refreshTokenRepo.DeleteByUserID(ctx, user.ID)

	s.log.Audit("contraseña restablecida",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)

	return nil
}

// ─── Helpers internos ─────────────────────────────────────────────────────────

// buildAuthResponse genera el access token, persiste el refresh token y construye la respuesta.
func (s *Service) buildAuthResponse(ctx context.Context, user *domain.User) (*input.AuthResponse, error) {
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	rawRefresh, err := generateSecureToken(64)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashSHA256(rawRefresh),
		ExpiresAt: time.Now().Add(s.jwt.refreshExpiry),
		CreatedAt: time.Now(),
	}

	if err := s.refreshTokenRepo.Save(ctx, refreshToken); err != nil {
		s.log.Error("error guardando refresh token", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return &input.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwt.accessExpiry.Seconds()),
	}, nil
}

// generateAccessToken firma un JWT con los datos del usuario.
func (s *Service) generateAccessToken(user *domain.User) (string, error) {
	claims := jwtClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwt.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "curn-networking",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwt.secret))
}

// hashPassword genera un hash argon2id con salt aleatorio.
// Formato de salida: $argon2id$v=19$m=65536,t=3,p=2$<salt_b64>$<hash_b64>
func (s *Service) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		s.argon2.iterations,
		s.argon2.memory,
		s.argon2.parallelism,
		s.argon2.keyLength,
	)

	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		s.argon2.memory, s.argon2.iterations, s.argon2.parallelism,
		saltB64, hashB64,
	)

	return encoded, nil
}

// verifyPassword compara la contraseña plana contra el hash almacenado.
func (s *Service) verifyPassword(password, encodedHash string) error {
	salt, expectedHash, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		return err
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		s.argon2.iterations,
		s.argon2.memory,
		s.argon2.parallelism,
		s.argon2.keyLength,
	)

	if !constantTimeEqual(actualHash, expectedHash) {
		return fmt.Errorf("contraseña incorrecta")
	}

	return nil
}

// decodeArgon2Hash extrae el salt y el hash del string codificado.
func decodeArgon2Hash(encoded string) (salt, hash []byte, err error) {
	var version int
	var memory, iterations uint32
	var parallelism uint8
	var saltB64, hashB64 string

	_, err = fmt.Sscanf(
		encoded,
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s",
		&version, &memory, &iterations, &parallelism, &saltB64,
	)

	// Separamos manualmente el salt y el hash por el último '$'
	parts := splitLast(encoded, "$")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("formato de hash inválido")
	}
	hashB64 = parts[1]
	saltB64 = splitLast(parts[0], "$")[1]

	salt, err = base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, nil, err
	}

	hash, err = base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return nil, nil, err
	}

	return salt, hash, nil
}

// constantTimeEqual compara dos slices en tiempo constante para evitar timing attacks.
func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// generateSecureToken genera un token aleatorio criptográficamente seguro.
func generateSecureToken(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// hashSHA256 devuelve el hash SHA-256 en hex del string dado.
// Se usa para almacenar tokens en BD sin guardarlos en texto plano.
func hashSHA256(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// splitLast divide el string por el último separador encontrado.
func splitLast(s, sep string) []string {
	idx := len(s) - 1
	for idx >= 0 {
		if string(s[idx]) == sep {
			return []string{s[:idx], s[idx+1:]}
		}
		idx--
	}
	return []string{s}
}
