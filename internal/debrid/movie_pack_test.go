package debrid

import (
	"errors"
	"github.com/braedonsaunders/ether/internal/core"
	"testing"
)

// Collection identity is independent of file size, listing order, and explicit
// addon selectors. This also exercises the shared archive-member name matcher.
func TestMoviePackIdentity(t *testing.T) {
	target := &core.ReleaseTarget{MovieTitles: []string{"The Mummy"}, MovieYear: 1999}
	files := []File{
		{Path: "The Mummy Returns.mkv", Size: 9000},
		{Path: "The Mummy (1959)/The Mummy.mkv", Size: 8000},
		{Path: "The Mummy Returns (2001).mkv", Size: 7000},
		{Path: "The Mummy (1999)/The.Mummy.1999.1080p.mkv", Size: 5000},
	}
	f, err := ChooseFile(files, target)
	if err != nil || f != files[3] {
		t.Fatalf("movie: %+v %v", f, err)
	}
	if _, err := ChooseFile(files[:3], target); !errors.Is(err, core.ErrWrongContent) {
		t.Fatalf("wrong pack accepted: %v", err)
	}
	for i := range files {
		copy := *target
		copy.FileIndex = &i
		_, err := ChooseFile(files, &copy)
		if (err == nil) != (i == 3) {
			t.Errorf("index %d: %v", i, err)
		}
	}
	copy := *target
	copy.ExactFile = files[0].Path
	if _, err := ChooseFile(files, &copy); !errors.Is(err, core.ErrWrongContent) {
		t.Fatalf("wrong explicit path: %v", err)
	}
	// A bare correctly titled member is still subject to runtime validation at Play.
	if _, err := ChooseFile([]File{{Path: "The Mummy.mkv", Size: 4000}}, target); err != nil {
		t.Fatal(err)
	}
}
