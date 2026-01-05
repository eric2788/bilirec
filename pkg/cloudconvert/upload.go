package cloudconvert

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func (c *Client) CreateUploadTask(payload *ImportUploadRequest) (*ImportUploadResponse, error) {
	req := c.client.R().
		SetContext(c.ctx).
		SetBody(payload)

	res, err := req.Post("/import/upload")
	if err != nil {
		return nil, err
	}

	var taskRes ImportUploadResponse
	if err := json.Unmarshal(res.Body(), &taskRes); err != nil {
		return nil, err
	}
	return &taskRes, nil
}

func (c *Client) UploadFileToTask(f *os.File, task *ImportUploadTask) error {
	defer f.Close()
	// Ensure file offset at beginning
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	req := c.client.R().
		SetContext(c.ctx).
		SetFormData(c.toFormData(task.Result.Form.Parameters)).
		SetFileReader("file", f.Name(), f)

	_, err := req.Post(task.Result.Form.URL)
	return err
}

func (c *Client) toFormData(params map[string]any) map[string]string {
	formData := make(map[string]string)
	for key, value := range params {
		formData[key] = fmt.Sprint(value)
	}
	return formData
}
