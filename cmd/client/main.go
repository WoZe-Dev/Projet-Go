package main

import (
	"errors"
	"os"

	"leboncoin/internal/client"

	// Hors cours : huh (Charmbracelet, imposé par le sujet) affiche des formulaires dans le terminal.
	"charm.land/huh/v2"
	// Hors cours : lipgloss (Charmbracelet) met le texte en forme (couleurs).
	"charm.land/lipgloss/v2"
)

var (
	logoStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EC5A13")) // orange leboncoin
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))                 // vert
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))                  // rouge
)

// Hors cours : huh.Option[string] utilise les generics : [string] est le type de la valeur choisie.
var (
	visitorOptions = []huh.Option[string]{
		huh.NewOption("Se connecter", "login"),
		huh.NewOption("Créer un compte", "register"),
		huh.NewOption("Confirmer mon inscription", "confirm"),
		huh.NewOption("Mot de passe oublié", "forgot"),
		huh.NewOption("Quitter", "quit"),
	}
	memberOptions = []huh.Option[string]{
		huh.NewOption("Se déconnecter", "logout"),
		huh.NewOption("Quitter", "quit"),
	}
)

func main() {
	api := client.New(os.Getenv("SERVER_URL"))
	connectedEmail := ""

	lipgloss.Println(logoStyle.Render("leboncoin"))

	for {
		var choice string
		var err error

		if connectedEmail == "" {
			choice, err = choose("Bienvenue sur leboncoin", visitorOptions)
		} else {
			choice, err = choose("Connecté en tant que "+connectedEmail, memberOptions)
		}

		if err != nil {
			// Ctrl+C n'est pas une vraie erreur : on quitte simplement
			if !errors.Is(err, huh.ErrUserAborted) {
				showError(err)
			}
			return
		}

		if choice == "quit" {
			return
		}

		switch choice {
		case "login":
			connectedEmail = login(api)
		case "register":
			register(api)
		case "confirm":
			confirm(api, "")
		case "forgot":
			forgotPassword(api)
		case "logout":
			logout(api)
			connectedEmail = ""
		}
	}
}

// choose affiche un menu et retourne la valeur de l'option choisie.
func choose(title string, options []huh.Option[string]) (string, error) {
	var choice string

	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title(title).Options(options...).Value(&choice),
	)).Run()

	return choice, err
}

// login demande l'email et le mot de passe. Retourne l'email si la connexion réussit, "" sinon.
func login(api *client.Client) string {
	var email, password string

	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Email").Value(&email),
		huh.NewInput().Title("Mot de passe").EchoMode(huh.EchoModePassword).Value(&password),
	)).Run()
	if err != nil {
		return ""
	}

	if err := api.Login(email, password); err != nil {
		showError(err)
		return ""
	}

	showSuccess("Connexion réussie")

	return email
}

// register crée un compte puis enchaîne sur la saisie du code de confirmation.
func register(api *client.Client) {
	var email, password string

	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Email").Value(&email),
		huh.NewInput().Title("Mot de passe").Description("8 caractères minimum").EchoMode(huh.EchoModePassword).Value(&password),
	)).Run()
	if err != nil {
		return
	}

	message, err := api.Register(email, password)
	if err != nil {
		showError(err)
		return
	}

	showSuccess(message)
	confirm(api, email)
}

// confirm demande le code reçu par email. L'email est pré-rempli s'il est déjà connu.
func confirm(api *client.Client, email string) {
	var code string

	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Email").Value(&email),
		huh.NewInput().Title("Code de confirmation").Description("Reçu par email").Value(&code),
	)).Run()
	if err != nil {
		return
	}

	message, err := api.Confirm(email, code)
	if err != nil {
		showError(err)
		return
	}

	showSuccess(message)
}

// forgotPassword demande l'envoi d'un code par email puis enchaîne sur le nouveau mot de passe.
func forgotPassword(api *client.Client) {
	var email string

	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Email du compte").Value(&email),
	)).Run()
	if err != nil {
		return
	}

	message, err := api.ForgotPassword(email)
	if err != nil {
		showError(err)
		return
	}

	showSuccess(message)
	resetPassword(api, email)
}

// resetPassword change le mot de passe avec le code reçu par email.
func resetPassword(api *client.Client, email string) {
	var code, password string

	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Email").Value(&email),
		huh.NewInput().Title("Code de réinitialisation").Description("Reçu par email, valable 15 minutes").Value(&code),
		huh.NewInput().Title("Nouveau mot de passe").Description("8 caractères minimum").EchoMode(huh.EchoModePassword).Value(&password),
	)).Run()
	if err != nil {
		return
	}

	message, err := api.ResetPassword(email, code, password)
	if err != nil {
		showError(err)
		return
	}

	showSuccess(message)
}

// logout ferme la session sur le serveur.
func logout(api *client.Client) {
	message, err := api.Logout()
	if err != nil {
		showError(err)
		return
	}

	showSuccess(message)
}

// showSuccess affiche un message en vert.
func showSuccess(message string) {
	lipgloss.Println(successStyle.Render("✔ " + message))
}

// showError affiche une erreur en rouge.
func showError(err error) {
	lipgloss.Println(errorStyle.Render("✘ " + err.Error()))
}
