package torr

import (
	"errors"
	"sort"
	"sync"
	"time"

	utils2 "plenka/internal/torrserver/utils"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"

	"plenka/internal/torrserver/log"
	"plenka/internal/torrserver/settings"
	"plenka/internal/torrserver/torr/state"
	cacheSt "plenka/internal/torrserver/torr/storage/state"
	"plenka/internal/torrserver/torr/storage/torrstor"
	"plenka/internal/torrserver/torr/utils"
)

type Torrent struct {
	Title  string
	Poster string
	Data   string
	*torrent.TorrentSpec

	Stat      state.TorrentStat
	Timestamp int64
	Size      int64

	*torrent.Torrent
	muTorrent sync.Mutex

	bt    *BTServer
	cache *torrstor.Cache

	lastTimeSpeed       time.Time
	DownloadSpeed       float64
	UploadSpeed         float64
	BytesReadUsefulData int64
	BytesWrittenData    int64

	PreloadSize    int64
	PreloadedBytes int64

	// fullDownload is set while the whole torrent is downloaded and kept on disk;
	// the torrent then never expires.
	fullDownload bool
	// keepBefore is what was kept when the full download started, and preloaded what
	// the preloads of this session asked for: cancelling the full download keeps both.
	keepBefore map[int]bool
	preloaded  map[int]bool

	expiredTime time.Time

	closed <-chan struct{}

	progressTicker *time.Ticker
}

func NewTorrent(spec *torrent.TorrentSpec, bt *BTServer) (*Torrent, error) {
	// https://github.com/anacrolix/torrent/issues/747
	if bt == nil || bt.client == nil {
		return nil, errors.New("BT client not connected")
	}
	switch settings.BTsets.RetrackersMode {
	case 1:
		spec.Trackers = append(spec.Trackers, [][]string{utils.GetDefTrackers()}...)
	case 2:
		spec.Trackers = nil
	case 3:
		spec.Trackers = [][]string{utils.GetDefTrackers()}
	}

	trackers := utils.GetTrackerFromFile()
	if len(trackers) > 0 {
		spec.Trackers = append(spec.Trackers, [][]string{trackers}...)
	}

	if len(spec.InfoBytes) == 0 {
		if db := GetTorrentDB(spec.InfoHash); db != nil && db.TorrentSpec != nil {
			spec.InfoBytes = db.TorrentSpec.InfoBytes
		}
	}

	// The read lock is taken and released explicitly, NOT with defer: the code below
	// takes bt.mu.Lock() for the torrents map, and sync.RWMutex is not reentrant,
	// so holding an RLock across that would self-deadlock.
	bt.mu.RLock()
	if bt.client == nil {
		bt.mu.RUnlock()
		return nil, errors.New("BT client not connected")
	}
	goTorrent, _, err := bt.client.AddTorrentSpec(spec)
	bt.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	bt.mu.Lock()
	defer bt.mu.Unlock()
	if tor, ok := bt.torrents[spec.InfoHash]; ok {
		return tor, nil
	}

	timeout := time.Second * time.Duration(settings.BTsets.TorrentDisconnectTimeout)
	if timeout > time.Minute {
		timeout = time.Minute
	}

	torr := new(Torrent)
	torr.Torrent = goTorrent
	torr.Stat = state.TorrentAdded
	torr.lastTimeSpeed = time.Now()
	torr.bt = bt
	torr.closed = goTorrent.Closed()
	torr.TorrentSpec = spec
	torr.AddExpiredTime(timeout)
	torr.Timestamp = time.Now().Unix()

	go torr.watch()

	bt.torrents[spec.InfoHash] = torr
	return torr, nil
}

