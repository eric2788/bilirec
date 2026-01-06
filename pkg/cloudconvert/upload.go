package cloudconvert

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/eric2788/bilirec/pkg/monitor"
	"github.com/eric2788/bilirec/utils"
	"github.com/sirupsen/logrus"
)

func (c *Client) CreateUploadTask(redirect ...string) (*ImportUploadResponse, error) {
	req := c.client.R().SetContext(c.ctx)

	if len(redirect) > 0 {
		req.SetBody(&ImportUploadRequest{Redirect: redirect[0]})
	}

	res, err := req.Post("/import/upload")
	if err != nil {
		return nil, err
	}

	var taskRes ImportUploadResponse
	if err := json.Unmarshal(res.Body(), &taskRes); err != nil {
		return nil, err
	} else if res.StatusCode() < 200 || res.StatusCode() >= 400 {
		return nil, fmt.Errorf("video convert failed with status code %d: %s", res.StatusCode(), res.String())
	}
	return &taskRes, nil
}

func (c *Client) UploadFileToTask(f *os.File, task *ImportUploadTask) error {
	defer f.Close()
	// Ensure file offset at beginning
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	reader := utils.TernaryFunc(os.Getenv("DEBUG") == "true",
		func() io.ReadCloser {
			return monitor.NewProgressReader(f, func(read int64) {
				logrus.WithField("task", task.ID).Debugf("uploaded %.2f MB", float64(read)/1024/1024)
			})
		},
		func() io.ReadCloser {
			return f
		},
	)

	req := c.uploadClient.R().
		SetContext(c.ctx).
		SetFormData(c.toFormData(task.Result.Form.Parameters)).
		SetFileReader("file", f.Name(), reader)

	res, err := req.Post(task.Result.Form.URL)
	if err != nil {
		return err
	} else if res.StatusCode() < 200 || res.StatusCode() >= 400 {
		return fmt.Errorf("upload failed with status code %d: %s", res.StatusCode(), res.String())
	}
	return nil
}

func (c *Client) toFormData(params map[string]any) map[string]string {
	formData := make(map[string]string)
	for key, value := range params {
		formData[key] = fmt.Sprint(value)
	}
	return formData
}
