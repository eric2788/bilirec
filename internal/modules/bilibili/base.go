package bilibili

import (
	"github.com/go-resty/resty/v2"
)

func (c *Client) Do(do func(req *resty.Request) (*resty.Response, error)) (*resty.Response, error) {
	var req = c.Resty().R()
	return do(req)
}

func (c *Client) DoLive(do func(req *resty.Request) (*resty.Response, error)) (*resty.Response, error) {
	var req = c.liveClient.R()
	return do(req)
}

func (c *Client) DoLiveStream(do func(req *resty.Request) (*resty.Response, error)) (*resty.Response, error) {
	var req = c.liveStreamClient.R()
	return do(req)
}
