package main

var n_lookup = map[string]func(){
	string(ctrlKey('l')):       goto_editor_N,
	string([]byte{0x17, 0x17}): goto_editor_N,
	//  "\r": return_N, //return_O
	"i": insert_N,
	"s": s_N,
	"~": tilde_N,
	"r": r_N,
	"a": a_N,
	"A": A_N,
	"x": x_N,
	"w": w_N,

	"daw": daw_N,
	"dw":  dw_N,
	"caw": caw_N,
	"cw":  cw_N,
	"de":  de_N,
	"d$":  d_dollar_N,

	"gg": gg_N,

	//"gt": gt_N,

	string(ctrlKey('i')): info_N, //{{0x9}}
	"b":                  b_N,
	"e":                  e_N,
	"0":                  zero_N,
	"$":                  dollar_N,
	"I":                  I_N,
	"G":                  G_N,
	":":                  colon_N,
	"v":                  v_N,
	"p":                  p_N,
	"*":                  asterisk_N,
	"m":                  m_N,
	"n":                  n_N,
	//"u": u_N,
	"dd":         dd_N,
	string(0x4):  dd_N,        //ctrl-d
	string(0x2):  star_N,      //ctrl-b -probably want this go backwards (unimplemented) and use ctrl-e for this
	string(0x18): completed_N, //ctrl-x
}

//case 'i':
func insert_N() {
	org.mode = INSERT
	sess.showOrgMessage("\x1b[1m-- INSERT --\x1b[0m")
}

//case 's':
func s_N() {
	for i := 0; i < org.repeat; i++ {
		org.delChar()
	}
	//row.dirty = true; //in org.delChar but not sure it should be
	org.mode = INSERT
	sess.showOrgMessage("\x1b[1m-- INSERT --\x1b[0m") //[1m=bold
}

//case 'x':
func x_N() {
	for i := 0; i < org.repeat; i++ {
		org.delChar()
	}
	//row.dirty = true;
}

func daw_N() {
	for i := 0; i < org.repeat; i++ {
		org.delWord()
	}
}

func caw_N() {
	for i := 0; i < org.repeat; i++ {
		org.delWord()
	}
	org.mode = INSERT
	sess.showOrgMessage("\x1b[1m-- INSERT --\x1b[0m")
}

func dw_N() {
	for j := 0; j < org.repeat; j++ {
		start := org.fc
		org.moveEndWord2()
		end := org.fc
		org.fc = start
		t := &org.rows[org.fr].title
		*t = (*t)[:org.fc] + (*t)[end+1:]
	}
}

func cw_N() {
	for j := 0; j < org.repeat; j++ {
		start := org.fc
		org.moveEndWord2()
		end := org.fc
		org.fc = start
		t := &org.rows[org.fr].title
		*t = (*t)[:org.fc] + (*t)[end+1:]
	}
	org.mode = INSERT
	sess.showOrgMessage("\x1b[1m-- INSERT --\x1b[0m")
}

func de_N() {
	start := org.fc
	org.moveEndWord() //correct one to use to emulate vim
	end := org.fc
	org.fc = start
	t := &org.rows[org.fr].title
	*t = (*t)[:org.fc] + (*t)[end:]
}

func d_dollar_N() {
	org.deleteToEndOfLine()
}

//case 'r':
func r_N() {
	org.mode = REPLACE
}

//case '~'
func tilde_N() {
	for i := 0; i < org.repeat; i++ {
		org.changeCase()
	}
}

//case 'a':
func a_N() {
	org.mode = INSERT //this has to go here for MoveCursor to work right at EOLs
	org.moveCursor(ARROW_RIGHT)
	sess.showOrgMessage("\x1b[1m-- INSERT --\x1b[0m")
}

//case 'A':
func A_N() {
	org.moveCursorEOL()
	org.mode = INSERT //needs to be here for movecursor to work at EOLs
	org.moveCursor(ARROW_RIGHT)
	sess.showOrgMessage("\x1b[1m-- INSERT --\x1b[0m")
}

//case 'b':
func b_N() {
	org.moveBeginningWord()
}

//case 'e':
func e_N() {
	org.moveEndWord()
}

//case '0':
func zero_N() {
	org.fc = 0 // this was commented out - not sure why but might be interfering with O.repeat
}

//case '$':
func dollar_N() {
	org.moveCursorEOL()
}

//case 'I':
func I_N() {
	org.fc = 0
	org.mode = INSERT
	sess.showOrgMessage("\x1b[1m-- INSERT --\x1b[0m")
}

func gg_N() {
	org.fc = 0
	org.rowoff = 0
	org.fr = org.repeat - 1 //this needs to take into account O.rowoff
	if org.view == TASK {
		sess.drawPreviewWindow(org.rows[org.fr].id)
	}
	// } else {
	//   c := getContainerInfo(org.rows[org.fr].id)
	//   if c.id != 0 {
	//     sess.displayContainerInfo(c)
	//   }
	// }
}

