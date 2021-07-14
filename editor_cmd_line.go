package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

//var e_lookup_C = map[string]interface{}{
var e_lookup_C = map[string]func(*Editor){
	"write":    (*Editor).writeNote,
	"w":        (*Editor).writeNote,
	"wa":       (*Editor).writeAll,
	"qa":       (*Editor).quitAll,
	"read":     (*Editor).readFile,
	"readfile": (*Editor).readFile,
	"resize":   (*Editor).resize,
	"compile":  (*Editor).compile,
	"c":        (*Editor).compile,
	"run":      (*Editor).run,
	"r":        (*Editor).run,
	"test":     (*Editor).sync,
	"sync":     (*Editor).sync,
	"save":     (*Editor).saveNoteToFile,
	"savefile": (*Editor).saveNoteToFile,
	"syntax":   (*Editor).syntax,
	"spell":    (*Editor).spell,
	"number":   (*Editor).number,
	"num":      (*Editor).number,
	"ha":       (*Editor).printNote,
	"modified": (*Editor).modified, // debugging
	"quit":     (*Editor).quitActions,
	"q":        (*Editor).quitActions,
	"quit!":    (*Editor).quitActions,
	"q!":       (*Editor).quitActions,
	"x":        (*Editor).quitActions,
	"fmt":      (*Editor).goFormat,
}

/* EDITOR cpp COMMAND_LINE mode lookup
const std::unordered_map<std::string, efunc> E_lookup_C {
 // all below handled (right now) in editor command line switch statement
 // {"x", &Editor::E_write_close_C},
 // {"quit", &Editor::E_quit_C},
 // {"q",&Editor:: E_quit_C},
 // {"quit!", &Editor::E_quit0_C},
 // {"q!", &Editor::E_quit0_C},
  {"vim", &Editor::E_open_in_vim_C},
  {"spell",&Editor:: E_spellcheck_C},
  {"spellcheck", &Editor::E_spellcheck_C},

  {"createlink", &Editor::createLink},
  //{"cl", &Editor::createLink},
  {"getlinked", &Editor::getLinked},
  {"gl", &Editor::getLinked},
  {"hide", &Editor::hide},
};
*/

func (e *Editor) saveNoteToFile() {
	pos := strings.Index(e.command_line, " ")
	if pos == -1 {
		sess.showEdMessage("You need to provide a filename")
		return
	}
	filename := e.command_line[pos+1:]
	f, err := os.Create(filename)
	if err != nil {
		sess.showEdMessage("Error creating file %s: %v", filename, err)
		return
	}
	defer f.Close()

	_, err = f.Write(bytes.Join(e.bb, []byte("\n")))
	if err != nil {
		sess.showEdMessage("Error writing file %s: %v", filename, err)
		return
	}
	sess.showEdMessage("Note written to file %s", filename)
}

func (e *Editor) writeNote() {
	updateNote(e)

	//uses nvim to write note to file for sole purpose of setting isModified to false
	err := v.Command("w")
	if err != nil {
		sess.showEdMessage("Error in writing file in editor.WriteNote: %v", err)
	}

	if taskFolder(e.id) == "code" {
		e.code = e.bufferToString()
		updateCodeFile(e)
	}
	e.drawStatusBar() //need this since now refresh won't do it unless redraw =true
	sess.showEdMessage("")
}

func (e *Editor) readFile() {
	pos := strings.Index(e.command_line, " ")
	if pos == -1 {
		sess.showEdMessage("You need to provide a filename")
		return
	}

	filename := e.command_line[pos+1:]
	err := e.readFileIntoNote(filename)
	if err != nil {
		sess.showEdMessage("%v", err)
		return
	}
	sess.showEdMessage("Note generated from file: %s", filename)
}

func (e *Editor) resize() {
	pos := strings.Index(e.command_line, " ")
	if pos == -1 {
		sess.showEdMessage("You need to provide a filename")
		return
	}
	pct, err := strconv.Atoi(e.command_line[pos+1:])
	if err != nil {
		sess.showEdMessage("You need to provide a number 0 - 100")
		return
	}
	moveDivider(pct)
}

