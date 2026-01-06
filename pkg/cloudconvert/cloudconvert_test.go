package cloudconvert_test

import (
	"os"
	"testing"

	"github.com/eric2788/bilirec/pkg/cloudconvert"
	"github.com/eric2788/bilirec/pkg/monitor"
	"github.com/eric2788/bilirec/utils"
	"github.com/sirupsen/logrus"
)

func TestUploadFile(t *testing.T) {

	if os.Getenv("CLOUDCONVERT_API_KEY") == "" {
		t.Skip("CLOUDCONVERT_API_KEY not set, skipping test")
	}

	client := cloudconvert.NewClient(t.Context(), os.Getenv("CLOUDCONVERT_API_KEY"))
	f, err := os.Open("large.flv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	res, err := client.CreateUploadTask()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created upload task: %v", utils.PrettyPrintJSON(res))

	if err := client.UploadFileToTask(f, &res.Data); err != nil {
		t.Fatal(err)
	}

	t.Log("Upload successful")
	// now check the task status

	task, err := client.GetTask(res.Data.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(task.Data.Result.Files) == 0 {
		t.Fatal("No files found in task result")
	}
	t.Logf("Task Response: %v", utils.PrettyPrintJSON(task))
}

func TestVideoConvertTask(t *testing.T) {
	if os.Getenv("CLOUDCONVERT_API_KEY") == "" {
		t.Skip("CLOUDCONVERT_API_KEY not set, skipping test")
	} else if os.Getenv("CLOUDCONVERT_TASK_ID") == "" {
		t.Skip("CLOUDCONVERT_TASK_ID not set, skipping test")
	}

	client := cloudconvert.NewClient(t.Context(), os.Getenv("CLOUDCONVERT_API_KEY"))

	task, err := client.VideoConvert(&cloudconvert.VideoConvertPayload{
		Input:        os.Getenv("CLOUDCONVERT_TASK_ID"),
		InputFormat:  "flv",
		OutputFormat: "mp4",
		VideoCodec:   "copy",
		AudioCodec:   "copy",
		Filename:     "output.mp4",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Convert Task Response: %v", utils.PrettyPrintJSON(task))

	// do export here and get that task ID
	exporter, err := client.CreateExportURL(&cloudconvert.ExportURLRequest{
		Input: task.Data.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Export Task Response: %v", utils.PrettyPrintJSON(exporter))
	t.Logf("Export Task ID: %v", exporter.Data.ID)
}

func TestGetTaskDetails(t *testing.T) {
	if os.Getenv("CLOUDCONVERT_API_KEY") == "" {
		t.Skip("CLOUDCONVERT_API_KEY not set, skipping test")
	} else if os.Getenv("CLOUDCONVERT_TASK_ID") == "" {
		t.Skip("CLOUDCONVERT_TASK_ID not set, skipping test")
	}
	client := cloudconvert.NewClient(t.Context(), os.Getenv("CLOUDCONVERT_API_KEY"))
	task, err := client.GetTask(os.Getenv("CLOUDCONVERT_TASK_ID"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Task Details: %v", utils.PrettyPrintJSON(task))
}

func TestDownloadTaskFile(t *testing.T) {

	if os.Getenv("CLOUDCONVERT_API_KEY") == "" {
		t.Skip("CLOUDCONVERT_API_KEY not set, skipping test")
	} else if os.Getenv("CLOUDCONVERT_TASK_ID") == "" {
		t.Skip("CLOUDCONVERT_TASK_ID not set, skipping test")
	}

	client := cloudconvert.NewClient(t.Context(), os.Getenv("CLOUDCONVERT_API_KEY"))
	task, err := client.GetTask(os.Getenv("CLOUDCONVERT_TASK_ID"))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Task: %v", utils.PrettyPrintJSON(task))
	t.Logf("Task status: %v", task.Data.Status)

	switch task.Data.Status {
	case cloudconvert.TaskStatusWaiting, cloudconvert.TaskStatusProcessing:
		t.Skip("Task is not finished yet, skipping download test")
	case cloudconvert.TaskStatusError:
		t.Fatal("Task ended with error status")
	}

	if len(task.Data.Result.Files) == 0 {
		t.Fatal("No files found in export URL result")
	} else if task.Data.Result.Files[0].URL == "" {
		t.Fatal("No download URL found in export URL result")
	}

	t.Logf("Download URL: %v", task.Data.Result.Files[0].URL)

	s, err := client.DownloadAsFileStream(task.Data.Result.Files[0].URL)
	if err != nil {
		t.Fatal(err)
	}

	outputPath := "converted_" + task.Data.Result.Files[0].Filename

	outFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer outFile.Close()

	t.Logf("downloading to file: %v", outputPath)
	monitorReader := monitor.NewProgressReader(s, func(read int64) {
		t.Logf("downloaded %.2f MB", float64(read)/1024/1024)
	})

	if _, err := outFile.ReadFrom(monitorReader); err != nil {
		t.Fatal(err)
	}

}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	os.Setenv("DEBUG", "true")
}
