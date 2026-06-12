package session

import (
	"context"
	"net"
	"strconv"
	"strings"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	metadataXForwardedFor = "x-forwarded-for"
	metadataXRealIP       = "x-real-ip"
	metadataForwarded     = "forwarded"
	metadataRemoteAddr    = "athena-remote-addr"
)

func clientIPFromContext(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ip := firstForwardedForIP(md.Get(metadataXForwardedFor)); ip != "" {
			return ip
		}
		if ip := firstValidIP(md.Get(metadataXRealIP)); ip != "" {
			return ip
		}
		if ip := firstForwardedHeaderIP(md.Get(metadataForwarded)); ip != "" {
			return ip
		}
		if ip := firstValidIP(md.Get(metadataRemoteAddr)); ip != "" {
			return ip
		}
	}
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		return validIPString(p.Addr.String())
	}
	return ""
}

func firstForwardedForIP(values []string) string {
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if ip := validIPString(part); ip != "" {
				return ip
			}
		}
	}
	return ""
}

func firstValidIP(values []string) string {
	for _, value := range values {
		if ip := validIPString(value); ip != "" {
			return ip
		}
	}
	return ""
}

func firstForwardedHeaderIP(values []string) string {
	for _, value := range values {
		for _, forwardedSet := range strings.Split(value, ",") {
			for _, pair := range strings.Split(forwardedSet, ";") {
				key, val, ok := strings.Cut(strings.TrimSpace(pair), "=")
				if !ok || !strings.EqualFold(strings.TrimSpace(key), "for") {
					continue
				}
				if ip := validIPString(val); ip != "" {
					return ip
				}
			}
		}
	}
	return ""
}

func validIPString(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" || strings.EqualFold(value, "unknown") {
		return ""
	}
	if strings.HasPrefix(value, "[") {
		if host, _, err := net.SplitHostPort(value); err == nil {
			value = host
		} else {
			value = strings.Trim(value, "[]")
		}
	} else if strings.Count(value, ":") == 1 {
		if host, _, err := net.SplitHostPort(value); err == nil {
			value = host
		}
	}
	if strings.HasPrefix(value, "_") {
		return ""
	}
	if host, port, err := net.SplitHostPort(value); err == nil && port != "" {
		value = host
	}
	value = strings.Trim(value, "[]")
	if _, err := strconv.Atoi(value); err == nil {
		return ""
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return ""
	}
	return ip.String()
}
