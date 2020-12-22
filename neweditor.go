package main

import (
	"fmt"

	"github.com/slzatz/listmango/runes"
)

const (
	insert int = iota
	normal
	comandLine
)

const (
	changeRow int = iota
	replaceNote
	addRows
	deleteRows
)

type editor struct {
	mode             int
	rows             [][]rune
	cx, cy           int // Cx and Cy represent current cursor position
	fc, fr           int
	fcPrev, frPrev   int
	lineOffset       int //row the user is currently scrolled to
	prevLineOffset   int
	colOff           int //column user is currently scrolled to
	screenLines      int //number of lines for this Editor
	screenCols       int //number of columns for this Editor
	leftMargin       int //can vary (so could TOP_MARGIN - will do that later
	leftMarginOffset int // 0 if no line numbers
	topMargin        int
	code             string
	dirty            bool
	commandLine      string //for commands on the command line; string doesn't include ':'
	command          string // right now includes normal mode commands and command line commands
	lastCommand      string
	repeat           int
	repeatPrev       int
	insertedText     []rune //what is typed between going into insert and leaving insert
	indent           int
	smartIndent      int
	firstVisibleRow  int
	lastVisibleRow   int
	spellcheck       bool
	highlightSyntax  bool
	redraw           bool
}

type undoDiff struct {
	fc, fr       int
	repeat       int
	command      string // right now includes normal mode commands and command line commands
	rows         [][]rune
	numRows      int //the row where insert occurs counts 1 and then any rows added with returns
	insertedText []rune
	undoMethod   int
	//std::string deleted_text; //deleted chars - being recorded by not used right now or perhaps ever!
	//std::vector<std::pair<char, int>> diff; //c = changed; d = deleted; a = added
	//std::vector<std::pair<int, std::string>> changed_rows;
	//int undo_method; //CHANGE_ROW< REPLACE_NOTE< ADD_ROWS, DELETE_ROWS
	//int mode;
}

// ERow represents a line of text in a file

// Text expands tabs in an eRow to spaces
func (row []rune) Text() []rune {
	dest := []rune{}
	for _, r := range row {
		switch r {
		case '\t':
			dest = append(dest, tabSpaces...)
		default:
			dest = append(dest, r)
		}
	}
	return dest
}

// CxToRx transforms cursor positions to account for tab stops
func (row []rune) CxToRx(cx int) int {
	rx := 0
	for j := 0; j < cx; j++ {
		if row[j] == '\t' {
			rx = (rx + kiloTabStop - 1) - (rx % kiloTabStop)
		}
		rx++
	}
	return rx
}

// NewEditor returns a new blank editor
func newEditor() *editor {
	return &editor{
		dirty: false,
		//Cursor: Point{0, 0},
		cx:   0,
		cy:   0,
		fc:   0,
		fr:   0,
		rows: [][]rune{},
	}
}

// NewEditorFromFile creates an editor from a file system file
func NewEditorFromFile(filename string) (*Editor, error) {

	rows := []ERow{}

	if filename != "" {
		var err error
		if rows, err = Open(filename); err != nil {
			return nil, fmt.Errorf("Error opening file %s: %v", filename, err)
		}
	}

	return &Editor{
		FileName: filename,
		Dirty:    false,
		Cursor:   Point{0, 0},
		Rows:     rows,
	}, nil
}

// CursorUp moves the cursor up as long as row number is non zero
func (e *Editor) CursorUp() {
	if e.Cursor.Row > 0 {
		e.Cursor.Row--
	}
	e.ResetX()
}

// CursorDown moves the cursor down till one line past the last line
func (e *Editor) CursorDown() {
	if e.Cursor.Row < len(e.Rows) {
		e.Cursor.Row++
	}
	e.ResetX()
}

// CursorLeft moves the cursor left. If at col 0 & any line other thant
// the first line, it moves to the previous line
func (e *Editor) CursorLeft() {

	if e.Cursor.Col > 0 {
		e.Cursor.Col--
	} else if e.Cursor.Row > 0 {
		e.Cursor.Row--
		e.Cursor.Col = len(e.Rows[e.Cursor.Row])
	}
}

// CursorRight moves the cursor right & wraps past EOL to col 0
func (e *Editor) CursorRight() {
	// right moves only if we're within a valid line.
	// for past EOF, there's no movement
	if e.Cursor.Row >= len(e.Rows) {
		return
	}
	if e.Cursor.Col < len(e.Rows[e.Cursor.Row]) {
		e.Cursor.Col++
	} else if e.Cursor.Col == len(e.Rows[e.Cursor.Row]) {
		e.Cursor.Row++
		e.Cursor.Col = 0
	}
}

// CursorEnd moves the cursor to end of line
func (e *Editor) CursorEnd() {
	if e.Cursor.Row < len(e.Rows) {
		e.Cursor.Col = len(e.Rows[e.Cursor.Row])
	}
}

