package services

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"strings"
	"time"

	"url-shortener/internal/database"
	"url-shortener/internal/models"
)

const (
	shortCodeLength = 6
	charset         = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	maxRetries      = 5
)

type URLService struct {
	db *database.DB
}

func NewURLService(db *database.DB) *URLService {
	return &URLService{db: db}
}

func (s *URLService) ShortenURL(req models.ShortenURLRequest, userIP string) (*models.URL, error) {
	var shortCode string
	var err error

	// If custom code is provided, validate and use it
	if req.CustomCode != "" {
		if err := s.validateCustomCode(req.CustomCode); err != nil {
			return nil, err
		}
		shortCode = req.CustomCode
	} else {
		shortCode, err = s.generateUniqueShortCode()
		if err != nil {
			return nil, err
		}
	}

	// Create URL entry
	url := &models.URL{
		ShortCode:   shortCode,
		OriginalURL: req.OriginalURL,
		CreatedAt:   time.Now(),
		ExpiresAt:   req.ExpiresAt,
		UserIP:      userIP,
		IsCustom:    req.CustomCode != "",
	}

	query := `
		INSERT INTO urls (short_code, original_url, created_at, expires_at, user_ip, is_custom)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query, url.ShortCode, url.OriginalURL, url.CreatedAt,
		url.ExpiresAt, url.UserIP, url.IsCustom)
	if err != nil {
		return nil, fmt.Errorf("failed to save URL: %v", err)
	}

	id, _ := result.LastInsertId()
	url.ID = int(id)

	return url, nil
}

func (s *URLService) GetOriginalURL(shortCode string) (*models.URL, error) {
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

	if url.ExpiresAt != nil && time.Now().After(*url.ExpiresAt) {
		return nil, fmt.Errorf("URL has expired")
	}

	return url, nil
}

func (s *URLService) IncrementClickCount(shortCode string) error {
	query := `UPDATE urls SET click_count = click_count + 1 WHERE short_code = ?`
	_, err := s.db.Exec(query, shortCode)
	return err
}

func (s *URLService) GetUserURLs(userIP string) ([]models.URL, error) {
	query := `
		SELECT id, short_code, original_url, created_at, expires_at, click_count, user_ip, is_custom
		FROM urls 
		WHERE user_ip = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userIP)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []models.URL
	for rows.Next() {
		var url models.URL
		err := rows.Scan(
			&url.ID, &url.ShortCode, &url.OriginalURL, &url.CreatedAt,
			&url.ExpiresAt, &url.ClickCount, &url.UserIP, &url.IsCustom,
		)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}

	return urls, nil
}

func (s *URLService) generateUniqueShortCode() (string, error) {
	for i := 0; i < maxRetries; i++ {
		code := generateRandomCode(shortCodeLength)

		// Check if code already exists
		exists, err := s.shortCodeExists(code)
		if err != nil {
			return "", err
		}

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique short code after %d retries", maxRetries)
}

func (s *URLService) shortCodeExists(shortCode string) (bool, error) {
	query := `SELECT COUNT(*) FROM urls WHERE short_code = ?`
	var count int
	err := s.db.QueryRow(query, shortCode).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *URLService) validateCustomCode(code string) error {
	if len(code) < 3 || len(code) > 20 {
		return fmt.Errorf("custom code must be between 3 and 20 characters")
	}

	for _, char := range code {
		if !strings.ContainsRune(charset, char) {
			return fmt.Errorf("custom code can only contain alphanumeric characters")
		}
	}

	exists, err := s.shortCodeExists(code)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("custom code already exists")
	}

	return nil
}

// generateRandomCode generates a random code of specified length
func generateRandomCode(length int) string {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randomIndex, _ := rand.Int(rand.Reader, charsetLen)
		result[i] = charset[randomIndex.Int64()]
	}

	return string(result)
}
