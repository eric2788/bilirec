package cloudconvert

type JobSubmitResult struct {
	Job   *JobResponse
	Tasks map[string]*TaskData
}

func (r *JobSubmitResult) TaskID(name string) string {
	if r == nil || len(r.Tasks) == 0 || name == "" {
		return ""
	}
	if task, ok := r.Tasks[name]; ok {
		return task.ID
	}
	return ""
}

func (r *JobSubmitResult) TaskData(name string) *TaskData {
	if r == nil || len(r.Tasks) == 0 || name == "" {
		return nil
	}
	if task, ok := r.Tasks[name]; ok {
		return task
	}
	return nil
}
