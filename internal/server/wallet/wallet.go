package wallet

import (
	"context"

	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Server struct {
	walletpkg.UnimplementedWalletServiceServer
	walletClientSet walletapiclient.Clientset
}

func NewServer(walletClientSet walletapiclient.Clientset) *Server {
	return &Server{walletClientSet: walletClientSet}
}

func (s *Server) GetWalletStatus(ctx context.Context, _ *walletpkg.GetWalletStatusRequest) (*v1alpha1.WalletStatus, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetWalletStatus(ctx, &walletapiclient.GetWalletStatusRequest{})
}
