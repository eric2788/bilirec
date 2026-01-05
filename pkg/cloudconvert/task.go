package cloudconvert

import (
	"encoding/json"
	"strconv"
	"strings"
)

func (c *Client) GetTask(taskID string, includes ...string) (*TaskResponse, error) {
	req := c.client.R().
		SetContext(c.ctx).
		SetQueryParam("include", strings.Join(includes, ",")).
		SetPathParam("id", taskID)

	res, err := req.Get("/tasks/{id}")
	if err != nil {
		return nil, err
	}

	var taskRes TaskResponse
	if err := json.Unmarshal(res.Body(), &taskRes); err != nil {
		return nil, err
	}

	return &taskRes, nil
}

func (c *Client) ListTasks(f *TaskListFilter) (*TaskListResponse, error) {
	req := c.client.R().SetContext(c.ctx)

	if f.JobID != "" {
		req.SetQueryParam("filter[job_id]", f.JobID)
	}
	if f.Status != "" {
		req.SetQueryParam("filter[status]", f.Status)
	}
	if f.Operation != "" {
		req.SetQueryParam("filter[operation]", f.Operation)
	}
	if len(f.Include) > 0 {
		req.SetQueryParam("include", strings.Join(f.Include, ","))
	}
	if f.PerPage > 0 {
		req.SetQueryParam("per_page", strconv.Itoa(f.PerPage))
	}
	if f.Page > 0 {
		req.SetQueryParam("page", strconv.Itoa(f.Page))
	}

	res, err := req.Get("/tasks")
	if err != nil {
		return nil, err
	}
	var taskListRes TaskListResponse
	if err := json.Unmarshal(res.Body(), &taskListRes); err != nil {
		return nil, err
	}
	return &taskListRes, nil
}

func (c *Client) CancelTask(taskID string) error {
	req := c.client.R().
		SetContext(c.ctx).
		SetPathParam("id", taskID)
	_, err := req.Post("/tasks/{id}/cancel")
	return err
}

func (c *Client) RetryTask(taskID string) error {
	req := c.client.R().
		SetContext(c.ctx).
		SetPathParam("id", taskID)
	_, err := req.Post("/tasks/{id}/retry")
	return err
}

func (c *Client) DeleteTask(taskID string) error {
	req := c.client.R().
		SetContext(c.ctx).
		SetPathParam("id", taskID)
	_, err := req.Delete("/tasks/{id}")
	return err
}
