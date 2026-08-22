package assets

import (
	"github.com/useryege/athena/assets"
)

var (
	SwaggerJSON      string
	BadgeSVG         string
)

func init() {
	// load swagger.json
	data, err := assets.Embedded.ReadFile("swagger.json")
	if err != nil {
		panic(err)
	}
	SwaggerJSON = string(data)
	// load badge.svg
	data, err = assets.Embedded.ReadFile("badge.svg")
	if err != nil {
		panic(err)
	}
	BadgeSVG = string(data)
}
