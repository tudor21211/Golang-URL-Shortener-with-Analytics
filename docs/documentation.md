# URL Shortener Documentation

## Overview

This application converts long URLs into short links and tracks their usage. A URL like `https://example.com/very/long/path` becomes `http://localhost:8080/abc123`. When someone clicks the short link, they get redirected to the original URL and that click gets recorded.

Built with Go and SQLite, the application runs as a single binary without requiring external services like Redis or PostgreSQL. This makes it simple to deploy and maintain while still providing the core functionality you'd expect from a URL shortener.

You can use it through a web interface for manual URL shortening or integrate it into other applications via the REST API.

## Main Features

**URL Shortening**
- Generates random 6-character codes (abc123, xy4z91, etc.)
- Supports custom codes if you want something specific
- Optional expiration dates for temporary links (1 hour, 24 hours, 7 days, 30 days, or custom date/time)
- Expired URLs automatically return 404 errors

**Analytics**
- Total clicks and unique visitors per link
- Geographic breakdown by country
- Daily click trends
- Recent activity with timestamps and visitor info

**API Access**
- REST endpoints for creating URLs and getting analytics
- Rate limited to 10 requests per minute per IP
- QR code generation for each link

## Setup

Install Go 1.16 or higher, then run:

```bash
go mod tidy
go run ./cmd/main.go
```

The server starts on port 8080. Database gets created automatically.

## Usage

**Web Interface:**
1. Go to `http://localhost:8080`
2. Paste your URL, optionally set a custom code
3. Choose an expiration option (Never, 1h, 24h, 7d, 30d, or custom date/time)
4. Get your short link and QR code
5. View analytics and expiration dates from the dashboard

**API Examples:**

Create a short URL:
```bash
curl -X POST http://localhost:8080/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com/page"}'
```

Response:
```json
{
  "short_code": "abc123",
  "short_url": "http://localhost:8080/abc123",
  "original_url": "https://example.com/page",
  "created_at": "2026-01-28T10:30:00Z"
}
```

Create with custom code and expiration:
```bash
curl -X POST http://localhost:8080/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://example.com/page",
    "custom_code": "mylink",
    "expires_at": "2026-12-31T23:59:59Z"
  }'
```

Get analytics:
```bash
curl http://localhost:8080/api/v1/analytics/abc123
```

Returns JSON with total clicks, unique visitors, geographic breakdown, and recent activity.

Get QR code:
```bash
curl http://localhost:8080/api/v1/qr/abc123 -o qrcode.png
```

## How It Works

**Architecture:**
- Entry point in `cmd/main.go` sets up database, routes, and starts server
- Handlers process HTTP requests
- Services contain business logic (URL shortening, analytics)
- SQLite database stores URLs and clicks

**Request flow:**
1. Request hits middleware (rate limiting, security headers)
2. Router sends it to the appropriate handler
3. Handler validates input and calls service layer
4. Service reads/writes to database
5. Response goes back to client

**Database Schema:**

The application uses two tables:

`urls` table:
- `id` - Auto-incrementing primary key
- `short_code` - Unique 6-character identifier (indexed)
- `original_url` - The full URL to redirect to
- `created_at` - Timestamp when shortened
- `expires_at` - Optional expiration date
- `click_count` - Total number of clicks
- `user_ip` - IP address of creator
- `is_custom` - Whether the code was custom or random

`clicks` table:
- `id` - Auto-incrementing primary key
- `url_short_code` - References the short code (foreign key)
- `ip_address` - Visitor's IP
- `user_agent` - Browser and device information
- `referer` - Page they came from
- `country` and `city` - Geographic data
- `clicked_at` - When the click happened (indexed)

The `short_code` column has a unique index for fast lookups when redirecting. The `clicked_at` column is indexed for efficient analytics queries.

**URL Expiration:**

Users can set expiration dates when creating short URLs:

- **Preset Options:** Quick select from 1 hour, 24 hours, 7 days, or 30 days
- **Custom Date/Time:** Pick any specific expiration date and time
- **Never Expires:** Default behavior if no expiration is set

When a user tries to access an expired URL:
1. The system checks if the current time is after the `expires_at` timestamp
2. If expired, returns a 404 error with "URL has expired" message
3. No analytics are recorded for expired URLs
4. The URL remains in the database but cannot be accessed

Expiration dates are displayed:
- In the API response when creating a URL
- On the success page after shortening
- In the dashboard's "Expires" column
- In analytics responses

**Short code generation:**
- Random 6-character codes from a-z, A-Z, 0-9 (over 56 billion combinations)
- Checks for collisions and retries if needed
- Custom codes validated for length and allowed characters

**Security:**
- Rate limiting: 10 requests/minute per IP (applies to API, not redirects)
- Input validation on all URLs and custom codes
- SQL injection prevention via prepared statements
- Security headers on all responses (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection)

**Analytics Tracking:**

When someone clicks a short link:
1. The app looks up the short code in the database
2. If found and not expired, it issues an HTTP 301 redirect
3. At the same time, it records a click event with the visitor's IP, user agent, referrer, and timestamp
4. Geographic data is stored based on IP (currently placeholder, can be enhanced with GeoIP services)

Analytics are calculated on-demand when you view them. The system queries the clicks table and aggregates data by country, date, and other dimensions. This keeps the database simple and ensures you always see current data.

## Project Structure

```
cmd/main.go              - Application entry point
internal/
  database/              - Database setup and connection
  handlers/              - HTTP request handlers
  middleware/            - Rate limiting and security
  models/                - Data structures
  services/              - Business logic for URLs and analytics
tests/                   - Test suite
web/
  static/                - CSS, JavaScript, images
  templates/             - HTML templates
```

## Testing

Run tests with:
```bash
go test ./tests/... -v
```

The test suite covers URL shortening (random and custom codes), collision detection, expiration handling, analytics calculation, and API endpoints. Tests use an in-memory database.

## Deployment

For production, compile to a binary:
```bash
go build -o urlshortener ./cmd/main.go
```

The application needs:
- The compiled binary
- The `web/` directory for static files and templates
- Write permissions to create the SQLite database file

For higher traffic, consider:
- Using PostgreSQL instead of SQLite for better concurrent writes
- Adding Redis for caching frequently accessed URLs
- Running multiple instances behind a load balancer
- Using a CDN for static assets
