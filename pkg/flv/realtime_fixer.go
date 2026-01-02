package flv

import (
	"bytes"
	"sync"
)

// =====================================================
// REALTIME FIXER - 逐個 Tag 修復並輸出
// =====================================================

type RealtimeFixer struct {
	mu            sync.Mutex
	tsStore       *TimestampStore
	buffer        *bytes.Buffer
	headerWritten bool
	pendingTags   []*Tag
	dedupCache    *DedupCache // 🔥 新增:  去重緩存
	dupCount      int64       // 🔥 新增: 重複計數
}

func NewRealtimeFixer() *RealtimeFixer {
	return &RealtimeFixer{
		tsStore:       &TimestampStore{FirstChunk: true},
		buffer:        byteBufferPool.Get(), // 🔥 優化: 從 pool 取得
		headerWritten: false,
		pendingTags:   make([]*Tag, 0, 32),
		dedupCache:    NewDedupCache(MaxDedupCacheSize, DedupWindowMs), // 🔥 初始化去重
		dupCount:      0,
	}
}

// 🔥 新增: 獲取去重統計
func (rf *RealtimeFixer) GetDedupStats() (duplicates int64, cacheSize int, cacheCapacity int) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	size, capacity := rf.dedupCache.GetStats()
	return rf.dupCount, size, capacity
}

// Fix processes incoming bytes and returns fixed FLV data
func (rf *RealtimeFixer) Fix(input []byte) ([]byte, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	rf.buffer.Write(input)

	// 🔥 優化: 從 pool 取得輸出 buffer
	output := byteBufferPool.Get()
	output.Reset()

	// Write FLV header once
	if !rf.headerWritten && rf.buffer.Len() >= 9 {
		header := rf.buffer.Next(9)
		if !bytes.Equal(header[:3], []byte{'F', 'L', 'V'}) {
			return nil, ErrNotFlvFile
		}
		output.Write(header)
		// Write initial PreviousTagSize0 = 0
		output.Write([]byte{0, 0, 0, 0})
		rf.headerWritten = true
	}

	// Parse complete tags from buffer
	for {
		// Need PreviousTagSize (4) + TagHeader (11) minimum
		if rf.buffer.Len() < 15 {
			break
		}

		// 🔥 優化: 避免完整拷貝，使用切片
		bufLen := rf.buffer.Len()

		// Skip PreviousTagSize
		rf.buffer.Next(PrevTagSizeBytes)

		// Peek tag header
		if rf.buffer.Len() < TagHeaderSize {
			// 🔥 優化:  使用 Grow + 手動回退而非完整拷貝
			rf.buffer.Reset()
			remaining := input[len(input)-(bufLen):]
			rf.buffer.Write(remaining)
			break
		}

		// 🔥 優化: 從 pool 取得 header buffer
		headerBytes := headerBytesPool.GetBytes()
		rf.buffer.Read(headerBytes)

		tagType := headerBytes[0]
		dataSize := uint32(headerBytes[1])<<16 | uint32(headerBytes[2])<<8 | uint32(headerBytes[3])

		// Check if we have complete tag data
		if rf.buffer.Len() < int(dataSize) {
			// 恢復:  需要更多數據
			headerBytesPool.PutBytes(headerBytes)

			// 🔥 優化: 手動構建最小恢復
			tempBuf := byteBufferPool.Get()
			tempBuf.Reset()
			tempBuf.Write([]byte{0, 0, 0, 0}) // PrevTagSize
			tempBuf.Write(headerBytes)
			tempBuf.Write(rf.buffer.Bytes())

			rf.buffer.Reset()
			rf.buffer.Write(tempBuf.Bytes())

			tempBuf.Reset()
			byteBufferPool.Put(tempBuf)
			break
		}

		// 🔥 優化: 複用 tag 對象但數據還是要拷貝 (因為會被修改)
		tagData := make([]byte, dataSize)
		rf.buffer.Read(tagData)

		// Parse timestamp (24bit + 8bit extended)
		timestamp := int32(headerBytes[7])<<24 | int32(headerBytes[4])<<16 |
			int32(headerBytes[5])<<8 | int32(headerBytes[6])

		// Create tag
		tag := tagPool.Get().(*Tag)
		tag.Reset()
		tag.Type = tagType
		tag.DataSize = dataSize
		tag.Timestamp = timestamp
		tag.Data = tagData
		copy(tag.StreamID[:], headerBytes[8:11])

		// 返還 header buffer
		headerBytesPool.PutBytes(headerBytes)

		// Detect header/keyframe
		if len(tagData) >= 2 {
			switch tagType {
			case TagTypeVideo:
				firstByte := tagData[0]
				secondByte := tagData[1]
				tag.IsKeyframe = (firstByte & 0xF0) == 0x10
				tag.IsHeader = secondByte == 0x00
			case TagTypeAudio:
				if (tagData[0]>>4) == 10 && len(tagData) >= 2 { // AAC
					tag.IsHeader = tagData[1] == 0x00
				}
			}
		}

		// 🔥 新增: 去重檢查 (在修復時間戳之前)
		if rf.dedupCache.IsDuplicate(tag) {
			rf.dupCount++
			tagPool.Put(tag)
			continue // 跳過重複的 tag
		}

		// Fix timestamp
		rf.fixTimestamp(tag)

		// Write fixed tag
		if err := writeTagOptimized(output, tag); err != nil {
			return nil, err
		}

		// 🔥 優化:  返還 tag 到 pool (但保留 Data 因為已經寫入)
		tagPool.Put(tag)
	}

	// 🔥 新增: 定期清理過期去重記錄
	if rf.tsStore.LastOriginal > 0 {
		rf.dedupCache.CleanOld(rf.tsStore.LastOriginal)
	}

	// 🔥 優化:  返回複製的數據，這樣 output buffer 可以被復用
	result := make([]byte, output.Len())
	copy(result, output.Bytes())

	output.Reset()
	byteBufferPool.Put(output)

	return result, nil
}

// 🔥 優化:  釋放資源
func (rf *RealtimeFixer) Close() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.buffer != nil {
		rf.buffer.Reset()
		byteBufferPool.Put(rf.buffer)
		rf.buffer = nil
	}

	if rf.dedupCache != nil {
		rf.dedupCache.Reset()
	}
}

func (rf *RealtimeFixer) fixTimestamp(tag *Tag) {
	ts := rf.tsStore
	currentTimestamp := tag.Timestamp

	// First chunk special handling
	if ts.FirstChunk {
		ts.FirstChunk = false
		ts.CurrentOffset = currentTimestamp
	}

	diff := currentTimestamp - ts.LastOriginal

	// Detect timestamp jump
	if diff < -JumpThreshold || (ts.LastOriginal == 0 && diff < 0) {
		ts.CurrentOffset = currentTimestamp - ts.NextTimestampTarget
	} else if diff > JumpThreshold {
		ts.CurrentOffset = currentTimestamp - ts.NextTimestampTarget
	}

	ts.LastOriginal = currentTimestamp

	// Apply offset
	tag.Timestamp -= ts.CurrentOffset

	// Calculate next target
	ts.NextTimestampTarget = CalculateNextTarget(tag)
}
