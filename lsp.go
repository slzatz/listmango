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
	//completionUri protocol.URI
}

var lsp Lsp

func counter() func() int32 {
	var n int32
	//n = 3
	return func() int32 {
		n++
		return n
	}
}

var version = counter()
var idNum = counter()

var stdin io.WriteCloser
var stdoutRdr *bufio.Reader

var diagnostics = make(chan []protocol.Diagnostic)

//var completion = make(chan protocol.CompletionList)
var quit = make(chan struct{})

var logFile *os.File
var requestType = make(map[jsonrpc2.ID]string)

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
	//lsp.completionUri = "/home/slzatz/completion"

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
	//id := jsonrpc2.NewNumberID(1)
	id := jsonrpc2.NewNumberID(idNum())
	requestType[id] = "initialize"
	request, err := jsonrpc2.NewCall(id, "initialize", params)
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
	id := jsonrpc2.NewNumberID(idNum())
	requestType[id] = "shutdown"
	request, err := jsonrpc2.NewCall(id, "shutdown", nil)
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

	if lsp.name == "clangd" {
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

func sendCompletionRequest(line, character uint32) {

	progressToken := protocol.NewProgressToken("test")

	// Since it doesn't appear possible to send the text of the file
	// you would have to save a scratch file somewhere so that
	// you could do autocomplete without specifically having user save ??
	params := protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: lsp.fileUri},
			//URI: lsp.completionUri},
			Position: protocol.Position{
				Line:      line,
				Character: character}},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{
			WorkDoneToken: progressToken},
		PartialResultParams: protocol.PartialResultParams{
			PartialResultToken: progressToken},
		Context: nil,
	}

	id := jsonrpc2.NewNumberID(idNum())
	requestType[id] = "completion"
	request, err := jsonrpc2.NewCall(id, "textDocument/completion", params)
	if err != nil {
		log.Fatal(err)
	}
	send(request)
}

func sendHoverRequest(line, character uint32) {
	progressToken := protocol.NewProgressToken("test")
	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: lsp.fileUri},
			Position: protocol.Position{
				Line:      line,
				Character: character}},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{
			WorkDoneToken: progressToken},
	}
	id := jsonrpc2.NewNumberID(idNum())
	requestType[id] = "hover"
	request, err := jsonrpc2.NewCall(id, "textDocument/hover", params)
	if err != nil {
		log.Fatal(err)
	}
	send(request)
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
				//if call, ok := msg.(*jsonrpc2.Call); ok
				if _, ok := msg.(*jsonrpc2.Call); ok {
					sess.showEdMessage("Request received")
				} else {
					notification := msg.(*jsonrpc2.Notification)
					notification.UnmarshalJSON(data)
					if notification.Method() == "textDocument/publishDiagnostics" {
						var params protocol.PublishDiagnosticsParams
						err := json.Unmarshal(notification.Params(), &params)
						if err != nil {
							sess.showEdMessage("Error unmarshaling diagnostics: %v", err)
							return
						}
						diagnostics <- params.Diagnostics
					}
				}
			case *jsonrpc2.Response:
				msg.UnmarshalJSON(data)
				id := msg.ID()
				result := msg.Result()

				switch requestType[id] {
				case "initialize", "shutdown":
					continue
				case "completion":
					var completion protocol.CompletionList
					err := json.Unmarshal(result, &completion)
					if err != nil {
						sess.showEdMessage("Error: %v", err)
					}
					//sess.showOrgMessage("Completion: %+v", completion.Items[0].Label)
					p.drawCompletionItems(completion)
					//sess.showEdMessage("Response/Result received")
				case "hover":
					var hover protocol.Hover
					err := json.Unmarshal(result, &hover)
					if err != nil {
						sess.showEdMessage("Error: %v", err)
					}
					p.drawHover(hover)
				}
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
