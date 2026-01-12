package cloudconvert

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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

// UploadFileToTask uploads the given file to the specified import upload task.
// This function will use net/http instead of resty due to the need of streaming upload with pipe.
// It is because `resty@v2` will buffer the entire request body in memory which cause OOM for large files.
func (c *Client) UploadFileToTask(f *os.File, task *ImportUploadTask) error {
	defer f.Close()
	// Ensure file offset at beginning
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	reader := utils.TernaryFunc(os.Getenv("DEBUG") == "true",
		func() io.ReadCloser {
			var lastLogged int64
			return monitor.NewProgressReader(f, func(read int64) {
				if read-lastLogged >= 100*1024*1024 { // 100MB threshold
					logrus.
						WithField("task", task.ID).
						WithField("file", f.Name()).
						Debugf("uploaded %.2f MB", float64(read)/1024/1024)
					lastLogged = read
				}
			})
		},
		func() io.ReadCloser {
			return f
		},
	)

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	// Writer goroutine: writes fields and stream-copies the file into the multipart writer.
	go func() {
		defer pw.Close()
		defer mw.Close()

		// write form fields
		for k, v := range task.Result.Form.Parameters {
			if err := mw.WriteField(k, fmt.Sprint(v)); err != nil {
				pw.CloseWithError(err)
				return
			}
		}

		part, err := mw.CreateFormFile("file", f.Name())
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		buf := c.uploadPool.GetBytes()
		defer c.uploadPool.PutBytes(buf)

		if _, err := io.CopyBuffer(part, reader, buf); err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	// Create HTTP request using standard library (NOT resty)
	httpReq, err := http.NewRequestWithContext(c.ctx, "POST", task.Result.Form.URL, pr)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", mw.FormDataContentType())

	res, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer res.Body.Close()

	// Check response status
	if res.StatusCode < 200 || res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("upload failed with status code %d: %s", res.StatusCode, string(body))
	}

	return nil
}
