package main

import "strings"

var navigation = map[int]struct{}{
	ARROW_UP:    z0,
	ARROW_DOWN:  z0,
	ARROW_LEFT:  z0,
	ARROW_RIGHT: z0,
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
			colon_N()
			return
		case '\x1b':
			org.command = ""
			org.repeat = 0
			return
		case 'i', 'I', 'a', 'A', 's':
			org.insertRow(0, "", true, false, false, BASE_DATE)
			org.mode = INSERT
			org.command = ""
			org.repeat = 0
			return
		}
		return

	case FIND:
		switch c {
		case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
			org.moveCursor(c)
			return

		default:
			org.mode = NORMAL
			org.command = ""
			organizerProcessKey(c)
			return
		}

	case INSERT:
		switch c {
		case '\r': //also does in effect an escape into NORMAL mode
			org.writeTitle()
		case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
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
		return // ? necessary

	case NORMAL:

		if c == '\x1b' {
			if org.view == TASK {
				sess.drawPreviewWindow(org.rows[org.fr].id)
			}
			sess.showOrgMessage("")
			org.command = ""
			org.repeat = 0
			return
		}

		if c == '\r' { //also does escape into NORMAL mode
			org.writeTitle()
			return
		}

		/*leading digit is a multiplier*/
		//if (isdigit(c))  //equiv to if (c > 47 && c < 58)

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

		return // end of case NORMAL

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

			sess.showOrgMessage("\x1b[41mNot an outline command: %s\x1b[0m", s)
			org.mode = NORMAL
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
		return //end of case COMMAND_LINE

	} // end switch o.mode
} // end func organizerProcessKey(c int)
