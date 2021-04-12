package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var cmd_lookup = map[string]func(*Organizer, int){
	"open":           (*Organizer).openContext,
	"o":              (*Organizer).openContext,
	"openfolder":     (*Organizer).openFolder,
	"of":             (*Organizer).openFolder,
	"openkeyword":    (*Organizer).openKeyword,
	"ok":             (*Organizer).openKeyword,
	"quit":           (*Organizer).quitApp,
	"q":              (*Organizer).quitApp,
	"e":              (*Organizer).editNote,
	"resize":         (*Organizer).resize,
	"test":           (*Organizer).sync,
	"sync":           (*Organizer).sync,
	"new":            (*Organizer).newEntry,
	"n":              (*Organizer).newEntry,
	"refresh":        (*Organizer).refresh,
	"r":              (*Organizer).refresh,
	"find":           (*Organizer).find,
	"contexts":       (*Organizer).contexts,
	"context":        (*Organizer).contexts,
	"c":              (*Organizer).contexts,
	"folders":        (*Organizer).folders,
	"folder":         (*Organizer).folders,
	"f":              (*Organizer).folders,
	"keywords":       (*Organizer).keywords,
	"keyword":        (*Organizer).keywords,
	"k":              (*Organizer).keywords,
	"recent":         (*Organizer).recent,
	"log":            (*Organizer).log,
	"deletekeywords": (*Organizer).deleteKeywords,
	"delkw":          (*Organizer).deleteKeywords,
	"delk":           (*Organizer).deleteKeywords,
	"showall":        (*Organizer).showAll,
	"show":           (*Organizer).showAll,
	"cc":             (*Organizer).updateContainer,
	"ff":             (*Organizer).updateContainer,
	"kk":             (*Organizer).updateContainer,
	"write":          (*Organizer).write,
	"w":              (*Organizer).write,
	"deletemarks":    (*Organizer).deleteMarks,
	"delmarks":       (*Organizer).deleteMarks,
	"delm":           (*Organizer).deleteMarks,
	"copy":           (*Organizer).copyEntry,
	"save":           (*Organizer).save,
	//"showmarkdown":   (*Organizer).showMarkdown,
	//"showm":          (*Organizer).showMarkdown,
	/*
	  "x": F_x,
	 // {"linked", F_linked,
	 // {"l", F_linked,
	 // {"related", F_linked,
	  "save": F_savefile,
	  "sort": F_sort,
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
	 //{"restart_lsp", F_restart_lsp,
	 //{"shutdown_lsp", F_shutdown_lsp,
	  "lsp": F_lsp_start,
	  "browser": F_launch_lm_browser,
	  "launch": F_launch_lm_browser,
	  "quitb": F_quit_lm_browser,
	  "quitbrowser": F_quit_lm_browser,
	  "createlink": F_createLink,
	  "getlinked": F_getLinked,
	*/
}

func (o *Organizer) log(unused int) {
	getSyncItems(MAX)
	o.fc, o.fr, o.rowoff = 0, 0, 0
	o.altRowoff = 0
	o.mode = SYNC_LOG      //kluge INSERT, NORMAL, ...
	o.view = SYNC_LOG_VIEW //TASK, FOLDER, KEYWORD ...

	// show first row's note
	o.eraseRightScreen()
	o.note = readSyncLog(o.rows[o.fr].id)
	o.drawNoteReadOnly()
	o.clearMarkedEntries()
	o.showOrgMessage("")
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
			//o.context = k
			o.filter = k
			success = true
			break
		}
	}

	if !success {
		sess.showOrgMessage("%s is not a valid  context!", cl[:pos])
		o.mode = NORMAL
		return
	}

	sess.showOrgMessage("'%s' will be opened", o.filter)

	o.clearMarkedEntries()
	//o.folder = ""
	//o.keyword = ""
	o.taskview = BY_CONTEXT
	org.view = TASK
	o.mode = NORMAL // can be changed to NO_ROWS below
	o.fc, o.fr, o.rowoff = 0, 0, 0
	o.rows = filterEntries(o.taskview, o.filter, o.show_deleted, o.sort, MAX)
	if len(o.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		o.mode = NO_ROWS
	}
	//o.drawPreviewWindow()
	o.drawPreview()
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
			//o.folder = k
			o.filter = k
			success = true
			break
		}
	}

	if !success {
		sess.showOrgMessage("%s is not a valid  folder!", (*cl)[:pos])
		o.mode = NORMAL
		return
	}

	sess.showOrgMessage("'%s' will be opened", o.filter)

	o.clearMarkedEntries()
	o.taskview = BY_FOLDER
	org.view = TASK
	o.mode = NORMAL // can be changed to NO_ROWS below
	o.fc, o.fr, o.rowoff = 0, 0, 0
	o.rows = filterEntries(o.taskview, o.filter, o.show_deleted, o.sort, MAX)
	if len(o.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		o.mode = NO_ROWS
	}
	//o.drawPreviewWindow()
	o.drawPreview()
	return
}

