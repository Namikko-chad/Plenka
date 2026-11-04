package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"

	"plenka/internal/torrserver/log"
)

// migrateLegacy brings a data dir written by older builds to the current layout,
// where everything lives in config.db:
//   - settings.json (the engine settings were kept there by default) is copied into
//     config.db Settings/BitTorr and removed;
//   - the unused "Viewed" bucket and viewed.json are dropped.
//
// Saved torrents were always in config.db "Torrents" and are left as they are.
func migrateLegacy(db *TDB) {
	jsonPath := filepath.Join(Path, "settings.json")
	if buf, err := os.ReadFile(jsonPath); err == nil {
		var root map[string]json.RawMessage
		if err := json.Unmarshal(buf, &root); err != nil {
			log.TLogln("migrate settings.json:", err)
		} else {
			if sets, ok := root["BitTorr"]; ok && len(sets) > 0 && string(sets) != "null" {
				// settings.json was authoritative whenever it held the settings.
				db.Set("Settings", "BitTorr", sets)
				log.TLogln("migrated engine settings from settings.json to config.db")
			}
			if err := os.Remove(jsonPath); err != nil {
				log.TLogln("migrate settings.json:", err)
			}
		}
	}

	if err := os.Remove(filepath.Join(Path, "viewed.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.TLogln("remove viewed.json:", err)
	}
	if db.db != nil {
		_ = db.db.Update(func(tx *bolt.Tx) error {
			if tx.Bucket([]byte("Viewed")) == nil {
				return nil
			}
			return tx.DeleteBucket([]byte("Viewed"))
		})
	}
}
