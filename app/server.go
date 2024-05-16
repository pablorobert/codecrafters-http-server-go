package main

import (
	"fmt"
	"net"
	"os"
	"io"
	"strings"
)

type config struct {
	directory string
}

var settings config

func init() {
	settings = config{}
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	argsWithoutProgram := os.Args[1:]
	for i := 0; i < len(argsWithoutProgram); i++ {
		if (argsWithoutProgram[i] == "--directory") {
			settings.directory = argsWithoutProgram[i+1]
		}
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go Server(conn)
	}
	
}

func Server(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
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
	headers := lines[1:]
	if (header[0:3] == "GET") {
		ParseGetRequest(header, headers, conn)
	}
}

func ParseGetRequest(header string, headers []string, conn net.Conn) {
	parts := strings.Split(header, " ")
	request := parts[1]
	
	if request == "/" && len(request) == 1 {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		return
	}

	if request[0:1] == "/" && len(request) > 1 {
		if len(request) >= 11 && request[0:11] == "/user-agent" {
			UserAgentEndPoint(request[11:], headers, conn)
		} else if len(request) >= 7 && request[0:7] == "/files/" {
			FileEndPoint(request[7:], conn)
		} else if len(request) >= 6 && request[0:6] == "/echo/" {
			EchoEndPoint(request[6:], conn)
		} else {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}
	}
}

func UserAgentEndPoint(echo string, headers []string, conn net.Conn) {
	for _, header := range headers {
		if strings.Contains(header, "User-Agent") {
			parts := strings.Split(header, ":")
			userAgent := strings.TrimSpace(parts[1])
			str := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)
			conn.Write([]byte(str))
			return
		}
	}
}

func FileEndPoint(filePath string, conn net.Conn) {
	fullPath := settings.directory + filePath
	if fi, err := os.Stat(fullPath); err == nil {
		content, err := os.ReadFile(fullPath)
		if err != nil {
			os.Exit(-3)
			//"Erro ao ler conteudo do arquivo"
		}
		conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
		conn.Write([]byte("Content-Length: " + fmt.Sprintf("%d", fi.Size()) + "\r\n"))
		conn.Write([]byte("Content-Type: application/octet-stream\r\n\r\n"))
		conn.Write(content)
		return
	}
	conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
}

func EchoEndPoint(echo string, conn net.Conn) {
	str := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echo),echo)
	conn.Write([]byte(str))
}
