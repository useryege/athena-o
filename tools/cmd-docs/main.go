package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"

	athenaserver "github.com/useryege/athena/cmd/athena-server/commands"
)

func main() {
	// set HOME env var so that default values involve user's home directory do not depend on the running user.
	os.Setenv("HOME", "/home/user")
	os.Setenv("XDG_CONFIG_HOME", "/home/user/.config")

	identity := func(s string) string { return s }
	headerPrepender := func(filename string) string {
		// The default header looks like `Athena app get`. The leading capital letter is off-putting.
		// This header overrides the default. It's better visually and for search results.
		filename = filepath.Base(filename)
		filename = filename[:len(filename)-3] // Drop the '.md'
		return fmt.Sprintf("# `%s` Command Reference\n\n", strings.ReplaceAll(filename, "_", " "))
	}

	err := doc.GenMarkdownTreeCustom(athenaserver.NewCommand(), "./docs/operator-manual/server-commands", headerPrepender, identity)
	if err != nil {
		log.Fatal(err)
	}
}
