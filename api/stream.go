package api

import (
	"bufio"
	"encoding/json"
	"io"
)

// LogMessage represents a single log line from the Docker log stream.
type LogMessage struct {
	Stream string
	Data   []byte
}

// NewLogReader wraps a Docker multiplexed log stream into a plain io.ReadCloser.
// When tty is true the stream is passed through raw; otherwise the 8-byte Docker
// stream header per frame is stripped.
func NewLogReader(body io.ReadCloser, tty bool) io.ReadCloser {
	if tty {
		return body
	}
	return &demuxReader{src: body}
}

type demuxReader struct {
	src io.ReadCloser
	buf []byte
}

func (d *demuxReader) Read(p []byte) (int, error) {
	for len(d.buf) == 0 {
		hdr := make([]byte, 8)
		if _, err := io.ReadFull(d.src, hdr); err != nil {
			return 0, err
		}
		size := int(hdr[4])<<24 | int(hdr[5])<<16 | int(hdr[6])<<8 | int(hdr[7])
		if size == 0 {
			continue
		}
		frame := make([]byte, size)
		if _, err := io.ReadFull(d.src, frame); err != nil {
			return 0, err
		}
		d.buf = frame
	}
	n := copy(p, d.buf)
	d.buf = d.buf[n:]
	return n, nil
}

func (d *demuxReader) Close() error {
	return d.src.Close()
}

// JSONLineScanner reads newline-delimited JSON objects from a stream.
type JSONLineScanner[T any] struct {
	scanner *bufio.Scanner
	current T
	err     error
}

// NewJSONLineScanner creates a scanner that decodes T from each line of the stream.
func NewJSONLineScanner[T any](r io.Reader) *JSONLineScanner[T] {
	return &JSONLineScanner[T]{scanner: bufio.NewScanner(r)}
}

// Scan advances to the next JSON object. Returns false when done.
func (s *JSONLineScanner[T]) Scan() bool {
	if !s.scanner.Scan() {
		s.err = s.scanner.Err()
		return false
	}
	var v T
	if err := json.Unmarshal(s.scanner.Bytes(), &v); err != nil {
		s.err = err
		return false
	}
	s.current = v
	return true
}

// Value returns the most recently scanned value.
func (s *JSONLineScanner[T]) Value() T { return s.current }

// Err returns the first non-EOF error encountered.
func (s *JSONLineScanner[T]) Err() error { return s.err }
