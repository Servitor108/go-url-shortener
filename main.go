package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const codeLength = 7

type store struct {
	mu sync.RWMutex
	urls map[string]string
}

func newStore() *store { return &store{urls: make(map[string]string)} }

// insertIfAbsent atomically inserts a code only if it is not already present.
func (s *store) insertIfAbsent(code, target string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.urls[code]; exists { return false }
	s.urls[code] = target
	return true
}

func (s *store) get(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	target, ok := s.urls[code]
	return target, ok
}

type server struct { store *store }

type shortenRequest struct { URL string `json:"url"` }
type shortenResponse struct {
	Code string `json:"code"`
	ShortPath string `json:"short_path"`
}

func validTarget(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Host == "" { return false }
	return u.Scheme == "http" || u.Scheme == "https"
}

func newCode() (string, error) {
	var b strings.Builder
	b.Grow(codeLength)
	max := big.NewInt(int64(len(alphabet)))
	for i := 0; i < codeLength; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil { return "", err }
		b.WriteByte(alphabet[n.Int64()])
	}
	return b.String(), nil
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /shorten", s.shorten)
	mux.HandleFunc("GET /{code}", s.redirect)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *server) shorten(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req shortenRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error":"invalid JSON body"})
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if !validTarget(req.URL) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error":"url must be a valid http or https URL"})
		return
	}

	for attempts := 0; attempts < 5; attempts++ {
		code, err := newCode()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error":"could not generate short code"})
			return
		}
		if !s.store.insertIfAbsent(code, req.URL) { continue }
		writeJSON(w, http.StatusCreated, shortenResponse{Code:code, ShortPath:"/"+code})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error":"could not allocate unique short code"})
}

func (s *server) redirect(w http.ResponseWriter, r *http.Request) {
	target, ok := s.store.get(r.PathValue("code"))
	if !ok { http.NotFound(w, r); return }
	http.Redirect(w, r, target, http.StatusFound)
}

func main() {
	s := &server{store:newStore()}
	httpServer := &http.Server{
		Addr:":8080",
		Handler:s.routes(),
		ReadHeaderTimeout:5*time.Second,
	}
	fmt.Println("URL shortener listening on :8080")
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
