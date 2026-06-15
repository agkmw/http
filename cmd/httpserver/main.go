package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/agkmw/httpfromtcp/internal/headers"
	"github.com/agkmw/httpfromtcp/internal/request"
	"github.com/agkmw/httpfromtcp/internal/response"
	"github.com/agkmw/httpfromtcp/internal/server"
)

const template = `
<html>
  <head>
    <title>%d %s</title>
  </head>
  <body>
    <h1>%s</h1>
    <p>%s</p>
  </body>
</html>`

const port = 42069

func main() {
	server, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}

func handler(w *response.Writer, r *request.Request) {
	var (
		statusCode response.StatusCode
		statusText string
		heading    string
		body       string
	)

	switch {
	case strings.HasPrefix(r.RequestLine.RequestTarget, "/yourproblem"):
		statusCode = response.StatusBadRequest
		statusText = response.ReasonPhrase(statusCode)
		heading = statusText
		body = "Your request honestly kinda sucked."
	case strings.HasPrefix(r.RequestLine.RequestTarget, "/myproblem"):
		statusCode = response.StatusInternalServerError
		statusText = response.ReasonPhrase(statusCode)
		heading = statusText
		body = "Okay, you know what? This one is on me."
	case strings.HasPrefix(r.RequestLine.RequestTarget, "/httpbin"):
		chunkHandler(w, r)
		return
	case strings.HasPrefix(r.RequestLine.RequestTarget, "/video"):
		binHandler(w, r)
		return
	default:
		statusCode = response.StatusSuccess
		statusText = response.ReasonPhrase(statusCode)
		heading = "Success!"
		body = "Your request was an absolute banger."
	}

	w.WriteStatusLine(statusCode)
	data := fmt.Sprintf(
		template,
		statusCode,
		statusText,
		heading,
		body,
	)
	headers := response.GetDefaultHeaders(len(data))
	headers.Set("Content-Type", "text/html")
	w.WriteHeaders(headers)
	w.WriteBody([]byte(data))
}

func chunkHandler(w *response.Writer, r *request.Request) {
	path := strings.TrimPrefix(r.RequestLine.RequestTarget, "/httpbin")
	fmt.Println("path: ", path)
	url := fmt.Sprintf("https://httpbin.org/%s", path)
	resp, err := http.Get(url)
	if err != nil {
		herr := &server.HandlerError{
			StatusCode: response.StatusInternalServerError,
			Message:    err.Error(),
		}
		herr.Write(w)
		return
	}
	contentType := ""
	if strings.HasPrefix(path, "/html") {
		contentType = "text/html"
	}

	w.WriteStatusLine(response.StatusSuccess)

	h := response.GetDefaultHeaders(0)
	h.Remove("Content-Length")
	h.Add("Transfer-Encoding", "chunked")
	h.Add("Trailer", "x-content-sha256")
	h.Add("Trailer", "x-content-length")
	if contentType != "" {
		h.Set("content-type", contentType)
	}
	if err := w.WriteHeaders(h); err != nil {
		log.Printf("Error writing headers: %v", err)
		return
	}

	fullBody := make([]byte, 1024)
	for {
		buf := make([]byte, 1024)
		n, err := resp.Body.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) && n == 0 {
				break
			}
			log.Printf("Error reading response body: %v", err)
			return
		}
		fmt.Printf("read: %d bytes\n", n)

		m, err := w.WriteChunkedBody(buf[:n])
		if err != nil {
			log.Printf("Error writing chunked body: %v", err)
			return
		}
		fmt.Printf("wrote: %d bytes\n", m)

		_, err = w.WriteChunkedBodyDone()
		if err != nil {
			log.Printf("Error writing chunked body done: %v", err)
			return
		}
		fullBody = append(fullBody, buf[:n]...)
	}

	w.WriteChunkedBody([]byte{})

	hash := sha256.Sum256(fullBody)
	t := headers.NewHeaders()
	t.Set("x-content-sha256", string(hash[:]))
	t.Set("x-content-length", fmt.Sprintf("%d", len(fullBody)))
	if err := w.WriteTrailers(t); err != nil {
		log.Printf("Error writing trailers: %v", err)
		return
	}
}

func binHandler(w *response.Writer, r *request.Request) {
	h := headers.NewHeaders()

	f, err := os.Open("assets/vim.mp4")
	if err != nil {
		h.Set("Content-Type", "text/plain")
		w.WriteStatusLine(response.StatusInternalServerError)
		w.WriteHeaders(h)
		return
	}

	h.Set("Content-Type", "video/mp4")
	h.Set("Transfer-Encoding", "chunked")
	w.WriteStatusLine(response.StatusSuccess)
	w.WriteHeaders(h)

	buf := make([]byte, 1000)
	for {
		n, err := f.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) && n == 0 {
				break
			}
			log.Print(err)
			return
		}
		log.Printf("read: %d bytes", n)

		n, err = w.WriteChunkedBody(buf[:n])
		if err != nil {
			log.Print(err)
			return
		}
		log.Printf("wrote: %d bytes", n)

		_, err = w.WriteChunkedBodyDone()
		if err != nil {
			log.Print(err)
			return
		}
	}
	w.WriteChunkedBody([]byte{})
	_, err = w.WriteChunkedBodyDone()
	if err != nil {
		log.Print(err)
		return
	}
}
