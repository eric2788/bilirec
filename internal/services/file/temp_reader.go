package file

import "os"

type TempReader struct {
	*os.File
	path string
}

func NewTempReader(file *os.File) *TempReader {
	return &TempReader{
		File: file,
		path: file.Name(),
	}
}

func (tr *TempReader) Close() error {
	err := tr.File.Close()
	if err == nil {
		err = os.Remove(tr.path)
	}
	return err
}
