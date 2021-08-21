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

type Lsp struct {
	name    string
	rootUri protocol.URI
	fileUri protocol.URI
	//lang     string
	//fileName string
}

var lsp Lsp

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

///var drawCommands = make(chan string)
var diagnostics = make(chan []protocol.Diagnostic)
var quit = make(chan struct{})

func launchLsp(lspName string) {
	var cmd *exec.Cmd
	switch lspName {
	case "gopls":
		lsp.name = "gopls"
		lsp.rootUri = "file:///home/slzatz/go_fragments"
		lsp.fileUri = "file:///home/slzatz/go_fragments/main.go"
		cmd = exec.Command("gopls", "serve", "-rpc.trace", "-logfile", "/home/slzatz/gopls_log")
	case "clangd":
		lsp.name = "clangd"
		lsp.rootUri = "file:///home/slzatz/clangd_examples"
		lsp.fileUri = "file:///home/slzatz/clangd_examples/test.cpp"
		cmd = exec.Command("clangd", "--log=verbose")
	}
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

	go readMessages() /**************************/

	//Client sends initialize method and server replies with result (not method): Capabilities ...)
	initializeRequest := JsonRequest{
		Jsonrpc: "2.0",
		Id:      1,
		Method:  "initialize",
		Params:  struct{}{},
	}

	params := protocol.InitializeParams{}
	params.ProcessID = 0
	//params.RootURI = "file:///home/slzatz/go_fragments"
	params.RootURI = lsp.rootUri
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
	// clangd doesn't send anything here

	// Client sends notification method:did/Open and server sends some notification (no id) method "window/logMessage"
	jsonNotification.Method = "textDocument/didOpen"
	var textParams protocol.DidOpenTextDocumentParams
	//textParams.TextDocument.URI = "file:///home/slzatz/go_fragments/main.go"
	textParams.TextDocument.URI = lsp.fileUri
	textParams.TextDocument.LanguageID = "go"
	textParams.TextDocument.Text = " "
	textParams.TextDocument.Version = 1
	jsonNotification.Params = textParams
	b, err = json.Marshal(jsonNotification)
	if err != nil {
		log.Fatal(err)
	}
	s = string(b)
	header = fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s
	io.WriteString(stdin, s)

	//go readMessages()

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

	sess.showEdMessage("LSP %s launched", lsp.name)
}

func shutdownLsp() {

	// tell server the file is closed
	jsonNotification.Method = "textDocument/didClose"
	var closeParams protocol.DidCloseTextDocumentParams
	//closeParams.TextDocument.URI = "file:///home/slzatz/go_fragments/main.go"
	closeParams.TextDocument.URI = lsp.fileUri
	jsonNotification.Params = closeParams
	b, err := json.Marshal(jsonNotification)
	if err != nil {
		log.Fatal(err)
	}
	s := string(b)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s
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

	// exit notification sent to server - hangs with clangd
	jsonNotification.Method = "exit"
	jsonNotification.Params = nil
	b, err = json.Marshal(jsonNotification)
	if err != nil {
		log.Fatal(err)
	}
	s = string(b)
	header = fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s
	io.WriteString(stdin, s)

	// this is blocking for clangd
	if lsp.name != "clangd" {
		quit <- struct{}{}
	}
}

func sendDidChangeNotification(text string) {
	jsonNotification.Method = "textDocument/didChange"
	jsonNotification.Params = protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				//URI: "file:///home/slzatz/go_fragments/main.go"},
				URI: lsp.fileUri},
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

func readMessages() {
	var length int64
	for {
		select {
		default:
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
		case <-quit:
			sess.showEdMessage("Shutdown LSP")
			return
		}
	}
}

// not in use
func readMessageAndDiscard() {
	var length int64

	line, err := stdoutRdr.ReadString('\n')
	if err != nil {
		sess.showOrgMessage("Error reading header: %s\n%v", string(line), err)
		return
	}
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
