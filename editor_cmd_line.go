package main

import (
	"bufio"
	"io"
	"log"
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
}

/* EDITOR cpp COMMAND_LINE mode lookup
const std::unordered_map<std::string, efunc> E_lookup_C {
  {"write", &Editor::E_write_C},
  {"w", &Editor::E_write_C},
 // all below handled (right now) in editor command line switch statement
 // {"x", &Editor::E_write_close_C},
 // {"quit", &Editor::E_quit_C},
 // {"q",&Editor:: E_quit_C},
 // {"quit!", &Editor::E_quit0_C},
 // {"q!", &Editor::E_quit0_C},
  {"vim", &Editor::E_open_in_vim_C},
  {"spell",&Editor:: E_spellcheck_C},
  {"spellcheck", &Editor::E_spellcheck_C},
  {"read", &Editor::E_readfile_C},
  {"readfile", &Editor::E_readfile_C},

  {"compile", &Editor::E_compile_C},
  {"c", &Editor::E_compile_C},
  {"make", &Editor::E_compile_C},
  {"r", &Editor::E_runlocal_C}, // this does change the text/usually COMMAND_LINE doesn't
  {"runl", &Editor::E_runlocal_C}, // this does change the text/usually COMMAND_LINE doesn't
  {"runlocal", &Editor::E_runlocal_C}, // this does change the text/usually COMMAND_LINE doesn't
  {"run", &Editor::E_runlocal_C}, //compile and run on Compiler Explorer
  {"rr", &Editor::E_run_code_C}, //compile and run on Compiler Explorer
  {"runremote", &Editor::E_run_code_C}, //compile and run on Compiler Explorer
  {"save", &Editor::E_save_note},
  {"savefile", &Editor::E_save_note},
  {"createlink", &Editor::createLink},
  //{"cl", &Editor::createLink},
  {"getlinked", &Editor::getLinked},
  {"gl", &Editor::getLinked},
  {"resize", &Editor::moveDivider},
  {"hide", &Editor::hide},
};
*/

func (e *Editor) writeNote() {
	if e.is_subeditor {
		e.showMessage("You can't save the contents of the Output Window")
		return
	}

	//update_note(false);
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
		e.showMessage("%w", err)
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

	//var str string
	var dir string
	var cmd *exec.Cmd
	if getFolderTid(e.id) == 18 {
		dir = "/home/slzatz/clangd_examples/"
		//str = "make"
		cmd = exec.Command("make")
	} else {
		dir = "/home/slzatz/go_fragments/"
		//str = "go build main.go"
		cmd = exec.Command("go", "build", "main.go")
	}
	//cmd := exec.Command(str)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
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

	//if (text.str().empty())   text << "Go build successful";
	//std::vector<std::string> zz = str2vecWW(text.str(), false); //ascii_only = false

	//fr = fc = cy = cx = line_offset = prev_line_offset = first_visible_row = last_visible_row = 0;

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
	//chdir("/home/slzatz/listmango/");
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
		//  cmd = "/home/slzatz/clangd_examples/test_cpp";
		obj = "./test_cpp"
		dir = "/home/slzatz/clangd_examples/"
	} else {
		//  cmd = "/home/slzatz/go/src/example/main";
		obj = "./main"
		dir = "/home/slzatz/go_fragments/"
	}
	cmd = exec.Command(obj, args)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
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
}

func (e *Editor) sync() {
	var reportOnly bool
	if e.command_line == "test" {
		reportOnly = true
	}
	synchronize(reportOnly)
}
