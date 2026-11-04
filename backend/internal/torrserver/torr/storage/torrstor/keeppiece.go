package torrstor

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"plenka/internal/torrserver/log"
)

// KeepPiece is a piece kept on disk outside the cache: preloaded or fully downloaded
// data that is never evicted. It is written to "<id>.part" and renamed to "<id>" once
// the client has verified it, so a file named "<id>" is always a complete piece.
type KeepPiece struct {
	piece *Piece
	dir   string

	mu sync.Mutex
}

func newKeepPiece(p *Piece, dir string) *KeepPiece {
	return &KeepPiece{piece: p, dir: dir}
}

func (k *KeepPiece) final() string { return filepath.Join(k.dir, strconv.Itoa(k.piece.Id)) }
func (k *KeepPiece) part() string  { return k.final() + ".part" }

func (k *KeepPiece) name() string {
	if k.piece.Complete {
		return k.final()
	}
	return k.part()
}

func (k *KeepPiece) WriteAt(b []byte, off int64) (int, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := os.MkdirAll(k.dir, 0o777); err != nil {
		log.TLogln("Error create dir:", err)
		return 0, err
	}
	ff, err := os.OpenFile(k.name(), os.O_RDWR|os.O_CREATE, 0o666)
	if err != nil {
		log.TLogln("Error open file:", err)
		return 0, err
	}
	defer ff.Close()
	n, err := ff.WriteAt(b, off)
	k.piece.Size = min(k.piece.Size+int64(n), k.piece.cache.pieceLen(k.piece.Id))
	k.piece.Accessed = time.Now().Unix()
	return n, err
}

func (k *KeepPiece) ReadAt(b []byte, off int64) (int, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	ff, err := os.Open(k.name())
	if os.IsNotExist(err) {
		return 0, io.EOF
	}
	if err != nil {
		log.TLogln("Error open file:", err)
		return 0, err
	}
	defer ff.Close()
	k.piece.Accessed = time.Now().Unix()
	return ff.ReadAt(b, off)
}

// setComplete renames the file to match the completion the client reports.
func (k *KeepPiece) setComplete(done bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	from, to := k.part(), k.final()
	if !done {
		from, to = to, from
	}
	if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
		log.TLogln("Error rename piece:", err)
	}
	k.piece.Complete = done
}

// fill writes data the piece already had in the cache, so it is not downloaded again.
func (k *KeepPiece) fill(data []byte, complete bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := os.MkdirAll(k.dir, 0o777); err != nil {
		log.TLogln("Error create dir:", err)
		return
	}
	name := k.part()
	if complete {
		name = k.final()
	}
	if err := os.WriteFile(name, data, 0o666); err != nil {
		log.TLogln("Error write piece:", err)
		return
	}
	k.piece.Complete = complete
}

// adopt moves a disk cache file of the piece into the keep dir.
func (k *KeepPiece) adopt(from string, complete bool) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := os.MkdirAll(k.dir, 0o777); err != nil {
		return false
	}
	to := k.part()
	if complete {
		to = k.final()
	}
	return os.Rename(from, to) == nil
}

func (k *KeepPiece) Release() {
	k.mu.Lock()
	defer k.mu.Unlock()
	os.Remove(k.final())
	os.Remove(k.part())
	k.piece.Size = 0
	k.piece.Complete = false
}
