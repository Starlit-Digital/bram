package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cshaiku/bram/internal/structured"
	"io"
	"mime"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cshaiku/bram/internal/appinfo"
	"github.com/cshaiku/bram/internal/config"
	"github.com/cshaiku/bram/internal/peer"
)

type Server struct {
	slots  chan struct{}
	cfg    config.Config
	server *http.Server
}

type AskRequest struct {
	Peer   string `json:"peer,omitempty"`
	Prompt string `json:"prompt"`
}

type AskResponse struct {
	Peer       string `json:"peer"`
	Output     string `json:"output"`
	Stderr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

func New(cfg config.Config) *Server {
	mux := http.NewServeMux()
	s := &Server{cfg: cfg, slots: make(chan struct{}, 4)}
	mux.HandleFunc("/v1/health", negotiate(s.health))
	mux.HandleFunc("/v1/peers", negotiate(s.peers))
	mux.HandleFunc("/v1/ask", negotiate(s.ask))
	s.server = &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		return err
	}
	errc := make(chan error, 1)
	go func() {
		errc <- s.server.Serve(ln)
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"ok":      true,
		"name":    "bram",
		"version": appinfo.Version,
		"listen":  s.cfg.Listen,
	})
}

func (s *Server) peers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	names := make([]string, 0, len(s.cfg.Peers))
	for name := range s.cfg.Peers {
		names = append(names, name)
	}
	sort.Strings(names)
	writeJSON(w, r, http.StatusOK, map[string]any{
		"default_peer": s.cfg.DefaultPeer,
		"peers":        names,
	})
}

func (s *Server) ask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()

	var req AskRequest
	body, err := io.ReadAll(io.LimitReader(r.Body, structured.MaxBytes+1))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "unreadable request")
		return
	}
	body, err = structured.JSON(body, structured.MaxBytes)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON/GCF request")
		return
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Prompt == "" {
		writeError(w, r, http.StatusBadRequest, "prompt is required")
		return
	}
	name := req.Peer
	if name == "" {
		name = s.cfg.DefaultPeer
	}
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "peer is required")
		return
	}
	spec, ok := s.cfg.Peers[name]
	if !ok {
		writeError(w, r, http.StatusNotFound, fmt.Sprintf("peer %q is not configured", name))
		return
	}

	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		writeError(w, r, http.StatusTooManyRequests, "peer capacity reached (four active requests)")
		return
	}
	result, err := peer.Ask(r.Context(), spec, req.Prompt)
	resp := AskResponse{
		Peer:       name,
		Output:     result.Output,
		Stderr:     result.Stderr,
		ExitCode:   result.ExitCode,
		DurationMS: result.DurationMS,
	}
	if err != nil {
		resp.Error = err.Error()
		writeJSON(w, r, http.StatusBadGateway, resp)
		return
	}
	writeJSON(w, r, http.StatusOK, resp)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeJSON(w, r, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	format, err := responseFormat(r)
	if err != nil {
		format = "json"
	}
	body, err := structured.Marshal(v, format)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":"response encoding failed"}`)
	}
	contentType := "application/json"
	if bytes.HasPrefix(body, []byte("GCF")) {
		contentType = "application/gcf"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Vary", "Accept")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

func responseFormat(r *http.Request) (string, error) {
	if format := r.URL.Query().Get("format"); format != "" {
		if !structured.ValidFormat(format) {
			return "", fmt.Errorf("invalid response format")
		}
		return format, nil
	}
	for _, accept := range strings.Split(r.Header.Get("Accept"), ",") {
		kind, params, err := mime.ParseMediaType(strings.TrimSpace(accept))
		if err == nil && kind == "application/gcf" && params["q"] != "0" {
			return "gcf", nil
		}
	}
	return "json", nil
}
func negotiate(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := responseFormat(r); err != nil {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		handler(w, r)
	}
}