func (t *Torrent) WaitInfo() bool {
	if t == nil {
		return false
	}
	// drop() (Close, "drop all torrents") sets t.Torrent to nil at any moment, e.g. on
	// a drop request while stream requests wait here. Take the pointer once under
	// muTorrent and use only it below: it stays valid after Drop, and t.closed fires.
	t.muTorrent.Lock()
	torrent := t.Torrent
	t.muTorrent.Unlock()
	if torrent == nil {
		return false
	}

	// Close torrent if no info in 1 minute + TorrentDisconnectTimeout config option
	tm := time.NewTimer(time.Minute + time.Second*time.Duration(settings.BTsets.TorrentDisconnectTimeout))

	select {
	case <-torrent.GotInfo():
		if t.TorrentSpec != nil && len(t.TorrentSpec.InfoBytes) == 0 {
			t.TorrentSpec.InfoBytes = torrent.Metainfo().InfoBytes
		}
		if t.bt != nil {
			// bt.storage is replaced by Connect under bt.mu; snapshot it under the
			// lock and use the snapshot afterwards, because the storage object keeps
			// its own locking and is never closed. Reading the field unlocked was
			// reported by -race and can tear the interface value.
			t.bt.mu.RLock()
			storage := t.bt.storage
			t.bt.mu.RUnlock()
			if storage != nil {
				cache := storage.GetCache(t.Hash())
				if cache == nil {
					// torrent was dropped while waiting for info: CloseHash already removed the cache
					return false
				}
				t.cache = cache
				cache.SetTorrent(torrent)
			}
		}
		return true
	case <-t.closed:
		return false
	case <-tm.C:
		return false
	}
}

func (t *Torrent) GotInfo() bool {
	// log.TLogln("GotInfo state:", t.Stat)
	if t == nil {
		return false
	}
	// Every stream request calls this, often several at once for one file. Once the info
	// is in, only report it: flipping Stat through TorrentGettingInfo on each call raced
	// with Status, Preload and Close, and could turn a closed torrent back into a working one.
	t.muTorrent.Lock()
	switch {
	case t.Stat == state.TorrentClosed:
		t.muTorrent.Unlock()
		return false
	case t.Stat == state.TorrentPreload:
		// assume we have info in preload state
		// and dont override with TorrentWorking
		t.muTorrent.Unlock()
		return true
	case t.Stat == state.TorrentWorking && t.Torrent != nil && t.Torrent.Info() != nil:
		t.muTorrent.Unlock()
		t.AddExpiredTime(time.Second * time.Duration(settings.BTsets.TorrentDisconnectTimeout))
		return true
	}
	t.Stat = state.TorrentGettingInfo
	t.muTorrent.Unlock()

	if t.WaitInfo() {
		t.muTorrent.Lock()
		if t.Stat == state.TorrentGettingInfo {
			t.Stat = state.TorrentWorking
		}
		closed := t.Stat == state.TorrentClosed
		t.muTorrent.Unlock()
		if closed {
			return false
		}
		t.AddExpiredTime(time.Second * time.Duration(settings.BTsets.TorrentDisconnectTimeout))
		return true
	} else {
		t.Close()
		return false
	}
}

func (t *Torrent) AddExpiredTime(duration time.Duration) {
	newExpiredTime := time.Now().Add(duration)
	t.muTorrent.Lock()
	if t.expiredTime.Before(newExpiredTime) {
		t.expiredTime = newExpiredTime
	}
	t.muTorrent.Unlock()
}

func (t *Torrent) watch() {
	t.progressTicker = time.NewTicker(time.Second)
	defer t.progressTicker.Stop()

	for {
		select {
		case <-t.progressTicker.C:
			go t.progressEvent()
		case <-t.closed:
			return
		}
	}
}

func (t *Torrent) progressEvent() {
	if t.expired() {
		if t.TorrentSpec != nil {
			log.TLogln("Torrent close by timeout", t.TorrentSpec.InfoHash.HexString())
		}
		t.bt.RemoveTorrent(t.Hash())
		return
	}

	t.muTorrent.Lock()
	if t.Torrent != nil && t.Torrent.Info() != nil {
		st := t.Torrent.Stats()
		deltaDlBytes := st.BytesRead.Int64() - t.BytesReadUsefulData
		deltaUpBytes := st.BytesWritten.Int64() - t.BytesWrittenData
		deltaTime := time.Since(t.lastTimeSpeed).Seconds()

		t.DownloadSpeed = float64(deltaDlBytes) / deltaTime
		t.UploadSpeed = float64(deltaUpBytes) / deltaTime

		t.BytesReadUsefulData = st.BytesRead.Int64()
		t.BytesWrittenData = st.BytesWritten.Int64()

	} else {
		t.DownloadSpeed = 0
		t.UploadSpeed = 0
	}
	t.muTorrent.Unlock()

	t.lastTimeSpeed = time.Now()
	t.updateRA()
}

