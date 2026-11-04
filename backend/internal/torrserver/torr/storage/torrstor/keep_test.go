package torrstor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"plenka/internal/torrserver/settings"
)

func testCache(t *testing.T, dir string) *Cache {
	t.Helper()
	settings.BTsets = settings.DefaultBTSets()
	c := NewCache(8, nil)
	c.pieceLength, c.pieceCount, c.totalLength = 4, 3, 10
	c.keepDir = dir
	for i := 0; i < c.pieceCount; i++ {
		c.pieces[i] = NewPiece(i, c)
	}
	c.loadKept()
	return c
}

func TestKeepPiece(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keep")
	c := testCache(t, dir)

	// a piece the cache already has in memory moves to disk with its data
	p0 := c.pieces[0]
	p0.mPiece.buffer = []byte("abcd")
	p0.Size, p0.Complete = 4, true
	if !p0.keep() || p0.mPiece.buffer != nil {
		t.Fatal("piece 0 was not moved out of memory")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "0")); string(b) != "abcd" {
		t.Fatalf("piece 0 on disk: %q", b)
	}

	// a downloading piece is a .part file until it is complete
	p2 := c.pieces[2]
	p2.keep()
	if _, err := p2.WriteAt([]byte("xy"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "2.part")); err != nil {
		t.Fatal(err)
	}
	p2.MarkComplete()
	buf := make([]byte, 2)
	if _, err := p2.ReadAt(buf, 0); err != nil || !bytes.Equal(buf, []byte("xy")) {
		t.Fatalf("read %q %v", buf, err)
	}

	// kept pieces are not cache: neither counted nor evicted
	if c.getRemPieces(); c.filled != 0 {
		t.Fatalf("cache fill %d", c.filled)
	}
	c.removePiece(p0)
	if !p0.Complete {
		t.Fatal("kept piece was evicted")
	}

	// a half-written piece and a stray file of the wrong length
	os.WriteFile(filepath.Join(dir, "1.part"), []byte("q"), 0o666)
	os.WriteFile(filepath.Join(dir, "1"), []byte("toolong"), 0o666)

	// the next session finds them again
	c = testCache(t, dir)
	if p := c.pieces[0]; !p.isKept() || !p.Complete || p.Size != 4 {
		t.Fatalf("piece 0: kept %v complete %v size %d", p.isKept(), p.Complete, p.Size)
	}
	if p := c.pieces[1]; !p.isKept() || p.Complete || p.Size != 1 {
		t.Fatalf("piece 1: kept %v complete %v size %d", p.isKept(), p.Complete, p.Size)
	}
	if p := c.pieces[2]; !p.Complete || p.Size != 2 {
		t.Fatalf("last piece: complete %v size %d", p.Complete, p.Size)
	}
	if done, total := c.KeepProgress([]int{0, 1, 2}); done != 6 || total != 10 {
		t.Fatalf("progress %d / %d", done, total)
	}

	c.pieces[0].unkeep()
	if _, err := os.Stat(filepath.Join(dir, "0")); !os.IsNotExist(err) {
		t.Fatal("unkept piece is still on disk")
	}
}