//case 'G':
func G_N() {
	org.fc = 0
	org.fr = len(org.rows) - 1
	if org.view == TASK {
		sess.drawPreviewWindow(org.rows[org.fr].id)
	}
	// } else {
	//   c := getContainerInfo(org.rows[org.fr].id)
	//   if c.id != 0 {
	//     sess.displayContainerInfo(c);
	//   }
	// }
}

//case ':':
func colon_N() {
	sess.showOrgMessage(":")
	org.command_line = ""
	org.last_mode = org.mode
	org.mode = COMMAND_LINE
}

//case 'v':
func v_N() {
	org.mode = VISUAL
	org.highlight[0] = org.fc
	org.highlight[1] = org.fc
	sess.showOrgMessage("\x1b[1m-- VISUAL --\x1b[0m")
}

//case 'p':
func p_N() {
	if len(org.string_buffer) > 0 {
		org.pasteString()
	}
}

//case '*':
func asterisk_N() {
	org.getWordUnderCursor()
	org.findNextWord()
}

//case 'm':
func m_N() {

	if _, found := org.marked_entries[org.rows[org.fr].id]; found {
		delete(org.marked_entries, org.rows[org.fr].id)
	} else {
		org.marked_entries[org.rows[org.fr].id] = struct{}{}
	}

	/*
	  org.rows[org.fr].marked = !org.rows[org.fr].marked
	  if org.rows[org.fr].marked {
	    org.marked_entries[org.rows[org.fr].id] = struct{}{}
	  } else {
	    delete(org.marked_entries, org.rows[org.fr].id)
	  }
	*/

	sess.showOrgMessage("Toggle mark for item %d", org.rows[org.fr].id)
}

//case 'n':
func n_N() {
	org.findNextWord()
}

//dd and 0x4 -> ctrl-d
func dd_N() {
	toggleDeleted()
}

//0x2 -> ctrl-b
func star_N() {
	toggleStar()
}

//0x18 -> ctrl-x
func completed_N() {
	toggleCompleted()
}

//void outlineMoveNextWord() {
func w_N() {
	org.moveNextWord()
}

func info_N() {
	e := getEntryInfo(getId())
	sess.displayEntryInfo(&e)
	sess.drawPreviewBox()
}

func goto_editor_N() {
	if len(sess.editors) == 0 {
		sess.showOrgMessage("There are no active editors")
		return
	}

	sess.eraseRightScreen()
	sess.drawEditors()

	sess.editorMode = true
}

/*
void return_N(void) {
  orow& row = org.rows.at(org.fr);
  std::string msg;

  if(row.dirty){
    if (org.view == TASK) {
      updateTitle();
      msg = "Updated id {} to {} (+fts)";

      if (sess.lm_browser) {
        int folder_tid = getFolderTid(org.rows.at(org.fr).id);
        if (!(folder_tid == 18 || folder_tid == 14)) sess.updateHTMLFile("assets/" + CURRENT_NOTE_FILE);
      }
    } else if (org.view == CONTEXT || org.view == FOLDER) {
      updateContainerTitle();
      msg = "Updated id {} to {}";
    } else if (org.view == KEYWORD) {
      updateKeywordTitle();
      msg = "Updated id {} to {}";
    }
    org.command[0] = '\0'; //11-26-2019
    org.mode = NORMAL;
    if (org.fc > 0) org.fc--;
    row.dirty = false;
    sess.showOrgMessage3(msg, row.id, row.title);
    return;
  }

  // return means retrieve items by context or folder
  // do this in database mode
  if (org.view == CONTEXT) {
    org.context = row.title;
    org.folder = "";
    org.taskview = BY_CONTEXT;
    sess.showOrgMessage("\'%s\' will be opened", org.context.c_str());
    org.command_line = "o " + org.context;
  } else if (org.view == FOLDER) {
    org.folder = row.title;
    org.context = "";
    org.taskview = BY_FOLDER;
    sess.showOrgMessage("\'%s\' will be opened", org.folder.c_str());
    org.command_line = "o " + org.folder;
  } else if (org.view == KEYWORD) {
    org.keyword = row.title;
    org.folder = "";
    org.context = "";
    org.taskview = BY_KEYWORD;
    sess.showOrgMessage("\'%s\' will be opened", org.keyword.c_str());
    org.command_line = "ok " + org.keyword;
  }

  sess.command_history.push_back(org.command_line);
  sess.page_hx_idx++;
  sess.page_history.insert(sess.page_history.begin() + sess.page_hx_idx, org.command_line);
  org.marked_entries.clear();

  getItems(MAX);
}
*/
