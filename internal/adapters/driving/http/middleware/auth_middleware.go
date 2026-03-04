package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// contextKey constants used to store JWT claims in Gin context.
const (
	ContextKeyUserID = "userID"
	ContextKeyEmail  = "email"
)

// jwtClaims mirrors the claims emitted by the auth service.
type jwtClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// AuthRequired returns a Gin middleware that validates the JWT in the
// Authorization header and injects userID + email into the context.
func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "se requiere autenticación",
			})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "formato de autorización inválido, se espera 'Bearer <token>'",
			})
			return
		}

		tokenString := parts[1]

		claims := &jwtClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token inválido o expirado",
			})
			return
		}

		// Inject user data into context for downstream handlers.
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyEmail, claims.Email)

		c.Next()
	}
}
