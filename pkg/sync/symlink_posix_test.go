// +build !windows

package sync

import (
	"testing"
)

func TestSymlinkPOSIXBackslashInvalid(t *testing.T) {
	if _, err := normalizeSymlinkAndEnsureSane("file", "target\\path"); err == nil {
		t.Fatal("symlink with backslash in target treated as sane")
	}
}
