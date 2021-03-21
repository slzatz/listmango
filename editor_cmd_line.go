package main

import (
	"bufio"
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
	"read":     (*Editor).readFile,
	"readfile": (*Editor).readFile,
	"resize":   (*Editor).resize,
	"c":        (*Editor).compile,
	"r":        (*Editor).runLocal,
	"test":     (*Editor).sync,
	"sync":     (*Editor).sync,
	"save":     (*Editor).saveNoteToFile,
	"savefile": (*Editor).saveNoteToFile,
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
		e.showMessage("You need to provide a filename")
		return
	}
	filename := e.command_line[pos+1:]
	f, err := os.Create(filename)
	if err != nil {
		sess.showEdMessage("Error creating file %s: %v", filename, err)
		return
	}
	defer f.Close()

	_, err = f.WriteString(e.generateWWStringFromBuffer())
	if err != nil {
		sess.showEdMessage("Error writing file %s: %v", filename, err)
		return
	}
	sess.showEdMessage("Note written to file %s", filename)
}

func (e *Editor) writeNote() {
	if e.is_subeditor {
		e.showMessage("You can't save the contents of the Output Window")
		return
	}

	updateNote()

	folder_tid := getFolderTid(e.id)
	if folder_tid == 18 || folder_tid == 14 {
		e.code = e.rowsToString()
		updateCodeFile()
	}
	/*
		} else if sess.lm_browser {
			sess.updateHTMLFile("assets/" + CURRENT_NOTE_FILE)
		}
	*/
	e.dirty = 0
	e.drawStatusBar() //need this since now refresh won't do it unless redraw =true
	e.showMessage("")
}

func (e *Editor) readFile() {
	pos := strings.Index(e.command_line, " ")
	if pos == -1 {
		e.showMessage("You need to provide a filename")
		return
	}

	filename := e.command_line[pos+1:]
	err := e.readFileIntoNote(filename)
	if err != nil {
		e.showMessage("%v", err)
		return
	}
	e.showMessage("Note generated from file: %s", filename)
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
	sess.moveDivider(pct)
}

func (e *Editor) compile() {

	var dir string
	var cmd *exec.Cmd
	if getFolderTid(e.id) == 18 {
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

	rows := &e.linked_editor.rows
	*rows = nil
	*rows = append(*rows, "------------------------")

	for {
		bytes, _, err := buffer_out.ReadLine()
		if err == io.EOF {
			break
		}
		*rows = append(*rows, string(bytes))
	}

	for {
		bytes, _, err := buffer_err.ReadLine()
		if err == io.EOF {
			break
		}
		*rows = append(*rows, string(bytes))
	}
	if len(*rows) == 1 {
		*rows = append(*rows, "The code compiled successfully")
	}

	*rows = append(*rows, "------------------------")

	e.linked_editor.fr = 0
	e.linked_editor.fc = 0

	// added 02092021
	e.linked_editor.cy = 0
	e.linked_editor.cx = 0
	e.linked_editor.line_offset = 0
	e.linked_editor.prev_line_offset = 0
	e.linked_editor.first_visible_row = 0
	e.linked_editor.last_visible_row = 0
	// added 02092021

	e.linked_editor.refreshScreen(true)
}

func (e *Editor) runLocal() {

	var args string
	pos := strings.Index(e.command_line, " ")
	if pos != -1 {
		args = e.command_line[pos+1:]
	}

	var dir string
	var obj string
	var cmd *exec.Cmd
	if getFolderTid(e.id) == 18 {
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
		sess.showEdMessage("Error in runLocal creating stdout pipe: %v", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		sess.showEdMessage("Error in runLocal creating stderr pipe: %v", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		sess.showEdMessage("Error in runLocal starting command: %v", err)
		return
	}

	buffer_out := bufio.NewReader(stdout)
	buffer_err := bufio.NewReader(stderr)

	rows := &e.linked_editor.rows
	*rows = nil
	*rows = append(*rows, "------------------------")

	for {
		bytes, _, err := buffer_out.ReadLine()
		if err == io.EOF {
			break
		}
		*rows = append(*rows, string(bytes))
	}

	for {
		bytes, _, err := buffer_err.ReadLine()
		if err == io.EOF {
			break
		}
		*rows = append(*rows, string(bytes))
	}

	*rows = append(*rows, "------------------------")

	le := e.linked_editor
	le.fr = 0
	le.fc = 0

	// added 02092021
	le.cy = 0
	le.cx = 0
	le.line_offset = 0
	le.prev_line_offset = 0
	le.first_visible_row = 0
	le.last_visible_row = 0

	le.refreshScreen(true)
	/*
		e.linked_editor.fr = 0
		e.linked_editor.fc = 0

		// added 02092021
		e.linked_editor.cy = 0
		e.linked_editor.cx = 0
		e.linked_editor.line_offset = 0
		e.linked_editor.prev_line_offset = 0
		e.linked_editor.first_visible_row = 0
		e.linked_editor.last_visible_row = 0

		e.linked_editor.refreshScreen(true)
	*/
}

func (e *Editor) sync() {
	var reportOnly bool
	if e.command_line == "test" {
		reportOnly = true
	}
	synchronize(reportOnly)
}
