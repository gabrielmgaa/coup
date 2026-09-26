package web

import (
	"io/fs"
	"testing"
)

func TestTheEmbeddedSiteKeepsItsPlaceholder(t *testing.T) {
	if _, err := fs.Stat(Dist(), ".gitkeep"); err != nil {
		t.Errorf("the embedded dist lost .gitkeep, which keeps go:embed compiling on a fresh clone: %v", err)
	}
}
