package commands

import (
	log "github.com/sirupsen/logrus"
	solanaapiclient "github.com/useryege/athena/internal/solanadiscovery/apiclient"
)

// A Solana configuration error degrades only the Solana facade. The API
// process remains available for its unrelated services.
func newOptionalSolanaClientset(address, token string) solanaapiclient.Clientset {
	client, err := solanaapiclient.NewSolanaClientset(address, token)
	if err != nil {
		log.WithError(err).Warn("Solana discovery client is unavailable")
	}
	return client
}
