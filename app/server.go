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

type httpRequest struct {
	httpMethod string
	httpPath string
	paths []string
	httpVersion string
	headers map[string]string
	conn net.Conn
}

type httpResponse struct {
	httpStatusCode int
	httpStatusDescption string
	headers []string
	body string
}

type statusCodeMap struct {
	status int
	description string
}

var statusMap = map[int]string{
	200: "OK",
	201: "Created",
	400: "Bad Request",
	404: "Not Found",
	500: "Internal Server Error",
}

func NewHTTPResponse(status int) *httpResponse {
	description := statusMap[status]
	return &httpResponse{
		httpStatusCode: status,
		httpStatusDescption: description,
		headers: []string{},
	}
}

func (r *httpResponse) addHeader(header string) {
	r.headers = append(r.headers, header)
}

func (r *httpResponse) sendResponse(conn net.Conn) {
	firstLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", 
		r.httpStatusCode, 
		r.httpStatusDescption)
	conn.Write([]byte(firstLine))

	for _, v := range r.headers {
		conn.Write([]byte(fmt.Sprintf("%s\r\n", v)))
	}

	if (len(r.body) > 0) {
		conn.Write([]byte("\r\n"))
		conn.Write([]byte(r.body))
	}

	conn.Write([]byte("\r\n"))
}	

var httpHeaders map[string]string = make(map[string]string)

var settings config
func init() {
	settings = config{}
}

func NewHTTPRequest() httpRequest {
	return httpRequest{}
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for i := 1; i < len(os.Args); i++ {
		if (os.Args[i] == "--directory") {
			settings.directory = os.Args[i + 1]
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

	request := NewHTTPRequest()
	request.conn = conn
	lines := strings.Split(string(buf), "\r\n")

	ReadHTTPHeaders(&request, lines[1:])
	fmt.Sscanf(lines[0], "%s %s %s", &request.httpMethod, &request.httpPath, &request.httpVersion)
	
	if (request.httpPath == "/") {
		request.paths = []string{"/"}
	} else {
		path := request.httpPath[1:]
		request.paths = strings.Split(path, "/")
	}

	if (request.httpMethod == "GET") {
		ParseGetRequest(request)
	}

	if (request.httpMethod == "POST") {
		ParsePostRequest(request, lines[1:])
	}
}

func ReadHTTPHeaders(request *httpRequest, headers []string) {
	for i := 0; i < len(headers); i++ {
		if headers[i] == "" {
			break
		}
		parts := strings.Split(headers[i], ":")
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		httpHeaders[key] = strings.Replace(value, "\r\n", "", -1)
	}
	request.headers = httpHeaders
}

func ParseGetRequest(request httpRequest) {
	if request.paths[0] == "/" {
		NewHTTPResponse(200).sendResponse(request.conn)
		return
	} else {
		endpoint := request.paths[0]
		if endpoint == "user-agent" {
			UserAgentEndPoint(request)
			return
		}
		if endpoint == "files" {
			FileEndPoint(request)
			return
		}
		if endpoint == "echo" {
			EchoEndPoint(request)
			return
		}
		NewHTTPResponse(404).sendResponse(request.conn)
	}
}

func ParsePostRequest(request httpRequest, headers []string) {
	
	var fileContent string =""
	var buffer bytes.Buffer
	var length int
	var err error

	if contentLength, ok := request.headers["content-length"]; ok {
		length, err = strconv.Atoi(contentLength)
		if err != nil {
			length = 0
		}
	}

	idx := 0
	for i := 0; i < len(headers); i++ {
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

	endpoint := request.paths[0]
		
	if endpoint == "files" {
		PostFileEndPoint(request, fileContent)
		return
	}
	NewHTTPResponse(404).sendResponse(request.conn)
}

func UserAgentEndPoint(request httpRequest) {
	response := NewHTTPResponse(200)
	if userAgent, ok := request.headers["user-agent"]; ok {
		response.addHeader("Content-Type: text/plain")
		response.addHeader(fmt.Sprintf("Content-Length: %d", len(userAgent)))
		response.body = userAgent

		response.sendResponse(request.conn)
	}
}

func PostFileEndPoint(request httpRequest, fileContent string) {
	fullPath := settings.directory + request.paths[1]
	file, err := os.Create(fullPath)
	if err != nil {
		os.Exit(-2) //"Erro ao criar arquivo"
	}
	defer file.Close()
	_, err = file.WriteString(fileContent)
	if err != nil {
		os.Exit(-2) //"Erro ao escrever arquivo"
	}
	NewHTTPResponse(201).sendResponse(request.conn)
}

func FileEndPoint(request httpRequest) {
	filePath := request.paths[1]
	fullPath := settings.directory + filePath
	if fi, err := os.Stat(fullPath); err == nil {
		content, err := os.ReadFile(fullPath)
		if err != nil {
			os.Exit(-3) //"Erro ao ler conteudo do arquivo"
		}

		response := NewHTTPResponse(200)
		response.addHeader("Content-Type: application/octet-stream")
		response.addHeader("Content-Length: " + fmt.Sprintf("%d", fi.Size()))
		response.body = string(content)
		
		response.sendResponse(request.conn)
		return
	}
	NewHTTPResponse(404).sendResponse(request.conn)
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

func EchoEndPoint(request httpRequest) {
	response := NewHTTPResponse(200)
	response.addHeader("Content-Type: text/plain")
	echo := request.paths[1]

	if encoding, ok := request.headers["accept-encoding"]; ok {
		if (encoding == "gzip" || strings.Contains(encoding, "gzip")) {
			gzipped := GzipString(echo)
	
			response.addHeader("Content-Encoding: gzip")
			response.addHeader("Content-Length: " + fmt.Sprintf("%d", len(gzipped)))
			response.body = gzipped
		} else {
			response.addHeader("Content-Length: " + fmt.Sprintf("%d", len(echo)))
			response.body = echo
		}
	}
	response.sendResponse(request.conn)
}
