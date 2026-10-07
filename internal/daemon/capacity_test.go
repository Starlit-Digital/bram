package daemon

import (
	"github.com/cshaiku/bram/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCapacityRejectsThenReleases(t *testing.T) {
	s := New(config.Default())
	for i := 0; i < cap(s.slots); i++ {
		s.slots <- struct{}{}
	}
	r := httptest.NewRequest(http.MethodPost, "/v1/ask", strings.NewReader(`{"peer":"echo","prompt":"synthetic"}`))
	w := httptest.NewRecorder()
	s.ask(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("busy status %d", w.Code)
	}
	<-s.slots
	r = httptest.NewRequest(http.MethodPost, "/v1/ask", strings.NewReader(`{"peer":"echo","prompt":"synthetic"}`))
	w = httptest.NewRecorder()
	s.ask(w, r)
	if w.Code != http.StatusOK || len(s.slots) != cap(s.slots)-1 {
		t.Fatalf("status %d slots %d", w.Code, len(s.slots))
	}
}
