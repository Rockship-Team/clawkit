package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCosmoTokenExpired(t *testing.T) {
	t.Run("personal api key never expires", func(t *testing.T) {
		if cosmoTokenExpired("p_api_key_abc123") {
			t.Fatalf("personal API key should not be treated as expired")
		}
	})

	t.Run("malformed token counts as expired", func(t *testing.T) {
		if !cosmoTokenExpired("not.a.real.jwt.token") {
			t.Fatalf("malformed token should be treated as expired")
		}
		if !cosmoTokenExpired("nodots") {
			t.Fatalf("token without dots should be treated as expired")
		}
	})

	t.Run("future exp is not expired", func(t *testing.T) {
		exp := time.Now().Add(10 * time.Minute).Unix()
		tok := fakeJWT(t, exp)
		if cosmoTokenExpired(tok) {
			t.Fatalf("future-dated token should not be expired (exp=%d now=%d)", exp, time.Now().Unix())
		}
	})

	t.Run("near-exp token is treated as expired (60s buffer)", func(t *testing.T) {
		// Token exp is 30 seconds from now — within 60s buffer, should count as expired.
		exp := time.Now().Add(30 * time.Second).Unix()
		tok := fakeJWT(t, exp)
		if !cosmoTokenExpired(tok) {
			t.Fatalf("token expiring in 30s should be treated as expired due to 60s buffer")
		}
	})

	t.Run("past exp is expired", func(t *testing.T) {
		exp := time.Now().Add(-1 * time.Hour).Unix()
		tok := fakeJWT(t, exp)
		if !cosmoTokenExpired(tok) {
			t.Fatalf("past-dated token should be expired")
		}
	})
}

func TestCosmoDoRequest(t *testing.T) {
	t.Run("sends bearer auth and json content type", func(t *testing.T) {
		var gotAuth, gotCT, gotMethod, gotPath, gotBody string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotCT = r.Header.Get("Content-Type")
			gotMethod = r.Method
			gotPath = r.URL.Path
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			w.WriteHeader(200)
			w.Write([]byte(`{"ok":true}`))
		}))
		defer srv.Close()

		body, code, err := cosmoDoRequest(srv.URL, "/v1/contacts", "POST", "tok-xyz", []byte(`{"name":"x"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != 200 {
			t.Fatalf("code = %d, want 200", code)
		}
		if string(body) != `{"ok":true}` {
			t.Fatalf("body = %q, want %q", string(body), `{"ok":true}`)
		}
		if gotAuth != "Bearer tok-xyz" {
			t.Fatalf("Authorization header = %q, want %q", gotAuth, "Bearer tok-xyz")
		}
		if gotCT != "application/json" {
			t.Fatalf("Content-Type = %q, want %q", gotCT, "application/json")
		}
		if gotMethod != "POST" {
			t.Fatalf("method = %q, want POST", gotMethod)
		}
		if gotPath != "/v1/contacts" {
			t.Fatalf("path = %q, want /v1/contacts", gotPath)
		}
		if gotBody != `{"name":"x"}` {
			t.Fatalf("body = %q, want %q", gotBody, `{"name":"x"}`)
		}
	})

	t.Run("propagates non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"not found"}`))
		}))
		defer srv.Close()

		_, code, err := cosmoDoRequest(srv.URL, "/v1/missing", "GET", "tok", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != 404 {
			t.Fatalf("code = %d, want 404", code)
		}
	})

	t.Run("strips trailing slash on base url", func(t *testing.T) {
		var gotPath string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.WriteHeader(200)
		}))
		defer srv.Close()

		_, _, err := cosmoDoRequest(srv.URL+"/", "/v1/x", "GET", "t", nil)
		if err != nil {
			t.Fatal(err)
		}
		if gotPath != "/v1/x" {
			t.Fatalf("path = %q, want /v1/x (no double slash)", gotPath)
		}
	})
}

// fakeJWT builds a minimal unsigned JWT with the given exp claim. The signature
// segment is a placeholder — cosmoTokenExpired only decodes the payload.
func fakeJWT(t *testing.T, exp int64) string {
	t.Helper()
	claims := map[string]int64{"exp": exp}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	return strings.Join([]string{header, body, "sig"}, ".")
}
