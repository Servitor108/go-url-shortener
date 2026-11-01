package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShortenAndRedirect(t *testing.T) {
	s := &server{store:newStore()}
	handler := s.routes()
	body := bytes.NewBufferString(`{"url":"https://example.com/docs"}`)
	create := httptest.NewRequest(http.MethodPost, "/shorten", body)
	create.Header.Set("Content-Type", "application/json")
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, create)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d: %s", createResponse.Code, http.StatusCreated, createResponse.Body.String())
	}
	var result shortenResponse
	if err := json.Unmarshal(createResponse.Body.Bytes(), &result); err != nil { t.Fatal(err) }
	if result.Code == "" { t.Fatal("expected generated short code") }

	redirect := httptest.NewRequest(http.MethodGet, "/"+result.Code, nil)
	redirectResponse := httptest.NewRecorder()
	handler.ServeHTTP(redirectResponse, redirect)
	if redirectResponse.Code != http.StatusFound { t.Fatalf("GET status = %d, want %d", redirectResponse.Code, http.StatusFound) }
	if got := redirectResponse.Header().Get("Location"); got != "https://example.com/docs" {
		t.Fatalf("Location = %q", got)
	}
}

func TestShortenRejectsNonHTTPURL(t *testing.T) {
	s := &server{store:newStore()}
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(`{"url":"javascript:alert(1)"}`))
	res := httptest.NewRecorder()
	s.routes().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest { t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest) }
}

func TestUnknownCodeReturns404(t *testing.T) {
	s := &server{store:newStore()}
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	res := httptest.NewRecorder()
	s.routes().ServeHTTP(res, req)
	if res.Code != http.StatusNotFound { t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound) }
}

func TestStoreInsertIsCollisionSafe(t *testing.T) {
	s := newStore()
	if !s.insertIfAbsent("abc1234", "https://first.example") { t.Fatal("first insert failed") }
	if s.insertIfAbsent("abc1234", "https://second.example") { t.Fatal("duplicate code was accepted") }
	got, ok := s.get("abc1234")
	if !ok || got != "https://first.example" { t.Fatalf("stored target = %q, %v", got, ok) }
}