func (e *Editor) compile() {

	var dir string
	var cmd *exec.Cmd
	//if getFolderTid(e.id) == 18 {
	if taskContext(e.id) == "cpp" {
		dir = "/home/slzatz/clangd_examples/"
		cmd = exec.Command("make")
	} else {
		dir = "/home/slzatz/go_fragments/"
		cmd = exec.Command("go", "build", "main.go")
	}
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sess.showEdMessage("Error in compile creating stdout pipe: %v", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		sess.showEdMessage("Error in compile creating stderr pipe: %v", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		sess.showEdMessage("Error in compile starting command: %v", err)
		return
	}

	buffer_out := bufio.NewReader(stdout)
	buffer_err := bufio.NewReader(stderr)

	var rows []string
	rows = append(rows, "------------------------")

	for {
		bytes, _, err := buffer_out.ReadLine()
		if err == io.EOF {
			break
		}
		rows = append(rows, string(bytes))
	}

	for {
		bytes, _, err := buffer_err.ReadLine()
		if err == io.EOF {
			break
		}
		rows = append(rows, string(bytes))
	}
	if len(rows) == 1 {
		rows = append(rows, "The code compiled successfully")
	}

	rows = append(rows, "------------------------")

	op := e.output
	op.rowOffset = 0
	op.rows = rows
	op.drawText()
	// no need to call drawFrame or drawStatusBar
}

func (e *Editor) run() {

	var args string
	pos := strings.Index(e.command_line, " ")
	if pos != -1 {
		args = e.command_line[pos+1:]
	}

	var dir string
	var obj string
	var cmd *exec.Cmd
	//if getFolderTid(e.id) == 18 {
	if taskContext(e.id) == "cpp" {
		obj = "./test_cpp"
		dir = "/home/slzatz/clangd_examples/"
	} else {
		obj = "./main"
		dir = "/home/slzatz/go_fragments/"
	}
	cmd = exec.Command(obj, args)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sess.showEdMessage("Error in run creating stdout pipe: %v", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		sess.showEdMessage("Error in run creating stderr pipe: %v", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		sess.showEdMessage("Error in run starting command: %v", err)
		return
	}

	buffer_out := bufio.NewReader(stdout)
	buffer_err := bufio.NewReader(stderr)

	var rows []string
	rows = append(rows, "------------------------")

	for {
		bytes, _, err := buffer_out.ReadLine()
		if err == io.EOF {
			break
		}
		rows = append(rows, string(bytes))
	}

	for {
		bytes, _, err := buffer_err.ReadLine()
		if err == io.EOF {
			break
		}
		rows = append(rows, string(bytes))
	}

	rows = append(rows, "------------------------")

	op := e.output
	op.rowOffset = 0
	op.rows = rows
	op.drawText()
	// no need to call drawFrame or drawStatusBar
}

func (e *Editor) sync() {
	var reportOnly bool
	if e.command_line == "test" {
		reportOnly = true
	}
	synchronize(reportOnly)
}

func (e *Editor) syntax() {
	e.highlightSyntax = !e.highlightSyntax
	if e.highlightSyntax {
		e.left_margin_offset = LEFT_MARGIN_OFFSET
		e.checkSpelling = false // can't syntax highlight(including markdown) and check spelling
	}
	e.drawText()
	// no need to call drawFrame or drawStatusBar
	sess.showEdMessage("Syntax highlighting is %v", e.highlightSyntax)
}

func (e *Editor) printNote() {
	err := v.Command("ha")
	if err != nil {
		sess.showEdMessage("Error printing: %v", err)
	}
}

// was for debugging
func (e *Editor) modified() {
	var result bool
	err := v.BufferOption(0, "modified", &result) //or e.vbuf
	if err != nil {
		sess.showEdMessage("%s", err)
		return
	}
	sess.showEdMessage("Modified = %t", result)
}

func (e *Editor) quitActions() {
	cmd := e.command_line
	if cmd == "x" {
		updateNote(e)

	} else if cmd == "q!" || cmd == "quit!" {
		// do nothing = allow editor to be closed

	} else if e.isModified() {
		e.mode = NORMAL
		e.command = ""
		e.command_line = ""
		sess.showEdMessage("No write since last change")
		return
	}
	deleteBufferOpts := map[string]bool{
		"force":  true,
		"unload": false,
	}
	err := v.DeleteBuffer(e.vbuf, deleteBufferOpts)
	if err != nil {
		sess.showOrgMessage("DeleteBuffer error %v", err)
	} else {
		sess.showOrgMessage("DeleteBuffer successful")
	}

	index := -1
	for i, w := range windows {
		if w == e {
			index = i
			break
		}
	}
	copy(windows[index:], windows[index+1:])
	windows = windows[:len(windows)-1]

	if e.output != nil {
		index = -1
		for i, w := range windows {
			if w == e.output {
				index = i
				break
			}
		}
		copy(windows[index:], windows[index+1:])
		windows = windows[:len(windows)-1]
	}

	//if len(windows) > 0 {
	if sess.numberOfEditors() > 0 {
		// easier to just go to first window which has to be an editor (at least right now)
		for _, w := range windows {
			if ed, ok := w.(*Editor); ok { //need the type assertion
				p = ed //p is the global current editor
				break
			}
		}

		//p = windows[0].(*Editor)
		err = v.SetCurrentBuffer(p.vbuf)
		if err != nil {
			sess.showOrgMessage("Error setting current buffer: %v", err)
		}
		sess.positionWindows()
		sess.eraseRightScreen()
		sess.drawRightScreen()

	} else { // we've quit the last remaining editor(s)
		// unless commented out earlier sess.p.quit <- causes panic
		//sess.p = nil
		sess.editorMode = false
		sess.eraseRightScreen()

		if sess.divider < 10 {
			sess.cfg.ed_pct = 80
			moveDivider(80)
		}

		org.drawPreview()
		sess.returnCursor() //because main while loop if started in editor_mode -- need this 09302020
	}

}

func (e *Editor) writeAll() {
	for _, w := range windows {
		if ed, ok := w.(*Editor); ok {
			err := v.SetCurrentBuffer(ed.vbuf)
			if err != nil {
				sess.showEdMessage("Problem setting current buffer: %d", ed.vbuf)
				return
			}
			ed.writeNote()
		}
	}
	err := v.SetCurrentBuffer(e.vbuf)
	if err != nil {
		sess.showEdMessage("Problem setting current buffer")
		return
	}
	e.command_line = ""
	e.mode = NORMAL
}

func (e *Editor) quitAll() {

	deleteBufferOpts := map[string]bool{
		"force":  true,
		"unload": false,
	}

	for _, w := range windows {
		if ed, ok := w.(*Editor); ok {
			if ed.isModified() {
				continue
			} else {
				err := v.DeleteBuffer(ed.vbuf, deleteBufferOpts)
				if err != nil {
					sess.showOrgMessage("DeleteBuffer error %v", err)
				} else {
					sess.showOrgMessage("DeleteBuffer successful")
				}
				index := -1
				for i, w := range windows {
					if w == ed {
						index = i
						break
					}
				}
				copy(windows[index:], windows[index+1:])
				windows = windows[:len(windows)-1]

				if ed.output != nil {
					index = -1
					for i, w := range windows {
						if w == ed.output {
							index = i
							break
						}
					}
					copy(windows[index:], windows[index+1:])
					windows = windows[:len(windows)-1]
				}
			}
		}
	}

	if sess.numberOfEditors() > 0 { // we could not quit some editors because they were in modified state
		for _, w := range windows {
			if ed, ok := w.(*Editor); ok { //need this type assertion to have statement below
				p = ed //p is the global representing the current editor
				break
			}
		}

		err := v.SetCurrentBuffer(p.vbuf)
		if err != nil {
			sess.showOrgMessage("Error setting current buffer: %v", err)
		}
		sess.positionWindows()
		sess.eraseRightScreen()
		sess.drawRightScreen()
		sess.showEdMessage("Some editors had no write since the last change")

	} else { // we've been able to quit all editors because none were in modified state
		sess.editorMode = false
		sess.eraseRightScreen()

		if sess.divider < 10 {
			sess.cfg.ed_pct = 80
			moveDivider(80)
		}

		org.drawPreview()
		sess.returnCursor() //because main while loop if started in editor_mode -- need this 09302020
	}
}

func (e *Editor) spell() {
	e.checkSpelling = !e.checkSpelling
	if e.checkSpelling {
		e.highlightSyntax = false // when you check spelling syntax highlighting off
		err := v.Command("set spell")
		if err != nil {
			sess.showEdMessage("Error in setting spelling %v", err)
		}
	} else {
		err := v.Command("set nospell")
		if err != nil {
			sess.showEdMessage("Error in setting no spelling %v", err)
		}
	}
	e.drawText()
	sess.showEdMessage("Spelling is %t", e.checkSpelling)
}

func (e *Editor) number() {
	e.numberLines = !e.numberLines
	if e.numberLines {
		e.left_margin_offset = LEFT_MARGIN_OFFSET
	} else {
		e.left_margin_offset = 0
	}
	e.drawText()
	sess.showEdMessage("Line numbering is %t", e.numberLines)
}

func (e *Editor) goFormat() {
	bb := [][]byte{}
	cmd := exec.Command("gofmt")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sess.showEdMessage("Problem in gofmt stdout: %v", err)
		return
	}
	buf_out := bufio.NewReader(stdout)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		sess.showEdMessage("Problem in gofmt stdin: %v", err)
		return
	}
	err = cmd.Start()
	if err != nil {
		sess.showEdMessage("Problem in cmd.Start (gofmt) stdin: %v", err)
		return
	}

	for _, row := range e.bb {
		io.WriteString(stdin, string(row)+"\n")
	}
	stdin.Close()

	for {
		bytes, err := buf_out.ReadBytes('\n')

		if err == io.EOF {
			break
		}

		/*
			if len(bytes) == 0 {
				break
			}
		*/

		bb = append(bb, bytes[:len(bytes)-1])
	}
	e.bb = bb

	err = v.SetBufferLines(e.vbuf, 0, -1, true, e.bb)
	if err != nil {
		sess.showEdMessage("Error in SetBufferLines in dbfuc: %v", err)
	}
	e.drawText()
	/*
		err = v.Command(fmt.Sprintf("w temp/buf%d", e.vbuf))
		if err != nil {
			sess.showEdMessage("Error in writing file in dbfunc: %v", err)
		}
	*/

}
