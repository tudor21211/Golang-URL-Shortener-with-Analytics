package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"url-shortener/internal/models"
	"url-shortener/internal/services"

	"github.com/gorilla/mux"
	"github.com/skip2/go-qrcode"
)

type URLHandler struct {
	urlService       *services.URLService
	analyticsService *services.AnalyticsService
}

func NewURLHandler(urlService *services.URLService, analyticsService *services.AnalyticsService) *URLHandler {
	return &URLHandler{
		urlService:       urlService,
		analyticsService: analyticsService,
	}
}

// ShortenURL handles POST /api/v1/shorten
func (h *URLHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	var req models.ShortenURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	if !strings.HasPrefix(req.OriginalURL, "http://") && !strings.HasPrefix(req.OriginalURL, "https://") {
		h.respondWithError(w, http.StatusBadRequest, "Invalid URL", "URL must start with http:// or https://")
		return
	}

	clientIP := h.getClientIP(r)

	url, err := h.urlService.ShortenURL(req, clientIP)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Failed to shorten URL", err.Error())
		return
	}

	response := models.ShortenURLResponse{
		ShortCode:   url.ShortCode,
		ShortURL:    fmt.Sprintf("%s://%s/%s", h.getScheme(r), r.Host, url.ShortCode),
		OriginalURL: url.OriginalURL,
		CreatedAt:   url.CreatedAt,
		ExpiresAt:   url.ExpiresAt,
	}

	h.respondWithJSON(w, http.StatusCreated, response)
}

func (h *URLHandler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	if shortCode == "" {
		h.respondWithError(w, http.StatusBadRequest, "Missing short code", "")
		return
	}

	url, err := h.urlService.GetOriginalURL(shortCode)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "URL not found", err.Error())
		return
	}
	//Here i added an analytics recording block
	click := models.Click{
		URLShortCode: shortCode,
		IPAddress:    h.getClientIP(r),
		UserAgent:    r.UserAgent(),
		Referer:      r.Referer(),
		ClickedAt:    time.Now(),
	}

	click.Country, click.City = h.analyticsService.GetLocationFromIP(click.IPAddress)

	go func() {
		if err := h.analyticsService.RecordClick(click); err != nil {
			fmt.Printf("Failed to record click: %v\n", err)
		}
		if err := h.urlService.IncrementClickCount(shortCode); err != nil {
			fmt.Printf("Failed to increment click count: %v\n", err)
		}
	}()

	http.Redirect(w, r, url.OriginalURL, http.StatusMovedPermanently)
}

// GetAnalytics handles GET /api/v1/analytics/{shortCode}
func (h *URLHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	if shortCode == "" {
		h.respondWithError(w, http.StatusBadRequest, "Missing short code", "")
		return
	}

	analytics, err := h.analyticsService.GetAnalytics(shortCode)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Analytics not found", err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, analytics)
}

// GetUserURLs handles GET /api/v1/urls
func (h *URLHandler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	clientIP := h.getClientIP(r)

	urls, err := h.urlService.GetUserURLs(clientIP)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to retrieve URLs", err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, urls)
}

func (h *URLHandler) GenerateQRCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	if shortCode == "" {
		h.respondWithError(w, http.StatusBadRequest, "Missing short code", "")
		return
	}

	_, err := h.urlService.GetOriginalURL(shortCode)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "URL not found", err.Error())
		return
	}

	shortURL := fmt.Sprintf("%s://%s/%s", h.getScheme(r), r.Host, shortCode)

	png, err := qrcode.Encode(shortURL, qrcode.Medium, 256)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to generate QR code", err.Error())
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s-qr.png", shortCode))
	w.Write(png)
}

func (h *URLHandler) GenerateQRCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	if shortCode == "" {
		h.respondWithError(w, http.StatusBadRequest, "Missing short code", "")
		return
	}

	_, err := h.urlService.GetOriginalURL(shortCode)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "URL not found", err.Error())
		return
	}

	shortURL := fmt.Sprintf("%s://%s/%s", h.getScheme(r), r.Host, shortCode)
	
	png, err := qrcode.Encode(shortURL, qrcode.Medium, 256)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to generate QR code", err.Error())
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s-qr.png", shortCode))
	w.Write(png)
}

