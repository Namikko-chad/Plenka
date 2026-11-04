package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"
)

// A data dir from older builds keeps the engine settings in settings.json and viewed
// marks in config.db "Viewed"; InitSets moves the settings into config.db and drops
// the rest.
func TestInitSetsMigratesLegacyLayout(t *testing.T) {
	dir := t.TempDir()
	Path = dir
	legacy := `{"BitTorr": {"CacheSize": 134217728, "ReaderReadAHead": 80, "EnableDLNA": true, "StoreSettingsInJson": true, "TMDBSettings": {"APIKey": "k"}}}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := bolt.Open(filepath.Join(dir, "config.db"), 0o666, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte("Viewed"))
		_ = b.Put([]byte("abc"), []byte(`{"1":{}}`))
		b, _ = tx.CreateBucketIfNotExists([]byte("Torrents"))
		return b.Put([]byte("abc"), []byte(`{"title":"kept"}`))
	})
	db.Close()

	if err := InitSets(); err != nil {
		t.Fatal(err)
	}
	if BTsets.CacheSize != 134217728 || BTsets.ReaderReadAHead != 80 || BTsets.DefaultTrackers == "" {
		t.Errorf("settings: %+v", BTsets)
	}
	if _, err := os.Stat(filepath.Join(dir, "settings.json")); !os.IsNotExist(err) {
		t.Errorf("settings.json not removed: %v", err)
	}
	SetBTSets(BTsets) // re-save drops the removed fields
	var raw map[string]any
	if err := json.Unmarshal(tdb.Get("Settings", "BitTorr"), &raw); err != nil || raw["EnableDLNA"] != nil || raw["CacheSize"] == nil {
		t.Errorf("saved settings: %v %v", raw, err)
	}
	if tdb.List("Viewed") != nil {
		t.Errorf("Viewed bucket kept")
	}
	if len(tdb.List("Torrents")) != 1 {
		t.Errorf("torrents lost")
	}
	CloseDB()

	// A second start with no legacy files keeps the settings.
	if err := InitSets(); err != nil {
		t.Fatal(err)
	}
	defer CloseDB()
	if BTsets.CacheSize != 134217728 {
		t.Errorf("settings after restart: %+v", BTsets)
	}
}

func TestNormalizeAndDefaults(t *testing.T) {
	s := &BTSets{ReaderReadAHead: 1, PreloadCache: 300, RetrackersMode: -1, PeersListenPort: 70000, UseDisk: true, DownloadRateLimit: -5}
	normalize(s)
	if s.CacheSize != 64<<20 || s.ReaderReadAHead != 5 || s.PreloadCache != 100 || s.RetrackersMode != 0 ||
		s.PeersListenPort != 65535 || s.UseDisk || s.DownloadRateLimit != 0 || s.ConnectionsLimit != 25 || s.TorrentDisconnectTimeout != 30 {
		t.Errorf("normalize: %+v", s)
	}
	d := DefaultBTSets()
	if d.CacheSize != 64<<20 || d.ReaderReadAHead != 95 || d.PreloadCache != 50 || d.RetrackersMode != 1 || !d.ResponsiveMode || !d.EnableLPD {
		t.Errorf("defaults: %+v", d)
	}
}
