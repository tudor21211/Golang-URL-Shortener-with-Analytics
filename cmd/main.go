package main

import (
	"log"
	"net/http"

	"url-shortener/internal/database"
	"url-shortener/internal/handlers"
	"url-shortener/internal/middleware"
	"url-shortener/internal/services"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	urlService := services.NewURLService(db)
	analyticsService := services.NewAnalyticsService(db)
	urlHandler := handlers.NewURLHandler(urlService, analyticsService)
	router := mux.NewRouter()
}