func (o *Organizer) openKeyword(pos int) {
	if pos == 0 {
		sess.showOrgMessage("You did not provide a keyword!")
		o.mode = NORMAL
		return
	}
	keyword := o.command_line[pos+1:]
	if keywordExists(keyword) == -1 {
		o.mode = o.last_mode
		sess.showOrgMessage("keyword '%s' does not exist!", keyword)
		return
	}

	o.filter = keyword

	sess.showOrgMessage("'%s' will be opened", o.filter)

	o.clearMarkedEntries()
	o.taskview = BY_KEYWORD
	org.view = TASK
	o.mode = NORMAL // can be changed to NO_ROWS below
	o.fc, o.fr, o.rowoff = 0, 0, 0
	o.rows = filterEntries(o.taskview, o.filter, o.show_deleted, o.sort, MAX)
	if len(o.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		o.mode = NO_ROWS
	}
	//o.drawPreviewWindow()
	o.drawPreview()
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
	for _, e := range editors {
		if e.id == id {
			active = true
			p = e
			break
		}
	}

	if !active {
		p = &Editor{}
		editors = append(editors, p)
		p.id = id
		p.top_margin = TOP_MARGIN + 1

		folder_tid := getFolderTid(o.rows[o.fr].id)
		if folder_tid == 18 || folder_tid == 14 {
			p.linked_editor = &Editor{}
			editors = append(editors, p.linked_editor)
			p.linked_editor.id = id
			p.linked_editor.is_subeditor = true
			p.linked_editor.is_below = true
			p.linked_editor.linked_editor = p
			//p.linked_editor.rows = []string{" "}
			p.left_margin_offset = LEFT_MARGIN_OFFSET
		} else if folder_tid == 21 {
			p.left_margin_offset = LEFT_MARGIN_OFFSET
		}
		readNoteIntoEditor(id)

		ok, err := v.AttachBuffer(0, false, make(map[string]interface{})) // 0 => current buffer
		if err != nil {
			sess.showOrgMessage("Error when attaching buffer: %v", err)
		}
		if !ok {
			sess.showOrgMessage("Problem when attaching buffer")
		}
	}

	sess.positionEditors()
	sess.eraseRightScreen() //erases editor area + statusbar + msg
	sess.drawEditors()
	p.mode = NORMAL

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
	moveDivider(pct)
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
		if o.taskview == BY_FIND {
			//o.rows = nil
			o.fc, o.fr, o.rowoff = 0, 0, 0
			o.rows = searchEntries(sess.fts_search_terms, o.show_deleted, false)
			if len(o.rows) == 0 {
				sess.showOrgMessage("No results were returned")
				o.mode = NO_ROWS
			}
			if unused != -1 { //complete kluge has to do with refreshing when syncing
				//o.drawPreviewWindow()
				o.drawPreview()
			}
		} else {
			o.fc, o.fr, o.rowoff = 0, 0, 0
			o.rows = filterEntries(o.taskview, o.filter, o.show_deleted, o.sort, MAX)
			if len(o.rows) == 0 {
				sess.showOrgMessage("No results were returned")
				o.mode = NO_ROWS
			}
			if unused != -1 { //complete kluge has to do with refreshing when syncing
				//o.drawPreviewWindow()
				o.drawPreview()
			}
		}
		sess.showOrgMessage("Entries will be refreshed")
	} else {
		getContainers()
		if org.mode != NO_ROWS && unused != -1 {
			c := getContainerInfo(o.rows[o.fr].id)
			sess.displayContainerInfo(&c)
			sess.drawPreviewBox()
		}
		sess.showOrgMessage("view refreshed")
	}
	o.mode = o.last_mode
	o.clearMarkedEntries()
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

	//o.context = ""
	//o.folder = ""
	o.filter = ""
	o.taskview = BY_FIND
	//o.rows = nil
	o.fc, o.fr, o.rowoff = 0, 0, 0

	sess.showOrgMessage("Searching for '%s'", searchTerms)
	org.mode = FIND
	o.rows = searchEntries(searchTerms, o.show_deleted, false)
	if len(o.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		o.mode = NO_ROWS
	}
	//o.drawPreviewWindow()
	o.drawPreview()
}

