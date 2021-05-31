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
	"\x0a":               (*Editor).controlJ,
	"\x0b":               (*Editor).controlK,
	"\x02":               (*Editor).decorateWord,
	leader + "b":         (*Editor).decorateWord,
	"\x05":               (*Editor).decorateWord,
	string(ctrlKey('i')): (*Editor).decorateWord,
	"\x17=":              (*Editor).changeSplit,
	"\x17_":              (*Editor).changeSplit,
	"\x06":               (*Editor).findMatchForBrace,
	leader + "l":         showVimMessageLog,
	leader + "m":         (*Editor).showMarkdown,
	leader + "s":         (*Editor).nextStyle,
}

// needs rewriting
func (e *Editor) changeSplit(flag int) {
	if e.output == nil {
		return
	}

	op := e.output
	var outputHeight int
	if flag == '=' {
		outputHeight = sess.textLines / 2
	} else if flag == '_' {
		outputHeight = LINKED_NOTE_HEIGHT
	} else {
		return
	}
	e.screenlines = sess.textLines - outputHeight - 1
	op.screenlines = outputHeight
	op.top_margin = sess.textLines - outputHeight + 2

	//op.drawText()
	//e.drawText()
	sess.eraseRightScreen()
	sess.drawRightScreen()
}

func (e *Editor) moveOutputWindowRight() {
	if e.output == nil { // && e.is_subeditor && e.is_below) {
		return
	}
	//top_margin = TOP_MARGIN + 1;
	//screenlines = total_screenlines - 1;
	e.output.is_below = false

	sess.positionWindows()
	sess.eraseRightScreen()
	sess.drawRightScreen()
	//editorSetMessage("top_margin = %d", top_margin);

}

func (e *Editor) moveOutputWindowBelow() {
	if e.output == nil { // && e.is_subeditor && e.is_below) {
		return
	}
	//top_margin = TOP_MARGIN + 1;
	//screenlines = total_screenlines - 1;
	e.output.is_below = true

	sess.positionWindows()
	sess.eraseRightScreen()
	sess.drawRightScreen()
	//editorSetMessage("top_margin = %d", top_margin);
}

// should scroll output down
func (e *Editor) controlJ() {
	op := e.output
	if op == nil {
		e.command = ""
		return
	}
	if op.rowOffset < len(op.rows)-1 {
		op.rowOffset++
		op.drawText()
	}
	e.command = ""
}

// should scroll output up
func (e *Editor) controlK() {
	if e.output == nil {
		e.command = ""
		return
	}
	if e.output.rowOffset > 0 {
		e.output.rowOffset--
	}
	e.output.drawText()
	e.command = ""
}

func controlH() {
	if p.isModified() {
		sess.showEdMessage("You need to save first")
		return
	}

	if sess.numberOfEditors() == 1 {

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

	eds := sess.editors()
	index := 0
	for i, e := range eds {
		if e == p {
			index = i
			break
		}
	}

	sess.showEdMessage("index: %d; length: %d", index, len(eds))

	if index > 0 {
		p = eds[index-1]
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
	if p.isModified() {
		sess.showEdMessage("You need to save first")
		return
	}

	eds := sess.editors()
	index := 0
	for i, e := range eds {
		if e == p {
			index = i
			break
		}
	}
	sess.showEdMessage("index: %d; length: %d", index, len(eds))

	if index < len(eds)-1 {
		p = eds[index+1]
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

func showLastVimMessage() {
	_ = v.SetCurrentBuffer(messageBuf)

	// don't have to erase message buffer before adding to it
	//_ = v.SetBufferLines(messageBuf, 0, -1, true, [][]byte{})
	_ = v.FeedKeys("\x1bG$\"apqaq", "t", false) //qaq ->record macro to register 'a'  followed by q (stop recording) clears register
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
		sess.showEdMessage("message: %v", message)
	} else {
		sess.showEdMessage("No message: len bb %v; Current Buf %v", len(bb), currentBuf)
	}
}

// for testing - displays vimMessageLog in preview windows
func showVimMessageLog() {

	// below don't want to erase log of messages
	//_ = v.SetBufferLines(messageBuf, 0, -1, true, [][]byte{})
	_ = v.SetCurrentBuffer(messageBuf)

	// \"ap pastes register a into messageBuf buffer
	// qaq is wierd way to erase a register (record macro to register then immediatly stop recording)
	_ = v.FeedKeys("\x1bG$\"apqaq", "t", false)
	v.SetCurrentBuffer(p.vbuf)

	// z needs some dimenstions like screenCols - takes from current editor
	z := *p
	z.vbuf = messageBuf
	z.bb, _ = v.BufferLines(messageBuf, 0, -1, true)

	p.renderedNote = z.generateWWStringFromBuffer2()
	p.mode = PREVIEW_MARKDOWN
	p.previewLineOffset = 0
	p.drawPreview()
}

func (e *Editor) showMarkdown() {
	if len(e.bb) == 0 {
		return
	}

	//note := readNoteIntoString(e.id)

	//note = generateWWString(note, e.screencols, -1, "\n")
	note := e.generateWWStringFromBuffer2()
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
