//go:build !linux

package torrents

func freeDisk(string) (int64, bool) { return 0, false }