// CursorPageUp moves the cursor one screen up
func (e *Editor) CursorPageUp(screenRows int, rowOffset int) {
	e.Cursor.Row = rowOffset
	for j := 0; j < screenRows; j++ {
		e.CursorUp()
	}
}

// CursorPageDown moves the cursor one screen down
func (e *Editor) CursorPageDown(screenRows int, rowOffset int) {
	e.Cursor.Row = rowOffset + screenRows - 1
	if e.Cursor.Row > len(e.Rows) {
		e.Cursor.Row = len(e.Rows)
	}
	for j := 0; j < screenRows; j++ {
		e.CursorDown()
	}
}

// CursorHome moves the cursor to col 0
func (e *Editor) CursorHome() {
	e.Cursor.Col = 0
}

// ResetX sets the cursor X position to a valid position after moving y
func (e *Editor) ResetX() {

	// if we moved past last row, set cursor to 0
	if e.Cursor.Row >= len(e.Rows) {
		e.Cursor.Col = 0
		return
	}

	// we allow moving to 1 pos past the last character
	rowLen := len(e.Rows[e.Cursor.Row])
	if e.Cursor.Col > rowLen {
		e.Cursor.Col = len(e.Rows[e.Cursor.Row])
	}
}

// InsertChar inserts a character at a given location
func (e *Editor) InsertChar(c rune) {

	// if we're at the last line, insert a new row
	if e.Cursor.Row == len(e.Rows) {
		e.InsertRow(len(e.Rows), "")
	}

	// store a reference to the working row to improve readability
	src := e.Rows[e.Cursor.Row]

	dest := make([]rune, len(src)+1)
	copy(dest, src[:e.Cursor.Col])
	copy(dest[e.Cursor.Col+1:], src[e.Cursor.Col:])
	dest[e.Cursor.Col] = c

	e.Rows[e.Cursor.Row] = dest
	e.Dirty = true
	e.Cursor.Col++
}

// InsertRow inserts a row at a given index
func (e *Editor) InsertRow(rowidx int, s string) {
	if rowidx < 0 || rowidx > len(e.Rows) {
		return
	}

	row := []rune(s)

	e.Rows = append(e.Rows, ERow{})
	copy(e.Rows[rowidx+1:], e.Rows[rowidx:])
	e.Rows[rowidx] = row

	e.Dirty = true
}

// InsertNewline inserts a new line at the cursor position
func (e *Editor) InsertNewline() {
	if e.Cursor.Col == 0 {
		e.InsertRow(e.Cursor.Row, "")
		e.Dirty = true
		return
	}

	moveChars := string(e.Rows[e.Cursor.Row][e.Cursor.Col:])
	e.Rows[e.Cursor.Row] = e.Rows[e.Cursor.Row][:e.Cursor.Col]
	e.InsertRow(e.Cursor.Row+1, moveChars)
	e.Dirty = true

	e.Cursor.Row++
	e.Cursor.Col = 0
}

// DelChar deletes a character at current cursor location
func (e *Editor) DelChar() {
	// if the cursor is at the line beyond the end of text
	// then move it to the last line
	if e.Cursor.Row == len(e.Rows) {
		if len(e.Rows) == 0 {
			return
		}
		e.Cursor.Row = len(e.Rows) - 1
		e.Cursor.Col = len(e.Rows[e.Cursor.Row])
	}

	// if at the beginning of the text, then do nothing
	if e.Cursor.Col == 0 && e.Cursor.Row == 0 {
		return
	}

	// different handling for at the beginning of the line or middle of line
	if e.Cursor.Col > 0 {
		row := e.Rows[e.Cursor.Row]
		copy(row[e.Cursor.Col-1:], row[e.Cursor.Col:])
		row = row[:len(row)-1]
		e.Rows[e.Cursor.Row] = row
		e.Cursor.Col--
	} else {
		e.Cursor.Col = len(e.Rows[e.Cursor.Row-1])
		e.Rows[e.Cursor.Row-1] = append(e.Rows[e.Cursor.Row-1], e.Rows[e.Cursor.Row]...)
		e.DelRow(e.Cursor.Row)
		e.Cursor.Row--
	}
	e.Dirty = true
}

// DelRow deletes a given row
func (e *Editor) DelRow(rowidx int) {
	if rowidx < 0 || rowidx >= len(e.Rows) {
		return
	}

	copy(e.Rows[rowidx:], e.Rows[rowidx+1:])
	e.Rows = e.Rows[:len(e.Rows)-1]
	e.Dirty = true
}

// Find searches the whole editor for a text
func (e *Editor) Find(search ERow) []Point {
	ret := []Point{}

	for i := 0; i < len(e.Rows); i++ {
		if idx := runes.Index(runes.ToLower(e.Rows[i]), runes.ToLower(search)); idx != -1 {
			ret = append(ret, Point{Col: idx, Row: i})
		}
	}
	return ret
}

// SetCursor sets the cursor to a specific point
func (e *Editor) SetCursor(p Point) {
	e.Cursor.Col = p.Col
	e.Cursor.Row = p.Row
}
