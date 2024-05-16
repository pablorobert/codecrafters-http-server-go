package main

import (
	"fmt"
	"net"
	"os"
	"io"
	"strings"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")
	
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	
	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil {
		if err != io.EOF {
			fmt.Println("read error:", err)
			conn.Close()
			os.Exit(1)
		}
	}

	str := string(buf)
	lines := strings.Split(str, "\r\n")
	header := lines[0]
	if (header[0:3] == "GET") {
		ParseGetRequest(header, conn)
	}
}

func ParseGetRequest(header string, conn net.Conn) {
	parts := strings.Split(header, " ")
	request := parts[1]
	
	if request[0:1] == "/" && len(request) > 1 {
		if len(request) > 6 && request[0:6] == "/echo/" {
			EchoEndPoint(request[6:], conn)
		} else {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}
	} else if (request == "/") {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	}
}

func EchoEndPoint(echo string, conn net.Conn) {
	str := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echo),echo)
	fmt.Println(str)
	conn.Write([]byte(str))
}
