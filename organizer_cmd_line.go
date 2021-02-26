package main

import (
	"fmt"
	"strings"
)

var cmd_lookup = map[string]func(int){
	"open":        F_open, //open_O
	"o":           F_open,
	"openfolder":  F_openfolder,
	"of":          F_openfolder,
	"openkeyword": F_openkeyword,
	"ok":          F_openkeyword,
	"quit":        F_quit_app,
	"q":           F_quit_app,
	"e":           F_edit,
	/*
	  "deletekeywords": F_deletekeywords,
	  "delkw": F_deletekeywords,
	  "delk": F_deletekeywords,
	  "addkeywords": F_addkeyword,
	  "addkw": F_addkeyword,
	  "addk": F_addkeyword,
	  "k": F_keywords,
	  "keyword": F_keywords,
	  "keywords": F_keywords,
	*/
	"write": F_write,
	"w":     F_write,
	/*
	  "x": F_x,
	  "refresh": F_refresh,
	  "r": F_refresh,
	  "n": F_new,
	  "new": F_new,
	  "e": F_edit,
	  "edit": F_edit,
	  "contexts": F_contexts,
	  "context": F_contexts,
	  "c": F_contexts,
	  "folders": F_folders,
	  "folder": F_folders,
	  "f": F_folders,
	  "recent": F_recent,
	 // {"linked", F_linked,
	 // {"l", F_linked,
	 // {"related", F_linked,
	  "find": F_find,
	  "fin": F_find,
	  "search": F_find,
	  "sync": F_sync,
	  "test": F_sync_test,
	  "updatefolder": F_updatefolder,
	  "uf": F_updatefolder,
	  "updatecontext": F_updatecontext,
	  "uc": F_updatecontext,
	  "delmarks": F_delmarks,
	  "delm": F_delmarks,
	  "save": F_savefile,
	  "sort": F_sort,
	  "show": F_showall,
	  "showall": F_showall,
	  "set": F_set,
	  "syntax": F_syntax,
	  "vim": F_open_in_vim,
	  "join": F_join,
	  "saveoutline": F_saveoutline,
	  //{"readfile": F_readfile,
	  //{"read": F_readfile,
	  "valgrind": F_valgrind,
	  "quit": F_quit_app,
	  "q": F_quit_app,
	  "quit!": F_quit_app_ex,
	  "q!": F_quit_app_ex,
	  //{"merge", F_merge,
	  "help": F_help,
	  "h": F_help,
	  "copy": F_copy_entry,
	 //{"restart_lsp", F_restart_lsp,
	 //{"shutdown_lsp", F_shutdown_lsp,
	  "lsp": F_lsp_start,
	  "browser": F_launch_lm_browser,
	  "launch": F_launch_lm_browser,
	  "quitb": F_quit_lm_browser,
	  "quitbrowser": F_quit_lm_browser,
	  "createlink": F_createLink,
	  "getlinked": F_getLinked,
	  "resize": F_resize
	*/
}

func F_open(pos int) { //C_open - by context
	if pos == 0 {
		sess.showOrgMessage("You did not provide a context!")
		org.mode = NORMAL
		return
	}

	cl := org.command_line
	var success bool
	success = false
	for k, _ := range org.context_map {
		if strings.HasPrefix(k, cl[pos+1:]) {
			org.context = k
			success = true
			break
		}
	}

	if !success {
		sess.showOrgMessage(fmt.Sprintf("%s is not a valid  context!", cl[:pos]))
		org.mode = NORMAL
		return
	}

	sess.showOrgMessage("'%s' will be opened", org.context)

	org.marked_entries = nil
	org.folder = ""
	org.taskview = BY_CONTEXT
	getItems(MAX)
	org.mode = NORMAL
	return
}

func F_openfolder(pos int) { //C_open - by context
	if pos == 0 {
		sess.showOrgMessage("You did not provide a folder!")
		org.mode = NORMAL
		return
	}

	cl := &org.command_line
	var success bool
	success = false
	for k, _ := range org.folder_map {
		if strings.HasPrefix(k, (*cl)[pos+1:]) {
			org.folder = k
			success = true
			break
		}
	}

	if !success {
		sess.showOrgMessage("%s is not a valid  folder!", (*cl)[:pos])
		org.mode = NORMAL
		return
	}

	sess.showOrgMessage("'%s' will be opened", org.folder)

	org.marked_entries = nil
	org.context = ""
	org.taskview = BY_FOLDER
	getItems(MAX)
	org.mode = NORMAL
	return
}

