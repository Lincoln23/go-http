package main

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// MaxReadBytes represents the maximum number of bytes that we read from conn.Read().
const MaxReadBytes = 1024

type request struct {
	requestLine string
	headers     map[string]string
	requestBody string
}

func (r *request) String() string {
	// for k, v := range r.headers {
	// 	fmt.Printf("KEY: %s, VALUE: %s\n", k, v)
	// }
	return fmt.Sprintf("REQUEST LINE: %s\nHEADERS: %v\n", r.requestLine, r.headers)
}

func (r *request) readRequestLine(conn net.Conn, buffer []byte) ([]byte, error) {
	lineBuffer, buffer, err := readLine(conn, buffer)
	fmt.Println(string(buffer))
	if err != nil {
		return nil, err // errors.Wrap(err, "error calling readLine")
	}
	r.requestLine = string(lineBuffer)
	return buffer, nil
}

func (r *request) readHTTPHeader(conn net.Conn, buffer []byte) ([]byte, error) {
	lineBuffer, buffer, err := readLine(conn, buffer)
	if err != nil {
		return nil, errors.Wrap(err, "error calling readLine")
	}
	idx := findSubstring(lineBuffer, ":", 0)

	// Note that we will need to trim off the leading space and the colon.
	r.headers[strings.ToLower(string(lineBuffer[:idx]))] = string(lineBuffer[idx+2:])
	return buffer, nil
}

func (r *request) readHTTPHeaders(conn net.Conn, buffer []byte) ([]byte, error) {
	// Have we seen the last CRLF.
	tmp := make([]byte, MaxReadBytes)
	var err error

	for {
		// We could potentially optimize this further by checking if buffer
		// starts with `\r`.
		for {
			if len(buffer) < 2 {
				n, err := conn.Read(tmp)
				if err != nil {
					return nil, err
				}
				buffer = append(buffer, tmp[:n]...)
			} else {
				break
			}
		}

		if buffer[0] == '\r' && buffer[1] == '\n' {
			buffer = buffer[2:]
			break
		}

		buffer, err = r.readHTTPHeader(conn, buffer)
		if err != nil {
			return nil, err
		}
	}

	return buffer, nil
}

func (r *request) readRequestBody(conn net.Conn, buffer []byte) ([]byte, error) {
	valStr, ok := r.headers["content-length"]
	if !ok {
		return nil, nil
	}
	valInt, err := strconv.Atoi(valStr)
	if err != nil {
		return nil, err
	}
	if valInt == 0 {
		return nil, nil
	}
	for {
		if len(buffer) >= valInt {
			break
		}

		tmp := make([]byte, MaxReadBytes)
		n, err := conn.Read(tmp)
		if err != nil {
			fmt.Printf("BUST: %v\n", err)
			return nil, err
		}

		buffer = append(buffer, tmp[:n]...)
	}
	r.requestBody = string(buffer[:valInt])
	return nil, nil
}

// See https://tools.ietf.org/html/rfc2616#section-5 on parsing.
//
// Request       = Request-Line              ; Section 5.1
//                 *(( general-header        ; Section 4.5
//                  | request-header         ; Section 5.3
//                  | entity-header ) CRLF)  ; Section 7.1
//                 CRLF
//                 [ message-body ]          ; Section 4.3
func readRequest(conn net.Conn) (*request, error) {
	var err error
	var buffer []byte

	r := &request{
		headers: make(map[string]string),
	}

	// Parse Request-Line.
	// Format: `Method SP Request-URI SP HTTP-Version CRLF`.
	buffer, err = r.readRequestLine(conn, buffer)
	if err != nil {
		return nil, err //errors.Wrap(err, "error after calling readRequestLine")
	}

	// Parse HTTP Headers.
	buffer, err = r.readHTTPHeaders(conn, buffer)
	if err != nil {
		return nil, errors.Wrap(err, "error after calling readHTTPHeaders")
	}

	// Parse `message-body`.
	if _, err = r.readRequestBody(conn, buffer); err != nil {
		return nil, errors.Wrap(err, "error after calling readRequestBody")
	}
	return r, nil
}

func writeResponse(conn net.Conn) {
	b := []byte("HTTP/1.1 200 OK\r\nCache-Control: no-cache, private\r\nContent-Length: 5\r\nDate: Mon, 24 Nov 2014 10:21:21 GMT\r\n\r\nHello\r\n")
	conn.Write(b)
}

func main() {
	ln, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}
	for {
		// When we accept a new connection, the TCP handshake has already been completed.
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		req, err := readRequest(conn)
		if err != nil {
			if err == io.EOF {
				conn.Close()
				continue
			}
			panic(err)
		}

		fmt.Println(req)

		fmt.Println("BODY: " + req.requestBody)

		writeResponse(conn)
		conn.Close()
	}
}

// `readLine` extracts a string that ends with "\r\n" from `buffer`. If `buffer` does
// not have the desired markers, it will attempt to read at most `MaxReadBytes` bytes
// from the connection.
func readLine(conn net.Conn, buffer []byte) ([]byte, []byte, error) {
	startIndex := 0
	for {
		fmt.Printf("Finding in buffer: %v\n", string(buffer))
		if idx := findSubstring(buffer, "\r\n", startIndex); idx >= 0 {
			return buffer[:idx], buffer[idx+2:], nil
		}

		fmt.Printf("Reading more...\n")

		// TODO: Fix off-by-one error bug.
		// startIndex = len(buffer) - 1
		// if startIndex < 0 {
		// 	startIndex = 0
		// }

		tmp := make([]byte, MaxReadBytes)
		n, err := conn.Read(tmp)
		if err != nil {
			fmt.Printf("BUST: %v\n", err)
			return nil, nil, err
		}
		fmt.Println("tmp: " + string(tmp))

		buffer = append(buffer, tmp[:n]...)
	}
}

// `findSubstring` returns the index of `toFind` in `buffer` from `startIndex`.
// The first index in `buffer` is returned if found, and -1 otherwise. Note that
// this method is inefficient, and could be improved to O(m + n) using KMP.
func findSubstring(buffer []byte, toFind string, startIndex int) int {
	for i := startIndex; i < len(buffer)-len(toFind); i++ {
		found := true
		for j := 0; j < len(toFind); j++ {
			if buffer[i+j] != toFind[j] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}