func (o *Organizer) sync(unused int) {
	var log string
	if o.command_line == "test" {
		log = synchronize(true)
	} else {
		//o.refresh(0) //param is unused
		log = synchronize(false)
	}
	o.command_line = ""
	o.eraseRightScreen()
	o.note = generateWWString(log, org.totaleditorcols, 500, "\n")
	o.drawNoteReadOnly()
	o.mode = PREVIEW_SYNC_LOG
}

func (o *Organizer) contexts(pos int) {
	o.mode = NORMAL

	if pos == 0 {
		sess.eraseRightScreen()
		o.view = CONTEXT
		getContainers()
		if o.mode != NO_ROWS {
			// two lines below show first context's info
			//c := getContainerInfo(o.rows[o.fr].id)
			c := getContainerInfo(o.rows[0].id)
			sess.displayContainerInfo(&c)
			sess.drawPreviewBox()
			sess.showOrgMessage("Retrieved contexts")
		}
		return
	}

	var context string //new context for the entry
	success := false

	input := o.command_line[pos+1:]
	if len(input) < 3 {
		sess.showOrgMessage("You need to provide at least 3 characters to match existing context")
		return
	}

	for k, _ := range o.context_map {
		if strings.HasPrefix(k, input) {
			context = k
			success = true
			break
		}
	}

	if !success {
		sess.showOrgMessage("What you typed did not match any context")
		return
	}

	if len(o.marked_entries) > 0 {
		for entry_id := range o.marked_entries {
			updateTaskContext(context, entry_id) //true = update fts_dn
		}
		sess.showOrgMessage("Marked entries moved into context %s", context)
		return
	}
	updateTaskContext(context, o.rows[o.fr].id)
	sess.showOrgMessage("Moved current entry (since none were marked) into context %s", context)
}

func (o *Organizer) folders(pos int) {
	o.mode = NORMAL

	if pos == 0 {
		sess.eraseRightScreen()
		o.view = FOLDER
		getContainers()
		if o.mode != NO_ROWS {
			// two lines below show first folder's info
			c := getContainerInfo(o.rows[0].id)
			sess.displayContainerInfo(&c)
			sess.drawPreviewBox()
			sess.showOrgMessage("Retrieved folders")
		}
		return
	}

	var folder string //new folder for the entry
	success := false

	input := o.command_line[pos+1:]
	if len(input) < 3 {
		sess.showOrgMessage("You need to provide at least 3 characters to match existing folder")
		return
	}

	for k, _ := range o.folder_map {
		if strings.HasPrefix(k, input) {
			folder = k
			success = true
			break
		}
	}

	if !success {
		sess.showOrgMessage("What you typed did not match any folder")
		return
	}

	if len(o.marked_entries) > 0 {
		for entry_id, _ := range o.marked_entries {
			updateTaskFolder(folder, entry_id)
		}
		sess.showOrgMessage("Marked entries moved into folder %s", folder)
		return
	}
	updateTaskFolder(folder, o.rows[o.fr].id)
	sess.showOrgMessage("Moved current entry (since none were marked) into folder %s", folder)
}