func F_openkeyword(pos int) {
	if pos == 0 {
		sess.showOrgMessage("You did not provide a keyword!")
		org.mode = NORMAL
		return
	}
	//O.keyword = O.command_line.substr(pos+1);
	keyword := org.command_line[pos+1:]
	if keywordExists(keyword) == -1 {
		org.mode = org.last_mode
		sess.showOrgMessage("keyword '%s' does not exist!", keyword)
		return
	}

	org.keyword = keyword
	sess.showOrgMessage("'%s' will be opened", org.keyword)
	org.marked_entries = nil
	org.context = ""
	org.folder = ""
	org.taskview = BY_KEYWORD
	getItems(MAX)
	org.mode = NORMAL
	return
}

func F_write(pos int) {
	if org.view == TASK {
		updateRows()
	}
	org.mode = org.last_mode
	org.command_line = ""
}

func F_quit_app(pos int) {
	unsaved_changes := false
	for _, r := range org.rows {
		if r.dirty {
			unsaved_changes = true
			break
		}
	}
	if unsaved_changes {
		org.mode = NORMAL
		sess.showOrgMessage("No db write since last change")
	} else {
		sess.run = false

		/* need to figure out if need any of the below
		   context.close();
		   subscriber.close();
		   publisher.close();
		   subs_thread.join();
		   exit(0);
		*/
	}
}
func F_edit(id int) {

	if org.view != TASK {
		org.command = ""
		org.mode = NORMAL
		sess.showOrgMessage("Only tasks have notes to edit!")
		return
	}

	//pos is zero if no space and command modifier
	if id == 0 {
		id = getId()
	}
	if id == -1 {
		sess.showOrgMessage("You need to save item before you can create a note")
		org.command = ""
		org.mode = NORMAL
		return
	}

	sess.showOrgMessage("Edit note %d", id)
	sess.editorMode = true

	if len(sess.editors) > 0 {
		active := false
		for _, e := range sess.editors {
			if e.id == id {
				active = true
				sess.p = e
				break
			}
		}

		if !active {
			sess.p = &Editor{}
			sess.editors = append(sess.editors, sess.p)
			sess.p.id = id
			sess.p.top_margin = TOP_MARGIN + 1

			folder_tid := getFolderTid(org.rows[org.fr].id)
			if folder_tid == 18 || folder_tid == 14 {
				sess.p.linked_editor = &Editor{}
				sess.editors = append(sess.editors, sess.p.linked_editor)
				sess.p.linked_editor.id = id
				sess.p.linked_editor.is_subeditor = true
				sess.p.linked_editor.is_below = true
				sess.p.linked_editor.linked_editor = sess.p
				sess.p.left_margin_offset = LEFT_MARGIN_OFFSET
			}
			readNoteIntoEditor(id)
		}

	} else {
		sess.p = &Editor{}
		sess.editors = append(sess.editors, sess.p)
		sess.p.id = id
		sess.p.top_margin = TOP_MARGIN + 1

		folder_tid := getFolderTid(org.rows[org.fr].id)
		if folder_tid == 18 || folder_tid == 14 {
			sess.p.linked_editor = &Editor{}
			sess.editors = append(sess.editors, sess.p.linked_editor)
			sess.p.linked_editor.id = id
			sess.p.linked_editor.is_subeditor = true
			sess.p.linked_editor.is_below = true
			sess.p.linked_editor.linked_editor = sess.p
			sess.p.left_margin_offset = LEFT_MARGIN_OFFSET
		}
		readNoteIntoEditor(id) //if id == -1 does not try to retrieve note
	}
	sess.positionEditors()
	sess.eraseRightScreen() //erases editor area + statusbar + msg
	sess.drawEditors()

	if len(sess.p.rows) == 0 {
		//sess.p.mode = INSERT
		sess.p.mode = NORMAL
		// below all for undo
		sess.p.last_command = "i"
		sess.p.prev_fr = 0
		sess.p.prev_fc = 0
		sess.p.last_repeat = 1
		//sess.p.snapshot.push_back("");
		//sess.p.showMessage("\x1b[1m-- INSERT --\x1b[0m")
	} else {
		sess.p.mode = NORMAL
	}

	org.command = ""
	org.mode = NORMAL
}
