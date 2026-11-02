package auth

import (
	// Hors cours : database/sql (imposé par le sujet) exécute des requêtes SQL.
	// $1, $2... sont remplacés par les valeurs passées en paramètre, ce qui protège des injections SQL.
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrEmailTaken   = errors.New("email déjà utilisé")
	ErrUserNotFound = errors.New("utilisateur introuvable")
	ErrInvalidCode  = errors.New("code invalide")
)

// User est un utilisateur enregistré en base.
type User struct {
	ID           int
	Email        string
	PasswordHash string
	Confirmed    bool
}

// Store regroupe les requêtes SQL de l'authentification.
type Store struct {
	db *sql.DB
}

// NewStore crée un Store à partir d'une connexion à la base.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateUser enregistre un utilisateur non confirmé avec son code de confirmation.
// Si l'email existe déjà mais n'a jamais été confirmé, le mot de passe et le code sont remplacés :
// l'utilisateur peut se réinscrire s'il a perdu son code.
func (store *Store) CreateUser(email, passwordHash, confirmationCode string) error {
	result, err := store.db.Exec(
		`INSERT INTO users (email, password_hash, confirmation_code)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE
		SET password_hash = EXCLUDED.password_hash, confirmation_code = EXCLUDED.confirmation_code
		WHERE users.confirmed = FALSE`,
		email, passwordHash, confirmationCode,
	)
	if err != nil {
		return fmt.Errorf("création de l'utilisateur : %w", err)
	}

	return checkOneRow(result, ErrEmailTaken)
}

// ConfirmUser confirme le compte si le code correspond à celui envoyé par email.
func (store *Store) ConfirmUser(email, code string) error {
	result, err := store.db.Exec(
		`UPDATE users SET confirmed = TRUE, confirmation_code = NULL
		WHERE email = $1 AND confirmation_code = $2`,
		email, code,
	)
	if err != nil {
		return fmt.Errorf("confirmation de l'utilisateur : %w", err)
	}

	return checkOneRow(result, ErrInvalidCode)
}

// FindUserByEmail retourne l'utilisateur qui possède cet email.
func (store *Store) FindUserByEmail(email string) (User, error) {
	var user User

	err := store.db.QueryRow(
		`SELECT id, email, password_hash, confirmed FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Confirmed)

	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("recherche de l'utilisateur : %w", err)
	}

	return user, nil
}

// CreateSession enregistre un token de session valable 24 heures.
func (store *Store) CreateSession(userID int, token string) error {
	_, err := store.db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, NOW() + INTERVAL '24 hours')`,
		token, userID,
	)
	if err != nil {
		return fmt.Errorf("création de la session : %w", err)
	}

	return nil
}

// DeleteSession supprime une session (déconnexion).
func (store *Store) DeleteSession(token string) error {
	_, err := store.db.Exec(`DELETE FROM sessions WHERE token = $1`, token)
	if err != nil {
		return fmt.Errorf("suppression de la session : %w", err)
	}

	return nil
}

// SetResetCode enregistre un code de réinitialisation valable 15 minutes.
func (store *Store) SetResetCode(email, code string) error {
	result, err := store.db.Exec(
		`UPDATE users SET reset_code = $2, reset_code_expires_at = NOW() + INTERVAL '15 minutes'
		WHERE email = $1`,
		email, code,
	)
	if err != nil {
		return fmt.Errorf("enregistrement du code de réinitialisation : %w", err)
	}

	return checkOneRow(result, ErrUserNotFound)
}

// ResetPassword remplace le mot de passe si le code est correct et pas expiré,
// puis supprime les sessions ouvertes avec l'ancien mot de passe.
func (store *Store) ResetPassword(email, code, passwordHash string) error {
	var userID int

	err := store.db.QueryRow(
		`UPDATE users SET password_hash = $3, reset_code = NULL, reset_code_expires_at = NULL
		WHERE email = $1 AND reset_code = $2 AND reset_code_expires_at > NOW()
		RETURNING id`,
		email, code, passwordHash,
	).Scan(&userID)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidCode
	}
	if err != nil {
		return fmt.Errorf("réinitialisation du mot de passe : %w", err)
	}

	_, err = store.db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("suppression des sessions : %w", err)
	}

	return nil
}

// checkOneRow retourne notFound si la requête n'a modifié aucune ligne.
func checkOneRow(result sql.Result, notFound error) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("lecture du résultat : %w", err)
	}

	if rows == 0 {
		return notFound
	}

	return nil
}
