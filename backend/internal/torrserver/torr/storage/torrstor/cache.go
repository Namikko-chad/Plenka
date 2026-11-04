package torrstor

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anacrolix/torrent"

	"plenka/internal/torrserver/log"
	"plenka/internal/torrserver/settings"
	"plenka/internal/torrserver/torr/storage/state"
	"plenka/internal/torrserver/torr/utils"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

type Cache struct {
	storage.TorrentImpl
	storage *Storage

	capacity int64
	filled   int64
	hash     metainfo.Hash

	pieceLength int64
	pieceCount  int
	totalLength int64

	// keepDir holds the kept pieces (see KeepPiece): preloaded and fully downloaded
	// data, outside the capacity and never evicted.
	keepDir string

	// pieces content is immutable after Init; muPieces guards the map
	// reference itself (nilled in Close), so holders of a snapshot may
	// safely iterate it without the lock
	pieces   map[int]*Piece
	muPieces sync.RWMutex

	readers   map[*Reader]struct{}
	muReaders sync.RWMutex

	isRemove atomic.Bool
	isClosed atomic.Bool
	muRemove sync.Mutex
	// muPrio serializes clearPriority and setLoadPriority so that the priority
	// reset of a reader that has just closed cannot wipe the priorities a
	// freshly created reader has already set.
	muPrio  sync.Mutex
	torrent *torrent.Torrent
}

func NewCache(capacity int64, storage *Storage) *Cache {
	ret := &Cache{
		capacity: capacity,
		filled:   0,
		pieces:   make(map[int]*Piece),
		storage:  storage,
		readers:  make(map[*Reader]struct{}),
	}

	return ret
}

func (c *Cache) Init(info *metainfo.Info, hash metainfo.Hash) {
	log.TLogln("Create cache for:", info.Name, hash.HexString())
	if c.capacity == 0 {
		c.capacity = info.PieceLength * 4
	}

	c.pieceLength = info.PieceLength
	c.pieceCount = info.NumPieces()
	c.totalLength = info.TotalLength()
	c.hash = hash
	c.keepDir = filepath.Join(settings.KeepDir(), hash.HexString(), "keep")

	if settings.BTsets.UseDisk {
		name := filepath.Join(settings.BTsets.TorrentsSavePath, hash.HexString())
		err := os.MkdirAll(name, 0o777)
		if err != nil {
			log.TLogln("Error create dir:", err)
		}
	}

	for i := 0; i < c.pieceCount; i++ {
		c.pieces[i] = NewPiece(i, c)
	}
	c.loadKept()

	go c.priorityWatchdog()
}

// pieceLen is the length of piece id; the last one may be shorter.
func (c *Cache) pieceLen(id int) int64 {
	return min(c.pieceLength, c.totalLength-int64(id)*c.pieceLength)
}

// loadKept picks up the pieces kept on disk by an earlier session.
func (c *Cache) loadKept() {
	entries, err := os.ReadDir(c.keepDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name, part := strings.CutSuffix(e.Name(), ".part")
		id, err := strconv.Atoi(name)
		p, ok := c.pieces[id]
		info, ierr := e.Info()
		if err != nil || !ok || ierr != nil || e.IsDir() {
			continue
		}
		if !part && info.Size() != c.pieceLen(id) {
			os.Remove(filepath.Join(c.keepDir, e.Name()))
			continue
		}
		if p.dPiece != nil {
			p.dPiece.Release()
		}
		p.kept = true
		p.Size = min(info.Size(), c.pieceLen(id))
		p.Complete = !part
		p.Accessed = info.ModTime().Unix()
	}
}

// Keep moves pieces ids into the keep dir and downloads the missing ones at Normal
// priority (High with high), whatever the readers do, until they are complete.
func (c *Cache) Keep(ids []int, high bool) {
	if c == nil || c.torrent == nil {
		return
	}
	pieces := c.getPieces()
	for _, id := range ids {
		p, ok := pieces[id]
		if !ok {
			continue
		}
		if !p.keep() {
			c.torrent.Piece(id).UpdateCompletion()
		}
		if p.Complete {
			continue
		}
		if high || !p.wanted.Load() {
			p.wantHigh.Store(high)
		}
		p.wanted.Store(true)
		c.raiseWanted(id, p)
	}
}

// Unkeep deletes the kept pieces that stay does not list and stops downloading them;
// they go back to the cache, empty.
func (c *Cache) Unkeep(stay map[int]bool) {
	if c == nil || c.torrent == nil {
		return
	}
	for id, p := range c.getPieces() {
		if stay[id] {
			continue
		}
		wanted := p.wanted.Load()
		if !wanted && !p.isKept() {
			continue
		}
		p.unkeep()
		c.torrent.Piece(id).SetPriority(torrent.PiecePriorityNone)
		c.torrent.Piece(id).UpdateCompletion()
	}
	go c.cleanPieces()
}

