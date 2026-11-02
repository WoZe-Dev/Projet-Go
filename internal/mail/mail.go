// Package mail envoie les emails de l'application.
package mail

import (
	"fmt"
	// Hors cours : net/smtp (bibliothèque standard) envoie un email à un serveur SMTP.
	"net/smtp"
)

// Mailer envoie des emails via un serveur SMTP (Mailpit dans Docker).
type Mailer struct {
	Address string // exemple : "mailpit:1025"
	From    string
}

// Send envoie un email texte à un destinataire.
func (mailer Mailer) Send(to, subject, body string) error {
	message := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		mailer.From, to, subject, body,
	)

	// nil : Mailpit n'a pas besoin d'identifiant ni de mot de passe
	err := smtp.SendMail(mailer.Address, nil, mailer.From, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("envoi de l'email à %s : %w", to, err)
	}

	return nil
}
