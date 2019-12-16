package main

import (
	"fmt"
	"net"
	"strings"
)

type request struct {
	requestLine string
	headers     []string
	requestBody string
}

func (r *request) String() string {
	return fmt.Sprintf("REQUEST LINE: %s\nHEADERS: %s\n", r.requestLine, strings.Join(r.headers, ","))
}

func readLine(conn net.Conn, buffer []byte) (string, []byte, error) {
	startIndex := 1
	for {
		if idx := findIndexOfNewLine(buffer, startIndex); idx >= 0 {
			return string(buffer[:idx]), buffer[idx+2:], nil
		}
		// Note, we do -1 here just in case that the previous buffer ends with \r.
		startIndex = len(buffer) - 1
		if startIndex < 1 {
			startIndex = 1
		}

		tmp := make([]byte, MAX_READ_BYTES)
		n, err := conn.Read(tmp)
		if err != nil {
			return "", nil, err
		}
		buffer = append(buffer, tmp[:n]...)
	}
}

func findIndexOfNewLine(buffer []byte, startIndex int) int {
	for i := startIndex; i < len(buffer); i++ {
		if buffer[i] == '\n' && buffer[i-1] == '\r' {
			return i - 1
		}
	}
	return -1
}

func (r *request) readRequestLine(conn net.Conn, buffer []byte) ([]byte, error) {
	line, buffer, err := readLine(conn, buffer)
	if err != nil {
		return nil, err
	}
	r.requestLine = line
	return buffer, err
}

func (r *request) readHTTPHeader(conn net.Conn, buffer []byte) ([]byte, error) {
	line, buffer, err := readLine(conn, buffer)
	if err != nil {
		return nil, err
	}
	r.headers = append(r.headers, line)
	return buffer, err
}

// Maximum number of bytes that we read from conn.Read().
const MAX_READ_BYTES = 1

func (r *request) readHTTPHeaders(conn net.Conn, buffer []byte) ([]byte, error) {
	// Have we seen the last CRLF.
	tmp := make([]byte, MAX_READ_BYTES)
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
	// TODO!!

	// n, err := conn.Read(tmp)
	// if err != nil {
	// 	return nil, err
	// }
	// buffer = append(buffer, tmp[:n])
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

	r := &request{}

	// Parse Request-Line.
	// Format: `Method SP Request-URI SP HTTP-Version CRLF`.
	buffer, err = r.readRequestLine(conn, buffer)
	if err != nil {
		return nil, err
	}

	// Parse HTTP Headers.
	buffer, err = r.readHTTPHeaders(conn, buffer)
	if err != nil {
		return nil, err
	}

	// Parse `message-body`.
	if _, err = r.readRequestBody(conn, buffer); err != nil {
		return nil, err
	}
	return r, nil
}

func main() {
	fmt.Println("Hello Lincoln")
	ln, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}

	for {
		fmt.Println("Accepting...")
		// When we accept the handshake has already been completed.
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		fmt.Println("Done accepting...")

		req, err := readRequest(conn)
		if err != nil {
			panic(err)
		}

		fmt.Println(req)
		writeResponse(conn)

		conn.Close()
	}
}

func writeResponse(conn net.Conn) {
	b := []byte("HTTP/1.1 200 OK\r\nCache-Control: no-cache, private\r\nContent-Length: 5\r\nDate: Mon, 24 Nov 2014 10:21:21 GMT\r\n\r\nHello\r\n")
	conn.Write(b)
}
