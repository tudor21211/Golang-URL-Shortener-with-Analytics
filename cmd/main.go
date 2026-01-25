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
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	urlService := services.NewURLService(db)
	analyticsService := services.NewAnalyticsService(db)
	urlHandler := handlers.NewURLHandler(urlService, analyticsService)
	router := mux.NewRouter()

	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.RateLimitMiddleware)
	api.Use(middleware.CORSMiddleware)

	api.HandleFunc("/shorten", urlHandler.ShortenURL).Methods("POST")
	api.HandleFunc("/analytics/{shortCode}", urlHandler.GetAnalytics).Methods("GET")
	api.HandleFunc("/urls", urlHandler.GetUserURLs).Methods("GET")

	// Redirect route (no rate limiting for redirects)
	router.HandleFunc("/{shortCode}", urlHandler.RedirectURL).Methods("GET")

	// Web interface routes
	router.HandleFunc("/", urlHandler.HomePage).Methods("GET")
	router.HandleFunc("/dashboard", urlHandler.Dashboard).Methods("GET")
  
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	router.HandleFunc("/{shortCode}", urlHandler.RedirectURL).Methods("GET")

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
