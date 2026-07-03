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

type MP4Probe struct {
	HasFileType   bool  `json:"hasFileType"`
	HasMovie      bool  `json:"hasMovie"`
	HasVideoTrack bool  `json:"hasVideoTrack"`
	HasAudioTrack bool  `json:"hasAudioTrack"`
	DurationMs    int64 `json:"durationMs"`
}

func ProbeMP4(path string) (MP4Probe, error) {
	file, err := os.Open(path)
	if err != nil {
		return MP4Probe{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return MP4Probe{}, err
	}
	if info.IsDir() || info.Size() == 0 {
		return MP4Probe{}, nil
	}
	probe := MP4Probe{}
	if err := scanMP4Boxes(file, 0, uint64(info.Size()), 0, &probe); err != nil {
		return MP4Probe{}, err
	}
	return probe, nil
}

func mp4HasAudioTrack(path string) (bool, error) {
	probe, err := ProbeMP4(path)
	return probe.HasAudioTrack, err
}

func mp4HasVideoTrack(path string) (bool, error) {
	probe, err := ProbeMP4(path)
	return probe.HasVideoTrack, err
}

func scanMP4Boxes(file *os.File, start uint64, end uint64, depth int, probe *MP4Probe) error {
	if depth > mp4MaxProbeDepth {
		return nil
	}
	offset := start
	for offset+8 <= end {
		box, err := readMP4BoxHeader(file, offset, end)
		if err != nil {
			return err
		}
		if box.end <= offset {
			return fmt.Errorf("invalid mp4 box %q size at offset %d", box.kind, offset)
		}
		switch box.kind {
		case "ftyp":
			if depth == 0 {
				probe.HasFileType = true
			}
		case "mvhd":
			durationMs, err := readMP4MovieDurationMs(file, box)
			if err != nil {
				return err
			}
			if durationMs > probe.DurationMs {
				probe.DurationMs = durationMs
			}
		case "hdlr":
			boxHandlerType, err := readMP4HandlerType(file, box)
			if err != nil {
				return err
			}
			switch boxHandlerType {
			case mp4HandlerTypeSound:
				probe.HasAudioTrack = true
			case mp4HandlerTypeVideo:
				probe.HasVideoTrack = true
			}
		case "moov", "trak", "mdia":
			if box.kind == "moov" {
				probe.HasMovie = true
			}
			if err := scanMP4Boxes(file, box.payloadStart, box.end, depth+1, probe); err != nil {
				return err
			}
		}
		offset = box.end
	}
	return nil
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

func readMP4MovieDurationMs(file *os.File, box mp4BoxHeader) (int64, error) {
	if box.end < box.payloadStart+4 {
		return 0, fmt.Errorf("mp4 mvhd box is too small")
	}
	if _, err := file.Seek(int64(box.payloadStart), io.SeekStart); err != nil {
		return 0, err
	}
	header := make([]byte, 4)
	if _, err := io.ReadFull(file, header); err != nil {
		return 0, err
	}
	version := header[0]
	switch version {
	case 0:
		if box.end < box.payloadStart+20 {
			return 0, fmt.Errorf("mp4 mvhd version 0 box is too small")
		}
		payload := make([]byte, 16)
		if _, err := io.ReadFull(file, payload); err != nil {
			return 0, err
		}
		timescale := binary.BigEndian.Uint32(payload[8:12])
		duration := uint64(binary.BigEndian.Uint32(payload[12:16]))
		return mp4DurationMs(timescale, duration), nil
	case 1:
		if box.end < box.payloadStart+32 {
			return 0, fmt.Errorf("mp4 mvhd version 1 box is too small")
		}
		payload := make([]byte, 28)
		if _, err := io.ReadFull(file, payload); err != nil {
			return 0, err
		}
		timescale := binary.BigEndian.Uint32(payload[16:20])
		duration := binary.BigEndian.Uint64(payload[20:28])
		return mp4DurationMs(timescale, duration), nil
	default:
		return 0, fmt.Errorf("unsupported mp4 mvhd version %d", version)
	}
}

func mp4DurationMs(timescale uint32, duration uint64) int64 {
	if timescale == 0 || duration == 0 {
		return 0
	}
	return int64((duration * 1000) / uint64(timescale))
}
