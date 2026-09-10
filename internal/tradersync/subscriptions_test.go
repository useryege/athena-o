package tradersync

import (
	"strings"
	"testing"
)

func TestNoteUnicodeLimit(t *testing.T) {
	for _, s := range []string{"", strings.Repeat("🙂", 20)} {
		if e := ValidateNote(s); e != nil {
			t.Fatal(e)
		}
	}
	for _, s := range []string{strings.Repeat("🙂", 21), string([]byte{0xff})} {
		if ValidateNote(s) == nil {
			t.Fatal("invalid note accepted")
		}
	}
}
