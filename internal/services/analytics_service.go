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

	clicksByCountry, err := s.getClicksByCountry(shortCode)
	if err != nil {
		return nil, err
	}

	clicksByDay, err := s.getClicksByDay(shortCode, 30)
	if err != nil {
		return nil, err
	}

	recentClicks, err := s.getRecentClicks(shortCode, 10)
	if err != nil {
		return nil, err
	}

	analytics := &models.Analytics{
		URL:             url,
		TotalClicks:     totalClicks,
		UniqueVisitors:  uniqueVisitors,
		ClicksByCountry: clicksByCountry,
		ClicksByDay:     clicksByDay,
		RecentClicks:    recentClicks,
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

func (s *AnalyticsService) getClicksByCountry(shortCode string) (map[string]int, error) {
	query := `
		SELECT country, COUNT(*) as count 
		FROM clicks 
		WHERE url_short_code = ? AND country != ''
		GROUP BY country 
		ORDER BY count DESC
		LIMIT 10
	`

	rows, err := s.db.Query(query, shortCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var country string
		var count int
		if err := rows.Scan(&country, &count); err != nil {
			return nil, err
		}
		result[country] = count
	}

	return result, nil
}

func (s *AnalyticsService) getClicksByDay(shortCode string, days int) ([]models.DailyClicks, error) {
	query := `
		SELECT DATE(clicked_at) as date, COUNT(*) as count
		FROM clicks 
		WHERE url_short_code = ? AND clicked_at >= datetime('now', '-' || ? || ' days')
		GROUP BY DATE(clicked_at)
		ORDER BY date DESC
	`

	rows, err := s.db.Query(query, shortCode, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.DailyClicks
	for rows.Next() {
		var daily models.DailyClicks
		if err := rows.Scan(&daily.Date, &daily.Clicks); err != nil {
			return nil, err
		}
		result = append(result, daily)
	}

	return result, nil
}


func (s *AnalyticsService) getRecentClicks(shortCode string, limit int) ([]models.Click, error) {
	query := `
		SELECT id, url_short_code, ip_address, user_agent, referer, country, city, clicked_at
		FROM clicks 
		WHERE url_short_code = ?
		ORDER BY clicked_at DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, shortCode, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clicks []models.Click
	for rows.Next() {
		var click models.Click
		err := rows.Scan(&click.ID, &click.URLShortCode, &click.IPAddress,
			&click.UserAgent, &click.Referer, &click.Country, &click.City, &click.ClickedAt)
		if err != nil {
			return nil, err
		}
		clicks = append(clicks, click)
	}

	return clicks, nil
}


func (s *AnalyticsService) GetLocationFromIP(ip string) (country, city string) {
	if ip == "127.0.0.1" || ip == "::1" {
		return "Local", "Local"
	}
	return "Unknown", "Unknown"
}
