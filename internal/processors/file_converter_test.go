package processors_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/eric2788/bilirec/internal/processors"
	"github.com/eric2788/bilirec/pkg/pipeline"
)

func TestConvertToMp4(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	} else if !ffmpegAvailable() {
		t.Skip("ffmpeg not available, skipping test")
	}

	var file *os.File
	var err error

	if os.Getenv("CI") != "" {
		file, err = os.CreateTemp("", "output_video_*.mp4")
	} else {
		file, err = os.Create("output_video.mp4")
	}

	if err != nil {
		t.Fatalf("failed to create output file: %v", err)
	}
	defer file.Close()

	converter := pipeline.New(
		processors.NewMp4FileConverter(
			processors.FileConvertWithDestPath(file.Name()),
		),
	)
	if err := converter.Open(t.Context()); err != nil {
		t.Fatalf("failed to open converter: %v", err)
	}
	defer converter.Close()
	_, err = converter.Process(t.Context(), "original.flv")
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("original.flv does not exist, skipping test")
		}
		t.Fatalf("failed to convert file: %v", err)
	}

	t.Logf("converted file saved at %s", file.Name())
}

func ffmpegAvailable() bool {
	cmd := exec.Command("ffmpeg", "-h")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
