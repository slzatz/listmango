package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

//var stream jsonrpc2.Stream
var ctx context.Context
var conn io.ReadWriteCloser

type Lsp struct {
	name       string
	rootUri    protocol.URI
	fileUri    protocol.URI
	languageID protocol.LanguageIdentifier
}

var lsp Lsp

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

var diagnostics = make(chan []protocol.Diagnostic)
var quit = make(chan struct{})

var logFile *os.File

func launchLsp(lspName string) {

	/*
		//ctx = context.Background()
		client, server := net.Pipe()
		stream = jsonrpc2.NewStream(server)

		// below available in the tools jsonrpc2 I believe
		//headerStream := jsonrpc2.NewHeaderStream(fakenet.NewConn("stdio", os.Stdin, os.Stdout))

		//sess.showOrgMessage("+%v", headerStream)
		conn := jsonrpc2.NewConn(stream)
		sess.showOrgMessage("+%v", stream)
		sess.showOrgMessage("+%v", client)
		sess.showOrgMessage("+%v", conn)
	*/

	var cmd *exec.Cmd
	switch lspName {
	case "gopls":
		lsp.name = "gopls"
		lsp.rootUri = "file:///home/slzatz/go_fragments"
		lsp.fileUri = "file:///home/slzatz/go_fragments/main.go"
		lsp.languageID = "go"
		cmd = exec.Command("gopls", "serve", "-rpc.trace", "-logfile", "/home/slzatz/gopls_log")
	case "clangd":
		lsp.name = "clangd"
		lsp.rootUri = "file:///home/slzatz/clangd_examples"
		lsp.fileUri = "file:///home/slzatz/clangd_examples/test.cpp"
		lsp.languageID = "cpp"
		cmd = exec.Command("clangd", "--log=verbose")
		logFile, _ := os.Create("/home/slzatz/clangd_log")
		cmd.Stderr = logFile
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
	params := protocol.InitializeParams{
		ProcessID:    0,
		RootURI:      lsp.rootUri,
		Capabilities: clientcapabilities,
	}
	request, err := jsonrpc2.NewCall(jsonrpc2.NewNumberID(1), "initialize", params)
	if err != nil {
		log.Fatal(err)
	}
	send(request)

	//Client sends notification method:initialized and
	//server replies with notification (no id) method "window/showMessage"
	notify, err := jsonrpc2.NewNotification("initialized", struct{}{}) //has to be struct{}{} not nil
	if err != nil {
		log.Fatal(err)
	}
	send(notify)
	// clangd doesn't send anything back here

	// Client sends notification method:did/Open and
	//server sends some notification (no id) method "window/logMessage"
	textParams := protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI: lsp.fileUri,
			//LanguageID: "go",
			LanguageID: lsp.languageID,
			Text:       " ",
			Version:    1,
		},
	}
	notify, err = jsonrpc2.NewNotification("textDocument/didOpen", textParams)
	if err != nil {
		log.Fatal(err)
	}
	send(notify)

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

	sess.showOrgMessage("LSP %s launched", lsp.name)
}

func shutdownLsp() {
	// tell server the file is closed
	closeParams := protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: lsp.fileUri,
		},
	}
	notify, err := jsonrpc2.NewNotification("textDocument/didClose", closeParams)
	if err != nil {
		log.Fatal(err)
	}
	send(notify)

	// shutdown request sent to server
	request, err := jsonrpc2.NewCall(jsonrpc2.NewNumberID(2), "shutdown", nil)
	if err != nil {
		log.Fatal(err)
	}
	send(request)

	// exit notification sent to server - hangs with clangd
	notify, err = jsonrpc2.NewNotification("exit", nil)
	if err != nil {
		log.Fatal(err)
	}
	send(notify)

	if lsp.name == "clangd" { //"clangd" {
		logFile.Close()
		//quit <- struct{}{}
	} else {
		// this blocks for clangd so readMessages go routine doesn't terminate
		quit <- struct{}{}
	}
	sess.showOrgMessage("Shutdown LSP")

	lsp.name = ""
	lsp.rootUri = ""
	lsp.fileUri = ""
}

func sendDidChangeNotification(text string) {

	params := protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: lsp.fileUri},
			Version: version()},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{{Text: text}},
	}

	notify, err := jsonrpc2.NewNotification("textDocument/didChange", params)
	if err != nil {
		log.Fatal(err)
	}
	send(notify)
}

func readMessages() {
	var length int64
	name := lsp.name
	for {
		select {
		default:
			line, err := stdoutRdr.ReadString('\n')
			if err == io.EOF {
				// clangd never gets <-quit
				// but if you launch another lsp this is triggered
				sess.showEdMessage("ReadMessages(%s): Got EOF", name)
				return
			}
			if err != nil {
				sess.showEdMessage("ReadMessages: %s-%v", string(line), err)
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
				sess.showEdMessage("ReadMessages: %s-%v", string(line), err)
			}

			data := make([]byte, length)

			if _, err = io.ReadFull(stdoutRdr, data); err != nil {
				continue
			}

			msg, err := jsonrpc2.DecodeMessage(data)
			switch msg := msg.(type) {
			case jsonrpc2.Request:
				//if call, ok := msg.(*jsonrpc2.Call); ok {
				if _, ok := msg.(*jsonrpc2.Call); ok {
					sess.showEdMessage("Request received")
				} else {
					notification := msg.(*jsonrpc2.Notification)
					notification.UnmarshalJSON(data)
					if notification.Method() == "textDocument/publishDiagnostics" {
						var params protocol.PublishDiagnosticsParams
						err := json.Unmarshal(notification.Params(), &params)
						if err != nil {
							sess.showEdMessage("Error: %v", err)
						}
						diagnostics <- params.Diagnostics
					}
				}
			case *jsonrpc2.Response:
				//sess.showEdMessage("Response/Result received")
			}
		case <-quit: //clangd never gets here; gopls does
			return
		}
	}
}

func send(msg json.Marshaler) {
	b, err := msg.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}
	s := string(b)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(s))
	s = header + s

	io.WriteString(stdin, s)
}

//from go.lsp.dev.pkg/fakeroot
/*
func NewConn(name string, in io.ReadCloser, out io.WriteCloser) net.Conn {
	c := &fakeConn{
		name:   name,
		reader: newFeeder(in.Read),
		writer: newFeeder(out.Write),
		in:     in,
		out:    out,
	}
	go c.reader.run()
	go c.writer.run()
	return c
}
*/