func (t *Torrent) updateRA() {
	// t.muTorrent.Lock()
	// defer t.muTorrent.Unlock()
	// if t.Torrent != nil && t.Torrent.Info() != nil {
	// 	pieceLen := t.Torrent.Info().PieceLength
	// 	adj := pieceLen * int64(t.Torrent.Stats().ActivePeers) / int64(1+t.cache.Readers())
	// 	switch {
	// 	case adj < pieceLen:
	// 		adj = pieceLen
	// 	case adj > pieceLen*4:
	// 		adj = pieceLen * 4
	// 	}
	// 	go t.cache.AdjustRA(adj)
	// }
	adj := int64(16 << 20) // 16 MB fixed RA
	go t.cache.AdjustRA(adj)
}

func (t *Torrent) expired() bool {
	if t.cache == nil {
		return false
	}
	t.muTorrent.Lock()
	stat, expiredTime, full := t.Stat, t.expiredTime, t.fullDownload
	t.muTorrent.Unlock()
	return !full && t.cache.Readers() == 0 && expiredTime.Before(time.Now()) && (stat == state.TorrentWorking || stat == state.TorrentClosed)
}

func (t *Torrent) Files() []*torrent.File {
	if t.Torrent != nil && t.Torrent.Info() != nil {
		files := t.Torrent.Files()
		return files
	}
	return nil
}

func (t *Torrent) Hash() metainfo.Hash {
	if t.Torrent != nil {
		return t.Torrent.InfoHash()
	}
	if t.TorrentSpec != nil {
		return t.TorrentSpec.InfoHash
	}
	return [20]byte{}
}

func (t *Torrent) Length() int64 {
	if t.Info() == nil {
		return 0
	}
	return t.Torrent.Length()
}

func (t *Torrent) NewReader(file *torrent.File) *torrstor.Reader {
	if t.Stat == state.TorrentClosed {
		return nil
	}
	reader := t.cache.NewReader(file)
	return reader
}

func (t *Torrent) CloseReader(reader *torrstor.Reader) {
	t.cache.CloseReader(reader)
	t.AddExpiredTime(time.Second * time.Duration(settings.BTsets.TorrentDisconnectTimeout))
}

// DownloadAll downloads every file of the torrent and keeps it on disk (see
// torrstor.Cache.Keep), outside the cache, until CancelDownloadAll or the torrent is
// removed. It needs the metadata; it returns false without it.
func (t *Torrent) DownloadAll() bool {
	t.muTorrent.Lock()
	tor := t.Torrent
	if tor == nil || tor.Info() == nil || t.cache == nil {
		t.muTorrent.Unlock()
		return false
	}
	if t.fullDownload {
		t.muTorrent.Unlock()
		return true
	}
	t.fullDownload = true
	t.keepBefore = t.cache.KeptIDs()
	t.muTorrent.Unlock()

	t.cache.Keep(allPieces(tor), false)
	return true
}

// DownloadNeed is the disk space DownloadAll still needs.
func (t *Torrent) DownloadNeed() int64 {
	t.muTorrent.Lock()
	tor := t.Torrent
	t.muTorrent.Unlock()
	if tor == nil || tor.Info() == nil {
		return 0
	}
	done, total := t.cache.KeepProgress(allPieces(tor))
	return total - done
}

// KeepPending is what the preload and the full download still have to fetch.
func (t *Torrent) KeepPending() int64 {
	return t.cache.KeepPending()
}

func allPieces(tor *torrent.Torrent) []int {
	ids := make([]int, tor.NumPieces())
	for i := range ids {
		ids[i] = i
	}
	return ids
}

// CancelDownloadAll stops a full download: the pieces it fetched are deleted, except
// what was kept before it and what preloads asked for, and the torrent expires again
// after the idle timeout.
func (t *Torrent) CancelDownloadAll() {
	t.muTorrent.Lock()
	was := t.fullDownload
	t.fullDownload = false
	stay := make(map[int]bool, len(t.keepBefore)+len(t.preloaded))
	for id := range t.keepBefore {
		stay[id] = true
	}
	for id := range t.preloaded {
		stay[id] = true
	}
	t.keepBefore = nil
	t.muTorrent.Unlock()
	if !was {
		return
	}
	t.cache.Unkeep(stay)
	t.AddExpiredTime(time.Second * time.Duration(settings.BTsets.TorrentDisconnectTimeout))
}

