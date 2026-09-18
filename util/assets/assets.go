package assets

import (
	"github.com/useryege/athena/assets"
)

var (
	BadgeSVG string
)

func init() {
	// load badge.svg
	data, err := assets.Embedded.ReadFile("badge.svg")
	if err != nil {
		panic(err)
	}
	BadgeSVG = string(data)
}
