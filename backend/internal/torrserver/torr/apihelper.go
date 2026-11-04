package torr

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"

	"plenka/internal/torrserver/log"
	sets "plenka/internal/torrserver/settings"
	"plenka/internal/torrserver/torr/utils"
)

var bts *BTServer

// bts is read by request handlers without any synchronisation of their own, so it
// must be written exactly once. Connect calls this on every settings save with the
// same receiver, and the repeated assignment was reported by -race as a data race
// against every /stat request.
func InitApiHelper(bt *BTServer) {
	if bts == bt {
		return
	}
	bts = bt
}

// AddTorrent loads spec into the engine (or returns the already loaded torrent).
// Empty title, poster and data are filled from the saved copy, if any.
func AddTorrent(spec *torrent.TorrentSpec, title, poster string, data string) (*Torrent, error) {
	torr, err := NewTorrent(spec, bts)
	if err != nil {
		log.TLogln("error add torrent:", err)
		return nil, err
	}

	torDB := GetTorrentDB(spec.InfoHash)

	if torr.Title == "" {
		torr.Title = title
		if title == "" && torDB != nil {
			torr.Title = torDB.Title
		}
		if torr.Title == "" && torr.Torrent != nil && torr.Torrent.Info() != nil {
			torr.Title = torr.Info().Name
		}
	}

	if torr.Poster == "" {
		torr.Poster = poster
		if torr.Poster == "" && torDB != nil {
			torr.Poster = torDB.Poster
		}
	}

	if torr.Data == "" {
		torr.Data = data
		if torr.Data == "" && torDB != nil {
			torr.Data = torDB.Data
		}
	}
	if torDB != nil && torDB.Timestamp != 0 {
		torr.Timestamp = torDB.Timestamp
	}

	return torr, nil
}

func SaveTorrentToDB(torr *Torrent) {
	log.TLogln("save to db:", torr.Hash())
	AddTorrentDB(torr)
}

// hashFromHex parses an info hash that came from an API request. Malformed
// input is reported instead of panicking like metainfo.NewHashFromHex.
func hashFromHex(hashHex string) (hash metainfo.Hash, ok bool) {
	return hash, hash.FromHexString(hashHex) == nil
}

// FindTorrent returns the loaded torrent with this hash, else its saved copy
// (Stat TorrentInDB), else nil. Unlike the upstream GetTorrent it never starts
// loading a saved torrent. A loaded torrent's idle timer is extended.
func FindTorrent(hashHex string) *Torrent {
	hash, ok := hashFromHex(hashHex)
	if !ok {
		return nil
	}
	if tor := bts.GetTorrent(hash); tor != nil {
		timeout := time.Second * time.Duration(sets.BTsets.TorrentDisconnectTimeout)
		if timeout > time.Minute {
			timeout = time.Minute
		}
		tor.AddExpiredTime(timeout)
		return tor
	}
	return GetTorrentDB(hash)
}

// SetTorrent updates the title, poster and data of a loaded and/or saved torrent;
// empty values keep the current ones. The saved copy is rewritten.
func SetTorrent(hashHex, title, poster, data string) {
	hash, ok := hashFromHex(hashHex)
	if !ok {
		return
	}
	update := func(t *Torrent) {
		if title != "" {
			t.Title = title
		}
		if poster != "" {
			t.Poster = poster
		}
		if data != "" {
			t.Data = data
		}
	}
	if torr := bts.GetTorrent(hash); torr != nil {
		update(torr)
	}
	if torrDb := GetTorrentDB(hash); torrDb != nil {
		update(torrDb)
		AddTorrentDB(torrDb)
	}
}

func RemTorrent(hashHex string) {
	hash, ok := hashFromHex(hashHex)
	if !ok {
		return
	}

	// Download the torrent before deleting it to get the "closed" status
	torr := bts.GetTorrent(hash)
	if torr == nil {
		// If the torrent isn't in memory, just delete it from the database and the files
		RemTorrentDB(hash)
		removeFiles(hashHex)
		return
	}

	closedChan := torr.closed

	// Clear from memory
	if bts.RemoveTorrent(hash) {
		// Waiting for confirmation from the library via the closed channel
		select {
		case <-closedChan:
			// The library has confirmed the closure
			log.TLogln("Torrent closed by library:", hashHex)
		case <-time.After(5 * time.Second):
			log.TLogln("Warning: timeout waiting for torrent close:", hashHex)
		}

		// Now we can safely delete the files from the disk
		removeFiles(hashHex)
	}

	// Delete from the database
	RemTorrentDB(hash)
}

// removeFiles deletes what the torrent has on disk: the disk cache and the kept
// (preloaded, fully downloaded) pieces, both under KeepDir/<hash>.
func removeFiles(hashHex string) {
	if hashHex == "" || hashHex == "/" {
		return
	}
	name := filepath.Join(sets.KeepDir(), hashHex)
	if _, err := os.Stat(name); err == nil {
		log.TLogln("Removing files for:", hashHex)
		os.RemoveAll(name)
	}
}

func ListTorrent() []*Torrent {
	btlist := bts.ListTorrents()
	dblist := ListTorrentsDB()

	for hash, t := range dblist {
		if _, ok := btlist[hash]; !ok {
			btlist[hash] = t
		}
	}
	var ret []*Torrent

	for _, t := range btlist {
		ret = append(ret, t)
	}

	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Timestamp != ret[j].Timestamp {
			return ret[i].Timestamp > ret[j].Timestamp
		} else {
			return ret[i].Title > ret[j].Title
		}
	})

	return ret
}

func DropTorrent(hashHex string) {
	hash, ok := hashFromHex(hashHex)
	if !ok {
		return
	}
	bts.RemoveTorrent(hash)
}

func SetSettings(set *sets.BTSets) {
	sets.SetBTSets(set)
	utils.InvalidateTrackersCache()
	log.TLogln("drop all torrents")
	dropAllTorrent()
	time.Sleep(time.Second * 1)
	log.TLogln("disconect")
	bts.Disconnect()
	log.TLogln("connect")
	reconnect()
	time.Sleep(time.Second * 1)
	log.TLogln("end set settings")
}

// reconnect retries Connect after Disconnect. The uTP socket of the closed client
// closes lazily, only after its peer connections are gone, and until then
// PeersListenPort is busy: "listen udp4 :port: bind: address already in use".
// A failed Connect would leave bts.client nil until restart.
func reconnect() {
	var err error
	for i := 0; i < 30; i++ {
		if err = bts.Connect(); err == nil {
			if i > 0 {
				log.TLogln("connect ok, attempt", i+1)
			}
			return
		}
		if i == 0 {
			log.TLogln("connect error, retrying:", err)
		}
		time.Sleep(time.Second)
	}
	log.TLogln("connect failed:", err)
}

func SetDefSettings() {
	sets.SetDefaultConfig()
	utils.InvalidateTrackersCache()
	log.TLogln("drop all torrents")
	dropAllTorrent()
	time.Sleep(time.Second * 1)
	log.TLogln("disconect")
	bts.Disconnect()
	log.TLogln("connect")
	reconnect()
	time.Sleep(time.Second * 1)
	log.TLogln("end set default settings")
}

func dropAllTorrent() {
	for _, torr := range bts.ListTorrents() {
		torr.drop()
		<-torr.closed
	}
}

// Preload runs torr.Preload for the file index.
func Preload(torr *Torrent, index int) {
	torr.Preload(index)
}
