package services

import (
	"fmt"

	"url-shortener/internal/database"
	"url-shortener/internal/models"
)

type URLService struct {
	db *database.DB
}

func NewURLService(db *database.DB) *URLService {
	return &URLService{db: db}
}

func (s *URLService) ShortenURL(req models.ShortenURLRequest, userIP string) (*models.URL, error) {
	return nil, fmt.Errorf("ShortenURL not implemented")
}

func (s *URLService) GetOriginalURL(shortCode string) (*models.URL, error) {
	return nil, fmt.Errorf("GetOriginalURL not implemented")
}

func (s *URLService) IncrementClickCount(shortCode string) error {
	return fmt.Errorf("IncrementClickCount not implemented")
}

func (s *URLService) GetUserURLs(userIP string) ([]models.URL, error) {
	return nil, fmt.Errorf("GetUserURLs not implemented")
}
