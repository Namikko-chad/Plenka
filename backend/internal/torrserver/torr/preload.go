package torr

import (
	"fmt"
	"time"

	"plenka/internal/torrserver/log"
	"plenka/internal/torrserver/settings"
	"plenka/internal/torrserver/torr/state"
	utils2 "plenka/internal/torrserver/utils"

	"github.com/anacrolix/torrent"
)

// Preload downloads the start (PreloadCache % of the length) and the last 8/16 MB of
// every file to disk, as a full download would, the file index first. That data is
// kept outside the cache (see torrstor.Cache.Keep): it is not evicted, outlives the
// torrent being unloaded and goes only with the torrent. The torrent is in the
// preload state until all of it is in, or the torrent is closed.
func (t *Torrent) Preload(index int) {
	if settings.BTsets.PreloadCache <= 0 {
		return
	}

	if t.Stat == state.TorrentGettingInfo {
		if !t.WaitInfo() {
			return
		}
		// wait change status
		time.Sleep(100 * time.Millisecond)
	}

	t.muTorrent.Lock()
	if t.Stat != state.TorrentWorking || t.Torrent == nil || t.Torrent.Info() == nil {
		t.muTorrent.Unlock()
		return
	}
	// drop() (Close, "drop all torrents" in SetSettings) sets t.Torrent to nil at any moment,
	// and "drop all torrents" leaves Stat as is. Use only this pointer below: it stays valid after Drop.
	torrent := t.Torrent
	t.Stat = state.TorrentPreload
	t.muTorrent.Unlock()

	defer func() {
		t.muTorrent.Lock()
		if t.Stat == state.TorrentPreload {
			t.Stat = state.TorrentWorking
		}
		t.muTorrent.Unlock()
	}()

	// preloadPieces takes muTorrent (through Status): call it unlocked
	first, rest := t.preloadPieces(torrent, index)
	all := append(append([]int(nil), first...), rest...)
	done, total := t.cache.KeepProgress(all)
	t.muTorrent.Lock()
	if t.preloaded == nil {
		t.preloaded = make(map[int]bool, len(all))
	}
	for _, id := range all {
		t.preloaded[id] = true
	}
	t.PreloadedBytes, t.PreloadSize = done, total
	t.muTorrent.Unlock()

	t.cache.Keep(first, true)
	t.cache.Keep(rest, false)

	timeout := time.Second * time.Duration(settings.BTsets.TorrentDisconnectTimeout)
	if timeout > time.Minute {
		timeout = time.Minute
	}
	hash := torrent.InfoHash().HexString()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for tick := 0; ; tick++ {
		done, total := t.cache.KeepProgress(all)
		t.muTorrent.Lock()
		t.PreloadedBytes, t.PreloadSize = done, total
		active := t.Stat == state.TorrentPreload && t.Torrent == torrent
		t.muTorrent.Unlock()
		if done >= total {
			st := torrent.Stats()
			log.TLogln("End preload:", hash, utils2.Format(float64(total)),
				"Peers:", st.ActivePeers, "/", st.TotalPeers, "[ Seeds:", st.ConnectedSeeders, "]")
			return
		}
		if !active {
			log.TLogln("Preload stopped:", hash, utils2.Format(float64(done)), "/", utils2.Format(float64(total)))
			return
		}
		t.AddExpiredTime(timeout)
		if tick%10 == 0 {
			st := torrent.Stats()
			log.TLogln("Preload:", fmt.Sprint(hash, " ",
				utils2.Format(float64(done)), " / ", utils2.Format(float64(total)),
				" Speed: ", utils2.Format(t.DownloadSpeed),
				" Peers: ", st.ActivePeers, " / ", st.TotalPeers,
				" [ Seeds: ", st.ConnectedSeeders, " ]"))
		}
		<-ticker.C
	}
}

// PreloadNeed is the disk space Preload(index) still needs. It needs the metadata.
func (t *Torrent) PreloadNeed(index int) int64 {
	t.muTorrent.Lock()
	tor := t.Torrent
	t.muTorrent.Unlock()
	if tor == nil || tor.Info() == nil || settings.BTsets.PreloadCache <= 0 {
		return 0
	}
	first, rest := t.preloadPieces(tor, index)
	done, total := t.cache.KeepProgress(append(first, rest...))
	return total - done
}

// preloadPieces lists the pieces Preload fetches: first for the file index (the first
// file when there is no such file), rest for the other files.
func (t *Torrent) preloadPieces(tor *torrent.Torrent, index int) (first, rest []int) {
	files := tor.Files()
	if len(files) == 0 {
		return nil, nil
	}
	file := t.findFileIndex(tor, index)
	if file == nil {
		file = files[0]
	}
	pieceLength := tor.Info().PieceLength
	pct := int64(settings.BTsets.PreloadCache)
	seen := make(map[int]bool)
	first = filePreloadPieces(nil, seen, file.Offset(), file.Length(), pieceLength, pct)
	for _, f := range files {
		if f != file {
			rest = filePreloadPieces(rest, seen, f.Offset(), f.Length(), pieceLength, pct)
		}
	}
	return first, rest
}

// filePreloadPieces appends to ids the pieces holding the first pct % of a file at
// offset and its last 8/16 MB (the container index: mkv cues, mp4 moov), skipping
// those in seen.
func filePreloadPieces(ids []int, seen map[int]bool, offset, length, pieceLength, pct int64) []int {
	add := func(from, to int64) {
		if to <= from {
			return
		}
		for id := int(from / pieceLength); id <= int((to-1)/pieceLength); id++ {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	end := offset + length
	head := offset + length*pct/100
	add(offset, head)
	add(max(head, end-max(pieceLength, 8<<20)), end)
	return ids
}

func (t *Torrent) findFileIndex(torrent *torrent.Torrent, index int) *torrent.File {
	st := t.Status()
	var stFile *state.TorrentFileStat
	for _, f := range st.FileStats {
		if index == f.Id {
			stFile = f
			break
		}
	}
	if stFile == nil {
		return nil
	}
	for _, file := range torrent.Files() {
		if file.Path() == stFile.Path {
			return file
		}
	}
	return nil
}
