package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/slzatz/listmango/rawmode"
	"github.com/slzatz/listmango/terminal"
)

func ctrlKey(b byte) rune {
  return rune(b & 0x1f)
}
z0 := struct{}{}
navigation := map[int]struct{} {
                   ARROW_UP:z0,
                   ARROW_DOWN:z0,
                   ARROW_LEFT:z0,
                   ARROW_RIGHT:z0,
                  'h':z0,
                  'j':z0,
                  'k':z0,
                  'l':z0,
                  }

 //type Editor struct{ ...}
// cmd1_map = make(map[string]func(*Editor, int),4)
cmd1_map := map[string]func(*Editor, int){
                   "i":(*Editor).E_i,
                   "I":(*Editor).E_a,
                   "a":(*Editor).E_a,
                   "A":(*Editor).E_A,
                 }
// to call it's cmd1_map["i"](e, repeat)

// SafeExit restores terminal using the original terminal config stored
// in the global session variable
func SafeExit(err error) {
	fmt.Fprint(os.Stdout, "\x1b[2J\x1b[H")

	if err1 := rawmode.Restore(s.OrigTermCfg); err1 != nil {
		fmt.Fprintf(os.Stderr, "Error: disabling raw mode: %s\r\n", err)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\r\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

var s = Session{}

func main_() {

	// parse config flags & parameters
	flag.Parse()
	filename := flag.Arg(0)

	// enable raw mode
	origCfg, err := rawmode.Enable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling raw mode: %v", err)
		os.Exit(1)
	}
	s.OrigTermCfg = origCfg

	s.editorMode = false

	// get the screen dimensions and create a view
	s.screenLines, s.screenCols, err := rawmode.GetWindowSize()
	if err != nil {
		SafeExit(fmt.Errorf("couldn't get window size: %v", err))
	}

	s.setStatusMessage("hello")

	for {
		//s.View.RefreshScreen(s.Editor, s.StatusMessage, s.Prompt)

		// read key
		k, err := terminal.ReadKey()
		if err != nil {
			SafeExit(fmt.Errorf("Error reading from terminal: %s", err))
		}

		if s.editorMode {
			editorProcessKey(k)
		} else {
			organizerProcessKey(k)
		}

		// if it's been 5 secs since the last status message, reset
		if time.Now().Sub(s.StatusMessageTime) > time.Second*5 && s.State == stateEditing {
			s.setStatusMessage("")
		}
	}
}

func organizerProcessKey(c int) {

	switch o.mode {

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

   	case INSERT:
      switch c {
      case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
        org.moveCursor(c)
        return
        case '\x1b':
          org.command = ""
          org.mode = NORMAL
          if org.fc > 0 {
            org.fc--
          }
          sess.showOrgMessage("")
          return
        default:
          org.insertChar(c)
          return
        }
      }

  	case NORMAL:
      if c == '\x1b' {
        if (org.view == TASK) {
          sess.drawPreviewWindow(org.rows.at(org.fr).id)
        }
        sess.showOrgMessage("")
        org.command[0] = ""
        org.repeat = 0
        return
      }

      /*leading digit is a multiplier*/
      //if (isdigit(c))  //equiv to if (c > 47 && c < 58)

      if ((c > 47 && c < 58) && len(org.command) == 0)) {

        if (org.repeat == 0 && c == 48) {

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

      if cmd, found := n_lookup[org.command] found {
        cmd()
        org.command = ""
        org.repeat = 0
        return
      }

      //also means that any key sequence ending in something
      //that matches below will perform command

      // needs to be here because needs to pick up repeat
      //Arrows + h,j,k,l
      if _, found := navigation[c] {
        for j := 0; j < org.repeat; j++ {
          org.moveCursor(c)
        }
        org.command =  ""
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
        pos := strings.Index(org.command_line, ' ')
        cmd := org.command_line[0:pos]
        if cmd, found := cmd_lookup[cmd]; found  {
          if pos == -1 {
            pos = 0
          }
          cmd(pos)
          return
        }

        sess.showOrgMessage("\x1b[41mNot an outline command: %s\x1b[0m", cmd.c_str())
        org.mode = NORMAL
        return
      }

      if c == DEL_KEY || c == BACKSPACE {
        length = len(org.command_line)
        if length > 0 {
          org.command_line = org.command_line[:length-1]
      } else {
        org.command_line += string(c)
      }

      sess.showOrgMessage(":%s", org.command_line.c_str())
      return //end of case COMMAND_LINE

	}

func editorProcessKey(c int) bool {

	switch sess.p.mode {

    case NO_ROWS:
      switch c {
      case '\x1b':
        sess.p.command = ""
        sess.p.repeat = 0
        return false
      case ':':
        sess.p.mode = COMMAND_LINE
        sess.p.command_line = ""
        sess.p.command = ""
        sess.p.editorSetMessage(":")
       return false
      case 'i', 'I', 'a', 'A', 's', 'o', 'O':
        //p.editorInsertRow(0, std::string())
        sess.p.mode = INSERT
        sess.p.last_command = "i" //all the commands equiv to i
        sess.p.prev_fr = 0
        sess.p.prev_fc = 0
        sess.p.last_repeat = 1
        sess.p.snapshot = nil
        sess.p.snapshot = append(sess.p.snapshot, "")
        sess.p.editorSetMessage("\x1b[1m-- INSERT --\x1b[0m")
        //p.command[0] = '\0'
        //p.repeat = 0
        // ? p.redraw = true
        return true
      }
    case INSERT:
      switch c {

        case '\r':
          sess.p.editorInsertReturn()
          sess.p.last_typed += c
          return true

        // not sure this is in use
        case ctrlKey('s')
          sess.p.editorSaveNoteToFile("lm_temp")
          return false

        case HOME_KEY:
          sess.p.editorMoveCursorBOL()
          return false

        case END_KEY:
          sess.p.editorMoveCursorEOL()
          sess.p.editorMoveCursor(ARROW_RIGHT)
          return false

        case BACKSPACE:
          sess.p.editorBackspace()

          //not handling backspace correctly
          //when backspacing deletes more than currently entered text
          //A common case would be to enter insert mode  and then just start backspacing
          //because then dotting would actually delete characters
          //I could record a \b and then handle similar to handling \r
          if (!sess.p.last_typed.empty()) sess.p.last_typed.pop_back()
          return true
    
        case DEL_KEY:
          sess.p.editorDelChar()
          return true
    
        case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
          sess.p.editorMoveCursor(c)
          return false
    
        case ctrlKey('b'):
        //case ctrlKey('i'): ctrlKey('i') -> 9 same as tab
        case ctrlKey('e'):
          sess.p.push_current() //p.editorCreateSnapshot()
          sess.p.editorDecorateWord(c)
          return true
    
        // this should be a command line command
        case ctrlKey('z'):
          sess.p.smartindent = (sess.p.smartindent) ? 0 : SMARTINDENT;
          sess.p.editorSetMessage("smartindent = %d", sess.p.smartindent) 
          return false
    
        case '\x1b':

          /*
           * below deals with certain NORMAL mode commands that
           * cause entry to INSERT mode includes dealing with repeats
           */

          //i,I,a,A - deals with repeat
          if(cmd_map1.contains(sess.p.last_command)) { 
            sess.p.push_current() //
            for (int n=0; n<sess.p.last_repeat-1; n++) {
              for (char const &c : sess.p.last_typed) {sess.p.editorInsertChar(c)}
            }
          }

          //cmd_map2 -> E_o_escape and E_O_escape - here deals with deals with repeat > 1
          if (cmd_map2.contains(sess.p.last_command)) {
            (sess.p.*cmd_map2.at(sess.p.last_command))(sess.p.last_repeat - 1)
            sess.p.push_current()
          }

          //cw, caw, s
          if (cmd_map4.contains(sess.p.last_command)) {
            sess.p.push_current()
          }
          //'I' in VISUAL BLOCK mode
          if (sess.p.last_command == "VBI") {
            for (int n=0; n<sess.p.last_repeat-1; n++) {
              for (char const &c : sess.p.last_typed) {sess.p.editorInsertChar(c)}
            }
            int temp = sess.p.fr

            for (sess.p.fr=sess.p.fr+1; sess.p.fr<sess.p.vb0[1]+1; sess.p.fr++) {
              for (int n=0; n<sess.p.last_repeat; n++) { //NOTICE not p.last_repeat - 1
                sess.p.fc = sess.p.vb0[0] 
                for (char const &c : sess.p.last_typed) {sess.p.editorInsertChar(c)}
              }
            }
            sess.p.fr = temp
            sess.p.fc = sess.p.vb0[0]
          }

          //'A' in VISUAL BLOCK mode
          if (sess.p.last_command == "VBA") {
            for (int n=0; n<sess.p.last_repeat-1; n++) {
              for (char const &c : sess.p.last_typed) {sess.p.editorInsertChar(c);}
            }
            //{ 12302020
            int temp = sess.p.fr

            for (sess.p.fr=sess.p.fr+1; sess.p.fr<sess.p.vb0[1]+1; sess.p.fr++) {
              for (int n=0; n<sess.p.last_repeat; n++) { //NOTICE not p.last_repeat - 1
                int size = sess.p.rows.at(sess.p.fr).size()
                if (sess.p.vb0[2] > size) sess.p.rows.at(sess.p.fr).insert(size, sess.p.vb0[2]-size, ' ')
                sess.p.fc = sess.p.vb0[2]
                for (char const &c : sess.p.last_typed) {sess.p.editorInsertChar(c)}
              }
            }
            sess.p.fr = temp
            sess.p.fc = sess.p.vb0[0]
          //} 12302020
          }

          /*Escape whatever else happens falls through to here*/
          sess.p.mode = NORMAL
          sess.p.repeat = 0

          //? redundant - see 10 lines below
          sess.p.last_typed = std::string() 

          if (sess.p.fc > 0) sess.p.fc--

          // below - if the indent amount == size of line then it's all blanks
          // can hit escape with p.row == NULL or p.row[p.fr].size == 0
          if (!sess.p.rows.empty() && sess.p.rows[sess.p.fr].size()) {
            int n = sess.p.editorIndentAmount(sess.p.fr)
            if (n == sess.p.rows[sess.p.fr].size()) {
              sess.p.fc = 0
              for (int i = 0; i < n; i++) {
                sess.p.editorDelChar()
              }
            }
          }
          sess.p.editorSetMessage("") // commented out to debug push_current
          //editorSetMessage(p.last_typed.c_str())
          sess.p.last_typed.clear()//////////// 09182020
          return true //end case x1b:
    
        // deal with tab in insert mode - was causing segfault  
        case '\t':
          for (int i=0; i<4; i++) sess.p.editorInsertChar(' ')
          return true  

        default:
          sess.p.editorInsertChar(c)
          sess.p.last_typed += c
          return true
     
      } //end inner switch for outer case INSERT

      return true // end of case INSERT: - should not be executed

	}
}
