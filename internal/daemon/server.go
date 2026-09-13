package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"time"

	"bram/internal/appinfo"
	"bram/internal/config"
	"bram/internal/peer"
)

type Server struct {
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
	s := &Server{cfg: cfg}
	mux.HandleFunc("/v1/health", s.health)
	mux.HandleFunc("/v1/peers", s.peers)
	mux.HandleFunc("/v1/ask", s.ask)
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"name":    "bram",
		"version": appinfo.Version,
		"listen":  s.cfg.Listen,
	})
}

func (s *Server) peers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	names := make([]string, 0, len(s.cfg.Peers))
	for name := range s.cfg.Peers {
		names = append(names, name)
	}
	sort.Strings(names)
	writeJSON(w, http.StatusOK, map[string]any{
		"default_peer": s.cfg.DefaultPeer,
		"peers":        names,
	})
}

func (s *Server) ask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()

	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}
	name := req.Peer
	if name == "" {
		name = s.cfg.DefaultPeer
	}
	if name == "" {
		writeError(w, http.StatusBadRequest, "peer is required")
		return
	}
	spec, ok := s.cfg.Peers[name]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("peer %q is not configured", name))
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
		writeJSON(w, http.StatusBadGateway, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
