package response

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/agkmw/httpfromtcp/internal/headers"
)

var (
	ErrOutOfOrderCall = errors.New("error: out of order call")
)

type StatusCode int

const (
	StatusSuccess             StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

type writerState int

const (
	writerStateStatusLine writerState = iota
	writerStateHeaders
	writerStateBody
	writerStateChunkedBody
)

type Writer struct {
	conn        io.Writer
	writerState writerState
	hasTrailers bool
}

func NewWriter(conn io.Writer) *Writer {
	return &Writer{
		conn:        conn,
		writerState: writerStateStatusLine,
		hasTrailers: false,
	}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.writerState != writerStateStatusLine {
		return ErrOutOfOrderCall
	}
	_, err := fmt.Fprintf(w.conn, "HTTP/1.1 %d %s\r\n", statusCode, ReasonPhrase(statusCode))
	w.writerState = writerStateHeaders
	return err
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	if w.writerState != writerStateHeaders {
		return ErrOutOfOrderCall
	}
	if err := WriteHeaders(w.conn, headers); err != nil {
		return err
	}
	if headers.Get("Transfer-Encoding") == "chunked" {
		if headers.Get("Trailer") != "" {
			w.hasTrailers = true
		}
		w.writerState = writerStateChunkedBody
		return nil
	}
	w.writerState = writerStateBody
	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.writerState != writerStateBody {
		return 0, ErrOutOfOrderCall
	}
	n, err := fmt.Fprint(w.conn, string(p))
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.writerState != writerStateChunkedBody {
		return 0, ErrOutOfOrderCall
	}
	chunkSize := strconv.FormatInt(int64(len(p)), 16)
	n, err := fmt.Fprintf(w.conn, "%s\r\n", chunkSize)
	if err != nil {
		return 0, err
	}
	n, err = fmt.Fprintf(w.conn, "%s", p)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	n, err := fmt.Fprint(w.conn, "\r\n")
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	err := WriteHeaders(w.conn, h)
	if err != nil {
		return err
	}
	return nil
}

func ReasonPhrase(code StatusCode) string {
	switch code {
	case StatusSuccess:
		return "OK"
	case StatusBadRequest:
		return "Bad Request"
	case StatusInternalServerError:
		return "Internal Server Error"
	default:
		return ""
	}
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	headers := headers.NewHeaders()
	headers.Set("Content-Length", fmt.Sprintf("%d", contentLen))
	headers.Set("Connection", "close")
	headers.Set("Content-Type", "text/plain")
	return headers
}

func WriteHeaders(w io.Writer, headers headers.Headers) error {
	var err error
	for k, v := range headers {
		_, err = fmt.Fprintf(w, "%s: %s\r\n", k, v)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(w, "\r\n")
	return err
}
