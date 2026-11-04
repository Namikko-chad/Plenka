package store

import (
	"testing"
)

func TestReopenKeepsValues(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put("config", "k", "v"); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var v string
	if ok, err := s.Get("config", "k", &v); err != nil || !ok || v != "v" {
		t.Fatalf("value not kept: ok=%v v=%q err=%v", ok, v, err)
	}
}
