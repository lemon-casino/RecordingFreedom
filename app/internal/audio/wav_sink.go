package audio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const (
	wavHeaderSize      = 44
	wavFormatIEEEFloat = 3
	wavFloat32Bytes    = 4
)

type WAVSink struct {
	id         string
	path       string
	file       *os.File
	sampleRate int
	channels   int
	dataBytes  uint32
	closed     bool
}

func NewWAVSink(id string, path string) (*WAVSink, error) {
	if id == "" {
		return nil, errors.New("wav sink id is required")
	}
	if path == "" {
		return nil, errors.New("wav sink path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &WAVSink{id: id, path: path, file: file}, nil
}

func (s *WAVSink) ID() string {
	return s.id
}

func (s *WAVSink) Append(buffer ProcessedBuffer) error {
	if s.closed {
		return fmt.Errorf("wav sink %q is closed", s.id)
	}
	pcm := buffer.Buffer
	if pcm.SampleRate <= 0 {
		return errors.New("wav sink requires a positive sample rate")
	}
	if pcm.Channels <= 0 {
		return errors.New("wav sink requires a positive channel count")
	}
	if len(pcm.Samples)%pcm.Channels != 0 {
		return fmt.Errorf("wav sink got %d samples that do not align to %d channels", len(pcm.Samples), pcm.Channels)
	}
	if err := s.ensureHeader(pcm.SampleRate, pcm.Channels); err != nil {
		return err
	}
	if pcm.SampleRate != s.sampleRate || pcm.Channels != s.channels {
		return fmt.Errorf("wav sink format changed from %dHz/%dch to %dHz/%dch", s.sampleRate, s.channels, pcm.SampleRate, pcm.Channels)
	}
	if len(pcm.Samples) == 0 {
		return nil
	}
	if uint64(s.dataBytes)+uint64(len(pcm.Samples))*wavFloat32Bytes > math.MaxUint32 {
		return errors.New("wav sink exceeded 4GB RIFF data limit")
	}

	scratch := make([]byte, len(pcm.Samples)*wavFloat32Bytes)
	for index, sample := range pcm.Samples {
		binary.LittleEndian.PutUint32(scratch[index*wavFloat32Bytes:], math.Float32bits(sample))
	}
	written, err := s.file.Write(scratch)
	s.dataBytes += uint32(written)
	return err
}

func (s *WAVSink) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if s.sampleRate == 0 {
		s.sampleRate = RNNoiseSampleRate
		s.channels = 1
		if err := s.writeHeader(s.sampleRate, s.channels, 0); err != nil {
			_ = s.file.Close()
			return err
		}
	}
	if err := s.rewriteHeader(); err != nil {
		_ = s.file.Close()
		return err
	}
	return s.file.Close()
}

func (s *WAVSink) ensureHeader(sampleRate int, channels int) error {
	if s.sampleRate != 0 {
		return nil
	}
	s.sampleRate = sampleRate
	s.channels = channels
	return s.writeHeader(sampleRate, channels, 0)
}

func (s *WAVSink) rewriteHeader() error {
	if _, err := s.file.Seek(0, 0); err != nil {
		return err
	}
	if err := s.writeHeader(s.sampleRate, s.channels, s.dataBytes); err != nil {
		return err
	}
	_, err := s.file.Seek(0, 2)
	return err
}

func (s *WAVSink) writeHeader(sampleRate int, channels int, dataBytes uint32) error {
	if sampleRate <= 0 || channels <= 0 {
		return errors.New("wav header requires a positive sample rate and channel count")
	}
	blockAlign := uint16(channels * wavFloat32Bytes)
	byteRate := uint32(sampleRate) * uint32(blockAlign)
	header := make([]byte, wavHeaderSize)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], 36+dataBytes)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], wavFormatIEEEFloat)
	binary.LittleEndian.PutUint16(header[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], byteRate)
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)
	binary.LittleEndian.PutUint16(header[34:36], 32)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], dataBytes)
	_, err := s.file.Write(header)
	return err
}
