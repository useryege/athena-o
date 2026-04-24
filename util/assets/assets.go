package assets

import (
	"github.com/useryege/athena/assets"
)

var (
	BuiltinPolicyCSV string
	ModelConf        string
	SwaggerJSON      string
	BadgeSVG         string
)

func init() {
	// load builtin-policy.csv
	data, err := assets.Embedded.ReadFile("builtin-policy.csv")
	if err != nil {
		panic(err)
	}
	BuiltinPolicyCSV = string(data)
	// load model.conf
	data, err = assets.Embedded.ReadFile("model.conf")
	if err != nil {
		panic(err)
	}
	ModelConf = string(data)
	// load swagger.json
	data, err = assets.Embedded.ReadFile("swagger.json")
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
