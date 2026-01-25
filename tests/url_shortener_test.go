package tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"url-shortener/internal/database"
	"url-shortener/internal/middleware"
	"url-shortener/internal/models"
	"url-shortener/internal/services"
)

func setupTestDB(t *testing.T) *database.DB {
	os.Setenv("DB_PATH", ":memory:")
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("database init failed: %v", err)
	}
	return db
}

func TestURLServiceShortenURL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := services.NewURLService(db)

	req := models.ShortenURLRequest{
		OriginalURL: "https://example.com/test",
	}

	result, err := svc.ShortenURL(req, "127.0.0.1")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if len(result.ShortCode) != 6 {
		t.Errorf("short code length = %d, want 6", len(result.ShortCode))
	}

	if result.OriginalURL != req.OriginalURL {
		t.Errorf("original URL = %s, want %s", result.OriginalURL, req.OriginalURL)
	}
}

func TestURLServiceCustomCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := services.NewURLService(db)

	req := models.ShortenURLRequest{
		OriginalURL: "https://example.com/custom",
		CustomCode:  "mycode",
	}

	result, err := svc.ShortenURL(req, "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ShortCode != "mycode" {
		t.Errorf("short code = %s, want mycode", result.ShortCode)
	}

	if !result.IsCustom {
		t.Error("IsCustom should be true for custom codes")
	}
}

func TestURLServiceDuplicateCustomCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := services.NewURLService(db)

	req := models.ShortenURLRequest{
		OriginalURL: "https://example.com/first",
		CustomCode:  "duplicate",
	}

	_, err := svc.ShortenURL(req, "127.0.0.1")
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	req.OriginalURL = "https://example.com/second"
	_, err = svc.ShortenURL(req, "127.0.0.1")
	if err == nil {
		t.Error("expected error for duplicate custom code")
	}
}

func TestURLServiceGetOriginalURL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := services.NewURLService(db)

	req := models.ShortenURLRequest{
		OriginalURL: "https://example.com/retrieve",
		CustomCode:  "retrieve",
	}

	created, err := svc.ShortenURL(req, "127.0.0.1")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	retrieved, err := svc.GetOriginalURL(created.ShortCode)
	if err != nil {
		t.Fatalf("retrieval failed: %v", err)
	}

	if retrieved.OriginalURL != created.OriginalURL {
		t.Errorf("got %s, want %s", retrieved.OriginalURL, created.OriginalURL)
	}
}

func TestURLServiceExpiration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := services.NewURLService(db)

	past := time.Now().Add(-1 * time.Hour)
	req := models.ShortenURLRequest{
		OriginalURL: "https://example.com/expired",
		CustomCode:  "expired",
		ExpiresAt:   &past,
	}

	created, err := svc.ShortenURL(req, "127.0.0.1")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, err = svc.GetOriginalURL(created.ShortCode)
	if err == nil {
		t.Error("expected error for expired URL")
	}
}

func TestURLServiceIncrementClickCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := services.NewURLService(db)

	req := models.ShortenURLRequest{
		OriginalURL: "https://example.com/clicks",
		CustomCode:  "clicks",
	}

	created, _ := svc.ShortenURL(req, "127.0.0.1")

	err := svc.IncrementClickCount(created.ShortCode)
	if err != nil {
		t.Errorf("increment failed: %v", err)
	}

	updated, _ := svc.GetOriginalURL(created.ShortCode)
	if updated.ClickCount != 1 {
		t.Errorf("click count = %d, want 1", updated.ClickCount)
	}
}

func TestURLServiceGetUserURLs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := services.NewURLService(db)
	userIP := "192.168.1.1"

	for i := 0; i < 3; i++ {
		req := models.ShortenURLRequest{
			OriginalURL: "https://example.com/user",
		}
		_, err := svc.ShortenURL(req, userIP)
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}
	}

	urls, err := svc.GetUserURLs(userIP)
	if err != nil {
		t.Fatalf("get user urls failed: %v", err)
	}

	if len(urls) != 3 {
		t.Errorf("got %d URLs, want 3", len(urls))
	}
}

func TestAnalyticsServiceRecordClick(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	urlSvc := services.NewURLService(db)
	analyticsSvc := services.NewAnalyticsService(db)

	req := models.ShortenURLRequest{
		OriginalURL: "https://example.com/analytics",
		CustomCode:  "analytics",
	}
	created, _ := urlSvc.ShortenURL(req, "127.0.0.1")

	click := models.Click{
		URLShortCode: created.ShortCode,
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test/1.0",
		ClickedAt:    time.Now(),
	}

	err := analyticsSvc.RecordClick(click)
	if err != nil {
		t.Errorf("record click failed: %v", err)
	}
}

func TestAnalyticsServiceGetAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	urlSvc := services.NewURLService(db)
	analyticsSvc := services.NewAnalyticsService(db)

	req := models.ShortenURLRequest{
		OriginalURL: "https://example.com/stats",
		CustomCode:  "stats",
	}
	created, _ := urlSvc.ShortenURL(req, "127.0.0.1")

	for i := 0; i < 5; i++ {
		click := models.Click{
			URLShortCode: created.ShortCode,
			IPAddress:    "127.0.0.1",
			UserAgent:    "Test/1.0",
			Country:      "US",
			ClickedAt:    time.Now(),
		}
		analyticsSvc.RecordClick(click)
	}

	analytics, err := analyticsSvc.GetAnalytics(created.ShortCode)
	if err != nil {
		t.Fatalf("get analytics failed: %v", err)
	}

	if analytics.TotalClicks != 5 {
		t.Errorf("total clicks = %d, want 5", analytics.TotalClicks)
	}

	if analytics.UniqueVisitors != 1 {
		t.Errorf("unique visitors = %d, want 1", analytics.UniqueVisitors)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rateLimited := middleware.RateLimitMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	successCount := 0
	for i := 0; i < 15; i++ {
		rr := httptest.NewRecorder()
		rateLimited.ServeHTTP(rr, req)

		if rr.Code == http.StatusOK {
			successCount++
		}
	}

	if successCount > 10 {
		t.Errorf("rate limit not enforced: %d requests succeeded", successCount)
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := middleware.CORSMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	corsHandler.ServeHTTP(rr, req)

	origin := rr.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("CORS origin = %s, want *", origin)
	}

	methods := rr.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(methods, "GET") {
		t.Error("CORS methods should include GET")
	}
}
