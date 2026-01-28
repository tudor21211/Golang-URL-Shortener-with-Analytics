package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ShortenRequest struct {
	OriginalURL string     `json:"original_url"`
	CustomCode  string     `json:"custom_code,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type ShortenResponse struct {
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

func main() {
	baseURL := "http://localhost:8080"

	fmt.Println("=== Testing URL Expiration Feature ===\n")

	// Test 1: Create URL without expiration
	fmt.Println("Test 1: Creating URL without expiration...")
	req1 := ShortenRequest{
		OriginalURL: "https://www.example.com/no-expiration",
	}
	resp1 := createShortURL(baseURL, req1)
	if resp1 != nil {
		fmt.Printf("✓ Created: %s\n", resp1.ShortURL)
		fmt.Printf("  Expires: %v\n\n", resp1.ExpiresAt)
	}

	// Test 2: Create URL that expires in 1 hour
	fmt.Println("Test 2: Creating URL that expires in 1 hour...")
	expires1Hour := time.Now().Add(1 * time.Hour)
	req2 := ShortenRequest{
		OriginalURL: "https://www.example.com/expires-1-hour",
		ExpiresAt:   &expires1Hour,
	}
	resp2 := createShortURL(baseURL, req2)
	if resp2 != nil {
		fmt.Printf("✓ Created: %s\n", resp2.ShortURL)
		fmt.Printf("  Expires: %v\n\n", *resp2.ExpiresAt)
	}

	// Test 3: Create URL that expires in 7 days
	fmt.Println("Test 3: Creating URL that expires in 7 days...")
	expires7Days := time.Now().Add(7 * 24 * time.Hour)
	req3 := ShortenRequest{
		OriginalURL: "https://www.example.com/expires-7-days",
		CustomCode:  "weeklink",
		ExpiresAt:   &expires7Days,
	}
	resp3 := createShortURL(baseURL, req3)
	if resp3 != nil {
		fmt.Printf("✓ Created: %s\n", resp3.ShortURL)
		fmt.Printf("  Expires: %v\n\n", *resp3.ExpiresAt)
	}

	// Test 4: Create URL that already expired (for testing expiration check)
	fmt.Println("Test 4: Creating URL that's already expired...")
	expiredTime := time.Now().Add(-1 * time.Hour) // 1 hour ago
	req4 := ShortenRequest{
		OriginalURL: "https://www.example.com/already-expired",
		CustomCode:  "expiredlink",
		ExpiresAt:   &expiredTime,
	}
	resp4 := createShortURL(baseURL, req4)
	if resp4 != nil {
		fmt.Printf("✓ Created: %s\n", resp4.ShortURL)
		fmt.Printf("  Expires: %v\n", *resp4.ExpiresAt)

		// Try to access the expired link
		fmt.Println("  Testing access to expired link...")
		testURL := fmt.Sprintf("%s/%s", baseURL, resp4.ShortCode)
		resp, err := http.Get(testURL)
		if err != nil {
			fmt.Printf("  ✗ Error accessing link: %v\n\n", err)
		} else {
			if resp.StatusCode == 404 {
				fmt.Printf("  ✓ Correctly blocked - Got 404 (URL expired)\n\n")
			} else {
				fmt.Printf("  ✗ Unexpected status: %d\n\n", resp.StatusCode)
			}
			resp.Body.Close()
		}
	}

	// Test 5: Access a valid URL
	if resp2 != nil {
		fmt.Println("Test 5: Accessing valid non-expired URL...")
		testURL := fmt.Sprintf("%s/%s", baseURL, resp2.ShortCode)
		fmt.Printf("  Attempting to access: %s\n", testURL)

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		}

		resp, err := client.Get(testURL)
		if err != nil {
			fmt.Printf("  ✗ Error: %v\n\n", err)
		} else {
			if resp.StatusCode == 301 || resp.StatusCode == 302 {
				location := resp.Header.Get("Location")
				fmt.Printf("  ✓ Successfully redirects to: %s\n\n", location)
			} else {
				fmt.Printf("  ✗ Unexpected status: %d\n\n", resp.StatusCode)
			}
			resp.Body.Close()
		}
	}

	fmt.Println("=== Tests Complete ===")
	fmt.Println("\nYou can now:")
	fmt.Println("1. Visit http://localhost:8080 to test the web interface")
	fmt.Println("2. Visit http://localhost:8080/dashboard to see all created URLs with expiration dates")
}

func createShortURL(baseURL string, req ShortenRequest) *ShortenResponse {
	jsonData, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		return nil
	}

	resp, err := http.Post(baseURL+"/api/v1/shorten", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return nil
	}

	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Error: Got status %d - %s\n", resp.StatusCode, string(body))
		return nil
	}

	var result ShortenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		return nil
	}

	return &result
}
