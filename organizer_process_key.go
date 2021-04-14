package main

import (
	"strings"
)

var navigation = map[int]struct{}{
	ARROW_UP:    z0,
	ARROW_DOWN:  z0,
	ARROW_LEFT:  z0,
	ARROW_RIGHT: z0,
	PAGE_UP:     z0,
	PAGE_DOWN:   z0,
	'h':         z0,
	'j':         z0,
	'k':         z0,
	'l':         z0,
}

func organizerProcessKey(c int) {

	switch org.mode {

	case NO_ROWS:
		switch c {
		case ':':
			exCmd()
		case '\x1b':
			org.command = ""
			org.repeat = 0
		case 'i', 'I', 'a', 'A', 's':
			org.insertRow(0, "", true, false, false, BASE_DATE)
			org.mode = INSERT
			org.command = ""
			org.repeat = 0
		}

	case FIND:
		switch c {
		case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT, PAGE_UP, PAGE_DOWN:
			org.moveCursor(c)
		default:
			org.mode = NORMAL
			org.command = ""
			organizerProcessKey(c)
		}

	case INSERT:
		switch c {
		case '\r': //also does in effect an escape into NORMAL mode
			org.writeTitle()
		case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT, PAGE_UP, PAGE_DOWN:
			org.moveCursor(c)
		case '\x1b':
			org.command = ""
			org.mode = NORMAL
			if org.fc > 0 {
				org.fc--
			}
			sess.showOrgMessage("")
		case HOME_KEY:
			org.fc = 0
		case END_KEY:
			org.fc = len(org.rows[org.fr].title)
		case BACKSPACE:
			org.backspace()
		case DEL_KEY:
			org.delChar()
		case '\t':
			//do nothing
		default:
			org.insertChar(c)
		}
		//return // ? necessary

	case NORMAL:

		if c == '\x1b' {
			if org.view == TASK {
				//org.drawPreviewWindow()
				org.drawPreview()
			}
			sess.showOrgMessage("")
			org.command = ""
			org.repeat = 0
			return
		}

		if c == ctrlKey('l') && org.last_mode == ADD_CHANGE_FILTER {
			org.mode = ADD_CHANGE_FILTER
			sess.eraseRightScreen()
		}

		if c == '\r' { //also does escape into NORMAL mode
			row := &org.rows[org.fr]
			if row.dirty {
				org.writeTitle()
				return
			}
			switch org.view {
			case CONTEXT:
				org.taskview = BY_CONTEXT
			case FOLDER:
				org.taskview = BY_FOLDER
			case KEYWORD:
				org.taskview = BY_KEYWORD
			}

			org.filter = row.title
			sess.showOrgMessage("'%s' will be opened", org.filter)

			org.clearMarkedEntries()
			org.view = TASK
			org.mode = NORMAL // can be changed to NO_ROWS below
			org.fc, org.fr, org.rowoff = 0, 0, 0
			org.rows = filterEntries(org.taskview, org.filter, org.show_deleted, org.sort, MAX)
			if len(org.rows) == 0 {
				sess.showOrgMessage("No results were returned")
				org.mode = NO_ROWS
			}
			//org.drawPreviewWindow()
			org.drawPreview()
			return
		}

		/*leading digit is a multiplier*/

		if (c > 47 && c < 58) && len(org.command) == 0 {

			if org.repeat == 0 && c == 48 {
			} else if org.repeat == 0 {
				org.repeat = c - 48
				return
			} else {
				org.repeat = org.repeat*10 + c - 48
				return
			}
		}

		if org.repeat == 0 {
			org.repeat = 1
		}

		org.command += string(c)

		if cmd, found := n_lookup[org.command]; found {
			cmd()
			org.command = ""
			org.repeat = 0
			return
		}

		//also means that any key sequence ending in something
		//that matches below will perform command

		// needs to be here because needs to pick up repeat
		//Arrows + h,j,k,l
		if _, found := navigation[c]; found {
			for j := 0; j < org.repeat; j++ {
				org.moveCursor(c)
			}
			org.command = ""
			org.repeat = 0
			return
		}

	//return // end of case NORMAL

	case REPLACE:
		if org.repeat == 0 {
			org.repeat = 1
		}
		if c == '\x1b' {
			org.command = ""
			org.repeat = 0
			org.mode = NORMAL
			return
		}

		for i := 0; i < org.repeat; i++ {
			org.delChar()
			org.insertChar(c)
		}

		org.repeat = 0
		org.command = ""
		org.mode = NORMAL

		return

	case ADD_CHANGE_FILTER:

		switch c {

		case '\x1b':
			org.mode = NORMAL
			org.last_mode = ADD_CHANGE_FILTER
			org.command = ""
			org.command_line = ""
			org.repeat = 0

		case ARROW_UP, ARROW_DOWN, 'j', 'k':
			org.moveAltCursor(c)

		case '\r':
			altRow := &org.altRows[org.altFr] //currently highlighted container row
			row := &org.rows[org.fr]          //currently highlighted entry row
			if len(org.marked_entries) == 0 {
				switch org.altView {
				case KEYWORD:
					addTaskKeyword(altRow.id, row.id, true)
					sess.showOrgMessage("Added keyword %s to current entry", altRow.title)
				case FOLDER:
					updateTaskFolder(altRow.title, row.id)
					sess.showOrgMessage("Current entry folder changed to %s", altRow.title)
				case CONTEXT:
					updateTaskContext(altRow.title, row.id)
					sess.showOrgMessage("Current entry had context changed to %s", altRow.title)
				}
			} else {
				for id := range org.marked_entries {
					switch org.altView {
					case KEYWORD:
						addTaskKeyword(altRow.id, id, true)
					case FOLDER:
						updateTaskFolder(altRow.title, id)
					case CONTEXT:
						updateTaskContext(altRow.title, id)

					}
					sess.showOrgMessage("Marked entries' %d changed/added to %s", org.altView, altRow.title)
				}
			}
		}

	case COMMAND_LINE:
		if c == '\x1b' {
			org.mode = NORMAL
			sess.showOrgMessage("")
			return
		}

		if c == '\r' {
			pos := strings.Index(org.command_line, " ")
			var s string
			if pos != -1 {
				s = org.command_line[:pos]
			} else {
				pos = 0
				s = org.command_line
			}
			if cmd, found := cmd_lookup[s]; found {
				cmd(&org, pos)
				return
			}

			sess.showOrgMessage("\x1b[41mNot a recognized command: %s\x1b[0m", s)
			org.mode = org.last_mode
			return
		}

		if c == DEL_KEY || c == BACKSPACE {
			length := len(org.command_line)
			if length > 0 {
				org.command_line = org.command_line[:length-1]
			}
		} else {
			org.command_line += string(c)
		}

		sess.showOrgMessage(":%s", org.command_line)
		//return //end of case COMMAND_LINE

		//probably should be a org.view not org.mode but
		// for the moment this kluge works
	case SYNC_LOG:
		switch c {
		case ARROW_UP, 'k':
			if org.fr == 0 {
				return
			}
			org.fr--
			sess.eraseRightScreen()
			org.altRowoff = 0
			note := readSyncLog(org.rows[org.fr].id)
			org.note = generateWWString(note, org.totaleditorcols, 500, "\n")
			org.drawNoteReadOnly()
		case ARROW_DOWN, 'j':
			if org.fr == len(org.rows)-1 {
				return
			}
			org.fr++
			sess.eraseRightScreen()
			org.altRowoff = 0
			note := readSyncLog(org.rows[org.fr].id)
			org.note = generateWWString(note, org.totaleditorcols, 500, "\n")
			org.drawNoteReadOnly()
		case ':':
			sess.showOrgMessage(":")
			org.command_line = ""
			org.last_mode = org.mode
			org.mode = COMMAND_LINE

		// the two below only handle logs < 2x textLines
		case PAGE_DOWN:
			org.altRowoff++
			sess.eraseRightScreen()
			org.drawNoteReadOnly()
		case PAGE_UP:
			if org.altRowoff > 0 {
				org.altRowoff--
			}
			sess.eraseRightScreen()
			org.drawNoteReadOnly()
		case ctrlKey('d'):
			if len(org.marked_entries) == 0 {
				deleteSyncItem(org.rows[org.fr].id)
			} else {
				for id := range org.marked_entries {
					deleteSyncItem(id)
				}
			}
			org.log(0)
		case 'm':
			mark()
		}
	case PREVIEW_MARKDOWN, PREVIEW_SYNC_LOG:
		switch c {
		case '\x1b', 'm':
			if org.mode == PREVIEW_MARKDOWN {
				sess.editorMode = true
				p.refreshScreen()
			} else {
				org.drawPreview()
			}
			org.mode = NORMAL
		case ':':
			sess.showOrgMessage(":")
			org.command_line = ""
			org.last_mode = org.mode
			org.mode = COMMAND_LINE

		// the two below only handle logs < 2x textLines
		case PAGE_DOWN, ARROW_DOWN, 'j':
			org.altRowoff++
			sess.eraseRightScreen()
			org.drawNoteReadOnly()
		case PAGE_UP, ARROW_UP, 'k':
			if org.altRowoff > 0 {
				org.altRowoff--
			}
			sess.eraseRightScreen()
			org.drawNoteReadOnly()
		}
	} // end switch o.mode
} // end func organizerProcessKey(c int)
