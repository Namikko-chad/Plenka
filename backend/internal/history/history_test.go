package history

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"plenka/internal/store"
)

func newService(t *testing.T) *Service {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return New(st)
}

const hash = "0123456789abcdef0123456789abcdef01234567"

func TestSaveListOrderAndUpsert(t *testing.T) {
	s := newService(t)
	if _, err := s.Save(Entry{Hash: hash, FileID: 1, PositionMs: 10}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := s.Save(Entry{Hash: hash, FileID: 2, PositionMs: 20}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := s.Save(Entry{Hash: strings.ToUpper(hash), FileID: 1, PositionMs: 30}); err != nil {
		t.Fatal(err)
	}
	got, err := s.List(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 entries, got %d", len(got))
	}
	if got[0].FileID != 1 || got[0].PositionMs != 30 || got[0].Hash != hash {
		t.Fatalf("newest entry should be the upserted file 1, got %+v", got[0])
	}
}

func TestCapTrimsOldest(t *testing.T) {
	s := newService(t)
	for i := 0; i < maxEntries+5; i++ {
		if _, err := s.Save(Entry{Hash: hash, FileID: i}); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := s.List(0)
	if len(got) != maxEntries {
		t.Fatalf("want %d entries, got %d", maxEntries, len(got))
	}
}

func TestDeleteWholeTorrent(t *testing.T) {
	s := newService(t)
	other := strings.Repeat("f", 40)
	_, _ = s.Save(Entry{Hash: hash, FileID: 1})
	_, _ = s.Save(Entry{Hash: hash, FileID: 2})
	_, _ = s.Save(Entry{Hash: other, FileID: 1})
	if err := s.Delete(hash, -1); err != nil {
		t.Fatal(err)
	}
	got, _ := s.List(0)
	if len(got) != 1 || got[0].Hash != other {
		t.Fatalf("only the other torrent should remain, got %+v", got)
	}
}

func TestHTTPValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	newService(t).RegisterRoutes(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/history", strings.NewReader(`{"hash":"short","fileId":1}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad hash: want 400, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/history", strings.NewReader(`{"hash":"`+hash+`","fileId":1,"positionMs":5}`)))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"updatedAt"`) {
		t.Fatalf("save: want 200 with updatedAt, got %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/history/"+hash+"/1", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: want 204, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/history", nil))
	if strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("list after delete: want [], got %s", w.Body.String())
	}
}
