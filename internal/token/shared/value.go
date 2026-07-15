package shared

import "github.com/ethereum/go-ethereum/common"

type ChainID int64
type ProjectID int64
type Address = common.Address
type Hash = common.Hash

type Page struct {
	Number int32
	Size   int32
}
