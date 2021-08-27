package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
)

var cmd_lookup = map[string]func(*Organizer, int){
	"open":            (*Organizer).open,
	"o":               (*Organizer).open,
	"opencontext":     (*Organizer).openContext,
	"oc":              (*Organizer).openContext,
	"openfolder":      (*Organizer).openFolder,
	"of":              (*Organizer).openFolder,
	"openkeyword":     (*Organizer).openKeyword,
	"ok":              (*Organizer).openKeyword,
	"quit":            (*Organizer).quitApp,
	"q":               (*Organizer).quitApp,
	"e":               (*Organizer).editNote,
	"vertical resize": (*Organizer).verticalResize,
	"vert res":        (*Organizer).verticalResize,
	"test":            (*Organizer).sync,
	"sync":            (*Organizer).sync,
	"new":             (*Organizer).newEntry,
	"n":               (*Organizer).newEntry,
	"refresh":         (*Organizer).refresh,
	"r":               (*Organizer).refresh,
	"find":            (*Organizer).find,
	"contexts":        (*Organizer).contexts,
	"context":         (*Organizer).contexts,
	"c":               (*Organizer).contexts,
	"folders":         (*Organizer).folders,
	"folder":          (*Organizer).folders,
	"f":               (*Organizer).folders,
	"keywords":        (*Organizer).keywords,
	"keyword":         (*Organizer).keywords,
	"k":               (*Organizer).keywords,
	"recent":          (*Organizer).recent,
	"log":             (*Organizer).log,
	"deletekeywords":  (*Organizer).deleteKeywords,
	"delkw":           (*Organizer).deleteKeywords,
	"delk":            (*Organizer).deleteKeywords,
	"showall":         (*Organizer).showAll,
	"show":            (*Organizer).showAll,
	"cc":              (*Organizer).updateContainer,
	"ff":              (*Organizer).updateContainer,
	"kk":              (*Organizer).updateContainer,
	"write":           (*Organizer).write,
	"w":               (*Organizer).write,
	"deletemarks":     (*Organizer).deleteMarks,
	"delmarks":        (*Organizer).deleteMarks,
	"delm":            (*Organizer).deleteMarks,
	"copy":            (*Organizer).copyEntry,
	"savelog":         (*Organizer).savelog,
	"save":            (*Organizer).save,
	"image":           (*Organizer).setImage,
	"images":          (*Organizer).setImage,
	"lsp":             (*Organizer).launchLsp,
	"shutdown":        (*Organizer).shutdownLsp,
}

func (o *Organizer) log(unused int) {
	getSyncItems(MAX)
	o.fc, o.fr, o.rowoff = 0, 0, 0
	o.altRowoff = 0
	o.mode = SYNC_LOG //kluge INSERT, NORMAL, ...
	//o.view = SYNC_LOG_VIEW //TASK, FOLDER, KEYWORD ...
	o.view = -1 //TASK, FOLDER, KEYWORD ...

	// show first row's note
	o.eraseRightScreen()
	note := readSyncLog(o.rows[o.fr].id)
	o.note = strings.Split(note, "\n")
	o.drawPreviewWithoutImages()
	o.clearMarkedEntries()
	o.showOrgMessage("")
}

func (o *Organizer) open(pos int) {
	if pos == -1 {
		sess.showOrgMessage("You did not provide a context or folder!")
		o.mode = NORMAL
		return
	}

	cl := &o.command_line
	var success bool
	for k, _ := range o.context_map {
		if strings.HasPrefix(k, (*cl)[pos+1:]) {
			o.filter = k
			success = true
			o.taskview = BY_CONTEXT
			break
		}
	}

	if !success {
		for k, _ := range o.folder_map {
			if strings.HasPrefix(k, (*cl)[pos+1:]) {
				o.filter = k
				success = true
				o.taskview = BY_FOLDER
				break
			}
		}
	}

	if !success {
		sess.showOrgMessage("%s is not a valid context or folder!", (*cl)[:pos])
		o.mode = NORMAL
		return
	}

	sess.showOrgMessage("'%s' will be opened", o.filter)

	o.clearMarkedEntries()
	org.view = TASK
	o.mode = NORMAL // can be changed to NO_ROWS below
	o.fc, o.fr, o.rowoff = 0, 0, 0
	o.rows = filterEntries(o.taskview, o.filter, o.show_deleted, o.sort, MAX)
	if len(o.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		o.mode = NO_ROWS
	}
	o.drawPreview()
	return
}

