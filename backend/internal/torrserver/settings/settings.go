package settings

import (
	"fmt"
	"path/filepath"

	"plenka/internal/torrserver/log"
)

func IsDebug() bool {
	if BTsets != nil {
		return BTsets.EnableDebug
	}
	return false
}

var (
	tdb TorrServerDB
	// Path is the data directory holding config.db, trackers.txt, blocklist and the
	// default disk cache.
	Path string
)

// InitSets opens config.db in Path, migrates data left by older versions and loads
// the engine settings (defaults when none are saved).
func InitSets() error {
	bboltDB := NewTDB()
	if bboltDB == nil {
		err := fmt.Errorf("error open bboltDB: %s", filepath.Join(Path, "config.db"))
		log.TLogln(err.Error())
		return err
	}
	migrateLegacy(bboltDB.(*TDB))
	tdb = NewDBReadCache(bboltDB)
	loadBTSets()
	return nil
}

func CloseDB() {
	if tdb != nil {
		tdb.CloseDB()
		tdb = nil
	}
	globalBboltDB = nil
}
