package application

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/useryege/athena/pkg/abi/ERC20"
)

type ProjectSimulator interface {
	Simulate(msgCaller common.Address, tokenAddress common.Address, pairContract common.Address) (SimulateResult, error)
}

var _ ProjectSimulator = &projectSimulatorImpl{}

type projectSimulatorImpl struct {
	nodeClient *ethclient.Client
}

func NewProjectSimulator(nodeClient *ethclient.Client) ProjectSimulator {
	return &projectSimulatorImpl{
		nodeClient: nodeClient,
	}
}

type SimulateResult struct {
	CanMintFromDeadViaTransferFrom bool
	CanMintFromZeroViaTransferFrom bool
	CanMintFromPairViaTransferFrom bool
	CanMintViaTransfer             bool
}

func (s *projectSimulatorImpl) Simulate(msgCaller common.Address, tokenAddress common.Address, pairContract common.Address) (SimulateResult, error) {
	var result SimulateResult

	canMintFromDead, err := s.TransferFromMint(msgCaller, tokenAddress, common.HexToAddress(DeadAddress))
	if err != nil {
		return result, err
	}
	result.CanMintFromDeadViaTransferFrom = canMintFromDead

	canMintFromZero, err := s.TransferFromMint(msgCaller, tokenAddress, common.HexToAddress(ZeroAddress))
	if err != nil {
		return result, err
	}
	result.CanMintFromZeroViaTransferFrom = canMintFromZero

	canMintFromPair, err := s.TransferFromMint(msgCaller, tokenAddress, pairContract)
	if err != nil {
		return result, err
	}
	result.CanMintFromPairViaTransferFrom = canMintFromPair

	canMintViaTransfer, err := s.TransferMint(msgCaller, tokenAddress, pairContract)
	if err != nil {
		return result, err
	}
	result.CanMintViaTransfer = canMintViaTransfer

	return result, nil
}

func (s *projectSimulatorImpl) TransferFromMint(msgCaller common.Address, tokenAddress common.Address, mintFrom common.Address) (bool, error) {
	reader := &bind.CallOpts{Context: context.Background()}
	tokenInstance, err := ERC20.NewERC20(tokenAddress, s.nodeClient)
	if err != nil {
		return false, err
	}

	// get approve amount
	approveAmount, err := tokenInstance.Allowance(reader, mintFrom, msgCaller)
	if err != nil {
		return false, err
	}

	// calculate mint number MintNumber = (approveAmount + 10**18) * 2
	mintNumber := new(big.Int).Add(approveAmount, big.NewInt(1000000000000000000))
	mintNumber.Mul(mintNumber, big.NewInt(2))

	parsed, err := ERC20.ERC20MetaData.GetAbi()
	if err != nil {
		return false, err
	}
	data, err := parsed.Pack("transferFrom", mintFrom, msgCaller, mintNumber)
	if err != nil {
		return false, err
	}

	msg := ethereum.CallMsg{
		From: msgCaller,
		To:   &tokenAddress,
		Data: data,
	}
	_, err = s.nodeClient.EstimateGas(context.Background(), msg)
	isCanMint := err == nil
	if !isCanMint {
		// add one more check, mint to other address
		data, err := parsed.Pack("transferFrom", mintFrom, common.HexToAddress(MagicAddress), mintNumber)
		if err != nil {
			return false, err
		}
		msg := ethereum.CallMsg{
			From: msgCaller,
			To:   &tokenAddress,
			Data: data,
		}
		_, err = s.nodeClient.EstimateGas(context.Background(), msg)
		isCanMint = err == nil
	}
	return isCanMint, nil

}

func (s *projectSimulatorImpl) TransferMint(msgCaller common.Address, tokenAddress common.Address, to common.Address) (bool, error) {
	reader := &bind.CallOpts{Context: context.Background()}
	tokenInstance, err := ERC20.NewERC20(tokenAddress, s.nodeClient)
	if err != nil {
		return false, err
	}

	balance, err := tokenInstance.BalanceOf(reader, msgCaller)
	if err != nil {
		return false, err
	}

	// calculate mint number MintNumber = (balance + 10**18) * 2
	mintNumber := new(big.Int).Add(balance, big.NewInt(1000000000000000000))
	mintNumber.Mul(mintNumber, big.NewInt(2))

	parsed, err := ERC20.ERC20MetaData.GetAbi()
	if err != nil {
		return false, err
	}
	data, err := parsed.Pack("transfer", to, mintNumber)
	if err != nil {
		return false, err
	}

	msg := ethereum.CallMsg{
		From: msgCaller,
		To:   &tokenAddress,
		Data: data,
	}
	_, err = s.nodeClient.EstimateGas(context.Background(), msg)

	return err == nil, nil
}
