package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	// Hors cours : strings contient des fonctions pour manipuler du texte
	// (TrimSpace, ToLower, ToUpper, Count, Index, ContainsAny, TrimPrefix).
	"strings"

	"leboncoin/internal/api"
	"leboncoin/internal/mail"

	// Hors cours : bcrypt transforme un mot de passe en empreinte (hash) impossible à inverser.
	// On ne stocke jamais le mot de passe lui-même en base.
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	// bcrypt refuse les mots de passe de plus de 72 octets
	maxPasswordLength = 72
)

// Handler regroupe les routes HTTP de l'authentification.
type Handler struct {
	store  *Store
	mailer mail.Mailer
}

// NewHandler crée le Handler de l'authentification.
func NewHandler(store *Store, mailer mail.Mailer) *Handler {
	return &Handler{store: store, mailer: mailer}
}

// Routes enregistre les routes de l'authentification sur le routeur.
func (handler *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", handler.register)
	mux.HandleFunc("POST /api/auth/confirm", handler.confirm)
	mux.HandleFunc("POST /api/auth/login", handler.login)
	mux.HandleFunc("POST /api/auth/logout", handler.logout)
	mux.HandleFunc("POST /api/auth/forgot-password", handler.forgotPassword)
	mux.HandleFunc("POST /api/auth/reset-password", handler.resetPassword)
}

// register crée un compte non confirmé et envoie le code de confirmation par email.
func (handler *Handler) register(w http.ResponseWriter, r *http.Request) {
	var request api.Credentials
	if err := readJSON(r, &request); err != nil {
		writeMessage(w, http.StatusBadRequest, "Requête invalide")
		return
	}

	email := normalizeEmail(request.Email)
	if !validEmail(email) {
		writeMessage(w, http.StatusBadRequest, "Adresse email invalide")
		return
	}

	if !validPassword(request.Password) {
		writeMessage(w, http.StatusBadRequest, passwordRule())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		internalError(w, fmt.Errorf("hash du mot de passe : %w", err))
		return
	}

	code := generateCode()

	err = handler.store.CreateUser(email, string(hash), code)
	if errors.Is(err, ErrEmailTaken) {
		writeMessage(w, http.StatusConflict, "Un compte existe déjà avec cette adresse email")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	body := fmt.Sprintf("Bonjour,\n\nBienvenue sur leboncoin ! Votre code de confirmation est : %s\n", code)
	if err := handler.mailer.Send(email, "leboncoin - Confirmez votre inscription", body); err != nil {
		internalError(w, err)
		return
	}

	writeMessage(w, http.StatusCreated, "Compte créé. Un code de confirmation a été envoyé par email")
}

// confirm confirme un compte avec le code reçu par email.
func (handler *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	var request api.ConfirmRequest
	if err := readJSON(r, &request); err != nil {
		writeMessage(w, http.StatusBadRequest, "Requête invalide")
		return
	}

	err := handler.store.ConfirmUser(normalizeEmail(request.Email), normalizeCode(request.Code))
	if errors.Is(err, ErrInvalidCode) {
		writeMessage(w, http.StatusBadRequest, "Code de confirmation invalide")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	writeMessage(w, http.StatusOK, "Compte confirmé. Vous pouvez vous connecter")
}

// login vérifie l'email et le mot de passe puis crée une session.
func (handler *Handler) login(w http.ResponseWriter, r *http.Request) {
	var request api.Credentials
	if err := readJSON(r, &request); err != nil {
		writeMessage(w, http.StatusBadRequest, "Requête invalide")
		return
	}

	user, err := handler.store.FindUserByEmail(normalizeEmail(request.Email))
	if errors.Is(err, ErrUserNotFound) {
		// Même message que pour un mauvais mot de passe : on ne révèle pas si l'email existe
		writeMessage(w, http.StatusUnauthorized, "Email ou mot de passe incorrect")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password))
	if err != nil {
		writeMessage(w, http.StatusUnauthorized, "Email ou mot de passe incorrect")
		return
	}

	if !user.Confirmed {
		writeMessage(w, http.StatusForbidden, "Compte non confirmé : entrez le code reçu par email")
		return
	}

	token := generateToken()
	if err := handler.store.CreateSession(user.ID, token); err != nil {
		internalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, api.LoginResponse{Token: token})
}

// logout supprime la session dont le token est dans le header Authorization.
func (handler *Handler) logout(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	if err := handler.store.DeleteSession(token); err != nil {
		internalError(w, err)
		return
	}

	writeMessage(w, http.StatusOK, "Vous êtes déconnecté")
}

// forgotPassword envoie un code de réinitialisation par email (sans être connecté).
func (handler *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var request api.ForgotPasswordRequest
	if err := readJSON(r, &request); err != nil {
		writeMessage(w, http.StatusBadRequest, "Requête invalide")
		return
	}

	email := normalizeEmail(request.Email)
	code := generateCode()

	err := handler.store.SetResetCode(email, code)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		internalError(w, err)
		return
	}

	if err == nil {
		body := fmt.Sprintf("Bonjour,\n\nVotre code de réinitialisation leboncoin est : %s\nIl est valable 15 minutes.\n", code)
		if err := handler.mailer.Send(email, "leboncoin - Réinitialisation de votre mot de passe", body); err != nil {
			internalError(w, err)
			return
		}
	}

	// Même réponse que l'email existe ou non : on ne révèle pas quels emails ont un compte
	writeMessage(w, http.StatusOK, "Si un compte existe avec cette adresse, un code a été envoyé par email")
}

