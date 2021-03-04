package main

import "strings"
/*
var e_lookup2 = map[string]func(*Editor) {
  "\x17L":(*Editor).moveOutputWindowRight,
  "\x17J":(*Editor).moveOutputWindowBelow,
  "\x12":(*Editor).controlJ,
  "\x13":(*Editor).controlK,
}
*/

var e_lookup2 = map[string]interface{} {
  "\x17L":(*Editor).moveOutputWindowRight,
  "\x17J":(*Editor).moveOutputWindowBelow,
  "\x08":controlH,
  "\x0c":controlL,
  "\x02":(*Editor).decorateWord,
  "\x05":(*Editor).decorateWord,
  string(ctrlKey('i')):(*Editor).decorateWord,
  "\x17=":(*Editor).resize,
  "\x17_":(*Editor).resize,
}

func (e *Editor) resize(flag int) {
  if e.linked_editor == nil {
    return
  }

  le := e.linked_editor
  var subnote_height int
  if flag == '=' {
    subnote_height = sess.textLines/2
  } else if flag == '_' {
    subnote_height = LINKED_NOTE_HEIGHT
  } else {
    return
  }

  if !e.is_subeditor {
    e.screenlines = sess.textLines - subnote_height - 1
    le.screenlines = subnote_height
    le.top_margin = sess.textLines - subnote_height + 2
  } else {
    le.screenlines = sess.textLines - subnote_height - 1
    e.screenlines = subnote_height
    e.top_margin = sess.textLines - subnote_height + 2
  }
  le.refreshScreen(true)
  e.refreshScreen(true)
}

func (e *Editor) moveOutputWindowRight() {
  if (e.linked_editor == nil){ // && e.is_subeditor && e.is_below) {
    return
  }
  //top_margin = TOP_MARGIN + 1;
  //screenlines = total_screenlines - 1;
  e.linked_editor.is_below = false

  editor_slots := 0;
  for _, e := range sess.editors {
    if !e.is_below {
      editor_slots++
    }
  }

  s_cols := -1 + (sess.screenCols - sess.divider)/editor_slots
  i := -1 //i = number of columns of editors -1
  for _, e := range sess.editors {
    if !e.is_below {
      i++
    }
    e.left_margin = sess.divider +i*s_cols + i
    e.screencols = s_cols
    e.setLinesMargins()
 }
  sess.eraseRightScreen()
  sess.drawEditors()
  //editorSetMessage("top_margin = %d", top_margin);

}

func (e *Editor) moveOutputWindowBelow() {
  if (e.linked_editor == nil){ // && e.is_subeditor && e.is_below) {
    return
  }
  //top_margin = TOP_MARGIN + 1;
  //screenlines = total_screenlines - 1;
  e.linked_editor.is_below = true

  editor_slots := 0;
  for _, e := range sess.editors {
    if !e.is_below {
      editor_slots++
    }
  }

  s_cols := -1 + (sess.screenCols - sess.divider)/editor_slots
  i := -1 //i = number of columns of editors -1
  for _, e := range sess.editors {
    if !e.is_below {
      i++
    }
    e.left_margin = sess.divider +i*s_cols + i
    e.screencols = s_cols
    e.setLinesMargins()
 }
  sess.eraseRightScreen()
  sess.drawEditors()
  //editorSetMessage("top_margin = %d", top_margin);
}

func (e *Editor) controlJ() {
			if e.linked_editor.is_below {
				e = e.linked_editor
			}
				e.mode = NORMAL
			e.command = ""
}

func (e *Editor) controlK() {
			if e.is_below {
				e = e.linked_editor
			}
				e.mode = NORMAL
			e.command = ""
}