func (o *Organizer) openContext(pos int) {
	if pos == -1 {
		sess.showOrgMessage("You did not provide a context!")
		o.mode = NORMAL
		return
	}

	cl := o.command_line
	var success bool
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
	if pos == -1 {
		sess.showOrgMessage("You did not provide a folder!")
		o.mode = NORMAL
		return
	}

	cl := &o.command_line
	var success bool
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
	if pos == -1 {
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
	for _, w := range windows {
		if e, ok := w.(*Editor); ok {
			if e.id == id {
				active = true
				p = e
				break
			}
		}
	}

	if !active {
		p = NewEditor()
		windows = append(windows, p)
		p.id = id
		p.top_margin = TOP_MARGIN + 1

		if taskFolder(o.rows[o.fr].id) == "code" {
			p.output = &Output{}
			p.output.is_below = true
			p.output.id = id
			windows = append(windows, p.output)
		}
		readNoteIntoBuffer(p, id)

	}

	sess.positionWindows()
	sess.eraseRightScreen()      //erases editor area + statusbar + msg
	fmt.Print("\x1b_Ga=d\x1b\\") //delete any images
	sess.drawRightScreen()
	p.mode = NORMAL

	o.command = ""
	o.mode = NORMAL
}

func (o *Organizer) verticalResize(pos int) {
	if pos == -1 {
		sess.showOrgMessage("You need to provide a number 0 - 100")
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

}

func (o *Organizer) refresh(unused int) {
	if o.view == TASK {
		if o.taskview == BY_FIND {
			// if the view was BY_FIND put the mode back to FIND
			//o.last_mode = FIND
			o.mode = FIND
			o.fc, o.fr, o.rowoff = 0, 0, 0
			o.rows = searchEntries(sess.fts_search_terms, o.show_deleted, false)
			if len(o.rows) == 0 {
				sess.showOrgMessage("No results were returned")
				o.mode = NO_ROWS
			}
			if unused != -1 { //complete kluge has to do with refreshing when syncing
				o.drawPreview()
			}
		} else {
			o.mode = o.last_mode
			o.fc, o.fr, o.rowoff = 0, 0, 0
			o.rows = filterEntries(o.taskview, o.filter, o.show_deleted, o.sort, MAX)
			if len(o.rows) == 0 {
				sess.showOrgMessage("No results were returned")
				o.mode = NO_ROWS
			}
			if unused != -1 { //complete kluge has to do with refreshing when syncing
				o.drawPreview()
			}
		}
		sess.showOrgMessage("Entries will be refreshed")
	} else {
		o.mode = o.last_mode
		getContainers()
		if org.mode != NO_ROWS && unused != -1 {
			c := getContainerInfo(o.rows[o.fr].id)
			sess.displayContainerInfo(&c)
			sess.drawPreviewBox()
		}
		sess.showOrgMessage("view refreshed")
	}
	//o.mode = o.last_mode
	o.clearMarkedEntries()
}

func (o *Organizer) find(pos int) {

	if pos == -1 {
		sess.showOrgMessage("You did not enter something to find!")
		o.mode = NORMAL
		return
	}

	searchTerms := strings.ToLower(o.command_line[pos+1:])
	sess.fts_search_terms = searchTerms
	if len(searchTerms) < 3 {
		sess.showOrgMessage("You  need to provide at least 3 characters to search on")
		return
	}

	o.filter = ""
	o.taskview = BY_FIND
	o.fc, o.fr, o.rowoff = 0, 0, 0

	sess.showOrgMessage("Searching for '%s'", searchTerms)
	o.mode = FIND
	o.rows = searchEntries(searchTerms, o.show_deleted, false)
	if len(o.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		o.mode = NO_ROWS
	}
	o.drawPreview()
}

func (o *Organizer) sync(unused int) {
	var log string
	if o.command_line == "test" {
		// true => reportOnly
		log = synchronize(true)
	} else {
		log = synchronize(false)
	}
	o.command_line = ""
	o.eraseRightScreen()
	note := generateWWString(log, org.totaleditorcols)
	// below draw log as markeup
	r, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("/home/slzatz/listmango/darkslz.json"),
		glamour.WithWordWrap(0),
	)
	note, _ = r.Render(note)
	if note[0] == '\n' {
		note = note[1:]
	}
	o.note = strings.Split(note, "\n")
	o.drawPreviewWithoutImages()
	o.mode = PREVIEW_SYNC_LOG
}

func (o *Organizer) contexts(pos int) {
	o.mode = NORMAL

	if pos == -1 {
		sess.eraseRightScreen()
		o.view = CONTEXT
		getContainers()
		if o.mode != NO_ROWS {
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

	if pos == -1 {
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

	if pos == -1 {
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

func (o *Organizer) savelog(unused int) {
	if o.last_mode == PREVIEW_SYNC_LOG {
		title := fmt.Sprintf("%v", time.Now().Format("Mon Jan 2 15:04:05"))
		insertSyncEntry(title, strings.Join(o.note, "\n"))
		sess.showOrgMessage("Sync log save to database")
		o.command_line = ""
		o.mode = o.last_mode
	} // otherwise should save to file
}

func (o *Organizer) save(pos int) {
	if pos == -1 {
		sess.showOrgMessage("You need to provide a filename")
		return
	}
	filename := o.command_line[pos+1:]
	f, err := os.Create(filename)
	if err != nil {
		sess.showOrgMessage("Error creating file %s: %v", filename, err)
		return
	}
	defer f.Close()

	//_, err = f.Write(bytes.Join(e.bb, []byte("\n")))
	_, err = f.WriteString(strings.Join(o.note, "\n"))
	if err != nil {
		sess.showOrgMessage("Error writing file %s: %v", filename, err)
		return
	}
	sess.showOrgMessage("Note written to file %s", filename)
}

func (o *Organizer) setImage(pos int) {
	if pos == -1 {
		sess.showOrgMessage("You need to provide an option ('on' or 'off')")
		return
	}
	opt := o.command_line[pos+1:]
	if opt == "on" {
		sess.imagePreview = true
	} else if opt == "off" {
		sess.imagePreview = false
	} else {
		sess.showOrgMessage("Your choice of options is 'on' or 'off'")
	}
	o.mode = o.last_mode
	o.drawPreview()
	o.command_line = ""
}

func (o *Organizer) launchLsp(pos int) {
	var lsp string
	var cl string
	if pos != 0 {
		cl = o.command_line
		for _, v := range Lsps {
			if strings.HasPrefix(v, cl[pos+1:]) {
				lsp = v
				break
			}
		}
	} else {
		lsp = "gopls"
	}

	/*
		opt := o.command_line[pos+1 : pos+2]
		switch opt {
		case "g":
			lsp = "gopls"
		case "c":
			lsp = "clangd"
		}
	*/
	if lsp != "" {
		go launchLsp(lsp) // could be race to write to screen
	} else {
		sess.showOrgMessage("%q does not match an lsp", cl[pos+1:])
	}
	o.mode = o.last_mode
	o.command_line = ""
}

func (o *Organizer) shutdownLsp(unused int) {
	shutdownLsp()
	o.mode = o.last_mode
	o.command_line = ""
}
