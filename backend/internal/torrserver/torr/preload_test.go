package torr

import (
	"slices"
	"testing"
)

func TestFilePreloadPieces(t *testing.T) {
	const mb = 1 << 20
	seen := map[int]bool{}
	// 16 GB file of 16 MB pieces at 50 %: the first 512 pieces and the last one
	ids := filePreloadPieces(nil, seen, 0, 16<<30, 16*mb, 50)
	if len(ids) != 513 || ids[0] != 0 || ids[511] != 511 || ids[512] != 1023 {
		t.Fatalf("got %d pieces: %v … %v", len(ids), ids[:2], ids[len(ids)-2:])
	}

	// a second file sharing piece 1023 with the first one does not list it again
	ids = filePreloadPieces(nil, seen, 16<<30-mb, 33*mb, 16*mb, 50)
	if want := []int{1024, 1025}; !slices.Equal(ids, want) {
		t.Fatalf("second file: %v, want %v", ids, want)
	}

	// a small file is preloaded whole, with 0 % only its end
	if ids := filePreloadPieces(nil, map[int]bool{}, 0, 4*mb, mb, 0); !slices.Equal(ids, []int{0, 1, 2, 3}) {
		t.Fatalf("small file: %v", ids)
	}
}
