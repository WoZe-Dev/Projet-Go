// Package client appelle l'API HTTP du serveur depuis les applications en ligne de commande.
package client

import (
	// Hors cours : bytes.NewReader transforme des octets en corps de requête HTTP.
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"leboncoin/internal/api"
)

// Client envoie les requêtes au serveur et garde le token de session après la connexion.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// New crée un Client pour le serveur à l'adresse baseURL (exemple : "http://server:8080").
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		// Hors cours : http.Client envoie des requêtes HTTP (le cours ne montre que le côté serveur).
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Register crée un compte et retourne le message du serveur.
func (client *Client) Register(email, password string) (string, error) {
	var response api.MessageResponse
	err := client.post("/api/auth/register", api.Credentials{Email: email, Password: password}, &response)

	return response.Message, err
}

// Confirm confirme l'inscription avec le code reçu par email.
func (client *Client) Confirm(email, code string) (string, error) {
	var response api.MessageResponse
	err := client.post("/api/auth/confirm", api.ConfirmRequest{Email: email, Code: code}, &response)

	return response.Message, err
}

// Login connecte l'utilisateur et garde le token de session.
func (client *Client) Login(email, password string) error {
	var response api.LoginResponse
	if err := client.post("/api/auth/login", api.Credentials{Email: email, Password: password}, &response); err != nil {
		return err
	}

	client.token = response.Token

	return nil
}

// Logout supprime la session sur le serveur et oublie le token.
func (client *Client) Logout() (string, error) {
	var response api.MessageResponse
	err := client.post("/api/auth/logout", nil, &response)
	client.token = ""

	return response.Message, err
}

// ForgotPassword demande l'envoi d'un code de réinitialisation par email.
func (client *Client) ForgotPassword(email string) (string, error) {
	var response api.MessageResponse
	err := client.post("/api/auth/forgot-password", api.ForgotPasswordRequest{Email: email}, &response)

	return response.Message, err
}

// ResetPassword change le mot de passe avec le code reçu par email.
func (client *Client) ResetPassword(email, code, newPassword string) (string, error) {
	var response api.MessageResponse
	request := api.ResetPasswordRequest{Email: email, Code: code, NewPassword: newPassword}
	err := client.post("/api/auth/reset-password", request, &response)

	return response.Message, err
}

// post envoie data en JSON au chemin path, puis lit la réponse JSON dans result.
// Si le serveur répond avec un code d'erreur (400 et plus), son message devient l'erreur retournée.
func (client *Client) post(path string, data any, result any) error {
	// Hors cours : json.Marshal transforme une valeur Go en JSON (comme json.NewEncoder, mais en mémoire).
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("préparation de la requête : %w", err)
	}

	// Hors cours : http.NewRequest prépare une requête pour pouvoir ajouter des headers.
	request, err := http.NewRequest(http.MethodPost, client.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("préparation de la requête : %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if client.token != "" {
		request.Header.Set("Authorization", "Bearer "+client.token)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("serveur injoignable : %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		var message api.MessageResponse
		if err := json.NewDecoder(response.Body).Decode(&message); err != nil {
			return fmt.Errorf("réponse invalide du serveur (code %d)", response.StatusCode)
		}

		return errors.New(message.Message)
	}

	if err := json.NewDecoder(response.Body).Decode(result); err != nil {
		return fmt.Errorf("lecture de la réponse : %w", err)
	}

	return nil
}