func (o *Organizer) keywords(pos int) {

	o.mode = NORMAL

	if pos == 0 {
		sess.eraseRightScreen()
		o.view = KEYWORD
		getContainers()
		if o.mode != NO_ROWS {
			// two lines below show first keyword's info
			c := getContainerInfo(o.rows[0].id)
			sess.displayContainerInfo(&c)
			sess.drawPreviewBox()
			sess.showOrgMessage("Retrieved keywords")
		}
		return
	}

	keyword := o.command_line[pos+1:]
	keyword_id := keywordExists(keyword)
	if keyword_id == -1 {
		o.mode = o.last_mode
		sess.showOrgMessage("keyword '%s' does not exist!", keyword)
		return
	}

	if len(o.marked_entries) > 0 {
		for entry_id, _ := range o.marked_entries {
			addTaskKeyword(keyword_id, entry_id, true) //true = update fts_dn
		}
		sess.showOrgMessage("Added keyword %s to marked entries", keyword)
		return
	}

	addTaskKeyword(keyword_id, o.rows[o.fr].id, true)
	sess.showOrgMessage("Added keyword %s to current entry (since none were marked)", keyword)
}

func (o *Organizer) recent(unused int) {
	sess.showOrgMessage("Will retrieve recent items")
	o.clearMarkedEntries()
	// should just be o.filter
	//o.context = ""
	//o.folder = ""
	//o.keyword = ""
	o.filter = ""
	o.taskview = BY_RECENT
	org.view = TASK
	o.mode = NORMAL // can be changed to NO_ROWS below
	o.fc, o.fr, o.rowoff = 0, 0, 0
	o.rows = filterEntries(o.taskview, o.filter, o.show_deleted, o.sort, MAX)
	if len(o.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		o.mode = NO_ROWS
	}
	//o.drawPreviewWindow()
	o.drawPreview()
}

func (o *Organizer) deleteKeywords(unused int) {
	id := getId()
	res := deleteKeywords(id)
	o.mode = o.last_mode
	if res != -1 {
		sess.showOrgMessage("%d keyword(s) deleted from entry %d", res, id)
	}
}

func (o *Organizer) showAll(unused int) {

	if o.view != TASK {
		return
	}
	o.show_deleted = !o.show_deleted
	o.show_completed = !o.show_completed
	o.refresh(0)
	if o.show_deleted {
		sess.showOrgMessage("Showing completed/deleted")
	} else {
		sess.showOrgMessage("Hiding completed/deleted")
	}
}

func (o *Organizer) updateContainer(unused int) {
	//o.current_task_id = o.rows[o.fr].id
	sess.eraseRightScreen()
	switch o.command_line {
	case "cc":
		o.altView = CONTEXT
	case "ff":
		o.altView = FOLDER
	case "kk":
		o.altView = KEYWORD
	}
	getAltContainers() //O.mode = NORMAL is in get_containers
	if len(org.altRows) != 0 {
		o.mode = ADD_CHANGE_FILTER //this needs to change to somthing like UPDATE_TASK_MODIFIERS
		sess.showOrgMessage("Select context to add to marked or current entry")
	}
}

func (o *Organizer) deleteMarks(unused int) {
	o.clearMarkedEntries()
	o.mode = NORMAL
	o.command_line = ""
	sess.showOrgMessage("Marks cleared")
}

func (o *Organizer) copyEntry(unused int) {
	copyEntry()
	o.mode = NORMAL
	o.command_line = ""
	o.refresh(0)
	sess.showOrgMessage("Entry copied")
}

func (o *Organizer) save(unused int) {
	if o.last_mode == PREVIEW_SYNC_LOG {
		title := fmt.Sprintf("%v", time.Now().Format("Mon Jan 2 15:04:05"))
		insertSyncEntry(title, o.note)
		sess.showOrgMessage("Sync log save to database")
		o.command_line = ""
		o.mode = o.last_mode
	} // otherwise should save to file
}
