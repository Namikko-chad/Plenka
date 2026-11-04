package settings

import (
	"encoding/json"
	"path/filepath"

	"plenka/internal/torrserver/log"
)

type BTSets struct {
	// Cache
	CacheSize       int64 // in byte, def 64 MB
	ReaderReadAHead int   // in percent, 5%-100%, [...S__X__E...] [S-E] not clean
	PreloadCache    int   // in percent

	// Disk
	UseDisk           bool
	TorrentsSavePath  string
	RemoveCacheOnDrop bool

	// Torrent
	ForceEncrypt             bool
	RetrackersMode           int    // 0 - don`t add, 1 - add retrackers (def), 2 - remove retrackers 3 - replace retrackers
	TrackersListURL          string // optional custom remote trackers list URL; empty = use built-in mirrors; tried first, then mirrors
	DefaultTrackers          string // newline-separated announce URLs used as local/fallback list
	TorrentDisconnectTimeout int    // in seconds
	EnableDebug              bool   // debug logs

	// BT Config
	EnableIPv6        bool
	DisableTCP        bool
	DisableUTP        bool
	DisableUPNP       bool
	DisableDHT        bool
	DisablePEX        bool
	DisableUpload     bool
	DownloadRateLimit int // in kb, 0 - inf
	UploadRateLimit   int // in kb, 0 - inf
	ConnectionsLimit  int
	PeersListenPort   int

	// LPD
	EnableLPD bool
	LPDIPv6   bool

	// Reader
	ResponsiveMode bool // enable Responsive reader (don't wait pieceComplete)
}

func (v *BTSets) String() string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

// DefaultTrackersListURLs is the built-in remote trackers list mirrors, tried in order.
var DefaultTrackersListURLs = []string{
	"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best_ip.txt",
	"https://ngosang.github.io/trackerslist/trackers_best_ip.txt",
	"https://cdn.jsdelivr.net/gh/ngosang/trackerslist@master/trackers_best_ip.txt",
	"https://raw.githack.com/ngosang/trackerslist/master/trackers_best_ip.txt",
}

const DefaultTrackersText = `http://retracker.local/announce
http://bt4.t-ru.org/ann?magnet
http://retracker.mgts.by:80/announce
http://tracker.city9x.com:2710/announce
http://tracker.electro-torrent.pl:80/announce
http://tracker.internetwarriors.net:1337/announce
http://tracker2.itzmx.com:6961/announce
udp://opentor.org:2710
udp://public.popcorn-tracker.org:6969/announce
udp://tracker.opentrackr.org:1337/announce
http://bt.svao-ix.ru/announce
udp://explodie.org:6969/announce
wss://tracker.btorrent.xyz
wss://tracker.openwebtorrent.com`

var BTsets *BTSets

// KeepDir is where preloaded and fully downloaded pieces are kept, whatever UseDisk
// says: TorrentsSavePath when set, else "torrents" in the data dir. Each torrent keeps
// them in <KeepDir>/<hash>/keep.
func KeepDir() string {
	if BTsets != nil && BTsets.TorrentsSavePath != "" {
		return BTsets.TorrentsSavePath
	}
	return filepath.Join(Path, "torrents")
}

// SetBTSets clamps sets to valid values (zero or out-of-range values fall back to
// defaults), makes it the current settings and saves it to config.db.
func SetBTSets(sets *BTSets) {
	normalize(sets)
	BTsets = sets
	save()
}

func normalize(sets *BTSets) {
	if sets.CacheSize <= 0 {
		sets.CacheSize = 64 * 1024 * 1024
	}
	if sets.ConnectionsLimit <= 0 {
		sets.ConnectionsLimit = 25
	}
	if sets.TorrentDisconnectTimeout <= 0 {
		sets.TorrentDisconnectTimeout = 30
	}
	sets.ReaderReadAHead = clamp(sets.ReaderReadAHead, 5, 100)
	sets.PreloadCache = clamp(sets.PreloadCache, 0, 100)
	sets.RetrackersMode = clamp(sets.RetrackersMode, 0, 3)
	sets.DownloadRateLimit = clamp(sets.DownloadRateLimit, 0, 1<<30)
	sets.UploadRateLimit = clamp(sets.UploadRateLimit, 0, 1<<30)
	sets.PeersListenPort = clamp(sets.PeersListenPort, 0, 65535)
	if sets.TorrentsSavePath == "" {
		sets.UseDisk = false
	}
}

func clamp(v, lo, hi int) int {
	return max(lo, min(v, hi))
}

// DefaultBTSets returns the default engine settings.
func DefaultBTSets() *BTSets {
	return &BTSets{
		CacheSize:                64 * 1024 * 1024, // 64 MB
		PreloadCache:             50,
		ConnectionsLimit:         25,
		RetrackersMode:           1,
		DefaultTrackers:          DefaultTrackersText,
		TorrentDisconnectTimeout: 30,
		ReaderReadAHead:          95, // 95%
		ResponsiveMode:           true,
		EnableLPD:                true,
	}
}

// SetDefaultConfig makes the default settings current and saves them.
func SetDefaultConfig() {
	BTsets = DefaultBTSets()
	save()
}

func save() {
	buf, err := json.Marshal(BTsets)
	if err != nil {
		log.TLogln("Error marshal btsets", err)
		return
	}
	tdb.Set("Settings", "BitTorr", buf)
}

func loadBTSets() {
	buf := tdb.Get("Settings", "BitTorr")
	if len(buf) > 0 {
		sets := new(BTSets)
		err := json.Unmarshal(buf, sets)
		if err == nil {
			// Upgrade older configs that never had tracker list fields.
			// Empty TrackersListURL now means "use built-in mirrors".
			if sets.TrackersListURL == "" && sets.DefaultTrackers == "" {
				sets.DefaultTrackers = DefaultTrackersText
			}
			normalize(sets)
			BTsets = sets
			return
		}
		log.TLogln("Error unmarshal btsets", err)
	}
	// initialize defaults on error
	SetDefaultConfig()
}
