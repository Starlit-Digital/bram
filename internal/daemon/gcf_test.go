package daemon

import (
	"bytes"
	"encoding/json"
	"github.com/cshaiku/bram/internal/config"
	"github.com/cshaiku/bram/internal/structured"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGCFNegotiatedAPI(t *testing.T) {
	cfg := config.Default()
	cfg.Peers["fail"] = config.Peer{Command: "/usr/bin/false"}
	server := New(cfg)
	cases := []struct {
		peer, prompt string
		code         int
	}{{"echo", "雪|00123", 200}, {"missing", "hi", 404}, {"echo", "", 400}, {"fail", "hi", 502}}
	for _, tc := range cases {
		wire, err := structured.Marshal(AskRequest{Peer: tc.peer, Prompt: tc.prompt}, "gcf")
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/v1/ask", bytes.NewReader(wire))
		req.Header.Set("Content-Type", "application/gcf")
		req.Header.Set("Accept", "application/gcf; q=1")
		out := httptest.NewRecorder()
		server.server.Handler.ServeHTTP(out, req)
		if out.Code != tc.code || out.Header().Get("Content-Type") != "application/gcf" {
			t.Fatalf("code/type: %d %s %s", out.Code, out.Header(), out.Body)
		}
		decoded, err := structured.JSON(out.Body.Bytes(), structured.MaxBytes)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		if err = json.Unmarshal(decoded, &result); err != nil {
			t.Fatal(err)
		}
		if tc.code == 200 && result["output"] != tc.prompt {
			t.Fatalf("lost prompt: %s", decoded)
		}
		if tc.code != 200 && result["error"] == nil {
			t.Fatal("missing structured error")
		}
	}
}
func TestJSONCompatibilityAndInvalidGCF(t *testing.T) {
	server := New(config.Default())
	for _, path := range []string{"/v1/health", "/v1/peers"} {
		req := httptest.NewRequest("GET", path, nil)
		out := httptest.NewRecorder()
		server.server.Handler.ServeHTTP(out, req)
		if out.Code != 200 || !json.Valid(out.Body.Bytes()) {
			t.Fatal(out.Body)
		}
	}
	for _, body := range []string{"GCF profile=graph\n", "{} trailing", "GCF profile=generic session=x\n"} {
		req := httptest.NewRequest("POST", "/v1/ask?format=gcf", bytes.NewBufferString(body))
		out := httptest.NewRecorder()
		server.server.Handler.ServeHTTP(out, req)
		if out.Code != 400 {
			t.Fatalf("accepted invalid body %q", body)
		}
	}
	req := httptest.NewRequest("GET", "/v1/health?format=bad", nil)
	out := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(out, req)
	if out.Code != 400 || !json.Valid(out.Body.Bytes()) {
		t.Fatal("invalid format must fail as JSON")
	}
}
