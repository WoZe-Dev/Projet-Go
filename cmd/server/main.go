package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"leboncoin/internal/auth"
	"leboncoin/internal/mail"

	// Hors cours : driver PostgreSQL pour database/sql.
	// Import anonyme (module 8) : on veut seulement son effet de bord, qui enregistre le driver "postgres".
	_ "github.com/lib/pq"
)

func main() {
	// Hors cours : os.Getenv lit une variable d'environnement (définies dans compose.yaml).
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// sql.Open ne se connecte pas encore : Ping vérifie que la base répond
	if err := db.Ping(); err != nil {
		log.Fatalf("connexion à la base impossible : %v", err)
	}

	mailer := mail.Mailer{
		Address: os.Getenv("SMTP_ADDRESS"),
		From:    os.Getenv("MAIL_FROM"),
	}

	mux := http.NewServeMux()
	auth.NewHandler(auth.NewStore(db), mailer).Routes(mux)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	log.Println("Serveur démarré sur :8080")
	log.Fatal(server.ListenAndServe())
}
