package torrstor

import (
	"sync"
	"sync/atomic"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
	"plenka/internal/torrserver/settings"
)

type Piece struct {
	storage.PieceImpl `json:"-"`

	Id   int   `json:"-"`
	Size int64 `json:"size"`

	Complete bool  `json:"complete"`
	Accessed int64 `json:"accessed"`

	mPiece *MemPiece  `json:"-"`
	dPiece *DiskPiece `json:"-"`
	kPiece *KeepPiece `json:"-"`

	// kept pieces live in kPiece, outside the cache: they are never evicted.
	// mu guards moving a piece there against concurrent reads and writes.
	kept bool
	mu   sync.RWMutex
	// wanted pieces are being preloaded or fully downloaded, at High priority with
	// wantHigh, else Normal.
	wanted   atomic.Bool
	wantHigh atomic.Bool

	cache *Cache `json:"-"`
}

func NewPiece(id int, cache *Cache) *Piece {
	p := &Piece{
		Id:    id,
		cache: cache,
	}

	if !settings.BTsets.UseDisk {
		p.mPiece = NewMemPiece(p)
	} else {
		p.dPiece = NewDiskPiece(p)
	}
	p.kPiece = newKeepPiece(p, cache.keepDir)
	return p
}

func (p *Piece) WriteAt(b []byte, off int64) (n int, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.kept {
		return p.kPiece.WriteAt(b, off)
	}
	if !settings.BTsets.UseDisk {
		return p.mPiece.WriteAt(b, off)
	} else {
		return p.dPiece.WriteAt(b, off)
	}
}

func (p *Piece) ReadAt(b []byte, off int64) (n int, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.kept {
		return p.kPiece.ReadAt(b, off)
	}
	if !settings.BTsets.UseDisk {
		return p.mPiece.ReadAt(b, off)
	} else {
		return p.dPiece.ReadAt(b, off)
	}
}

func (p *Piece) MarkComplete() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.kept {
		p.kPiece.setComplete(true)
		return nil
	}
	p.Complete = true
	return nil
}

func (p *Piece) MarkNotComplete() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.kept {
		p.kPiece.setComplete(false)
		return nil
	}
	p.Complete = false
	return nil
}

func (p *Piece) Completion() storage.Completion {
	return storage.Completion{
		Complete: p.Complete,
		Ok:       true,
	}
}

// isKept reports whether the piece lives in the keep dir.
func (p *Piece) isKept() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.kept
}

// keep moves the piece out of the cache into the keep dir, with the data it already
// has. It reports false when that data could not be moved and was dropped.
func (p *Piece) keep() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.kept {
		return true
	}
	p.kept = true
	if p.Size == 0 {
		return true
	}
	size, complete := p.Size, p.Complete
	moved := false
	if p.mPiece != nil {
		if data := p.mPiece.take(); data != nil {
			p.kPiece.fill(data[:min(int64(len(data)), p.cache.pieceLen(p.Id))], complete)
			moved = true
		}
	} else if p.dPiece != nil {
		moved = p.kPiece.adopt(p.dPiece.name, complete)
	}
	if !moved {
		if p.dPiece != nil {
			p.dPiece.Release()
		}
		p.Size, p.Complete = 0, false
		return false
	}
	p.Size, p.Complete = size, complete
	return true
}

// unkeep deletes the kept data; the piece goes back to the cache, empty.
func (p *Piece) unkeep() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.wanted.Store(false)
	if !p.kept {
		return
	}
	p.kPiece.Release()
	p.kept = false
}

func (p *Piece) Release() {
	if !settings.BTsets.UseDisk {
		p.mPiece.Release()
	} else {
		p.dPiece.Release()
	}
	// if !p.cache.isClosed {
	p.cache.torrent.Piece(p.Id).SetPriority(torrent.PiecePriorityNone)
	p.cache.torrent.Piece(p.Id).UpdateCompletion()
	//}
}
