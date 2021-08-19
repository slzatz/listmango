package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go.lsp.dev/protocol"
	//"github.com/go-language-server/protocol"
	//"go.lsp.dev/jsonrpc2"
)

type JsonRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type JsonResult struct {
	Jsonrpc string                    `json:"jsonrpc"`
	Id      int                       `json:"id"`
	Result  protocol.InitializeResult `json:"result"`
}

type JsonNotification struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

var jsonNotification = JsonNotification{
	Jsonrpc: "2.0",
	Method:  "initialize",
	Params:  struct{}{},
}

func counter() func() int32 {
	var n int32
	n = 3
	return func() int32 {
		n++
		return n
	}
}

var version = counter()

var stdin io.WriteCloser
var stdoutRdr *bufio.Reader

var drawCommands = make(chan string)
var diagnostics = make(chan []protocol.Diagnostic)

func launchLsp() {
	cmd := exec.Command("gopls", "serve", "-rpc.trace", "-logfile", "/home/slzatz/gopls_log")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sess.showOrgMessage("Failed to create stdout pipe: %v", err)
		return
	}
	stdin, err = cmd.StdinPipe()
	if err != nil {
		sess.showOrgMessage("Failed to launch LSP: %v", err)
		return
	}
	err = cmd.Start()
	if err != nil {
		sess.showOrgMessage("Failed to start LSP: %v", err)
		return
	}
	stdoutRdr = bufio.NewReaderSize(stdout, 10000)

	//Client sends initialize method and server replies with result (not method): Capabilities ...)
	initializeRequest := JsonRequest{
		Jsonrpc: "2.0",
		Id:      1,
		Method:  "initialize",
		Params:  struct{}{},
	}

	params := protocol.InitializeParams{}
	params.ProcessID = 0
	params.RootURI = "file:///home/slzatz/go_fragments"
	params.Capabilities = clientcapabilities
	initializeRequest.Params = params
	b, err := json.Marshal(initializeRequest)
	if err != nil {
		sess.showOrgMessage("Failed to marshal client capabilities json request: %v", err)
		return
	}
	s := string(b)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s

	io.WriteString(stdin, s)
	readMessageAndDiscard()

	//Client sends notification method:initialized and server replies with notification (no id) method "window/showMessage"
	jsonNotification.Method = "initialized"
	jsonNotification.Params = struct{}{}
	b, err = json.Marshal(jsonNotification)
	if err != nil {
		log.Fatal(err)
	}
	s = string(b)
	header = fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s
	io.WriteString(stdin, s)
	//fmt.Println("#4")
	pp := make([]byte, 10000)
	//fmt.Printf("buffer_out0 = %v\n", buffer_out0.Size())
	_, err = stdoutRdr.Read(pp)
	if err != nil {
		log.Fatal(err)
	}
	//fullRead := string(pp)
	/*
		fmt.Printf("Number of bytes read = %d\n", n)
		fmt.Printf("\n\n-------------------------------\n\n")
		fmt.Printf("Full Read = %s", fullRead)
	*/
	// Client sends notification method:did/Open and server replies with notification (no id) method "window/logMessage"
	// It looks like this is a notification and should not have an id
	//jsonMethod.Method = "textDocument/didOpen"
	jsonNotification.Method = "textDocument/didOpen"
	//jsonMethod.Id = 3
	var textParams protocol.DidOpenTextDocumentParams
	textParams.TextDocument.URI = "file:///home/slzatz/go_fragments/main.go"
	textParams.TextDocument.LanguageID = "go"
	textParams.TextDocument.Text = " "
	textParams.TextDocument.Version = 1
	//jsonMethod.Params = textParams
	jsonNotification.Params = textParams
	//b, err = json.Marshal(jsonMethod)
	b, err = json.Marshal(jsonNotification)
	if err != nil {
		log.Fatal(err)
	}
	s = string(b)
	header = fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s
	//fmt.Printf("\n\n%s\n\n", s)
	io.WriteString(stdin, s)
	ppp := make([]byte, 10000)
	//time.Sleep(2 * time.Second)
	//fmt.Println("#5")
	_, err = stdoutRdr.Read(ppp)
	if err != nil {
		log.Fatal(err)
	}
	go readMessages()

	// draining off any diagnostics before issuing didChange
	timeout := time.After(2 * time.Second)
L:
	for {
		select {
		case <-diagnostics:
		case <-timeout:
			break L
		default:
		}
	}

	sess.showEdMessage("LSP launched")
}

