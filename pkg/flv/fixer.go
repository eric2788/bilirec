package flv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sync"

	"github.com/eric2788/bilirec/pkg/pool"
)

const (
	TagTypeAudio  = 0x08
	TagTypeVideo  = 0x09
	TagTypeScript = 0x12

	JumpThreshold = 500

	AudioDurationFallback = 22
	AudioDurationMin      = 20
	AudioDurationMax      = 24

	VideoDurationFallback = 33
	VideoDurationMin      = 15
	VideoDurationMax      = 50

	// 🔥 優化: Buffer 大小常量
	DefaultBufferSize = 8 * 1024  // 8KB - Raspberry Pi 友好
	MaxBufferSize     = 64 * 1024 // 64KB - 最大緩衝
	TagHeaderSize     = 11
	FlvHeaderSize     = 9
	PrevTagSizeBytes  = 4
)

var (
	FlvHeader = []byte{'F', 'L', 'V', 0x01, 0x05, 0x00, 0x00, 0x00, 0x09}

	ErrNotFlvFile = errors.New("not a valid FLV file")
	ErrInvalidTag = errors.New("invalid FLV tag")

	// 🔥 優化: sync.Pool 用於復用 buffer 和對象
	byteBufferPool = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 0, DefaultBufferSize))
		},
	}

	tagPool = sync.Pool{
		New: func() any {
			return &Tag{}
		},
	}

	headerBytesPool = pool.NewBufferPool(TagHeaderSize)
	smallBytesPool  = pool.NewBufferPool(PrevTagSizeBytes)
)

// Tag represents a complete FLV tag
type Tag struct {
	Type       byte
	DataSize   uint32
	Timestamp  int32
	StreamID   [3]byte
	Data       []byte
	IsHeader   bool
	IsKeyframe bool
}

// 🔥 優化:  重置 Tag 以便復用
func (t *Tag) Reset() {
	t.Type = 0
	t.DataSize = 0
	t.Timestamp = 0
	t.StreamID = [3]byte{0, 0, 0}
	t.Data = nil
	t.IsHeader = false
	t.IsKeyframe = false
}

// TimestampStore tracks timestamp fixing state (session-based)
type TimestampStore struct {
	FirstChunk          bool
	LastOriginal        int32
	CurrentOffset       int32
	NextTimestampTarget int32
}

func (ts *TimestampStore) Reset() {
	ts.FirstChunk = true
	ts.LastOriginal = 0
	ts.CurrentOffset = 0
	ts.NextTimestampTarget = 0
}

// =====================================================
// Helper:  Write Tag to Stream
// =====================================================

func writeTagOptimized(w io.Writer, tag *Tag) error {
	// 🔥 優化: 從 pool 取得 header buffer
	header := headerBytesPool.GetBuffer()
	defer headerBytesPool.PutBuffer(header)

	header[0] = tag.Type

	header[1] = byte(tag.DataSize >> 16)
	header[2] = byte(tag.DataSize >> 8)
	header[3] = byte(tag.DataSize)

	header[4] = byte(tag.Timestamp >> 16)
	header[5] = byte(tag.Timestamp >> 8)
	header[6] = byte(tag.Timestamp)
	header[7] = byte(tag.Timestamp >> 24)

	copy(header[8:11], tag.StreamID[:])

	if _, err := w.Write(header); err != nil {
		return err
	}

	if _, err := w.Write(tag.Data); err != nil {
		return err
	}

	// 🔥 優化:  從 pool 取得 prevTagSize buffer
	prevTagSize := smallBytesPool.GetBuffer()
	defer smallBytesPool.PutBuffer(prevTagSize)

	binary.BigEndian.PutUint32(prevTagSize, uint32(11+len(tag.Data)))
	if _, err := w.Write(prevTagSize); err != nil {
		return err
	}

	return nil
}

func WriteTag(w io.Writer, tag *Tag) error {
	return writeTagOptimized(w, tag)
}

// =====================================================
// Advanced: Group-based Timestamp Calculation
// =====================================================

func CalculateNextTargetAdvanced(tags []*Tag) int32 {
	videoTags := make([]*Tag, 0)
	audioTags := make([]*Tag, 0)

	for _, tag := range tags {
		switch tag.Type {
		case TagTypeVideo:
			videoTags = append(videoTags, tag)
		case TagTypeAudio:
			audioTags = append(audioTags, tag)
		}
	}

	videoDuration := int32(0)
	if len(videoTags) >= 2 {
		duration := videoTags[1].Timestamp - videoTags[0].Timestamp
		if duration >= VideoDurationMin && duration <= VideoDurationMax {
			videoDuration = videoTags[len(videoTags)-1].Timestamp + duration
		} else {
			videoDuration = videoTags[len(videoTags)-1].Timestamp + VideoDurationFallback
		}
	} else if len(videoTags) == 1 {
		videoDuration = videoTags[0].Timestamp + VideoDurationFallback
	}

	audioDuration := int32(0)
	if len(audioTags) >= 2 {
		duration := audioTags[1].Timestamp - audioTags[0].Timestamp
		if duration >= AudioDurationMin && duration <= AudioDurationMax {
			audioDuration = audioTags[len(audioTags)-1].Timestamp + duration
		} else {
			audioDuration = audioTags[len(audioTags)-1].Timestamp + AudioDurationFallback
		}
	} else if len(audioTags) == 1 {
		audioDuration = audioTags[0].Timestamp + AudioDurationFallback
	}

	return int32(math.Max(float64(videoDuration), float64(audioDuration)))
}

func CalculateNextTarget(tag *Tag) int32 {
	duration := int32(VideoDurationFallback)
	if tag.Type == TagTypeAudio {
		duration = AudioDurationFallback
	}
	return tag.Timestamp + duration
}
