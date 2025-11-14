package models

import (
	"time"
)

 
type URL struct {
	ID          int        `json:"id" db:"id"`
	ShortCode   string     `json:"short_code" db:"short_code"`
	OriginalURL string     `json:"original_url" db:"original_url"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	ClickCount  int        `json:"click_count" db:"click_count"`
	UserIP      string     `json:"user_ip" db:"user_ip"`
	IsCustom    bool       `json:"is_custom" db:"is_custom"`
}

type Click struct {
	ID           int       `json:"id" db:"id"`
	URLShortCode string    `json:"url_short_code" db:"url_short_code"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	Referer      string    `json:"referer" db:"referer"`
	Country      string    `json:"country" db:"country"`
	City         string    `json:"city" db:"city"`
	ClickedAt    time.Time `json:"clicked_at" db:"clicked_at"`
}

// ShortenURLRequest represents the request to shorten a URL
type ShortenURLRequest struct {
	OriginalURL string     `json:"original_url" validate:"required,url"`
	CustomCode  string     `json:"custom_code,omitempty" validate:"alphanum,min=3,max=20"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ShortenURLResponse represents the response after shortening a URL
type ShortenURLResponse struct {
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Analytics represents analytics data for a URL
type Analytics struct {
	URL             *URL           `json:"url"`
	TotalClicks     int            `json:"total_clicks"`
	UniqueVisitors  int            `json:"unique_visitors"`
	ClicksByCountry map[string]int `json:"clicks_by_country"`
	ClicksByDay     []DailyClicks  `json:"clicks_by_day"`
	RecentClicks    []Click        `json:"recent_clicks"`
}

// DailyClicks represents clicks grouped by day
type DailyClicks struct {
	Date   string `json:"date"`
	Clicks int    `json:"clicks"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
