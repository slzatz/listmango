package main

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

var e_lookup2 = map[string]interface{}{
	"\x17L":              (*Editor).moveOutputWindowRight,
	"\x17J":              (*Editor).moveOutputWindowBelow,
	"\x08":               controlH,
	"\x0c":               controlL,
	"\x02":               (*Editor).decorateWord,
	leader + "b":         (*Editor).decorateWord,
	"\x05":               (*Editor).decorateWord,
	string(ctrlKey('i')): (*Editor).decorateWord,
	"\x17=":              (*Editor).changeSplit,
	"\x17_":              (*Editor).changeSplit,
	"\x06":               (*Editor).findMatchForBrace,
	leader + "+":         showVimMessage,
	leader + "m":         (*Editor).showMarkdown,
	leader + "s":         (*Editor).nextStyle,
}

func (e *Editor) changeSplit(flag int) {
	if e.linked_editor == nil {
		return
	}

	le := e.linked_editor
	var subnote_height int
	if flag == '=' {
		subnote_height = sess.textLines / 2
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
	le.refreshScreen()
	e.refreshScreen()
}

func (e *Editor) moveOutputWindowRight() {
	if e.linked_editor == nil { // && e.is_subeditor && e.is_below) {
		return
	}
	//top_margin = TOP_MARGIN + 1;
	//screenlines = total_screenlines - 1;
	e.linked_editor.is_below = false

	editor_slots := 0
	for _, e := range editors {
		if !e.is_below {
			editor_slots++
		}
	}

	s_cols := -1 + (sess.screenCols-sess.divider)/editor_slots
	i := -1 //i = number of columns of editors -1
	for _, e := range editors {
		if !e.is_below {
			i++
		}
		e.left_margin = sess.divider + i*s_cols + i
		e.screencols = s_cols
		e.setLinesMargins()
	}
	sess.eraseRightScreen()
	sess.drawEditors()
	//editorSetMessage("top_margin = %d", top_margin);

}

func (e *Editor) moveOutputWindowBelow() {
	if e.linked_editor == nil { // && e.is_subeditor && e.is_below) {
		return
	}
	//top_margin = TOP_MARGIN + 1;
	//screenlines = total_screenlines - 1;
	e.linked_editor.is_below = true

	editor_slots := 0
	for _, e := range editors {
		if !e.is_below {
			editor_slots++
		}
	}

	s_cols := -1 + (sess.screenCols-sess.divider)/editor_slots
	i := -1 //i = number of columns of editors -1
	for _, e := range editors {
		if !e.is_below {
			i++
		}
		e.left_margin = sess.divider + i*s_cols + i
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
	if len(editors) == 1 {

		if sess.divider < 10 {
			sess.cfg.ed_pct = 80
			moveDivider(80)
		}

		sess.editorMode = false //needs to be here

		org.drawPreview()
		org.mode = NORMAL
		sess.returnCursor()
		return
	}

	temp := []*Editor{}
	for _, e := range editors {
		if !e.is_subeditor {
			temp = append(temp, e)
		}
	}

	index := 0
	for i, e := range temp {
		if e == p {
			index = i
			break
		}
	}

	sess.showEdMessage("index: %d; length: %d", index, len(temp))

	if index > 0 {
		p = temp[index-1]
		err := v.SetCurrentBuffer(p.vbuf)
		if err != nil {
			sess.showEdMessage("Problem setting current buffer")
		}
		p.mode = NORMAL
		return
	} else {

		if sess.divider < 10 {
			sess.cfg.ed_pct = 80
			moveDivider(80)
		}

		sess.editorMode = false //needs to be here

		org.drawPreview()
		org.mode = NORMAL
		sess.returnCursor()
		return
	}
}

func controlL() {

	temp := []*Editor{}
	for _, e := range editors {
		if !e.is_below {
			temp = append(temp, e)
		}
	}

	index := 0
	for i, e := range temp {
		if e == p {
			index = i
			break
		}
	}

	sess.showEdMessage("index: %d; length: %d", index, len(temp))

	if index < len(temp)-1 {
		p = temp[index+1]
		p.mode = NORMAL
		err := v.SetCurrentBuffer(p.vbuf)
		if err != nil {
			sess.showEdMessage("Problem setting current buffer")
		}
	}

	return
}

func (e *Editor) decorateWord(c int) {
	if len(e.bb) == 0 {
		return
	}

	// here probably easier to convert to string
	row := string(e.bb[e.fr])
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
		end = end + e.fc - 1
	}

	var undo bool
	if strings.HasPrefix(row[beg:], "**") {
		row = row[:beg] + row[beg+2:]
		end -= 4
		row = row[:end+1] + row[end+3:]
		e.fc -= 2
		if c == ctrlKey('b') || c == 'b' {
			undo = true
		}
	} else if row[beg] == '*' {
		row = row[:beg] + row[beg+1:]
		end -= 2
		e.fc -= 1
		row = row[:end+1] + row[end+2:]
		if c == ctrlKey('i') || c == 'i' {
			undo = true
		}
	} else if row[beg] == '`' {
		row = row[:beg] + row[beg+1:]
		end -= 2
		e.fc -= 1
		row = row[:end+1] + row[end+2:]
		if c == ctrlKey('e') || c == 'e' {
			undo = true
		}
	}
	if undo {
		v.SetBufferLines(e.vbuf, e.fr, e.fr+1, false, [][]byte{})          //true - out of bounds indexes are not clamped
		v.SetBufferLines(e.vbuf, e.fr, e.fr, false, [][]byte{[]byte(row)}) //true - out of bounds indexes are not clamped
		v.SetWindowCursor(w, [2]int{e.fr + 1, e.fc})                       //set screen cx and cy from pos
		return
	}

	// needed if word at end of row ????
	if end == len(row) {
		row += " "
	}

	switch c {
	case ctrlKey('b'), 'b':
		row = row[:beg] + "**" + row[beg:end+1] + "**" + row[1+end:]
		e.fc += 2
	case ctrlKey('i'), 'i':
		row = row[:beg] + "*" + row[beg:end+1] + "*" + row[1+end:]
		e.fc++
	case ctrlKey('e'), 'e':
		row = row[:beg] + "`" + row[beg:end+1] + "`" + row[1+end:]
		e.fc++
	}

	v.SetBufferLines(e.vbuf, e.fr, e.fr+1, false, [][]byte{})          //true - out of bounds indexes are not clamped
	v.SetBufferLines(e.vbuf, e.fr, e.fr, false, [][]byte{[]byte(row)}) //true - out of bounds indexes are not clamped
	v.SetWindowCursor(w, [2]int{e.fr + 1, e.fc})                       //set screen cx and cy from pos
}

func showVimMessage() {
	_ = v.SetCurrentBuffer(messageBuf)
	_ = v.SetBufferLines(messageBuf, 0, -1, true, [][]byte{})
	_ = v.FeedKeys("\x1b\"apqaq", "t", false)
	bb, _ := v.BufferLines(messageBuf, 0, -1, true)
	var message string
	var i int
	for i = len(bb) - 1; i >= 0; i-- {
		message = string(bb[i])
		if message != "" {
			break
		}
	}
	v.SetCurrentBuffer(p.vbuf)
	currentBuf, _ := v.CurrentBuffer()
	if message != "" {
		sess.showEdMessage("len bb: %v; i: %v; message: %v", len(bb), i, message)
	} else {
		sess.showEdMessage("No message: len bb %v; Current Buf %v", len(bb), currentBuf)
	}
}

func (e *Editor) showMarkdown() {
	if len(e.bb) == 0 {
		return
	}

	note := readNoteIntoString(e.id)

	note = generateWWString(note, e.screencols, 500, "\n")
	r, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("/home/slzatz/listmango/darkslz.json"),
		glamour.WithWordWrap(0),
	)
	note, _ = r.Render(note)

	// for some` reason get extra line at top
	ix := strings.Index(note, "\n") //works for ix = -1
	e.renderedNote = note[ix+1:]

	e.mode = PREVIEW_MARKDOWN
	e.previewLineOffset = 0
	e.drawPreview()

}

func (e *Editor) nextStyle() {
	sess.styleIndex++
	if sess.styleIndex > len(sess.style)-1 {
		sess.styleIndex = 0
	}
	sess.showEdMessage("New style is %q", sess.style[sess.styleIndex])
}