func shutdownLsp(dc chan string) {
	time.Sleep(time.Second * 2)

	select {
	case xyz := <-dc:
		fmt.Println(xyz)
	default:
		fmt.Println("D -> There wasn't anything on the channel")
	}

	// tell server the file is closed
	jsonNotification.Method = "textDocument/didClose"
	var closeParams protocol.DidCloseTextDocumentParams
	closeParams.TextDocument.URI = "file:///home/slzatz/go_fragments/main.go"
	jsonNotification.Params = closeParams
	b, err := json.Marshal(jsonNotification)
	if err != nil {
		log.Fatal(err)
	}
	s := string(b)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s
	//fmt.Printf("\n\n%s\n\n", s)
	io.WriteString(stdin, s)

	// shutdown request sent to server
	shutdownRequest := JsonRequest{
		Jsonrpc: "2.0",
		Id:      2,
		Method:  "shutdown",
		Params:  nil,
	}
	b, err = json.Marshal(shutdownRequest)
	if err != nil {
		log.Fatal(err)
	}
	s = string(b)
	header = fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s
	io.WriteString(stdin, s)
	fmt.Printf("\n\n%s\n\n", s)

	// exit notification semt to server
	jsonNotification.Method = "exit"
	jsonNotification.Params = nil
	b, err = json.Marshal(jsonNotification)
	if err != nil {
		log.Fatal(err)
	}
	s = string(b)
	header = fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s
	fmt.Printf("\n\n%s\n\n", s)
	io.WriteString(stdin, s)

	time.Sleep(3 * time.Second)
}

func sendDidChangeNotification(text string) {
	jsonNotification.Method = "textDocument/didChange"
	jsonNotification.Params = protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: "file:///home/slzatz/go_fragments/main.go"},
			Version: version()},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{{Text: text}},
	}
	b, err := json.Marshal(jsonNotification)
	if err != nil {
		log.Fatalf("\n%s\n%v", string(b), err)
	}
	s := string(b)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s
	io.WriteString(stdin, s)

}

func readMessageAndDiscard() {
	var length int64
	line, err := stdoutRdr.ReadString('\n')
	/*
		if err == io.EOF {
			fmt.Printf("\n\nGot EOF presumably from shutdown\n\n")
			break
		}
	*/

	if err != nil {
		sess.showOrgMessage("Error reading header: %s\n%v", string(line), err)
		return
	}

	/*
		if line == "" {
			continue
		}
	*/

	colon := strings.IndexRune(line, ':')
	if colon < 0 {
		return
	}

	//name, value := line[:colon], strings.TrimSpace(line[colon+1:])
	value := strings.TrimSpace(line[colon+1:])

	if length, err = strconv.ParseInt(value, 10, 32); err != nil {
		return
	}

	if length <= 0 {
		return
	}

	// to read the last two chars of '\r\n\r\n'
	line, err = stdoutRdr.ReadString('\n')
	if err != nil {
		sess.showOrgMessage("Error reading header: %s\n%v", string(line), err)
		return
	}

	data := make([]byte, length)

	if _, err = io.ReadFull(stdoutRdr, data); err != nil {
		sess.showOrgMessage("In Discard, Error ReadFull %v", err)
	}
}

func readMessages() {
	var length int64
	for {
		line, err := stdoutRdr.ReadString('\n')
		if err == io.EOF {
			fmt.Printf("\n\nGot EOF presumably from shutdown\n\n")
			break
		}
		if err != nil {
			log.Fatalf("\nRead -> %s\n%v", string(line), err)
		}

		if line == "" {
			continue
		}

		colon := strings.IndexRune(line, ':')
		if colon < 0 {
			continue
		}

		value := strings.TrimSpace(line[colon+1:])

		if length, err = strconv.ParseInt(value, 10, 32); err != nil {
			continue
		}

		if length <= 0 {
			continue
		}

		// to read the last two chars of '\r\n\r\n'
		line, err = stdoutRdr.ReadString('\n')
		if err != nil {
			log.Fatalf("\nRead -> %s\n%v", string(line), err)
		}

		data := make([]byte, length)

		if _, err = io.ReadFull(stdoutRdr, data); err != nil {
			continue
		}

		var v JsonNotification
		err = json.Unmarshal(data, &v)
		if err != nil {
			log.Fatalf("\nB -> %s\n%v", string(data[2:]), err)
		}

		if v.Method == "textDocument/publishDiagnostics" {
			type JsonPubDiag struct {
				Jsonrpc string                            `json:"jsonrpc"`
				Method  string                            `json:"method"`
				Params  protocol.PublishDiagnosticsParams `json:"params"`
			}
			var vv JsonPubDiag
			err = json.Unmarshal(data, &vv)

			diagnostics <- vv.Params.Diagnostics
		}
	}
}
