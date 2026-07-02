package recpackage

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	mp4HandlerTypeSound = "soun"
	mp4HandlerTypeVideo = "vide"
	mp4MaxProbeDepth    = 8
)

func mp4HasAudioTrack(path string) (bool, error) {
	return mp4HasHandlerType(path, mp4HandlerTypeSound)
}

func mp4HasVideoTrack(path string) (bool, error) {
	return mp4HasHandlerType(path, mp4HandlerTypeVideo)
}

func mp4HasHandlerType(path string, handlerType string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return false, err
	}
	if info.IsDir() || info.Size() == 0 {
		return false, nil
	}
	return scanMP4BoxesForHandlerType(file, 0, uint64(info.Size()), 0, handlerType)
}

func scanMP4BoxesForHandlerType(file *os.File, start uint64, end uint64, depth int, handlerType string) (bool, error) {
	if depth > mp4MaxProbeDepth {
		return false, nil
	}
	offset := start
	for offset+8 <= end {
		box, err := readMP4BoxHeader(file, offset, end)
		if err != nil {
			return false, err
		}
		if box.end <= offset {
			return false, fmt.Errorf("invalid mp4 box %q size at offset %d", box.kind, offset)
		}
		switch box.kind {
		case "hdlr":
			boxHandlerType, err := readMP4HandlerType(file, box)
			if err != nil {
				return false, err
			}
			if boxHandlerType == handlerType {
				return true, nil
			}
		case "moov", "trak", "mdia":
			matched, err := scanMP4BoxesForHandlerType(file, box.payloadStart, box.end, depth+1, handlerType)
			if err != nil || matched {
				return matched, err
			}
		}
		offset = box.end
	}
	return false, nil
}

type mp4BoxHeader struct {
	kind         string
	payloadStart uint64
	end          uint64
}

func readMP4BoxHeader(file *os.File, offset uint64, limit uint64) (mp4BoxHeader, error) {
	if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
		return mp4BoxHeader{}, err
	}
	header := make([]byte, 8)
	if _, err := io.ReadFull(file, header); err != nil {
		return mp4BoxHeader{}, err
	}
	size := uint64(binary.BigEndian.Uint32(header[0:4]))
	kind := string(header[4:8])
	headerSize := uint64(8)
	if size == 1 {
		extendedSize := make([]byte, 8)
		if _, err := io.ReadFull(file, extendedSize); err != nil {
			return mp4BoxHeader{}, err
		}
		size = binary.BigEndian.Uint64(extendedSize)
		headerSize = 16
	}
	if size == 0 {
		size = limit - offset
	}
	if size < headerSize {
		return mp4BoxHeader{}, fmt.Errorf("mp4 box %q at offset %d has invalid size %d", kind, offset, size)
	}
	boxEnd := offset + size
	if boxEnd > limit || boxEnd < offset {
		return mp4BoxHeader{}, fmt.Errorf("mp4 box %q at offset %d exceeds file bounds", kind, offset)
	}
	return mp4BoxHeader{
		kind:         kind,
		payloadStart: offset + headerSize,
		end:          boxEnd,
	}, nil
}

func readMP4HandlerType(file *os.File, box mp4BoxHeader) (string, error) {
	if box.end < box.payloadStart+12 {
		return "", fmt.Errorf("mp4 hdlr box is too small")
	}
	if _, err := file.Seek(int64(box.payloadStart+8), io.SeekStart); err != nil {
		return "", err
	}
	handlerType := make([]byte, 4)
	if _, err := io.ReadFull(file, handlerType); err != nil {
		return "", err
	}
	return string(handlerType), nil
}
