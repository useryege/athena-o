package rpc

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/useryege/athena/util/ethws"
	"net/http"
	"net/url"
	"time"
)

// DialHTTP creates the immutable HTTP endpoint borrowed by SourceRPC. Its owner
// closes the returned client. Live subscription/reconnect belongs only to Session.
func DialHTTP(ctx context.Context, endpoint, proxyURL string) (*ethclient.Client, error) {
	target, err := url.Parse(endpoint)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		return nil, errors.New("collector HTTP endpoint must use http or https")
	}
	if err = ethws.ValidateProxyURL(proxyURL); err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	if proxyURL != "" {
		proxy, _ := url.Parse(proxyURL)
		transport.Proxy = http.ProxyURL(proxy)
	}
	client, err := gethrpc.DialOptions(ctx, endpoint, gethrpc.WithHTTPClient(&http.Client{Transport: transport, Timeout: 5 * time.Second}))
	if err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	return ethclient.NewClient(client), nil
}