// KeptIDs lists the pieces in the keep dir or on their way there.
func (c *Cache) KeptIDs() map[int]bool {
	ids := make(map[int]bool)
	if c == nil {
		return ids
	}
	for id, p := range c.getPieces() {
		if p.isKept() || p.wanted.Load() {
			ids[id] = true
		}
	}
	return ids
}

// KeepProgress sums the lengths of pieces ids and of those complete in the keep dir.
func (c *Cache) KeepProgress(ids []int) (done, total int64) {
	if c == nil {
		return 0, 0
	}
	pieces := c.getPieces()
	for _, id := range ids {
		p, ok := pieces[id]
		if !ok {
			continue
		}
		total += c.pieceLen(id)
		if p.Complete && p.isKept() {
			done += c.pieceLen(id)
		}
	}
	return done, total
}

// KeepPending is what the wanted pieces still have to fetch.
func (c *Cache) KeepPending() int64 {
	if c == nil {
		return 0
	}
	var sum int64
	for id, p := range c.getPieces() {
		if p.wanted.Load() && !p.Complete {
			sum += c.pieceLen(id)
		}
	}
	return sum
}

// rearmWanted puts back the priority of wanted pieces that lost it.
func (c *Cache) rearmWanted() {
	for id, p := range c.getPieces() {
		if !p.wanted.Load() || p.Complete {
			continue
		}
		c.raiseWanted(id, p)
	}
}

func (c *Cache) raiseWanted(id int, p *Piece) {
	prio := torrent.PiecePriorityNormal
	if p.wantHigh.Load() {
		prio = torrent.PiecePriorityHigh
	}
	if c.torrent.PieceState(id).Priority < prio {
		c.torrent.Piece(id).SetPriority(prio)
	}
}

// priorityWatchdog re-arms piece priorities while readers are active.
//
// setLoadPriority is only reached through the cache cleanup path, which is
// driven by piece reads and writes (see mempiece.go and diskpiece.go). Should
// priorities ever end up cleared while a reader still needs data, nothing is
// downloaded, so no piece I/O happens, so cleanup never runs and the
// priorities are never restored - the torrent stalls indefinitely with peers
// connected. Re-arming them periodically breaks that cycle regardless of how
// the priorities were lost.
func (c *Cache) priorityWatchdog() {
	for {
		time.Sleep(5 * time.Second)
		if c.isClosed.Load() {
			return
		}
		if c.torrent == nil {
			continue
		}
		if c.GetUseReaders() > 0 {
			c.getRemPieces()
		}
		c.rearmWanted()
	}
}

func (c *Cache) SetTorrent(torr *torrent.Torrent) {
	if c == nil {
		return
	}
	c.torrent = torr
}

func (c *Cache) getPieces() map[int]*Piece {
	c.muPieces.RLock()
	defer c.muPieces.RUnlock()
	return c.pieces
}

func (c *Cache) readersSnapshot() []*Reader {
	c.muReaders.RLock()
	defer c.muReaders.RUnlock()
	list := make([]*Reader, 0, len(c.readers))
	for r := range c.readers {
		list = append(list, r)
	}
	return list
}

func (c *Cache) Piece(m metainfo.Piece) storage.PieceImpl {
	if val, ok := c.getPieces()[m.Index()]; ok {
		return val
	}
	return &PieceFake{}
}

func (c *Cache) Close() error {
	if c.torrent != nil {
		log.TLogln("Close cache for:", c.torrent.Name(), c.hash)
	} else {
		log.TLogln("Close cache for:", c.hash)
	}
	c.isClosed.Store(true)

	c.storage.removeCache(c.hash)

	if settings.BTsets.RemoveCacheOnDrop {
		name := filepath.Join(settings.BTsets.TorrentsSavePath, c.hash.HexString())
		if name != "" && name != "/" {
			for _, v := range c.getPieces() {
				// Kept pieces are not cache: they stay until the torrent is removed.
				if v.dPiece != nil && !v.isKept() {
					os.Remove(v.dPiece.name)
				}
			}
			os.Remove(name)
		}
	}

	c.muReaders.Lock()
	c.readers = nil
	c.muReaders.Unlock()

	c.muPieces.Lock()
	c.pieces = nil
	c.muPieces.Unlock()

	utils.FreeOSMemGC()
	return nil
}

