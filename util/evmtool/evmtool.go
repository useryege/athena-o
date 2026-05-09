package evmtool

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
)

func GetToken0(contractA common.Address, contractB common.Address) common.Address {
	if bytes.Compare(contractA[:], contractB[:]) < 0 {
		return contractA
	}
	return contractB
}

func GetToken1(contractA common.Address, contractB common.Address) common.Address {
	if bytes.Compare(contractA[:], contractB[:]) > 0 {
		return contractA
	}
	return contractB
}
