package config

var ReadOnly *GlobalReadOnly = nil

// for global readonly access
type GlobalReadOnly struct {
	config *Config
}

func (g *GlobalReadOnly) UploadBufferSize() int {
	return g.config.uploadBufferSize
}

func (g *GlobalReadOnly) DownloadBufferSize() int {
	return g.config.downloadBufferSize
}

func (g *GlobalReadOnly) DownloadWriterBufferSize() int {
	return g.config.streamWriterBufferSize
}

func (g *GlobalReadOnly) LiveStreamWriterBufferSize() int {
	return g.config.liveStreamWriterBufferSize
}
