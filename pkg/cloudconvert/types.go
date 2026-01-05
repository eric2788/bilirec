package cloudconvert

type VideoConvertPayload struct {
	Input        string `json:"input"`                  // import task ID
	InputFormat  string `json:"input_format,omitempty"` // e.g. "flv"
	OutputFormat string `json:"output_format"`          // e.g. "mp4"`

	// Video / audio options to mimic: ffmpeg -i a.flv -c copy a.mp4
	VideoCodec string `json:"video_codec,omitempty"` // "copy"
	AudioCodec string `json:"audio_codec,omitempty"` // "copy"

	// Optional: output file name, timeout, etc.
	Filename string `json:"filename,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
}

type ConvertTaskResponse struct {
	Data ConvertTaskData `json:"data"`
}

type ConvertTaskData struct {
	ID             string                 `json:"id"`
	Operation      string                 `json:"operation"`
	Status         string                 `json:"status"`
	Message        *string                `json:"message"`
	CreatedAt      string                 `json:"created_at"`
	StartedAt      string                 `json:"started_at"`
	EndedAt        *string                `json:"ended_at"`
	DependsOnTasks map[string]interface{} `json:"depends_on_tasks"`
	Engine         string                 `json:"engine"`
	EngineVersion  string                 `json:"engine_version"`
	Payload        map[string]interface{} `json:"payload"`
	Result         interface{}            `json:"result"`
}

type ImportExportBaseData struct {
	ID        string  `json:"id"`
	Operation string  `json:"operation"`
	Status    string  `json:"status"`
	Message   *string `json:"message"`
	CreatedAt string  `json:"created_at"`
	StartedAt string  `json:"started_at"`
	EndedAt   string  `json:"ended_at"`
}

type ImportUploadRequest struct {
	Redirect string `json:"redirect,omitempty"`
}

type ImportUploadForm struct {
	URL        string         `json:"url"`
	Parameters map[string]any `json:"parameters"`
}

type ImportUploadResult struct {
	Form ImportUploadForm `json:"form"`
}

type ImportUploadTask struct {
	ImportExportBaseData
	Code    *string            `json:"code"`
	Payload map[string]any     `json:"payload"`
	Result  ImportUploadResult `json:"result"`
}

type ImportUploadResponse struct {
	Data ImportUploadTask `json:"data"`
}

type ExportURLRequest struct {
	Input                any  `json:"input"` // string or []string
	ArchiveMultipleFiles bool `json:"archive_multiple_files,omitempty"`
	Inline               bool `json:"inline,omitempty"`
}

type ExportURLResponse struct {
	Data ExportURLData `json:"data"`
}

type ExportURLData struct {
	ImportExportBaseData
	Result ExportURLResult `json:"result"`
}

type ExportURLResult struct {
	Files []ExportedFile `json:"files"`
}

type ExportedFile struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

type TaskResponse struct {
	Data TaskData `json:"data"`
}

type TaskListResponse struct {
	Data  []TaskData    `json:"data"`
	Links TaskListLinks `json:"links"`
	Meta  TaskListMeta  `json:"meta"`
}

type TaskData struct {
	ID             string            `json:"id"`
	JobID          string            `json:"job_id"`
	Operation      string            `json:"operation"`
	Status         string            `json:"status"`
	Credits        *int              `json:"credits"`
	Message        *string           `json:"message"`
	Code           *string           `json:"code"`
	CreatedAt      string            `json:"created_at"`
	StartedAt      string            `json:"started_at"`
	EndedAt        *string           `json:"ended_at"`
	DependsOnTasks map[string]string `json:"depends_on_tasks"`
	Engine         string            `json:"engine"`
	EngineVersion  string            `json:"engine_version"`
	Payload        TaskPayload       `json:"payload"`
	Result         TaskResult        `json:"result"`
}

type TaskListLinks struct {
	First string  `json:"first"`
	Last  *string `json:"last"`
	Prev  *string `json:"prev"`
	Next  *string `json:"next"`
}

type TaskListMeta struct {
	CurrentPage int    `json:"current_page"`
	From        int    `json:"from"`
	Path        string `json:"path"`
	PerPage     int    `json:"per_page"`
	To          int    `json:"to"`
}

type TaskPayload struct {
	InputFormat   string `json:"input_format"`
	OutputFormat  string `json:"output_format"`
	PageRange     string `json:"page_range"`
	OptimizePrint bool   `json:"optimize_print"`
}

type TaskListFilter struct {
	JobID     string
	Status    string
	Operation string
	Include   []string
	PerPage   int
	Page      int
}

type TaskResult struct {
	Files []TaskResultFile `json:"files"`
}

type TaskResultFile struct {
	Filename string `json:"filename"`
}