// HomePage handles GET / - i added a simple web interface, maybe will change it in the future
func (h *URLHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>URL Shortener</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .container { text-align: center; }
        input[type="url"], input[type="text"] { width: 300px; padding: 10px; margin: 5px; }
        button { padding: 10px 20px; background: #007bff; color: white; border: none; cursor: pointer; }
        button:hover { background: #0056b3; }
        .result { margin: 20px 0; padding: 20px; background: #f8f9fa; border-radius: 5px; }
        .error { color: red; }
        .success { color: green; }
    </style>
</head>
<body>
    <div class="container">
        <h1>URL Shortener</h1>
        <p>Convert your long URLs into short, manageable links</p>
        
        <form id="shortenForm">
            <div>
                <input type="url" id="originalUrl" placeholder="Enter your long URL here..." required>
            </div>
            <div>
                <input type="text" id="customCode" placeholder="Custom short code (optional)">
            </div>
            <div>
                <button type="submit">Shorten URL</button>
            </div>
        </form>
        
        <div id="result" class="result" style="display: none;"></div>
        
        <div style="margin-top: 40px;">
            <a href="/dashboard">View Dashboard</a>
        </div>
    </div>

    <script>
        document.getElementById('shortenForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const originalUrl = document.getElementById('originalUrl').value;
            const customCode = document.getElementById('customCode').value;
            const resultDiv = document.getElementById('result');
            
            try {
                const response = await fetch('/api/v1/shorten', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        original_url: originalUrl,
                        custom_code: customCode || undefined
                    })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    resultDiv.innerHTML = ` + "`" + `
                        <div class="success">
                            <h3>Success!</h3>
                            <p><strong>Short URL:</strong> <a href="${data.short_url}" target="_blank">${data.short_url}</a></p>
                            <p><strong>Original URL:</strong> ${data.original_url}</p>
                            <p><strong>Created:</strong> ${new Date(data.created_at).toLocaleString()}</p>
                            <p><strong>QR Code:</strong> <a href="/api/v1/qr/${data.short_code}" target="_blank">Download QR Code</a></p>
                            <p><img src="/api/v1/qr/${data.short_code}" alt="QR Code" style="margin-top: 10px; border: 1px solid #ddd; padding: 10px;"></p>
                        </div>
                    ` + "`" + `;
                } else {
                    resultDiv.innerHTML = ` + "`" + `<div class="error">Error: ${data.message}</div>` + "`" + `;
                }
                
                resultDiv.style.display = 'block';
            } catch (error) {
                resultDiv.innerHTML = ` + "`" + `<div class="error">Network error: ${error.message}</div>` + "`" + `;
                resultDiv.style.display = 'block';
            }
        });
    </script>
</body>
</html>`

	t, _ := template.New("home").Parse(tmpl)
	t.Execute(w, nil)
}

// Dashboard handles GET /dashboard
func (h *URLHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	clientIP := h.getClientIP(r)

	urls, err := h.urlService.GetUserURLs(clientIP)
	if err != nil {
		http.Error(w, "Failed to load dashboard", http.StatusInternalServerError)
		return
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Dashboard - URL Shortener</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f2f2f2; }
        .short-url { color: #007bff; text-decoration: none; }
        .short-url:hover { text-decoration: underline; }
        .analytics-btn { padding: 5px 10px; background: #28a745; color: white; text-decoration: none; border-radius: 3px; margin-right: 5px; }
        .analytics-btn:hover { background: #218838; }
        .qr-btn { padding: 5px 10px; background: #17a2b8; color: white; text-decoration: none; border-radius: 3px; }
        .qr-btn:hover { background: #138496; }
    </style>
</head>
<body>
    <h1>Dashboard</h1>
    <p><a href="/">← Back to Home</a></p>
    
    {{if .}}
    <table>
        <thead>
            <tr>
                <th>Short Code</th>
                <th>Original URL</th>
                <th>Clicks</th>
                <th>Created</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .}}
            <tr>
                <td><a href="/{{.ShortCode}}" class="short-url" target="_blank">{{.ShortCode}}</a></td>
                <td>{{.OriginalURL}}</td>
                <td>{{.ClickCount}}</td>
                <td>{{.CreatedAt.Format "Jan 2, 2006 15:04"}}</td>
                <td>
                    <a href="/api/v1/analytics/{{.ShortCode}}" class="analytics-btn" target="_blank">Analytics</a>
                    <a href="/api/v1/qr/{{.ShortCode}}" class="qr-btn" target="_blank">QR Code</a>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{else}}
    <p>No URLs created yet. <a href="/">Create your first short URL</a></p>
    {{end}}
</body>
</html>`

	t, _ := template.New("dashboard").Parse(tmpl)
	t.Execute(w, urls)
}

func (h *URLHandler) getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	ip := r.RemoteAddr
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}
	return ip
}

func (h *URLHandler) getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func (h *URLHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (h *URLHandler) respondWithError(w http.ResponseWriter, code int, error, message string) {
	errorResponse := models.ErrorResponse{
		Error:   error,
		Message: message,
		Code:    code,
	}
	h.respondWithJSON(w, code, errorResponse)
}