// resetPassword remplace le mot de passe avec le code reçu par email.
func (handler *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var request api.ResetPasswordRequest
	if err := readJSON(r, &request); err != nil {
		writeMessage(w, http.StatusBadRequest, "Requête invalide")
		return
	}

	if !validPassword(request.NewPassword) {
		writeMessage(w, http.StatusBadRequest, passwordRule())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		internalError(w, fmt.Errorf("hash du mot de passe : %w", err))
		return
	}

	err = handler.store.ResetPassword(normalizeEmail(request.Email), normalizeCode(request.Code), string(hash))
	if errors.Is(err, ErrInvalidCode) {
		writeMessage(w, http.StatusBadRequest, "Code invalide ou expiré")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	writeMessage(w, http.StatusOK, "Mot de passe modifié. Vous pouvez vous connecter")
}

// normalizeEmail enlève les espaces autour et met en minuscules : "Alice@Mail.com " devient "alice@mail.com".
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// normalizeCode enlève les espaces autour et met en majuscules, comme les codes générés.
func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// validEmail vérifie qu'il y a un seul @, avec du texte avant et après, et aucun espace.
func validEmail(email string) bool {
	if strings.Count(email, "@") != 1 || strings.ContainsAny(email, " \t\r\n") {
		return false
	}

	at := strings.Index(email, "@")

	return at > 0 && at < len(email)-1
}

// validPassword vérifie la longueur du mot de passe.
func validPassword(password string) bool {
	return len(password) >= minPasswordLength && len(password) <= maxPasswordLength
}

// passwordRule retourne le message qui explique la règle des mots de passe.
func passwordRule() string {
	return fmt.Sprintf("Le mot de passe doit contenir entre %d et %d caractères", minPasswordLength, maxPasswordLength)
}

// readJSON lit le corps JSON de la requête dans destination.
func readJSON(r *http.Request, destination any) error {
	// Hors cours : json.NewDecoder fait l'inverse de json.NewEncoder (module 10) : il lit du JSON.
	return json.NewDecoder(r.Body).Decode(destination)
}

// writeJSON envoie data en JSON avec le code HTTP status.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("écriture de la réponse : %v", err)
	}
}

// writeMessage envoie un message JSON {"message": "..."} avec le code HTTP status.
func writeMessage(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, api.MessageResponse{Message: message})
}

// internalError affiche l'erreur dans les logs du serveur et renvoie un message générique au client.
func internalError(w http.ResponseWriter, err error) {
	log.Printf("erreur interne : %v", err)
	writeMessage(w, http.StatusInternalServerError, "Erreur interne du serveur")
}
