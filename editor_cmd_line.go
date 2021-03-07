package main

import (
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
	/*
	  folder_tid := getFolderTid(id);
	  if (folder_tid == 18 || folder_tid == 14) {
	    code = editorRowsToString();
	    updateCodeFile();
	  } else if (sess.lm_browser) {
	    sess.updateHTMLFile("assets/" + CURRENT_NOTE_FILE);
	  }
	*/

	e.dirty = 1
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
