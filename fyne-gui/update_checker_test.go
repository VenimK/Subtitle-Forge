package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchReleaseFileUsesFirstRouteAndSetsOctetStreamAccept(t *testing.T) {
	var apiHits, browserHits int

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiHits++
		if got := r.Header.Get("Accept"); got != "application/octet-stream" {
			t.Errorf("Accept header = %q, want application/octet-stream", got)
		}
		w.Write([]byte("bytes from the api route"))
	}))
	defer api.Close()

	browser := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		browserHits++
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer browser.Close()

	resp, err := fetchReleaseFile(&http.Client{}, []string{api.URL, browser.URL})
	if err != nil {
		t.Fatalf("fetchReleaseFile: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != "bytes from the api route" {
		t.Errorf("body = %q, want the bytes from the api route", body)
	}
	if browserHits != 0 {
		t.Errorf("browser route was requested %d times, want 0", browserHits)
	}
}

func TestFetchReleaseFileRetriesGatewayStatus(t *testing.T) {
	var hits int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 2 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.Write([]byte("bytes after a retry"))
	}))
	defer server.Close()

	resp, err := fetchReleaseFile(&http.Client{}, []string{"", server.URL})
	if err != nil {
		t.Fatalf("fetchReleaseFile: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != "bytes after a retry" {
		t.Errorf("body = %q, want the bytes served after the retry", body)
	}
	if hits != 2 {
		t.Errorf("requests = %d, want 2 (one gateway response, then a retry)", hits)
	}
}

func TestFetchReleaseFileDoesNotRetryOtherStatus(t *testing.T) {
	var hits int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := fetchReleaseFile(&http.Client{}, []string{server.URL})
	if err == nil {
		t.Fatal("expected an error for a non-gateway status")
	}
	if !strings.Contains(err.Error(), "bad status") {
		t.Errorf("error = %v, want a bad status error", err)
	}
	if hits != 1 {
		t.Errorf("requests = %d, want 1 (no retry for 404)", hits)
	}
}
