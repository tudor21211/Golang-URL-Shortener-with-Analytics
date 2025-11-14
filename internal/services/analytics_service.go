package services

import (
	"database/sql"
	"fmt"

	"url-shortener/internal/database"
	"url-shortener/internal/models"
)

type AnalyticsService struct {
	db *database.DB
}

func NewAnalyticsService(db *database.DB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

func (s *AnalyticsService) RecordClick(click models.Click) error {
	query := `
		INSERT INTO clicks (url_short_code, ip_address, user_agent, referer, country, city, clicked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, click.URLShortCode, click.IPAddress, click.UserAgent,
		click.Referer, click.Country, click.City, click.ClickedAt)

	return err
}

func (s *AnalyticsService) GetAnalytics(shortCode string) (*models.Analytics, error) {
	url, err := s.getURLByShortCode(shortCode)
	if err != nil {
		return nil, err
	}

	totalClicks, err := s.getTotalClicks(shortCode)
	if err != nil {
		return nil, err
	}

	uniqueVisitors, err := s.getUniqueVisitors(shortCode)
	if err != nil {
		return nil, err
	}


	analytics := &models.Analytics{
		URL:             url,
		TotalClicks:     totalClicks,
		UniqueVisitors:  uniqueVisitors,
	}

	return analytics, nil
}

func (s *AnalyticsService) getURLByShortCode(shortCode string) (*models.URL, error) {
	query := `
		SELECT id, short_code, original_url, created_at, expires_at, click_count, user_ip, is_custom
		FROM urls 
		WHERE short_code = ?
	`

	url := &models.URL{}
	err := s.db.QueryRow(query, shortCode).Scan(
		&url.ID, &url.ShortCode, &url.OriginalURL, &url.CreatedAt,
		&url.ExpiresAt, &url.ClickCount, &url.UserIP, &url.IsCustom,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("short code not found")
		}
		return nil, err
	}

	return url, nil
}

func (s *AnalyticsService) getTotalClicks(shortCode string) (int, error) {
	query := `SELECT COUNT(*) FROM clicks WHERE url_short_code = ?`
	var count int
	err := s.db.QueryRow(query, shortCode).Scan(&count)
	return count, err
}

func (s *AnalyticsService) getUniqueVisitors(shortCode string) (int, error) {
	query := `SELECT COUNT(DISTINCT ip_address) FROM clicks WHERE url_short_code = ?`
	var count int
	err := s.db.QueryRow(query, shortCode).Scan(&count)
	return count, err
}


func (s *AnalyticsService) GetLocationFromIP(ip string) (country, city string) {
	if ip == "127.0.0.1" || ip == "::1" {
		return "Local", "Local"
	}
	return "Unknown", "Unknown"
}
