package torrents

import "testing"

func TestFreeSpaceLinux(t *testing.T) {
	if free, ok := freeDisk(t.TempDir()); !ok || free <= 0 {
		t.Errorf("disk: %d %v", free, ok)
	}
	if _, ok := freeDisk("/no/such/dir"); ok {
		t.Error("missing dir reported free space")
	}
}
