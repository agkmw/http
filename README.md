# httpfromtcp — Zero-dependency HTTP/1.1 server from raw TCP, with chunked streaming and binary data

[![Course Link](https://img.shields.io/badge/Boot.dev-Learn%20HTTP-blue)](https://www.boot.dev/courses/learn-http-protocol-golang)

httpfromtcp is an educational Go project that implements HTTP/1.1 from scratch — request parsing, response generation, chunked transfer encoding, and trailer headers — entirely on top of raw `net.Conn` without importing `net/http`. It demonstrates how HTTP really works at the wire format by building a working server that streams video files, proxies third-party APIs with content verification, and serves dynamic responses, all through hand-rolled state machines over TCP byte streams.

## Features

- **Zero-dependency HTTP/1.1 wire parsing**: Incremental state-machine parser (Initialized → Headers → Body → Done) that validates request lines, RFC 9110 header tokens, and Content-Length body framing directly from TCP reads — no `net/http`, no `bufio`, no third-party libraries
- **Chunked transfer encoding with trailers**: Full RFC 7230 chunked response writer that streams data in `hex-size\r\ndata` frames and supports post-body trailer headers (used to attach SHA-256 hashes and content length metadata to proxied responses)
- **Reverse proxy with content verification**: Proxies `/httpbin/*` requests through to httpbin.org, streams the response body as chunked encoding, and appends `x-content-sha256` and `x-content-length` trailers computed over the full payload
- **Binary video streaming over HTTP**: Streams a local MP4 file (assets/vim.mp4) as a chunked HTTP response with the correct `Content-Type: video/mp4` — a real demonstration of serving binary data through a hand-rolled HTTP stack
- **Concurrent TCP server with graceful shutdown**: Goroutine-per-connection accept loop, `atomic.Bool` shutdown signaling, and transient `Accept()` error resilience — no worker pools, no mutex contention in the hot path
- **Response state machine enforcement**: `response.Writer` tracks a strict state sequence (StatusLine → Headers → Body | ChunkedBody → Trailers) and returns `ErrOutOfOrderCall` on protocol violations

## Project Structure

```text
cmd/
├── httpserver/
│   └── main.go       — server entry point: prefix-based routing, reverse proxying, and file streaming
├── tcplistener/
│   └── main.go       — debug tool: raw TCP HTTP/1.1 request parsing and inspection
└── udpsender/
    └── main.go       — UDP client: interactive stdin-to-datagram sender
internal/
├── headers/
│   └── headers.go    — header parser: incremental field-line parsing, RFC 9110 token validation, and duplicate combination
├── request/
│   └── request.go    — request parser: state-machine driven HTTP/1.1 request deserialization
├── response/
│   └── response.go   — response writer: status line, headers, fixed body, chunked encoding, and trailers
└── server/
    └── server.go     — TCP server: accept loop, connection dispatch, and graceful shutdown
```

## Getting Started

```bash
# Start the server on port 42069
go run ./cmd/httpserver

# Static responses
curl -v http://localhost:42069/
curl -v http://localhost:42069/yourproblem   # 400
curl -v http://localhost:42069/myproblem     # 500

# Chunked proxy response with SHA-256 trailers
curl -v http://localhost:42069/httpbin/anything

# Download sample video
mkdir assets
curl -o assets/vim.mp4 https://storage.googleapis.com/qvault-webapp-dynamic-assets/lesson_videos/vim-vs-neovim-prime.mp4

# Chunked binary video stream
curl -v http://localhost:42069/video -o /dev/null

# Inspect raw TCP request parsing
go run ./cmd/tcplistener &
printf "GET / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 11\r\n\r\nHello World" | nc localhost 42069
```
