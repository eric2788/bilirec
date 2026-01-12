package cloudconvert

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/eric2788/bilirec/pkg/pool"
	"github.com/go-resty/resty/v2"
)

type Client struct {
	client       *resty.Client
	streamClient *http.Client
	ctx          context.Context

	uploadPool *pool.BytesPool
}

func NewClient(ctx context.Context, apiKey string) *Client {

	client := resty.New().
		SetBaseURL("https://api.cloudconvert.com/v2/").
		SetHeader("Content-Type", "application/json").
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(5)).
		SetTimeout(30 * time.Second).
		SetAuthScheme("Bearer").
		SetAuthToken(apiKey)

	streamClient := &http.Client{
		Timeout: 0, // No timeout for streaming client
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				MaxVersion: tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				},
			},
		},
	}

	return &Client{
		client:       client,
		streamClient: streamClient,
		ctx:          ctx,
		uploadPool:   pool.NewBytesPool(5 * 1024 * 1024), // 5MB
	}
}
