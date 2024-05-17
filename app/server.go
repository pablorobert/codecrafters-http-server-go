package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"io"
	"strings"
	"bytes"
	"strconv"
	"compress/gzip"
)

type config struct {
	directory string
}

var httpHeaders map[string]string = make(map[string]string)

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
	requestHeader := lines[0]
	headers := lines[1:]
	ReadHTTPHeaders(headers)

	if (requestHeader[0:3] == "GET") {
		ParseGetRequest(requestHeader, headers, conn)
	}
	if (requestHeader[0:4] == "POST") {
		ParsePostRequest(requestHeader, headers, conn)
	}
}

func ReadHTTPHeaders(headers []string) {
	for i := 0; i < len(headers); i++ {
		if headers[i] == "" {
			break
		}
		breakHeader(headers[i])
	}
}

func breakHeader(header string) {
	parts := strings.Split(header, ":")
	key := strings.ToLower(strings.TrimSpace(parts[0]))
	value := strings.TrimSpace(parts[1])
	httpHeaders[key] = value
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

func ParsePostRequest(header string, headers []string, conn net.Conn) {
	parts := strings.Split(header, " ")
	request := parts[1]
	
	var fileContent string =""
	var buffer bytes.Buffer
	var length int
	var err error

	for _, header := range headers {
		if strings.Contains(header, "Content-Length") {
			parts = strings.Split(header, ": ")
			length, err = strconv.Atoi(parts[1])
			if err != nil {
				length = 0
			}
		}
	}

	idx := 0
	fmt.Println("len", len(headers))
	for i := 0; i < len(headers); i++ {
		fmt.Println(headers[i])
		if (headers[i] == "") {
			idx = i
		}
	}
	for i := idx; i < len(headers); i++ {
		if (len(headers[i]) == 0) {
			continue
		}
		buffer.WriteString(headers[i][0:length])
	}
	fileContent = buffer.String()

	if request[0:1] == "/" && len(request) > 1 {
		if len(request) >= 7 && request[0:7] == "/files/" {
			PostFileEndPoint(request[7:], fileContent, conn)
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

func PostFileEndPoint(filePath string, fileContent string, conn net.Conn) {
	fullPath := settings.directory + filePath
	fmt.Println(fullPath)
	file, err := os.Create(fullPath)
	/*err := os.WriteFile(fullPath, []byte(fileContent), 0644)*/
	if err != nil {
		os.Exit(-2)
		//"Erro ao criar arquivo"
	}
	defer file.Close()
	_, err = file.WriteString(fileContent)
	if err != nil {
		os.Exit(-2)
		//"Erro ao escrever arquivo"
	}
	conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
	//conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
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

func GzipString(str string) string {
	var b bytes.Buffer
    gz := gzip.NewWriter(&b)
    if _, err := gz.Write([]byte(str)); err != nil {
        log.Fatal(err)
    }
    if err := gz.Close(); err != nil {
        log.Fatal(err)
    }
	return b.String()
}

func EchoEndPoint(echo string, conn net.Conn) {
	encoding := httpHeaders["accept-encoding"]
	var str string
	if (encoding == "gzip" || strings.Contains(encoding, "gzip")) {
		encoding = "gzip"
		gzipped := GzipString(echo)

		str = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Encoding: %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
			encoding, len(gzipped), gzipped)
	} else {
		str = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
			len(echo),echo)
	}
	conn.Write([]byte(str))
}
