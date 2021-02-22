package main

import (
	"flag"
	"fmt"
	"os"
//	"time"
  "strings"

	"github.com/slzatz/listmango/rawmode"
	"github.com/slzatz/listmango/terminal"
)

/*
func ctrlKey(b byte) rune {
  return rune(b & 0x1f)
}
*/
var z0 = struct{}{}
var navigation = map[int]struct{} {
                   ARROW_UP:z0,
                   ARROW_DOWN:z0,
                   ARROW_LEFT:z0,
                   ARROW_RIGHT:z0,
                  'h':z0,
                  'j':z0,
                  'k':z0,
                  'l':z0,
                  }

// SafeExit restores terminal using the original terminal config stored
// in the global session variable
/*
func SafeExit(err error) {
	fmt.Fprint(os.Stdout, "\x1b[2J\x1b[H")

	if err1 := rawmode.Restore(sess.origTermCfg); err1 != nil {
		fmt.Fprintf(os.Stderr, "Error: disabling raw mode: %s\r\n", err)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\r\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
*/

var sess Session
var org Organizer

func main() {

	// parse config flags & parameters
	flag.Parse()

	// enable raw mode
	origCfg, err := rawmode.Enable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling raw mode: %v", err)
		os.Exit(1)
	}
	sess.origTermCfg = origCfg

	sess.editorMode = false

	// get the screen dimensions and create a view
	sess.screenLines, sess.screenCols, err = rawmode.GetWindowSize()
	if err != nil {
		//SafeExit(fmt.Errorf("couldn't get window size: %v", err))
    os.Exit(1)
	}

	sess.showOrgMessage("hello")
	//filename := flag.Arg(0)

  org.cx = 0 //cursor x position
  org.cy = 0 //cursor y position
  org.fc = 0 //file x position
  org.fr = 0 //file y position
  org.rowoff = 0  //number of rows scrolled off the screen
  org.coloff = 0  //col the user is currently scrolled to  
  org.sort = "modified" //Entry sort column
  org.show_deleted = false //not treating these separately right now
  org.show_completed = true
  org.message = "" //very bottom of screen; ex. -- INSERT --
  org.highlight[0], org.highlight[1] = -1, -1
  org.mode = NORMAL //0=normal; 1=insert; 2=command line; 3=visual line; 4=visual; 5='r' 
  org.last_mode = NORMAL
  org.command = ""
  org.command_line = ""
  org.repeat = 0 //number of times to repeat commands like x,s,yy also used for visual line mode x,y

  org.view = TASK; // not necessary here since set when searching database
  org.taskview = BY_FOLDER
  org.folder = "todo"
  org.context = "No Context"
  org.keyword = ""

  org.context_map = make(map[string]int)
  org.folder_map = make(map[string]int)

  // ? where this should be.  Also in signal.
  sess.textLines = sess.screenLines - 2 - TOP_MARGIN // -2 for status bar and message bar
  //sess.divider = sess.screencols - sess.cfg.ed_pct * sess.screencols/100
  sess.divider = sess.screenCols - (60 * sess.screenCols/100)
  sess.totaleditorcols = sess.screenCols - sess.divider - 1 // was 2 

  generateContextMap()
  generateFolderMap()
  sess.eraseScreenRedrawLines()
  getItems(MAX)

  sess.refreshOrgScreen();
  sess.drawOrgStatusBar();
  sess.showOrgMessage("rows: %d  columns: %d", sess.screenLines, sess.screenCols);
  sess.returnCursor();
  sess.run = true

	for sess.run {
		//sess.View.RefreshScreen(sess.Editor, sess.StatusMessage, sess.Prompt)

		// read key
		key, err := terminal.ReadKey()
		if err != nil {
			//SafeExit(fmt.Errorf("Error reading from terminal: %s", err))
      os.Exit(1)
		}

    var k int
    if key.Regular != 0 {
      k = int(key.Regular)
    } else {
      k = key.Special
    }

		if sess.editorMode {
			editorProcessKey(k)
		} else {
			organizerProcessKey(k)
      org.scroll()
      sess.refreshOrgScreen()
      sess.drawOrgStatusBar();
		}

    sess.returnCursor()
		// if it's been 5 secs since the last status message, reset
		//if time.Now().Sub(sess.StatusMessageTime) > time.Second*5 && sess.State == stateEditing {
		//	sess.setStatusMessage("")
		//}
	}
  sess.quitApp()
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

  case NORMAL:
      if c == '\x1b' {
        if (org.view == TASK) {
          sess.drawPreviewWindow(org.rows[org.fr].id)
        }
        sess.showOrgMessage("")
        org.command = ""
        org.repeat = 0
        return
      }

      /*leading digit is a multiplier*/
      //if (isdigit(c))  //equiv to if (c > 47 && c < 58)

      if ((c > 47 && c < 58) && len(org.command) == 0) {

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
        pos := strings.Index(org.command_line, " ")
        var s string
        if pos != - 1 {
          s = org.command_line[:pos]
        } else {
          pos = 0
          s = org.command_line
        }
        if cmd, found := cmd_lookup[s]; found {
          cmd(pos)
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
        sess.p.showMessage(":")
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
        sess.p.showMessage("\x1b[1m-- INSERT --\x1b[0m")
        //p.command[0] = '\0'
        //p.repeat = 0
        // ? p.redraw = true
        return true
      }
    case INSERT:
      switch c {

      case '\r':
          sess.p.insertReturn()
          sess.p.last_typed += string(c)
          return true

        case HOME_KEY:
          sess.p.moveCursorBOL()
          return false

        case END_KEY:
          sess.p.moveCursorEOL()
          sess.p.moveCursor(ARROW_RIGHT)
          return false

        case BACKSPACE:
          sess.p.backspace()

          //not handling backspace correctly
          //when backspacing deletes more than currently entered text
          //A common case would be to enter insert mode  and then just start backspacing
          //because then dotting would actually delete characters
          //I could record a \b and then handle similar to handling \r
          length := len(sess.p.last_typed)
          if length > 0 {
            sess.p.last_typed = sess.p.last_typed[:length-1]
          }
          return true
    
        case DEL_KEY:
          sess.p.delChar()
          return true
    
        case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
          sess.p.moveCursor(c)
          return false
    
        case ctrlKey('b'), ctrlKey('e'):
          //sess.p.push_current() //p.editorCreateSnapshot()
          //sess.p.editorDecorateWord(c)
          return true
    
          /*
        // this should be a command line command
        case ctrlKey('z'):
          if sess.p.smartindex != 0 {
            sess.p.smartindent = 0
          } else {
            sess.p.smartindex = SMARTINDENT
          }
          sess.p.editorSetMessage("smartindent = %d", sess.p.smartindent)
          return false
    */
        case '\x1b':

          /*
           * below deals with certain NORMAL mode commands that
           * cause entry to INSERT mode includes dealing with repeats
           */
         /*
          //i,I,a,A - deals with repeat
          if _, found := cmd_map1[sess.p.last_command]; found {
            sess.p.push_current() //
            for n := 0; n < sess.p.last_repeat-1; n++ {
              for pos, char := range sess.p.last_typed {
                sess.p.editorInsertChar(char)
              }
            }
          }

          //cmd_map2 -> E_o_escape and E_O_escape - here deals with deals with repeat > 1
          if cmd, found := cmd_map2[sess.p.last_command]; found {
            cmd(sess.p, sess.p.last_repeat - 1)
            sess.p.push_current()
          }

          //cw, caw, s
          if _, found := cmd_map4[sess.p.last_command]; found {
            sess.p.push_current()
          }
          //'I' in VISUAL BLOCK mode
          if sess.p.last_command == "VBI" {
            for n := 0; n < sess.p.last_repeat-1; n++ {
              for pos, char := range sess.p.last_typed {
                sess.p.editorInsertChar(char)
              }
            }
            temp := sess.p.fr

            for sess.p.fr=sess.p.fr+1; sess.p.fr<sess.p.vb0[1]+1; sess.p.fr++ {
              for n := 0; n<sess.p.last_repeat; n++ { //NOTICE not p.last_repeat - 1
                sess.p.fc = sess.p.vb0[0]
                for pos, char := range sess.p.last_typed {
                  sess.p.editorInsertChar(char)
                }
              }
            }
            sess.p.fr = temp
            sess.p.fc = sess.p.vb0[0]
          }

          //'A' in VISUAL BLOCK mode
          if sess.p.last_command == "VBA" {
            for n := 0; n < sess.p.last_repeat-1; n++ {
              for pos, char := range sess.p.last_typed {
                sess.p.editorInsertChar(char)
              }
            }
            //{ 12302020
            temp := sess.p.fr

            for sess.p.fr=sess.p.fr+1; sess.p.fr<sess.p.vb0[1]+1; sess.p.fr++ {
              for n := 0; n<sess.p.last_repeat; n++ { //NOTICE not p.last_repeat - 1
                length := len(sess.p.rows[sess.p.fr])
                if sess.p.vb0[2] > length {
                  sess.p.rows[sess.p.fr] + strings.Repeat(" ", sess.p.vb0[2] - length)
                }
                sess.p.fc = sess.p.vb0[2]
              for pos, char := range sess.p.last_typed {
                sess.p.editorInsertChar(char)
              }
              }
            }
            sess.p.fr = temp
            sess.p.fc = sess.p.vb0[0]
          //} 12302020
          }
           */
          /*Escape whatever else happens falls through to here*/
          sess.p.mode = NORMAL
          sess.p.repeat = 0


          if sess.p.fc > 0 {
            sess.p.fc--
          }

          /*
          // below - if the indent amount == size of line then it's all blanks
          // can hit escape with p.row == NULL or p.row[p.fr].size == 0
          if len(sess.p.rows) != 0 && len(sess.p.rows[sess.p.fr]) != 0 {
            n := sess.p.editorIndentAmount(sess.p.fr)
            if n == len(sess.p.rows[sess.p.fr]) {
              sess.p.fc = 0
              for i := 0; i < n; i++ {
                sess.p.editorDelChar()
              }
            }
          }
          sess.p.editorSetMessage("") // commented out to debug push_current
          //editorSetMessage(p.last_typed.c_str())
          sess.p.last_typed = "" /////////// 09182020
          */
          return true //end case x1b:
    
        // deal with tab in insert mode - was causing segfault  
        case '\t':
          for  i := 0; i < 4; i++{
            sess.p.insertChar(' ')
          }
          return true  

        default:
          sess.p.insertChar(c)
          sess.p.last_typed += string(c)
          return true
     
      } //end inner switch for outer case INSERT

      return true // end of case INSERT: - should not be executed

	}
  return true
}
