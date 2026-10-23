package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGenerateRoutesAllowedOriginsWithSpaces verifies that a hand-written
// OLLAMA_ORIGINS list ("a, b,") neither crashes route setup (gin-contrib/cors
// panics on an origin without a scheme prefix) nor silently drops an origin.
// See https://github.com/ollama/ollama/issues/18011.
func TestGenerateRoutesAllowedOriginsWithSpaces(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OLLAMA_ORIGINS", "https://a.example, https://b.example,")

	var handler http.Handler
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("GenerateRoutes panicked: %v", r)
			}
		}()

		var err error
		handler, err = (&Server{}).GenerateRoutes()
		if err != nil {
			t.Fatal(err)
		}
	}()

	for _, origin := range []string{"https://a.example", "https://b.example"} {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/version", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("Origin", origin)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Errorf("origin %s: status = %d, want %d", origin, recorder.Code, http.StatusOK)
		}
		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("origin %s: Access-Control-Allow-Origin = %q", origin, got)
		}
	}
}
