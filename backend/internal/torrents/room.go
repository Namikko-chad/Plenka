package torrents

import (
	"os"

	sets "plenka/internal/torrserver/settings"
	"plenka/internal/torrserver/torr"
)

// keepReserve stays free on disk after a preload or a full download.
const keepReserve = 512 << 20

// freeSpace reports the free disk space under the keep dir, where preloads and full
// downloads go. ok is false when it cannot be told; the download is then allowed.
// Tests replace it.
var freeSpace = func() (free int64, ok bool) {
	dir := sets.KeepDir()
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return 0, false
	}
	return freeDisk(dir)
}

// pendingKeeps is what the preloads and full downloads already running elsewhere still
// have to fetch: that space is free now but is already promised.
func pendingKeeps(except string) int64 {
	var sum int64
	for _, t := range torr.ListTorrent() {
		if t.Hash().HexString() == except || t.GetCache() == nil {
			continue
		}
		sum += t.KeepPending()
	}
	return sum
}
