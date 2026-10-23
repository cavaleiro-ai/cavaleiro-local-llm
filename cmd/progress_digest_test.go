package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/ollama/ollama/api"
)

// newDigestTestServer serves routes from a test server and points
// OLLAMA_HOST at it.
func newDigestTestServer(t *testing.T, routes map[string]http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && r.URL.Path == "/" {
			return
		}
		handler, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		handler(w, r)
	}))
	t.Setenv("OLLAMA_HOST", server.URL)
	t.Cleanup(server.Close)
}

func writeProgress(t *testing.T, w http.ResponseWriter, responses ...api.ProgressResponse) {
	t.Helper()
	for _, resp := range responses {
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Error(err)
			return
		}
		w.(http.Flusher).Flush()
	}
}

// TestListHandlersShortDigest verifies that `ollama list` and `ollama ps` do
// not panic when a server reports a digest shorter than the 12 characters
// they display. See https://github.com/ollama/ollama/issues/14250.
func TestListHandlersShortDigest(t *testing.T) {
	newDigestTestServer(t, map[string]http.HandlerFunc{
		"/api/tags": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(api.ListResponse{Models: []api.ListModelResponse{
				{Name: "short:latest", Digest: "abc", Size: 1024, ModifiedAt: time.Now()},
				{Name: "empty:latest", Digest: "", Size: 1024, ModifiedAt: time.Now()},
			}})
		},
		"/api/ps": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(api.ProcessResponse{Models: []api.ProcessModelResponse{
				{Name: "short:latest", Digest: "abc", Size: 1024, ExpiresAt: time.Now().Add(time.Hour)},
			}})
		},
	})

	for name, handler := range map[string]func(*cobra.Command, []string) error{
		"list": ListHandler,
		"ps":   ListRunningHandler,
	} {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetContext(t.Context())

			var err error
			var panicked any
			out := captureStdout(t, func() {
				defer func() { panicked = recover() }()
				err = handler(cmd, nil)
			})
			if panicked != nil {
				t.Fatalf("%s panicked on a short digest: %v", name, panicked)
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "short:latest") || !strings.Contains(out, "abc") {
				t.Fatalf("output missing model or digest:\n%s", out)
			}
		})
	}
}

// TestPushHandlerProgressWithoutStatus verifies that push progress made only
// of layer updates, without a status message, panics neither on a digest
// shorter than a sha256 digest nor on the spinner that was never created.
func TestPushHandlerProgressWithoutStatus(t *testing.T) {
	for name, digest := range map[string]string{
		"short digest": "sha256:abc",
		"full digest":  "sha256:" + strings.Repeat("ab", 32),
	} {
		t.Run(name, func(t *testing.T) {
			newDigestTestServer(t, map[string]http.HandlerFunc{
				"/api/push": func(w http.ResponseWriter, r *http.Request) {
					writeProgress(t, w,
						api.ProgressResponse{Digest: digest, Total: 10, Completed: 5},
						api.ProgressResponse{Digest: digest, Total: 10, Completed: 10},
					)
				},
			})

			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)
			t.Setenv("USERPROFILE", tmpDir)

			cmd := &cobra.Command{}
			cmd.Flags().Bool("insecure", false, "")
			cmd.SetContext(t.Context())

			var err error
			var panicked any
			var stdout string
			captureStderr(t, func() {
				stdout = captureStdout(t, func() {
					defer func() { panicked = recover() }()
					err = PushHandler(cmd, []string{"localhost:5000/user/model"})
				})
			})
			if panicked != nil {
				t.Fatalf("PushHandler panicked: %v", panicked)
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stdout, "You can find your model at") {
				t.Fatalf("unexpected output %q", stdout)
			}
		})
	}
}

// TestCreateHandlerProgressShortDigest verifies that create progress for a
// layer without a status message does not panic on a short digest.
func TestCreateHandlerProgressShortDigest(t *testing.T) {
	newDigestTestServer(t, map[string]http.HandlerFunc{
		"/api/create": func(w http.ResponseWriter, r *http.Request) {
			writeProgress(t, w,
				api.ProgressResponse{Digest: "sha256:abc", Total: 10, Completed: 10},
				api.ProgressResponse{Status: "success"},
			)
		},
	})

	modelfile := filepath.Join(t.TempDir(), "Modelfile")
	if err := os.WriteFile(modelfile, []byte("FROM foo"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("file", "", "")
	if err := cmd.Flags().Set("file", modelfile); err != nil {
		t.Fatal(err)
	}
	cmd.Flags().Bool("insecure", false, "")
	cmd.SetContext(t.Context())

	var err error
	var panicked any
	captureStderr(t, func() {
		captureStdout(t, func() {
			defer func() { panicked = recover() }()
			err = CreateHandler(cmd, []string{"test-model"})
		})
	})
	if panicked != nil {
		t.Fatalf("CreateHandler panicked: %v", panicked)
	}
	if err != nil {
		t.Fatal(err)
	}
}
