-- Seed de programas académicos y roles base para registro por nombre.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'roles') THEN
        INSERT INTO roles (name)
        VALUES ('estudiante'), ('egresado'), ('administrativo'), ('admin')
        ON CONFLICT (name) DO NOTHING;
    END IF;
END$$;
DO $$
DECLARE
    has_faculty BOOLEAN;
    has_level BOOLEAN;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'programs') THEN
        RAISE NOTICE 'La tabla programs no existe; se omite seed de programas.';
        RETURN;
    END IF;
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'programs' AND column_name = 'faculty'
    ) INTO has_faculty;
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'programs' AND column_name = 'level'
    ) INTO has_level;
    IF has_faculty AND has_level THEN
        INSERT INTO programs (name, faculty, level)
        SELECT s.name, s.faculty, s.level::program_level
        FROM (VALUES
            ('Administración de Empresas', 'Ciencias Contables y Administrativas', 'profesional'),
            ('Contaduría Pública', 'Ciencias Contables y Administrativas', 'profesional'),
            ('Tecnología en Gestión Contable y Financiera', 'Ciencias Contables y Administrativas', 'tecnologo'),
            ('Enfermería', 'Ciencias de la Salud', 'profesional'),
            ('Instrumentación Quirúrgica', 'Ciencias de la Salud', 'tecnologo'),
            ('Medicina', 'Ciencias de la Salud', 'profesional'),
            ('Odontología', 'Ciencias de la Salud', 'profesional'),
            ('Derecho', 'Ciencias Sociales y Humanas', 'profesional'),
            ('Trabajo Social', 'Ciencias Sociales y Humanas', 'profesional'),
            ('Licenciatura en Educación Infantil', 'Ciencias Sociales y Humanas', 'profesional'),
            ('Ingeniería de Sistemas', 'Ingeniería', 'profesional'),
            ('Tecnología en Desarrollo de Sistemas de Información y de Software', 'Ingeniería', 'tecnologo')
        ) AS s(name, faculty, level)
        WHERE NOT EXISTS (
            SELECT 1 FROM programs p WHERE LOWER(p.name) = LOWER(s.name)
        );
    ELSIF has_faculty THEN
        INSERT INTO programs (name, faculty)
        SELECT s.name, s.faculty
        FROM (VALUES
            ('Administración de Empresas', 'Ciencias Contables y Administrativas'),
            ('Contaduría Pública', 'Ciencias Contables y Administrativas'),
            ('Tecnología en Gestión Contable y Financiera', 'Ciencias Contables y Administrativas'),
            ('Enfermería', 'Ciencias de la Salud'),
            ('Instrumentación Quirúrgica', 'Ciencias de la Salud'),
            ('Medicina', 'Ciencias de la Salud'),
            ('Odontología', 'Ciencias de la Salud'),
            ('Derecho', 'Ciencias Sociales y Humanas'),
            ('Trabajo Social', 'Ciencias Sociales y Humanas'),
            ('Licenciatura en Educación Infantil', 'Ciencias Sociales y Humanas'),
            ('Ingeniería de Sistemas', 'Ingeniería'),
            ('Tecnología en Desarrollo de Sistemas de Información y de Software', 'Ingeniería')
        ) AS s(name, faculty)
        WHERE NOT EXISTS (
            SELECT 1 FROM programs p WHERE LOWER(p.name) = LOWER(s.name)
        );
    ELSIF has_level THEN
        INSERT INTO programs (name, level)
        SELECT s.name, s.level::program_level
        FROM (VALUES
            ('Administración de Empresas', 'profesional'),
            ('Contaduría Pública', 'profesional'),
            ('Tecnología en Gestión Contable y Financiera', 'tecnologo'),
            ('Enfermería', 'profesional'),
            ('Instrumentación Quirúrgica', 'tecnologo'),
            ('Medicina', 'profesional'),
            ('Odontología', 'profesional'),
            ('Derecho', 'profesional'),
            ('Trabajo Social', 'profesional'),
            ('Licenciatura en Educación Infantil', 'profesional'),
            ('Ingeniería de Sistemas', 'profesional'),
            ('Tecnología en Desarrollo de Sistemas de Información y de Software', 'tecnologo')
        ) AS s(name, level)
        WHERE NOT EXISTS (
            SELECT 1 FROM programs p WHERE LOWER(p.name) = LOWER(s.name)
        );
    ELSE
        INSERT INTO programs (name)
        SELECT s.name
        FROM (VALUES
            ('Administración de Empresas'),
            ('Contaduría Pública'),
            ('Tecnología en Gestión Contable y Financiera'),
            ('Enfermería'),
            ('Instrumentación Quirúrgica'),
            ('Medicina'),
            ('Odontología'),
            ('Derecho'),
            ('Trabajo Social'),
            ('Licenciatura en Educación Infantil'),
            ('Ingeniería de Sistemas'),
            ('Tecnología en Desarrollo de Sistemas de Información y de Software')
        ) AS s(name)
        WHERE NOT EXISTS (
            SELECT 1 FROM programs p WHERE LOWER(p.name) = LOWER(s.name)
        );
    END IF;
END$$;