func (c *Cache) removePiece(piece *Piece) {
	if !c.isClosed.Load() && !piece.isKept() {
		piece.Release()
	}
}

func (c *Cache) AdjustRA(readahead int64) {
	if c == nil {
		return
	}
	if settings.BTsets.CacheSize == 0 {
		c.capacity = readahead * 3
	}
	for _, r := range c.readersSnapshot() {
		r.SetReadahead(readahead)
	}
}

func (c *Cache) GetState() *state.CacheState {
	cState := new(state.CacheState)

	piecesState := make(map[int]state.ItemState, 0)
	var fill int64 = 0

	for _, p := range c.getPieces() {
		if p.Size > 0 {
			if !p.isKept() {
				fill += p.Size
			}
			piecesState[p.Id] = state.ItemState{
				Id:        p.Id,
				Size:      p.Size,
				Length:    c.pieceLength,
				Completed: p.Complete,
				Priority:  int(c.torrent.PieceState(p.Id).Priority),
			}
		}
	}

	readersState := make([]*state.ReaderState, 0)

	// Queued pieces: prioritized for download (a reader's window, a preload, a full
	// download) but with nothing received yet, so they are not in the loop above.
	for id := 0; id < c.pieceCount; id++ {
		if _, ok := piecesState[id]; ok {
			continue
		}
		if prio := c.torrent.PieceState(id).Priority; prio != torrent.PiecePriorityNone {
			piecesState[id] = state.ItemState{Id: id, Length: c.pieceLength, Priority: int(prio)}
		}
	}

	for _, r := range c.readersSnapshot() {
		rng := r.getPiecesRange()
		pc := r.getReaderPiece()
		readersState = append(readersState, &state.ReaderState{
			Start:  rng.Start,
			End:    rng.End,
			Reader: pc,
		})
	}

	c.filled = fill
	cState.Capacity = c.capacity
	cState.PiecesLength = c.pieceLength
	cState.PiecesCount = c.pieceCount
	cState.Hash = c.hash.HexString()
	cState.Filled = fill
	cState.Pieces = piecesState
	cState.Readers = readersState
	return cState
}

func (c *Cache) cleanPieces() {
	if c.isRemove.Load() || c.isClosed.Load() {
		return
	}

	// Protection against concurrent deletion
	if !c.muRemove.TryLock() {
		return // Cleanup is already in progress in another goroutine
	}
	defer c.muRemove.Unlock()

	c.isRemove.Store(true)
	defer func() { c.isRemove.Store(false) }()

	remPieces := c.getRemPieces()
	if c.filled > c.capacity {
		rems := (c.filled-c.capacity)/c.pieceLength + 1
		for _, p := range remPieces {
			c.removePiece(p)
			rems--
			if rems <= 0 {
				utils.FreeOSMemGC()
				return
			}
		}
	}
}

func (c *Cache) getRemPieces() []*Piece {
	readers := c.readersSnapshot()

	// Collect read ranges from active readers
	ranges := make([]Range, 0)
	for _, r := range readers {
		r.checkReader()
		if r.isUse {
			ranges = append(ranges, r.getPiecesRange())
		}
	}
	ranges = mergeRange(ranges)

	piecesRemove := make([]*Piece, 0)
	fill := int64(0)

	// Determine which chunks can be deleted
	for id, p := range c.getPieces() {
		// kept pieces are outside the cache: not counted, never evicted
		if p.isKept() {
			continue
		}
		if p.Size > 0 {
			fill += p.Size
		}
		if len(ranges) > 0 {
			if !inRanges(ranges, id) {
				if p.Size > 0 && !c.isIdInFileBE(ranges, id) {
					piecesRemove = append(piecesRemove, p)
				}
			}
		} else {
			// When preloading, clear everything except the beginning and end of the file
			if p.Size > 0 && !c.isIdInFileBE(ranges, id) {
				piecesRemove = append(piecesRemove, p)
			}
		}
	}

	c.clearPriority()
	c.setLoadPriority(ranges)

	// Complete pieces first, oldest first; a piece still downloading goes last. The torrent
	// client never re-requests chunks it has received, so evicting their bytes makes a
	// responsive reader that comes back to them read zeros.
	sort.Slice(piecesRemove, func(i, j int) bool {
		if piecesRemove[i].Complete != piecesRemove[j].Complete {
			return piecesRemove[i].Complete
		}
		return piecesRemove[i].Accessed < piecesRemove[j].Accessed
	})

	c.filled = fill
	return piecesRemove
}

