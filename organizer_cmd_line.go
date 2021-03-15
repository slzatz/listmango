package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

var cmd_lookup = map[string]func(*Organizer, int){
	"open":        (*Organizer).openContext,
	"o":           (*Organizer).openContext,
	"openfolder":  (*Organizer).openFolder,
	"of":          (*Organizer).openFolder,
	"openkeyword": (*Organizer).openKeyword,
	"ok":          (*Organizer).openKeyword,
	"quit":        (*Organizer).quitApp,
	"q":           (*Organizer).quitApp,
	"e":           (*Organizer).editNote,
	"resize":      (*Organizer).resize,
	"test":        (*Organizer).sync,
	"sync":        (*Organizer).sync,
	"new":         (*Organizer).newEntry,
	"n":           (*Organizer).newEntry,
	"refresh":     (*Organizer).refresh,
	"r":           (*Organizer).refresh,
	"find":        (*Organizer).find,
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
	"write": (*Organizer).write,
	"w":     (*Organizer).write,
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

func (o *Organizer) openContext(pos int) {
	if pos == 0 {
		sess.showOrgMessage("You did not provide a context!")
		o.mode = NORMAL
		return
	}

	cl := o.command_line
	var success bool
	success = false
	for k, _ := range o.context_map {
		if strings.HasPrefix(k, cl[pos+1:]) {
			o.context = k
			success = true
			break
		}
	}

	if !success {
		sess.showOrgMessage(fmt.Sprintf("%s is not a valid  context!", cl[:pos]))
		o.mode = NORMAL
		return
	}

	sess.showOrgMessage("'%s' will be opened", o.context)

	o.marked_entries = nil
	o.folder = ""
	o.taskview = BY_CONTEXT
	getItems(MAX)
	o.mode = NORMAL
	return
}

func (o *Organizer) openFolder(pos int) {
	if pos == 0 {
		sess.showOrgMessage("You did not provide a folder!")
		o.mode = NORMAL
		return
	}

	cl := &o.command_line
	var success bool
	success = false
	for k, _ := range o.folder_map {
		if strings.HasPrefix(k, (*cl)[pos+1:]) {
			o.folder = k
			success = true
			break
		}
	}

	if !success {
		sess.showOrgMessage("%s is not a valid  folder!", (*cl)[:pos])
		o.mode = NORMAL
		return
	}

	sess.showOrgMessage("'%s' will be opened", o.folder)

	o.marked_entries = nil
	o.context = ""
	o.taskview = BY_FOLDER
	getItems(MAX)
	o.mode = NORMAL
	return
}

func (o *Organizer) openKeyword(pos int) {
	if pos == 0 {
		sess.showOrgMessage("You did not provide a keyword!")
		o.mode = NORMAL
		return
	}
	//O.keyword = O.command_line.substr(pos+1);
	keyword := o.command_line[pos+1:]
	if keywordExists(keyword) == -1 {
		o.mode = o.last_mode
		sess.showOrgMessage("keyword '%s' does not exist!", keyword)
		return
	}

	o.keyword = keyword
	sess.showOrgMessage("'%s' will be opened", o.keyword)
	o.marked_entries = nil
	o.context = ""
	o.folder = ""
	o.taskview = BY_KEYWORD
	getItems(MAX)
	o.mode = NORMAL
	return
}

func (o *Organizer) write(pos int) {
	if o.view == TASK {
		updateRows()
	}
	o.mode = o.last_mode
	o.command_line = ""
}

func (o *Organizer) quitApp(_ int) {
	unsaved_changes := false
	for _, r := range o.rows {
		if r.dirty {
			unsaved_changes = true
			break
		}
	}
	if unsaved_changes {
		o.mode = NORMAL
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
func (o *Organizer) editNote(id int) {

	if o.view != TASK {
		o.command = ""
		o.mode = NORMAL
		sess.showOrgMessage("Only entries have notes to edit!")
		return
	}

	//pos is zero if no space and command modifier
	if id == 0 {
		id = getId()
	}
	if id == -1 {
		sess.showOrgMessage("You need to save item before you can create a note")
		o.command = ""
		o.mode = NORMAL
		return
	}

	//sess.showOrgMessage("Edit note %d", id)
	sess.editorMode = true

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

		folder_tid := getFolderTid(o.rows[o.fr].id)
		if folder_tid == 18 || folder_tid == 14 {
			sess.p.linked_editor = &Editor{}
			sess.editors = append(sess.editors, sess.p.linked_editor)
			sess.p.linked_editor.id = id
			sess.p.linked_editor.is_subeditor = true
			sess.p.linked_editor.is_below = true
			sess.p.linked_editor.linked_editor = sess.p
			sess.p.linked_editor.rows = []string{" "}
			sess.p.left_margin_offset = LEFT_MARGIN_OFFSET
		}
		readNoteIntoEditor(id)

		ok, err := v.AttachBuffer(0, false, make(map[string]interface{})) // 0 => current buffer
		if err != nil {
			log.Fatal(err)
		}
		if !ok {
			log.Fatal()
		}
	}

	sess.positionEditors()
	sess.eraseRightScreen() //erases editor area + statusbar + msg
	sess.drawEditors()
	sess.p.mode = NORMAL

	o.command = ""
	o.mode = NORMAL
}

func (o *Organizer) resize(pos int) {
	if pos == 0 {
		sess.showOrgMessage("You need to provide a number")
		return
	}
	pct, err := strconv.Atoi(o.command_line[pos+1:])
	if err != nil {
		sess.showOrgMessage("You need to provide a number 0 - 100")
		o.mode = NORMAL
		return
	}
	sess.moveDivider(pct)
	o.mode = NORMAL
}
func (o *Organizer) newEntry(unused int) {
	//org.outlineInsertRow(0, "", true, false, false, now());

	/*
		type Row struct {
			id        int
			title     string
			fts_title string
			star      bool
			deleted   bool
			completed bool
			modified  string

			// below not in db
			dirty  bool
			marked bool
		}
	*/

	row := Row{
		id:       -1,
		star:     true,
		dirty:    true,
		modified: time.Now().Format("3:04:05 pm"),
	}

	//fmt.Fprintf(&lg, "UTC time is %v\n", time.Now().UTC())

	o.rows = append(o.rows, Row{})
	copy(o.rows[1:], o.rows[0:])
	o.rows[0] = row

	o.fc, o.fr, o.rowoff = 0, 0, 0
	o.command = ""
	o.repeat = 0
	sess.showOrgMessage("\x1b[1m-- INSERT --\x1b[0m")
	sess.eraseRightScreen() //erases the note area
	o.mode = INSERT

	/*
	  int fd;
	  std::string fn = "assets/" + CURRENT_NOTE_FILE;
	  if ((fd = open(fn.c_str(), O_WRONLY|O_CREAT|O_TRUNC, 0666)) != -1) {
	    sess.lock.l_type = F_WRLCK;
	    if (fcntl(fd, F_SETLK, &sess.lock) != -1) {
	    write(fd, " ", 1);
	    sess.lock.l_type = F_UNLCK;
	    fcntl(fd, F_SETLK, &sess.lock);
	    } else sess.showOrgMessage("Couldn't lock file");
	  } else sess.showOrgMessage("Couldn't open file");
	*/
}

func (o *Organizer) refresh(unused int) {
	if o.view == TASK {
		getItems(MAX)
		sess.showOrgMessage("Entries will be refreshed")
	}

	if o.taskview == BY_FIND {
		o.searchDB(sess.fts_search_terms, false)
	} else {
		getItems(MAX)
	}
	/*
	 } else {
	   sess.showOrgMessage("contexts/folders will be refreshed");
	   getContainers();
	    if (org.mode != NO_ROWS) {
	      Container c = getContainerInfo(org.rows.at(org.fr).id);
	      sess.displayContainerInfo(c);
	    }
	  }
	*/
	o.mode = o.last_mode
}

func (o *Organizer) find(pos int) {

	if pos == 0 {
		sess.showOrgMessage("You did not something to find!")
		o.mode = NORMAL
		return
	}

	searchTerms := strings.ToLower(o.command_line[pos+1:])
	sess.fts_search_terms = searchTerms
	if len(searchTerms) < 3 {
		sess.showOrgMessage("You  need to provide at least 3 characters to search on")
		return
	}

	o.context = ""
	o.folder = ""
	o.taskview = BY_FIND

	sess.showOrgMessage("Searching for '%s'", searchTerms)
	//sess.fts_search_terms = searchTerms
	o.searchDB(searchTerms, false)
}

func (o *Organizer) sync(unused int) {
	var reportOnly bool
	if o.command_line == "test" {
		reportOnly = true
	}
	synchronize(reportOnly)
	o.mode = NORMAL
}
