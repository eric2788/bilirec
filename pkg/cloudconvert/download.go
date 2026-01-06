package cloudconvert

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) CreateExportURL(payload *ExportURLRequest) (*TaskResponse, error) {
	req := c.client.R().
		SetContext(c.ctx).
		SetBody(payload)

	res, err := req.Post("/export/url")
	if err != nil {
		return nil, err
	}

	var exportRes TaskResponse
	if err := json.Unmarshal(res.Body(), &exportRes); err != nil {
		return nil, err
	} else if res.StatusCode() < 200 || res.StatusCode() >= 400 {
		return nil, fmt.Errorf("video convert failed with status code %d: %s", res.StatusCode(), res.String())
	}

	return &exportRes, nil
}

func (c *Client) DownloadAsFileStream(url string) (io.ReadCloser, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	} else if res.StatusCode < 200 || res.StatusCode >= 400 {
		return nil, fmt.Errorf("video convert failed with status code %d", res.StatusCode)
	}
	return res.Body, nil
}
