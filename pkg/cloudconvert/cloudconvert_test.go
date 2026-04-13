package cloudconvert_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/eric2788/bilirec/pkg/cloudconvert"
	"github.com/eric2788/bilirec/utils"
	"github.com/sirupsen/logrus"
)

const (
	importTaskName  = "import-source"
	commandTaskName = "command-faststart"
	convertTaskName = "convert-output"
	exportTaskName  = "export-output"
)

func TestUploadFile(t *testing.T) {
	if os.Getenv("CLOUDCONVERT_API_KEY") == "" {
		t.Skip("CLOUDCONVERT_API_KEY not set, skipping test")
	}

	const sourcePath = "test.flv"

	client := cloudconvert.NewClient(t.Context(), os.Getenv("CLOUDCONVERT_API_KEY"))
	f, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	job, err := client.NewJobBuilder().
		AddTask(cloudconvert.NewImportUploadTask(importTaskName, &cloudconvert.ImportUploadRequest{})).
		AddTask(cloudconvert.NewExportURLTask(exportTaskName, &cloudconvert.ExportURLRequest{
			Input: importTaskName,
		})).
		Submit()
	if err != nil {
		t.Fatal(err)
	}

	exportTaskID := job.TaskID(exportTaskName)
	if exportTaskID == "" {
		t.Fatalf("export task id not found for task name %s", exportTaskName)
	}
	uploadTask := job.TaskData(importTaskName)

	if err := client.UploadFileToTask(f, uploadTask.Result.Form); err != nil {
		t.Fatal(err)
	}

	t.Logf("Upload successful, import task id=%s", uploadTask.ID)
	t.Logf("Export task id=%s (use CLOUDCONVERT_TASK_ID to download)", exportTaskID)
}

func TestVideoConvertTask(t *testing.T) {
	if os.Getenv("CLOUDCONVERT_API_KEY") == "" {
		t.Skip("CLOUDCONVERT_API_KEY not set, skipping test")
	}

	const sourcePath = "test.flv"

	client := cloudconvert.NewClient(t.Context(), os.Getenv("CLOUDCONVERT_API_KEY"))
	f, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	inputFormat := filepath.Ext(sourcePath)
	if inputFormat != "" {
		inputFormat = inputFormat[1:]
	}

	job, err := client.NewJobBuilder().
		AddTask(cloudconvert.NewImportUploadTask(importTaskName, &cloudconvert.ImportUploadRequest{})).
		AddTask(cloudconvert.NewVideoConvertTask(convertTaskName, &cloudconvert.VideoConvertPayload{
			Input:        importTaskName,
			InputFormat:  inputFormat,
			OutputFormat: "mp4",
			VideoCodec:   "copy",
			AudioCodec:   "copy",
			Filename:     "output.mp4",
		})).
		AddTask(cloudconvert.NewExportURLTask(exportTaskName, &cloudconvert.ExportURLRequest{
			Input: convertTaskName,
		})).
		Submit()
	if err != nil {
		t.Fatal(err)
	}

	convertTaskID := job.TaskID(convertTaskName)
	exportTaskID := job.TaskID(exportTaskName)
	if convertTaskID == "" {
		t.Fatalf("convert task id not found for task name %s", convertTaskName)
	}
	if exportTaskID == "" {
		t.Fatalf("export task id not found for task name %s", exportTaskName)
	}

	importTask := job.TaskData(importTaskName)
	if importTask == nil {
		t.Fatalf("import task data not found for task name %s", importTaskName)
	}

	if err := client.UploadFileToTask(f, importTask.Result.Form); err != nil {
		t.Fatal(err)
	}

	t.Logf("Job Response: %v", utils.PrettyPrintJSON(job.Job))
	t.Logf("Convert Task ID: %v", convertTaskID)
	t.Logf("Export Task ID: %v", exportTaskID)
}

func TestGetTaskDetails(t *testing.T) {
	if os.Getenv("CLOUDCONVERT_API_KEY") == "" {
		t.Skip("CLOUDCONVERT_API_KEY not set, skipping test")
	}
	taskID := os.Getenv("CLOUDCONVERT_TASK_ID")
	if taskID == "" {
		t.Skip("CLOUDCONVERT_TASK_ID not set, skipping test")
	}

	client := cloudconvert.NewClient(t.Context(), os.Getenv("CLOUDCONVERT_API_KEY"))

	task, err := client.GetTask(taskID)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Task Details: %v", utils.PrettyPrintJSON(task))
}

func TestDownloadExportTask(t *testing.T) {
	if os.Getenv("CLOUDCONVERT_API_KEY") == "" {
		t.Skip("CLOUDCONVERT_API_KEY not set, skipping test")
	}
	taskID := os.Getenv("CLOUDCONVERT_TASK_ID")
	if taskID == "" {
		t.Skip("CLOUDCONVERT_TASK_ID not set, skipping test")
	}

	client := cloudconvert.NewClient(t.Context(), os.Getenv("CLOUDCONVERT_API_KEY"))

	task, err := client.GetTask(taskID)
	if err != nil {
		t.Fatal(err)
	}

	if task.Data.Status != cloudconvert.TaskStatusFinished {
		t.Fatalf("task %s is not finished (status: %s)", taskID, task.Data.Status)
	}
	if len(task.Data.Result.Files) == 0 || task.Data.Result.Files[0].URL == "" {
		t.Fatalf("task %s has no downloadable files", taskID)
	}

	t.Logf("Download URL: %v", task.Data.Result.Files[0].URL)
}

func TestUploadCommandFaststartDownload(t *testing.T) {
	if os.Getenv("CLOUDCONVERT_API_KEY") == "" {
		t.Skip("CLOUDCONVERT_API_KEY not set, skipping test")
	}

	const sourcePath = "test.flv"

	client := cloudconvert.NewClient(t.Context(), os.Getenv("CLOUDCONVERT_API_KEY"))
	f, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	job, err := client.NewJobBuilder().
		AddTask(cloudconvert.NewImportUploadTask(importTaskName, &cloudconvert.ImportUploadRequest{})).
		AddTask(cloudconvert.NewCommandTask(commandTaskName, &cloudconvert.CommandPayload{
			Input:     importTaskName,
			Engine:    "ffmpeg",
			Command:   "ffmpeg",
			Arguments: fmt.Sprintf("-i /input/%s/test.flv -map 0 -map_metadata 0 -c copy -movflags +faststart /output/output.mp4", importTaskName),
		})).
		AddTask(cloudconvert.NewExportURLTask(exportTaskName, &cloudconvert.ExportURLRequest{
			Input: commandTaskName,
		})).
		Submit()
	if err != nil {
		t.Fatal(err)
	}

	commandTaskID := job.TaskID(commandTaskName)
	exportTaskID := job.TaskID(exportTaskName)

	if commandTaskID == "" {
		t.Fatalf("command task id not found for task name %s", commandTaskName)
	}
	if exportTaskID == "" {
		t.Fatalf("export task id not found for task name %s", exportTaskName)
	}

	importTask := job.TaskData(importTaskName)

	if err := client.UploadFileToTask(f, importTask.Result.Form); err != nil {
		t.Fatal(err)
	}

	t.Logf("Upload successful, command task id=%s", commandTaskID)
	t.Logf("Export task id=%s (use CLOUDCONVERT_TASK_ID to download)", exportTaskID)
}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	os.Setenv("DEBUG", "true")
}
