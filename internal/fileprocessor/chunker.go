package fileprocessor

import (
	"errors"
	"io"
)

type Chunker struct {
	ChunkSize int
	ChunkCount int
}

func NewChunkerWithSize(chunkSize int) *Chunker {
	return &Chunker{ChunkSize: chunkSize}
}

func NewChunkerWithCount(chunkCount int) *Chunker {
	return &Chunker{ChunkSize: chunkCount}
}

func (c *Chunker) ChunkData(r io.Reader) ([][]byte, error) {
	// Option 1: Split into a fixed number of chunks.
	if c.ChunkCount > 0 {
		// Read the entire data to determine total size.
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		totalSize := len(data)
		if c.ChunkCount > totalSize {
			// If more chunks are requested than there are bytes, adjust to one byte per chunk.
			c.ChunkCount = totalSize
		}

		baseChunkSize := totalSize / c.ChunkCount
		remainder := totalSize % c.ChunkCount

		var chunks [][]byte
		offset := 0
		for i := 0; i < c.ChunkCount; i++ {
			// Distribute the remainder across the first few chunks.
			extra := 0
			if i < remainder {
				extra = 1
			}
			currentChunkSize := baseChunkSize + extra
			chunk := data[offset : offset+currentChunkSize]
			chunks = append(chunks, chunk)
			offset += currentChunkSize
		}
		return chunks, nil
	}

	// Option 2: Split into chunks of a fixed byte size.
	if c.ChunkSize > 0 {
		var chunks [][]byte
		buf := make([]byte, c.ChunkSize)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				// Copy the bytes read into a new slice.
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				chunks = append(chunks, chunk)
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
		}
		return chunks, nil
	}

	return nil, errors.New("no valid chunking option provided: set either ChunkSize or ChunkCount")
}