func controlH() {
		if len(sess.editors) == 1 {

			if sess.divider < 10 {
				sess.cfg.ed_pct = 80
				sess.moveDivider(80)
			}

			sess.editorMode = false //needs to be here

			sess.drawPreviewWindow(org.rows[org.fr].id)
			org.mode = NORMAL
			sess.returnCursor()
			return
		}

		temp := []*Editor{}
		for _, e := range sess.editors {
			if !e.is_subeditor {
				temp = append(temp, e)
			}
		}

		index := 0
		for i, e := range temp {
			if e == sess.p {
				index = i
				break
			}
		}

		sess.p.showMessage("index: %d; length: %d", index, len(temp))

		if index > 0 {
			sess.p = temp[index-1]
			err := v.SetCurrentBuffer(sess.p.vbuf)
			if err != nil {
				sess.p.showMessage("Problem setting current buffer")
			}
			sess.p.mode = NORMAL
			return
		} else {

			if sess.divider < 10 {
				sess.cfg.ed_pct = 80
				sess.moveDivider(80)
			}

			sess.editorMode = false //needs to be here

			sess.drawPreviewWindow(org.rows[org.fr].id)
			org.mode = NORMAL
			sess.returnCursor()
			return
		}
  }

func controlL() {

		temp := []*Editor{}
		for _, e := range sess.editors {
			if !e.is_below {
				temp = append(temp, e)
			}
		}

		index := 0
		for i, e := range temp {
			if e == sess.p {
				index = i
				break
			}
		}

		sess.p.showMessage("index: %d; length: %d", index, len(temp))

		if index < len(temp)-1 {
			sess.p = temp[index+1]
			sess.p.mode = NORMAL
			err := v.SetCurrentBuffer(sess.p.vbuf)
			if err != nil {
				sess.p.showMessage("Problem setting current buffer")
			}
		}

		return
  }

func (e *Editor) decorateWord(c int) {
  if len(e.rows) == 0 {
    return
  }

  row := e.rows[e.fr]
  if row[e.fc] == ' ' {
    return
  }

  //find beginning of word
  var beg int
  if e.fc != 0 {
    beg = strings.LastIndex(row[:e.fc], " ") //LastIndexAny and delimiters would be better
    if beg == -1 {
      beg = 0
    } else {
      beg++
    }
  }

  end := strings.Index(row[e.fc:], " ")
  if end == -1 {
    end = len(row) - 1
  } else {
    end = end + e.fc -1
  }

  var undo bool
  if strings.HasPrefix(row[beg:], "**") {
    row = row[:beg] + row[beg+2:]
    end -= 4
    row = row[:end+1] + row[end+3:]
    e.fc -=2
    if c == ctrlKey('b') {
      undo = true
    }
  } else if row[beg] == '*' {
    row = row[:beg] + row[beg+1:]
    end -= 2
    e.fc -= 1
    row = row[:end+1] + row[end+2:]
    if c == ctrlKey('i') {
      undo = true
    }
  } else if row[beg] == '`' {
    row = row[:beg] + row[beg+1:]
    end -= 2
    e.fc -= 1
    row = row[:end+1] + row[end+2:]
    if c == ctrlKey('e') {
      undo = true
    }
  }
  if undo {
    e.rows[e.fr] = row
	  v.SetBufferLines(e.vbuf, e.fr, e.fr+1, false, [][]byte{}) //true - out of bounds indexes are not clamped
	  v.SetBufferLines(e.vbuf, e.fr, e.fr, false, [][]byte{[]byte(row)}) //true - out of bounds indexes are not clamped
    v.SetWindowCursor(w, [2]int{e.fr+1, e.fc}) //set screen cx and cy from pos
    return
  }

  // needed if word at end of row ????
  if end == len(row) {
    row += " "
  }

  switch c {
    case ctrlKey('b'):
      row = row[:beg] + "**" + row[beg:end+1] + "**" + row[1+end:]
      e.fc +=2
    case ctrlKey('i'):
      row = row[:beg] + "*" + row[beg:end+1] + "*" + row[1+end:]
      e.fc++
    case ctrlKey('e'):
      row = row[:beg] + "`" + row[beg:end+1] + "`" + row[1+end:]
      e.fc++
  }

  e.rows[e.fr] = row
  //v.SetWindowCursor(w, [2]int{e.fr-1, e.fc}) //set screen cx and cy from pos
	v.SetBufferLines(e.vbuf, e.fr, e.fr+1, false, [][]byte{}) //true - out of bounds indexes are not clamped
	v.SetBufferLines(e.vbuf, e.fr, e.fr, false, [][]byte{[]byte(row)}) //true - out of bounds indexes are not clamped
  v.SetWindowCursor(w, [2]int{e.fr+1, e.fc}) //set screen cx and cy from pos
  //v.SetWindowCursor(w, [2]int{1, 4}) //set screen cx and cy from pos {r, c} r is 1-based, c - o-based
}

