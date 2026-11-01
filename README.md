# Go URL Shortener

A small REST API that validates a destination URL, generates a short code, stores the mapping, and redirects visitors to the original URL.

## Run

Requires Go 1.22 or later.

```bash
go test -race ./...
go run .
```

The server listens on `http://localhost:8080`.

## Create a short URL

```bash
curl -i -X POST http://localhost:8080/shorten \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/articles/go"}'
```

Example response:

```json
{
  "code": "a1B2c3D",
  "short_path": "/a1B2c3D"
}
```

Visit the returned path:

```bash
curl -i http://localhost:8080/a1B2c3D
```

## API

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | Health check |
| POST | `/shorten` | Validate a URL and create a short code |
| GET | `/{code}` | Redirect to the original URL |

The `POST /shorten` endpoint accepts a JSON body with a `url` field. Only HTTP and HTTPS destinations are accepted.

## Engineering concepts

- REST handlers using Go's standard `net/http` package.
- Strict JSON decoding and request body size limits.
- URL scheme validation.
- Cryptographically secure code generation with unbiased character selection.
- Mutex-protected storage and atomic collision-safe insertion.
- Handler tests using `net/http/httptest`.

## Design limitations

This is an educational in-memory implementation. Mappings disappear on restart, and there is no authentication, expiry, analytics, abuse detection, or persistence. A production service would typically add durable storage, rate limiting, observability, ownership controls, and operational safeguards.
