package auth

import (
	// Hors cours : crypto/rand génère des valeurs aléatoires impossibles à deviner,
	// contrairement à math/rand. Indispensable pour des codes de sécurité.
	"crypto/rand"
)

// generateCode retourne un code aléatoire de 8 caractères (lettres A-Z et chiffres 2-7),
// par exemple "K7DZF4QX". Il sert à confirmer une inscription ou à réinitialiser un mot de passe.
func generateCode() string {
	return rand.Text()[:8]
}

// generateToken retourne un jeton de session aléatoire de 26 caractères.
func generateToken() string {
	return rand.Text()
}
