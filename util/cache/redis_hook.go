package cache

import (
	"context"
	"errors"
	"net"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

type athenaRedisHooks struct {
	reconnectCallback func()
}

func NewAthenaRedisHook(reconnectCallback func()) *athenaRedisHooks {
	return &athenaRedisHooks{reconnectCallback: reconnectCallback}
}

func (hook *athenaRedisHooks) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := next(ctx, network, addr)
		return conn, err
	}
}

func (hook *athenaRedisHooks) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		var dnsError *net.DNSError
		err := next(ctx, cmd)
		if err != nil && errors.As(err, &dnsError) {
			log.Warnf("Reconnect to redis because error: \"%v\"", err)
			hook.reconnectCallback()
		}
		return err
	}
}

func (hook *athenaRedisHooks) ProcessPipelineHook(_ redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return nil
}