// FullDownload reports whether the whole torrent is being downloaded and kept.
func (t *Torrent) FullDownload() bool {
	t.muTorrent.Lock()
	defer t.muTorrent.Unlock()
	return t.fullDownload
}

func (t *Torrent) GetCache() *torrstor.Cache {
	return t.cache
}

func (t *Torrent) drop() {
	t.muTorrent.Lock()
	defer t.muTorrent.Unlock()
	if t.Torrent != nil {
		t.Torrent.Drop()
		t.Torrent = nil
	}
}

func (t *Torrent) Close() bool {
	if t == nil {
		return false
	}
	t.muTorrent.Lock()
	if t.Stat == state.TorrentClosed {
		t.muTorrent.Unlock()
		return true
	}
	t.Stat = state.TorrentClosed
	t.muTorrent.Unlock()

	if t.bt != nil {
		t.bt.mu.Lock()
		if _, ok := t.bt.torrents[t.Hash()]; ok {
			delete(t.bt.torrents, t.Hash())
		}
		t.bt.mu.Unlock()
	}

	t.drop()
	return true
}

func (t *Torrent) Status() *state.TorrentStatus {
	t.muTorrent.Lock()
	defer t.muTorrent.Unlock()

	st := new(state.TorrentStatus)

	st.Stat = t.Stat
	st.StatString = t.Stat.String()
	st.Title = t.Title
	st.Poster = t.Poster
	st.Data = t.Data
	st.Timestamp = t.Timestamp
	st.TorrentSize = t.Size

	if t.TorrentSpec != nil {
		st.Hash = t.TorrentSpec.InfoHash.HexString()
	}
	if t.Torrent != nil {
		st.Name = t.Torrent.Name()
		st.Hash = t.Torrent.InfoHash().HexString()
		st.LoadedSize = t.Torrent.BytesCompleted()

		st.PreloadedBytes = t.PreloadedBytes
		st.PreloadSize = t.PreloadSize
		st.FullDownload = t.fullDownload
		st.DownloadSpeed = t.DownloadSpeed
		st.UploadSpeed = t.UploadSpeed

		tst := t.Torrent.Stats()
		st.BytesWritten = tst.BytesWritten.Int64()
		st.BytesWrittenData = tst.BytesWrittenData.Int64()
		st.BytesRead = tst.BytesRead.Int64()
		st.BytesReadData = tst.BytesReadData.Int64()
		st.BytesReadUsefulData = tst.BytesReadUsefulData.Int64()
		st.ChunksWritten = tst.ChunksWritten.Int64()
		st.ChunksRead = tst.ChunksRead.Int64()
		st.ChunksReadUseful = tst.ChunksReadUseful.Int64()
		st.ChunksReadWasted = tst.ChunksReadWasted.Int64()
		st.PiecesDirtiedGood = tst.PiecesDirtiedGood.Int64()
		st.PiecesDirtiedBad = tst.PiecesDirtiedBad.Int64()
		st.TotalPeers = tst.TotalPeers
		st.PendingPeers = tst.PendingPeers
		st.ActivePeers = tst.ActivePeers
		st.ConnectedSeeders = tst.ConnectedSeeders
		st.HalfOpenPeers = tst.HalfOpenPeers

		if t.Torrent.Info() != nil {
			st.TorrentSize = t.Torrent.Length()

			// Files returns the torrent's own slice, which also backs each piece's file list,
			// so sort a copy: sorting in place reorders it under concurrent readers.
			files := append([]*torrent.File(nil), t.Files()...)
			sort.Slice(files, func(i, j int) bool {
				return utils2.CompareStrings(files[i].Path(), files[j].Path())
			})
			for i, f := range files {
				st.FileStats = append(st.FileStats, &state.TorrentFileStat{
					Id:     i + 1, // in web id 0 is undefined
					Path:   f.Path(),
					Length: f.Length(),
				})
			}

		}
	}

	return st
}

func (t *Torrent) CacheState() *cacheSt.CacheState {
	if t.Torrent != nil && t.cache != nil {
		st := t.cache.GetState()
		st.Torrent = t.Status()
		return st
	}
	return nil
}
