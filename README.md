# Golang-URL-Shortener-with-Analytics

A simple URL shortener service written in Go. This project uses SQLite as its database (no external database required).

Quick start (SQLite):

1. Install dependencies:

```powershell
cd "Golang-URL-Shortener-with-Analytics"
go mod tidy
```

2. Run the application:

```powershell
go run .\cmd\main.go
```

3. Open the web UI: http://localhost:8080

Notes:
- The app uses a file-based SQLite database (`url_shortener.db`) created in the project root.
- No PostgreSQL or other external DB is required.