var e_lookup = map[string]func(*Editor, int) {
                   "i":(*Editor).E_i,
                   "I":(*Editor).E_I,
                   "a":(*Editor).E_a,
                   "A":(*Editor).E_A,
                   "o":(*Editor).E_o,
                   "O":(*Editor).E_O,
                   "x":(*Editor).E_x,
                   "dw":(*Editor).E_dw,
                   "daw":(*Editor).E_daw,
                   "dd":(*Editor).E_dd,
                   "d$":(*Editor).E_deol,
                   "de":(*Editor).E_de,
                   "dG":(*Editor).E_dG,
                   "cw":(*Editor).E_cw,
                   "caw":(*Editor).E_caw,
                   "s":(*Editor).E_s,
                 }

func (e *Editor) E_i(repeat int) {
  switch repeat {
  case -1:
  }
}
func (e *Editor) E_I(repeat int) {
  e.moveCursorBOL();
  e.fc = e.indentAmount(e.fr);
}

func (e *Editor) E_a(repeat int) {
  e.moveCursor(ARROW_RIGHT)
}

func (e *Editor) E_A(repeat int) {
  e.moveCursorEOL();
  e.moveCursor(ARROW_RIGHT); //works even though not in INSERT mode
}

func (e *Editor) E_o(repeat int) {
  e.last_typed = ""
  e.insertNewLine(1)
}

func (e *Editor) E_O(repeat int) {
  e.last_typed = ""
  e.insertNewLine(0)
}

func (e *Editor) E_x(repeat int) {
  r := &e.rows[e.fr]
  if len(*r) == 0 {
    return
  }
  *r = (*r)[:e.fc] + (*r)[e.fc+1:]
  for i := 1; i < repeat; i++ {
    if e.fc == len(*r) - 1 {
      e.fc--
      break;
    }
    *r = (*r)[:e.fc] + (*r)[e.fc+1:]
  }
  e.dirty++
}

func (e *Editor) E_dw(repeat int) {
  for i := 0; i < repeat; i++ {
    start := e.fc
    //e.moveEndWord2() uses this in cpp - need to revisit
    e.moveEndWord()
    end := e.fc
    e.fc = start
    r := &e.rows[e.fr]
    *r = (*r)[:e.fc] +(*r)[end+1:]
  }
}

/*
func (e *Editor) resize(flag byte) {
  if e.linked_editor == nil {
    return
  }

  le := e.linked_editor
  var subnote_height int
  if flag == '=' {
    subnote_height = sess.textLines/2
  } else {
    subnote_height = LINKED_NOTE_HEIGHT
  }
  if !e.is_subeditor {
    e.screenlines = sess.textLines - subnote_height - 1
    le.screenlines = subnote_height
    le.top_margin = sess.textLines - subnote_height + 2
  } else {
    le.screenlines = sess.textLines - subnote_height - 1
    e.screenlines = subnote_height
    e.top_margin = sess.textLines - subnote_height + 2
  }
  le.refreshScreen(true)
  e.refreshScreen(true)
}
*/

func (e *Editor) E_daw(repeat int) {
}

func (e *Editor) E_dd(repeat int) {
}

func (e *Editor) E_deol(repeat int) {
}

func (e *Editor) E_de(repeat int) {
}

func (e *Editor) E_dG(repeat int) {
}

func (e *Editor) E_cw(repeat int) {
}

func (e *Editor) E_caw(repeat int) {
}

func (e *Editor) E_s(repeat int) {
}
