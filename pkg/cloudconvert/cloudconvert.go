package cloudconvert

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	client *resty.Client
	ctx    context.Context

	apiKey string
}

func NewClient(ctx context.Context, apiKey string) *Client {

	client := resty.New().
		SetBaseURL("https://api.cloudconvert.com/v2/").
		SetHeader("Content-Type", "application/json").
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(5)).
		SetTimeout(30 * time.Second).
		SetAuthScheme("Bearer").
		SetAuthToken(apiKey)

	return &Client{
		client: client,
		ctx:    ctx,
		apiKey: apiKey,
	}
}