func (c *Cache) setLoadPriority(ranges []Range) {
	readers := c.readersSnapshot()
	pieces := c.getPieces()
	if len(readers) == 0 || pieces == nil {
		return
	}
	c.muPrio.Lock()
	defer c.muPrio.Unlock()
	for _, r := range readers {
		if !r.isUse {
			continue
		}
		if c.isIdInFileBE(ranges, r.getReaderPiece()) {
			continue
		}
		readerPos := r.getReaderPiece()
		readerRAHPos := r.getReaderRAHPiece()
		end := r.getPiecesRange().End
		count := settings.BTsets.ConnectionsLimit / len(readers) // max concurrent loading blocks
		limit := 0
		for i := readerPos; i < end && limit < count; i++ {
			if !pieces[i].Complete {
				if i == readerPos {
					c.torrent.Piece(i).SetPriority(torrent.PiecePriorityNow)
				} else if i == readerPos+1 {
					c.torrent.Piece(i).SetPriority(torrent.PiecePriorityNext)
				} else if i > readerPos && i <= readerRAHPos {
					c.torrent.Piece(i).SetPriority(torrent.PiecePriorityReadahead)
				} else if i > readerRAHPos && i <= readerRAHPos+5 && c.torrent.PieceState(i).Priority != torrent.PiecePriorityHigh {
					c.torrent.Piece(i).SetPriority(torrent.PiecePriorityHigh)
				} else if i > readerRAHPos+5 && c.torrent.PieceState(i).Priority != torrent.PiecePriorityNormal {
					c.torrent.Piece(i).SetPriority(torrent.PiecePriorityNormal)
				}
				limit++
			}
		}
	}
}

func (c *Cache) isIdInFileBE(ranges []Range, id int) bool {
	// keep 8/16 MB
	FileRangeNotDelete := int64(c.pieceLength)
	if FileRangeNotDelete < 8<<20 {
		FileRangeNotDelete = 8 << 20
	}

	for _, rng := range ranges {
		ss := int(rng.File.Offset() / c.pieceLength)
		se := int((rng.File.Offset() + FileRangeNotDelete) / c.pieceLength)

		es := int((rng.File.Offset() + rng.File.Length() - FileRangeNotDelete) / c.pieceLength)
		ee := int((rng.File.Offset() + rng.File.Length()) / c.pieceLength)

		if id >= ss && id < se || id > es && id <= ee {
			return true
		}
	}
	return false
}

//////////////////
// Reader section
////////

func (c *Cache) NewReader(file *torrent.File) *Reader {
	return newReader(file, c)
}

func (c *Cache) GetUseReaders() int {
	if c == nil {
		return 0
	}
	c.muReaders.RLock()
	defer c.muReaders.RUnlock()
	readers := 0
	for reader := range c.readers {
		if reader.isUse {
			readers++
		}
	}
	return readers
}

func (c *Cache) Readers() int {
	if c == nil {
		return 0
	}
	c.muReaders.RLock()
	defer c.muReaders.RUnlock()
	return len(c.readers)
}

func (c *Cache) CloseReader(r *Reader) {
	r.cache.muReaders.Lock()
	delete(r.cache.readers, r)
	r.cache.muReaders.Unlock()
	// Reader.Close touches anacrolix internals, keep it outside muReaders
	r.Close()
	go c.clearPriority()
}

func (c *Cache) clearPriority() {
	if c.torrent == nil {
		return
	}
	// This used to sleep for a second before clearing priorities. A reader
	// created during that window could have its PiecePriorityNow/Next/Readahead
	// reset to None right after setLoadPriority had assigned them, starving the
	// player. A mutex provides the same ordering without the race window.
	c.muPrio.Lock()
	defer c.muPrio.Unlock()
	ranges := make([]Range, 0)
	for _, r := range c.readersSnapshot() {
		r.checkReader()
		if r.isUse {
			ranges = append(ranges, r.getPiecesRange())
		}
	}
	ranges = mergeRange(ranges)

	for id, p := range c.getPieces() {
		// preload / full download: keep downloading, whatever the readers do
		if p.wanted.Load() && !p.Complete {
			continue
		}
		if len(ranges) > 0 {
			if !inRanges(ranges, id) {
				if c.torrent.PieceState(id).Priority != torrent.PiecePriorityNone {
					c.torrent.Piece(id).SetPriority(torrent.PiecePriorityNone)
				}
			}
		} else {
			if c.torrent.PieceState(id).Priority != torrent.PiecePriorityNone {
				c.torrent.Piece(id).SetPriority(torrent.PiecePriorityNone)
			}
		}
	}
}

func (c *Cache) GetCapacity() int64 {
	if c == nil {
		return 0
	}
	return c.capacity
}
