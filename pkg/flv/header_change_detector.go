package flv

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// ErrVideoHeaderChanged signals that the video sequence header (SPS/PPS) has
// changed and the recording pipeline should rotate to a new file.
var ErrVideoHeaderChanged = errors.New("video sequence header changed, pipe rotation required")

// FlvHeaderChangedError is a rich error returned by HeaderChangeDetector when
// the video sequence header changes. It carries the new tag bytes so callers
// can inject them into the next segment without an extra getter call.
type FlvHeaderChangedError struct {
	// VideoHeaderTag is the full FLV tag bytes (TagHeader + Data + PrevTagSize)
	// for the new video sequence header that triggered rotation.
	VideoHeaderTag []byte
	// AudioHeaderTag is the most recently seen full audio sequence header FLV
	// tag bytes, or nil if no audio sequence header has been observed yet.
	AudioHeaderTag []byte
	// TagOffset is the byte offset of the video seq header tag within the
	// original data slice passed to DetectChange. Split-detector processors
	// can use this to excise the tag from the data before forwarding it.
	TagOffset int
	// TagEnd is the byte offset one past the end of the video seq header tag
	// (excluding the trailing PrevTagSize bytes, which are also excised).
	TagEnd int
}

func (e *FlvHeaderChangedError) Error() string {
	return ErrVideoHeaderChanged.Error()
}

// Is allows errors.Is(err, ErrVideoHeaderChanged) to return true, and also
// correctly matches any wrapped sentinel (such as ErrShouldCut) that itself
// wraps ErrVideoHeaderChanged.
func (e *FlvHeaderChangedError) Is(target error) bool {
	return target == ErrVideoHeaderChanged || errors.Is(target, ErrVideoHeaderChanged)
}

// HeaderChangeDetector performs byte-level comparison of FLV video (and
// optionally audio) sequence headers within a raw FLV byte stream.
//
// It is stateful: call Reset() when starting a new recording session.
type HeaderChangeDetector struct {
	lastVideoHeader []byte
	lastVideoTag    []byte
	lastAudioHeader []byte
	lastAudioTag    []byte
	pending         []byte
}

// NewHeaderChangeDetector returns a ready-to-use detector.
func NewHeaderChangeDetector() *HeaderChangeDetector {
	return &HeaderChangeDetector{}
}

// Reset clears remembered headers (call at the start of a new recording file).
func (d *HeaderChangeDetector) Reset() {
	d.lastVideoHeader = nil
	d.lastVideoTag = nil
	d.lastAudioHeader = nil
	d.lastAudioTag = nil
	d.pending = nil
}

// SeedVideoHeader pre-seeds the detector with a known video sequence header so
// it can detect changes from a mid-stream header injected by a prior rotation.
// Pass the full FLV tag bytes (TagHeader + TagData + PrevTagSize) as stored in
// FlvHeaderChangedError.VideoHeaderTag. Must be called after Reset().
func (d *HeaderChangeDetector) SeedVideoHeader(fullTagBytes []byte) {
	if len(fullTagBytes) <= TagHeaderSize+PrevTagSizeBytes {
		return
	}
	tagData := fullTagBytes[TagHeaderSize : len(fullTagBytes)-PrevTagSizeBytes]
	d.lastVideoHeader = append([]byte(nil), tagData...)
	d.lastVideoTag = append([]byte(nil), fullTagBytes...)
}

// LastVideoHeader returns a copy of the most recently seen video sequence
// header tag data, or nil if none has been seen yet.
// Callers can inject this into a new file on pipe rotation.
func (d *HeaderChangeDetector) LastVideoHeader() []byte {
	if d.lastVideoHeader == nil {
		return nil
	}
	return append([]byte(nil), d.lastVideoHeader...)
}

// LastVideoTag returns a copy of the most recently seen video sequence
// header FLV tag bytes (TagHeader + TagData + PreviousTagSize), or nil.
func (d *HeaderChangeDetector) LastVideoTag() []byte {
	if d.lastVideoTag == nil {
		return nil
	}
	return append([]byte(nil), d.lastVideoTag...)
}

// LastAudioHeader returns a copy of the most recently seen audio sequence
// header tag data, or nil if none has been seen yet.
func (d *HeaderChangeDetector) LastAudioHeader() []byte {
	if d.lastAudioHeader == nil {
		return nil
	}
	return append([]byte(nil), d.lastAudioHeader...)
}

// LastAudioTag returns a copy of the most recently seen audio sequence header
// FLV tag bytes (TagHeader + TagData + PreviousTagSize), or nil.
func (d *HeaderChangeDetector) LastAudioTag() []byte {
	if d.lastAudioTag == nil {
		return nil
	}
	return append([]byte(nil), d.lastAudioTag...)
}

