package request

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/agkmw/httpfromtcp/internal/headers"
)

type requestState int

const (
	requestStateInitialized requestState = iota
	requestStateParsingHeaders
	requestStateParsingBody
	requestStateDone
)

const bufferSize = 8

type Request struct {
	RequestLine  RequestLine
	Headers      headers.Headers
	Body         []byte
	requestState requestState
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	req := &Request{
		Headers:      headers.NewHeaders(),
		requestState: requestStateInitialized,
	}
	buf := make([]byte, bufferSize)
	totalRead := 0
	for req.requestState != requestStateDone {
		if totalRead == len(buf) {
			newBuf := make([]byte, 2*cap(buf))
			copy(newBuf, buf)
			buf = newBuf
		}

		n, err := reader.Read(buf[totalRead:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				if req.requestState != requestStateDone {
					return nil, io.ErrUnexpectedEOF
				}
				break
			}
			return nil, err
		}
		totalRead += n

		consumed, err := req.parse(buf[:totalRead])
		if err != nil {
			return nil, err
		}
		copy(buf, buf[consumed:totalRead])
		totalRead -= consumed
	}

	return req, nil
}

func parseRequestLine(data []byte) (RequestLine, int, error) {
	i := strings.Index(string(data), "\r\n")
	if i == -1 {
		return RequestLine{}, 0, nil
	}
	requestLine := string(data[:i])

	// The request line always contains just 3 parts
	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		return RequestLine{}, 0, fmt.Errorf("invalid request line")
	}

	// NOTE: RFC9110 Section-9
	// Ensure the "method" part is all-uppercase US-ASCII letters
	method := parts[0]
	for _, c := range method {
		if !isCapital(c) {
			return RequestLine{}, 0, fmt.Errorf("invalid method")
		}
	}

	// NOTE: RFC9112 Section-2.3
	// HTTP-version is case-sensitive
	httpParts := strings.Split(parts[2], "/")
	httpName := httpParts[0]
	if httpName != "HTTP" {
		return RequestLine{}, 0, fmt.Errorf("invalid http version")
	}
	// Only support HTTP/1.1 for now
	httpVersion := httpParts[1]
	if httpVersion != "1.1" {
		return RequestLine{}, 0, fmt.Errorf("invalid http version")
	}

	return RequestLine{
		HttpVersion:   httpVersion,
		RequestTarget: parts[1],
		Method:        method,
	}, i + 2, nil // i + 2 to consume the \r\n
}

// parse accepts all currently unparsed bytes from the buffer, and
// updates the "state" of the parser and the parsed RequestLine field.
// It returns the number of bytes it consumed (meaning successfully parsed) and
// an error if it encountered one.
func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0

	for r.requestState != requestStateDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return 0, err
		}
		if n == 0 {
			// No more full tokens found in the current buffer,
			// break to read more from the network.
			break
		}
		totalBytesParsed += n
	}

	return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.requestState {
	case requestStateDone:
		return 0, errors.New("error: trying to read data in a done state")

	case requestStateInitialized:
		requestLine, n, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, nil
		}
		r.requestState = requestStateParsingHeaders
		r.RequestLine = requestLine
		return n, nil

	case requestStateParsingHeaders:
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			r.requestState = requestStateParsingBody
		}
		return n, nil

	case requestStateParsingBody:
		contentLength := r.Headers.Get("Content-Length")
		if contentLength == "" {
			r.requestState = requestStateDone
			return 0, nil
		}

		conLen, err := strconv.ParseInt(contentLength, 10, 64)
		if err != nil {
			return 0, errors.New("error: invalid Content-Length value")
		}

		r.Body = append(r.Body, data...)
		bodyLen := len(r.Body)
		if bodyLen > int(conLen) {
			return 0, errors.New("error: length of the body is greater than the Content-Length header")
		}
		if bodyLen == int(conLen) {
			r.requestState = requestStateDone
		}
		return len(data), nil

	default:
		return 0, errors.New("error: unknown state")
	}
}

func isCapital(b rune) bool {
	return 'A' <= b && b <= 'Z'
}
