// Package abi contains the minimal event ABIs extracted from approved deployments.
// See provenance.json for the fixed sources, deployment runtime hashes and evidence.
package abi

import _ "embed"

//go:embed core_exchange.json
var CoreExchange []byte

//go:embed combos_exchange.json
var CombosExchange []byte

//go:embed proxy.json
var Proxy []byte

//go:embed combinatorial_module.json
var CombinatorialModule []byte

//go:embed binary_module.json
var BinaryModule []byte
