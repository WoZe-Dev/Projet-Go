// Package api contient les données échangées en JSON entre le serveur et les clients.
package api

// Credentials est envoyé pour l'inscription et la connexion.
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ConfirmRequest est envoyé pour confirmer une inscription avec le code reçu par email.
type ConfirmRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// ForgotPasswordRequest demande l'envoi d'un code de réinitialisation par email.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest change le mot de passe avec le code reçu par email.
type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

// LoginResponse est renvoyé après une connexion réussie.
type LoginResponse struct {
	Token string `json:"token"`
}

// MessageResponse est renvoyé pour un succès ou une erreur.
type MessageResponse struct {
	Message string `json:"message"`
}
