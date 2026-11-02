CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    -- Code envoyé par email à l'inscription, vidé une fois le compte confirmé
    confirmation_code TEXT UNIQUE,
    -- Code envoyé par email pour réinitialiser le mot de passe
    reset_code TEXT,
    reset_code_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Une session est créée à chaque connexion. Le client envoie le token
-- dans le header Authorization pour prouver qu'il est connecté.
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);
