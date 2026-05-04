package application

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

type ProjectFilter struct {
	Reader     *bind.CallOpts
	nodeClient *ethclient.Client
	inputCh    <-chan *Project
	outputCh   chan<- *Project
	wg         sync.WaitGroup
}

func NewProjectFilter(nodeClient *ethclient.Client, inputCh <-chan *Project, outputCh chan<- *Project) *ProjectFilter {
	return &ProjectFilter{
		nodeClient: nodeClient,
		inputCh:    inputCh,
		outputCh:   outputCh,
		Reader:     &bind.CallOpts{},
	}
}

func (p *ProjectFilter) Start(ctx context.Context) error {
	for i := 0; i < 10; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-p.inputCh:
					if !ok {
						return
					}
					filterStartedAt := time.Now()

					// calculate sender from transaction
					from, err := types.Sender(types.LatestSignerForChainID(event.Tx.ChainId()), event.Tx)
					if err != nil {
						log.WithFields(log.Fields{
							"component":   "Project Filter",
							"blockNumber": event.BlockNumber,
							"transaction": event.Tx.Hash(),
							"error":       err,
						}).Error("failed to get sender from creation transaction")
						continue
					}
					contractAddress := crypto.CreateAddress(from, event.Tx.Nonce())

					// check if the contract is a token contract
					// tokenMetadata, err := p.IsTokenContract(contractAddress)
					// if err != nil {
					// 	continue
					// }
					event.Contract = contractAddress

					// event.TokenMetadata = &tokenMetadata
					event.PerfTrace.FilterStartedAt = filterStartedAt
					event.PerfTrace.FilterCompletedAt = time.Now()
					p.outputCh <- event

					log.WithFields(log.Fields{
						"component":         "Project Filter",
						"blockNumber":       event.BlockNumber,
						"blockTime":         event.BlockTime,
						"transaction":       event.Tx.Hash(),
						"executionDuration": event.PerfTrace.FilterCompletedAt.Sub(event.PerfTrace.FilterStartedAt).Milliseconds(),
					}).Info("token contract detected")
				}
			}
		}()
	}
	return nil
}

// use to judge if the contract is a token contract
// func (p *ProjectFilter) IsTokenContract(addr common.Address) (TokenMetadata, error) {
// 	// create ERC20 instance
// 	tokenInstance, err := ERC20.NewERC20(addr, p.nodeClient)
// 	if err != nil {
// 		return TokenMetadata{}, err
// 	}

// 	// try to call totalSupply
// 	totalSupply, err := tokenInstance.TotalSupply(p.Reader)
// 	if err != nil {
// 		return TokenMetadata{}, err
// 	}

// 	// try to call balanceOf
// 	_, err = tokenInstance.BalanceOf(p.Reader, common.HexToAddress("0x0000000000000000000000000000000000000000"))
// 	if err != nil {
// 		return TokenMetadata{}, err
// 	}

// 	// try to call decimals
// 	decimals, err := tokenInstance.Decimals(p.Reader)
// 	if err != nil {
// 		return TokenMetadata{}, err
// 	}

// 	// try to call name
// 	name, err := tokenInstance.Name(p.Reader)
// 	if err != nil {
// 		return TokenMetadata{}, err
// 	}

// 	// try to call symbol
// 	symbol, err := tokenInstance.Symbol(p.Reader)
// 	if err != nil {
// 		return TokenMetadata{}, err
// 	}

// 	return TokenMetadata{
// 		Static: TokenStaticMetadata{
// 			Address:     addr,
// 			TotalSupply: totalSupply,
// 			Decimals:    decimals,
// 			Name:        name,
// 			Symbol:      symbol,
// 		},
// 	}, nil
// }

func (p *ProjectFilter) Stop() error {
	p.wg.Wait()
	return nil
}
