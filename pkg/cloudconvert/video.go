package cloudconvert

import (
	"encoding/json"
	"fmt"
)

func (c *Client) VideoConvert(payload *VideoConvertPayload) (*ConvertTaskResponse, error) {

	if payload.AudioCodec == "" {
		payload.AudioCodec = "copy"
	}

	if payload.VideoCodec == "" {
		payload.VideoCodec = "copy"
	}

	req := c.client.R().
		SetContext(c.ctx).
		SetBody(payload)

	res, err := req.Post("/convert")
	if err != nil {
		return nil, err
	} else if res.StatusCode() < 200 || res.StatusCode() >= 400 {
		return nil, fmt.Errorf("video convert failed with status code %d: %s", res.StatusCode(), res.String())
	}

	var convertRes ConvertTaskResponse
	if err := json.Unmarshal(res.Body(), &convertRes); err != nil {
		return nil, err
	}
	return &convertRes, nil
}