// DetectChange scans a raw FLV byte chunk for sequence-header tags.
// It returns ErrVideoHeaderChanged the first time a video sequence header
// with different binary content is found compared to the previously seen one.
// The detector remembers the most recently seen sequence headers, including
// the new video header that triggered ErrVideoHeaderChanged, so callers can
// retrieve it after rotation; call Reset() when starting a new recording file.
func (d *HeaderChangeDetector) DetectChange(data []byte) error {
	carryLen := len(d.pending)
	buf := make([]byte, 0, carryLen+len(data))
	if carryLen > 0 {
		buf = append(buf, d.pending...)
	}
	buf = append(buf, data...)

	offset := 0

	// Skip FLV file header + PreviousTagSize0 if this chunk starts with "FLV"
	if carryLen == 0 && len(buf) >= FlvHeaderSize && buf[0] == 'F' && buf[1] == 'L' && buf[2] == 'V' {
		offset = FlvHeaderSize + PrevTagSizeBytes
	}

	for offset+TagHeaderSize < len(buf) {
		tagType := buf[offset]
		dataSize := int(buf[offset+1])<<16 | int(buf[offset+2])<<8 | int(buf[offset+3])
		tagEnd := offset + TagHeaderSize + dataSize

		if tagEnd > len(buf) {
			break
		}

		tagData := buf[offset+TagHeaderSize : tagEnd]
		tagBytes := withPrevTagSize(buf[offset:tagEnd])

		switch tagType {
		case TagTypeVideo:
			if err := d.checkVideoHeader(tagData, tagBytes, offset, tagEnd); err != nil {
				if changed, ok := err.(*FlvHeaderChangedError); ok && carryLen > 0 {
					changed.TagOffset -= carryLen
					changed.TagEnd -= carryLen
					if changed.TagOffset < 0 {
						changed.TagOffset = 0
					}
					if changed.TagEnd < 0 {
						changed.TagEnd = 0
					}
				}
				d.pending = nil
				return err
			}
		case TagTypeAudio:
			if err := d.checkAudioHeader(tagData, tagBytes); err != nil {
				d.pending = nil
				return err
			}
		}

		// Advance: TagHeaderSize + dataSize + PreviousTagSize(4)
		offset = tagEnd + PrevTagSizeBytes
	}

	if offset < len(buf) {
		d.pending = append(d.pending[:0], buf[offset:]...)
	} else {
		d.pending = nil
	}

	return nil
}

func withPrevTagSize(tagBytes []byte) []byte {
	b := make([]byte, len(tagBytes)+PrevTagSizeBytes)
	copy(b, tagBytes)
	binary.BigEndian.PutUint32(b[len(tagBytes):], uint32(len(tagBytes)))
	return b
}

// checkVideoHeader inspects a video tag's payload for AVC sequence header
// changes. Returns ErrVideoHeaderChanged on a binary diff.
func (d *HeaderChangeDetector) checkVideoHeader(tagData []byte, tagBytes []byte, tagOffset int, tagEnd int) error {
	if len(tagData) < 2 {
		return nil
	}

	// tagData[0]: FrameType(4 bits) | CodecID(4 bits)
	// tagData[1]: AVCPacketType — only meaningful for AVC (CodecID == 7)
	codecID := tagData[0] & 0x0F
	if codecID != 7 { // not AVC/H.264 — skip
		return nil
	}

	const avcSequenceHeader = 0x00
	if tagData[1] != avcSequenceHeader {
		return nil
	}

	if d.lastVideoHeader == nil {
		// First time: remember and continue.
		d.lastVideoHeader = append([]byte(nil), tagData...)
		d.lastVideoTag = append([]byte(nil), tagBytes...)
		return nil
	}

	if !bytes.Equal(d.lastVideoHeader, tagData) {
		// Header changed — update state so the caller can retrieve the new
		// header via LastVideoHeader() after handling the error.
		d.lastVideoHeader = append([]byte(nil), tagData...)
		d.lastVideoTag = append([]byte(nil), tagBytes...)
		return &FlvHeaderChangedError{
			VideoHeaderTag: append([]byte(nil), tagBytes...),
			AudioHeaderTag: d.LastAudioTag(),
			TagOffset:      tagOffset,
			TagEnd:         tagEnd + PrevTagSizeBytes,
		}
	}

	return nil
}

// checkAudioHeader records AAC sequence header tag bytes.
func (d *HeaderChangeDetector) checkAudioHeader(tagData []byte, tagBytes []byte) error {
	if len(tagData) < 2 {
		return nil
	}
	// tagData[0] >> 4 == 10 → AAC; tagData[1] == 0 → AAC sequence header
	if (tagData[0]>>4) != 10 || tagData[1] != 0x00 {
		return nil
	}
	// Always update — both tagData and full tag bytes.
	d.lastAudioHeader = append([]byte(nil), tagData...)
	d.lastAudioTag = append([]byte(nil), tagBytes...)
	return nil
}
