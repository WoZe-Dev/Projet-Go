# leboncoin - plateforme d'achat en ligne de commande

Projet Go : une plateforme d'achat sur le thème de leboncoin.

## Lancer le projet

1. Démarrer la base de données, le serveur d'emails et le serveur :
   ```bash
   docker compose up -d
   ```
2. Lancer l'application client :
   ```bash
   docker compose run --rm client
   ```
3. Lire les emails envoyés (codes de confirmation et de réinitialisation) : http://localhost:8026

| Action | Commande |
|--------|----------|
| Arrêter le projet | `docker compose down` |
| Arrêter et vider la base | `docker compose down -v` |