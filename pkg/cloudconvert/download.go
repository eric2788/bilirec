package cloudconvert

import (
	"encoding/json"
	"io"
)

func (c *Client) CreateExportURL(payload *ExportURLRequest) (*ExportURLResponse, error) {
	req := c.client.R().
		SetContext(c.ctx).
		SetBody(payload)

	res, err := req.Post("/export/url")
	if err != nil {
		return nil, err
	}

	var exportRes ExportURLResponse
	if err := json.Unmarshal(res.Body(), &exportRes); err != nil {
		return nil, err
	}

	return &exportRes, nil
}

func (c *Client) DownloadAsFileStream(url string) (io.ReadCloser, error) {
	req := c.client.R().
		SetContext(c.ctx).
		SetDoNotParseResponse(true)

	res, err := req.Get(url)
	if err != nil {
		return nil, err
	}
	return res.RawBody(), nil
}
