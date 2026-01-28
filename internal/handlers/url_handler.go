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
        input[type="url"], input[type="text"], input[type="datetime-local"], select { width: 300px; padding: 10px; margin: 5px; }
        button { padding: 10px 20px; background: #007bff; color: white; border: none; cursor: pointer; }
        button:hover { background: #0056b3; }
        .result { margin: 20px 0; padding: 20px; background: #f8f9fa; border-radius: 5px; }
        .error { color: red; }
        .success { color: green; }
        .form-group { margin: 10px 0; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <h1>URL Shortener</h1>
        <p>Convert your long URLs into short, manageable links</p>
        
        <form id="shortenForm">
            <div class="form-group">
                <input type="url" id="originalUrl" placeholder="Enter your long URL here..." required>
            </div>
            <div class="form-group">
                <input type="text" id="customCode" placeholder="Custom short code (optional)">
            </div>
            <div class="form-group">
                <label for="expirationOption">Expiration (optional):</label>
                <select id="expirationOption">
                    <option value="">Never expires</option>
                    <option value="1h">1 hour</option>
                    <option value="24h">24 hours</option>
                    <option value="7d">7 days</option>
                    <option value="30d">30 days</option>
                    <option value="custom">Custom date/time</option>
                </select>
            </div>
            <div class="form-group" id="customExpirationDiv" style="display: none;">
                <label for="customExpiration">Custom expiration date:</label>
                <input type="datetime-local" id="customExpiration">
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
        // Show/hide custom expiration date input
        document.getElementById('expirationOption').addEventListener('change', (e) => {
            const customDiv = document.getElementById('customExpirationDiv');
            if (e.target.value === 'custom') {
                customDiv.style.display = 'block';
            } else {
                customDiv.style.display = 'none';
            }
        });

        document.getElementById('shortenForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const originalUrl = document.getElementById('originalUrl').value;
            const customCode = document.getElementById('customCode').value;
            const expirationOption = document.getElementById('expirationOption').value;
            const customExpiration = document.getElementById('customExpiration').value;
            const resultDiv = document.getElementById('result');
            
            // Calculate expiration date
            let expiresAt = null;
            if (expirationOption) {
                if (expirationOption === 'custom' && customExpiration) {
                    expiresAt = new Date(customExpiration).toISOString();
                } else if (expirationOption !== 'custom') {
                    const now = new Date();
                    switch (expirationOption) {
                        case '1h':
                            now.setHours(now.getHours() + 1);
                            break;
                        case '24h':
                            now.setHours(now.getHours() + 24);
                            break;
                        case '7d':
                            now.setDate(now.getDate() + 7);
                            break;
                        case '30d':
                            now.setDate(now.getDate() + 30);
                            break;
                    }
                    expiresAt = now.toISOString();
                }
            }
            
            try {
                const requestBody = {
                    original_url: originalUrl,
                    custom_code: customCode || undefined,
                    expires_at: expiresAt
                };

                const response = await fetch('/api/v1/shorten', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(requestBody)
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    let expirationInfo = '';
                    if (data.expires_at) {
                        expirationInfo = ` + "`" + `<p><strong>Expires:</strong> ${new Date(data.expires_at).toLocaleString()}</p>` + "`" + `;
                    }
                    
                    resultDiv.innerHTML = ` + "`" + `
                        <div class="success">
                            <h3>Success!</h3>
                            <p><strong>Short URL:</strong> <a href="${data.short_url}" target="_blank">${data.short_url}</a></p>
                            <p><strong>Original URL:</strong> ${data.original_url}</p>
                            <p><strong>Created:</strong> ${new Date(data.created_at).toLocaleString()}</p>
                            ${expirationInfo}
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
        body { font-family: Arial, sans-serif; max-width: 1400px; margin: 0 auto; padding: 20px; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; table-layout: fixed; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; word-wrap: break-word; }
        th { background-color: #f2f2f2; }
        th:nth-child(1) { width: 10%; } /* Short Code */
        th:nth-child(2) { width: 35%; } /* Original URL */
        th:nth-child(3) { width: 8%; } /* Clicks */
        th:nth-child(4) { width: 15%; } /* Created */
        th:nth-child(5) { width: 15%; } /* Expires */
        th:nth-child(6) { width: 17%; } /* Actions */
        .short-url { color: #007bff; text-decoration: none; }
        .short-url:hover { text-decoration: underline; }
        .url-cell { overflow: hidden; text-overflow: ellipsis; max-width: 0; }
        .analytics-btn { padding: 5px 10px; background: #28a745; color: white; text-decoration: none; border-radius: 3px; margin: 2px; display: inline-block; font-size: 12px; }
        .analytics-btn:hover { background: #218838; }
        .qr-btn { padding: 5px 10px; background: #17a2b8; color: white; text-decoration: none; border-radius: 3px; margin: 2px; display: inline-block; font-size: 12px; }
        .qr-btn:hover { background: #138496; }
        .expired { color: #dc3545; font-weight: bold; }
        .expires-soon { color: #ffc107; }
        .actions-cell { white-space: nowrap; }
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
                <th>Expires</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .}}
            <tr>
                <td><a href="/{{.ShortCode}}" class="short-url" target="_blank">{{.ShortCode}}</a></td>
                <td class="url-cell" title="{{.OriginalURL}}">{{.OriginalURL}}</td>
                <td>{{.ClickCount}}</td>
                <td>{{.CreatedAt.Format "Jan 2, 2006 15:04"}}</td>
                <td>
                    {{if .ExpiresAt}}
                        {{.ExpiresAt.Format "Jan 2, 2006 15:04"}}
                    {{else}}
                        Never
                    {{end}}
                </td>
                <td class="actions-cell">
                    <a href="/analytics/{{.ShortCode}}" class="analytics-btn">Analytics</a>
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

func (h *URLHandler) AnalyticsPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	if shortCode == "" {
		http.Error(w, "Missing short code", http.StatusBadRequest)
		return
	}

	// First check if the URL exists
	_, err := h.urlService.GetOriginalURL(shortCode)
	if err != nil {
		http.Error(w, "Short code not found", http.StatusNotFound)
		return
	}

	// Get analytics (even if no clicks yet)
	analytics, err := h.analyticsService.GetAnalytics(shortCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching analytics: %v", err), http.StatusInternalServerError)
		return
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Analytics - {{.URL.ShortCode}}</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; background: #f5f5f5; }
        .header { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header h1 { margin: 0 0 10px 0; color: #333; }
        .header .url-info { color: #666; font-size: 14px; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .stat-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .stat-card h3 { margin: 0 0 10px 0; color: #666; font-size: 14px; font-weight: normal; }
        .stat-card .value { font-size: 36px; font-weight: bold; color: #007bff; margin: 0; }
        .section { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .section h2 { margin: 0 0 20px 0; color: #333; font-size: 20px; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
        th { background-color: #f8f9fa; color: #666; font-weight: 600; }
        .country-bar { display: flex; align-items: center; margin: 10px 0; }
        .country-name { width: 100px; font-weight: 500; }
        .bar-container { flex: 1; background: #e9ecef; height: 30px; border-radius: 4px; overflow: hidden; margin: 0 10px; }
        .bar-fill { background: linear-gradient(90deg, #007bff, #0056b3); height: 100%; display: flex; align-items: center; padding-left: 10px; color: white; font-size: 12px; font-weight: bold; }
        .no-data { color: #999; font-style: italic; text-align: center; padding: 40px; }
        .back-link { display: inline-block; margin-bottom: 20px; color: #007bff; text-decoration: none; }
        .back-link:hover { text-decoration: underline; }
        .short-url { color: #007bff; word-break: break-all; }
        .chart { margin: 20px 0; }
        .day-item { display: flex; align-items: center; margin: 8px 0; }
        .day-date { width: 120px; font-size: 14px; color: #666; }
        .day-bar { flex: 1; background: #e9ecef; height: 24px; border-radius: 4px; overflow: hidden; margin: 0 10px; }
        .day-fill { background: #28a745; height: 100%; display: flex; align-items: center; padding-left: 8px; color: white; font-size: 12px; }
    </style>
</head>
<body>
    <a href="/dashboard" class="back-link">← Back to Dashboard</a>
    
    <div class="header">
        <h1>Analytics for {{.URL.ShortCode}}</h1>
        <div class="url-info">
            <p><strong>Short URL:</strong> <span class="short-url">{{.URL.ShortCode}}</span></p>
            <p><strong>Original URL:</strong> <span class="short-url">{{.URL.OriginalURL}}</span></p>
            <p><strong>Created:</strong> {{.URL.CreatedAt.Format "Jan 2, 2006 15:04"}}</p>
            {{if .URL.ExpiresAt}}<p><strong>Expires:</strong> {{.URL.ExpiresAt.Format "Jan 2, 2006 15:04"}}</p>{{end}}
        </div>
    </div>

    <div class="stats-grid">
        <div class="stat-card">
            <h3>Total Clicks</h3>
            <p class="value">{{.TotalClicks}}</p>
        </div>
        <div class="stat-card">
            <h3>Unique Visitors</h3>
            <p class="value">{{.UniqueVisitors}}</p>
        </div>
        <div class="stat-card">
            <h3>Countries</h3>
            <p class="value">{{len .ClicksByCountry}}</p>
        </div>
    </div>

    <div class="section">
        <h2>Clicks by Country</h2>
        {{if .ClicksByCountry}}
        <div class="chart">
            {{range $country, $count := .ClicksByCountry}}
            <div class="country-bar">
                <div class="country-name">{{$country}}</div>
                <div class="bar-container">
                    <div class="bar-fill" style="width: {{div (mul $count 100) $.TotalClicks}}%">
                        {{$count}} clicks
                    </div>
                </div>
            </div>
            {{end}}
        </div>
        {{else}}
        <p class="no-data">No geographic data available yet</p>
        {{end}}
    </div>

    <div class="section">
        <h2>Clicks Over Time (Last 30 Days)</h2>
        {{if .ClicksByDay}}
        <div class="chart">
            {{range .ClicksByDay}}
            <div class="day-item">
                <div class="day-date">{{.Date}}</div>
                <div class="day-bar">
                    <div class="day-fill" style="width: {{div (mul .Clicks 100) $.TotalClicks}}%">
                        {{.Clicks}}
                    </div>
                </div>
            </div>
            {{end}}
        </div>
        {{else}}
        <p class="no-data">No click history available yet</p>
        {{end}}
    </div>

    <div class="section">
        <h2>Recent Clicks</h2>
        {{if .RecentClicks}}
        <table>
            <thead>
                <tr>
                    <th>Time</th>
                    <th>IP Address</th>
                    <th>Location</th>
                    <th>User Agent</th>
                </tr>
            </thead>
            <tbody>
                {{range .RecentClicks}}
                <tr>
                    <td>{{.ClickedAt.Format "Jan 2, 15:04:05"}}</td>
                    <td>{{.IPAddress}}</td>
                    <td>{{if .Country}}{{.City}}, {{.Country}}{{else}}Unknown{{end}}</td>
                    <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{.UserAgent}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{else}}
        <p class="no-data">No clicks recorded yet</p>
        {{end}}
    </div>
</body>
</html>`

	t := template.New("analytics")
	t.Funcs(template.FuncMap{
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mul": func(a, b int) int {
			return a * b
		},
	})
	t, _ = t.Parse(tmpl)
	t.Execute(w, analytics)
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

	// Handle IPv6 addresses like [::1]:port
	if strings.HasPrefix(ip, "[") {
		// Extract IP from [ip]:port format
		endBracket := strings.Index(ip, "]")
		if endBracket > 0 {
			return ip[1:endBracket]
		}
	}

	// Handle IPv4 addresses like 127.0.0.1:port
	if strings.Contains(ip, ":") {
		lastColon := strings.LastIndex(ip, ":")
		return ip[:lastColon]
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
