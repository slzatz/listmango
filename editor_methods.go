package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"os"
	"strings"
)

func find_first_not_of(row *string, delimiters string, pos int) int {
	pos++
	for i, char := range (*row)[pos:] {
		if strings.Index(delimiters, string(char)) != -1 {
			continue
		} else {
			return pos + i
		}
	}
	return -1
}

//'automatically' happens in NORMAL and INSERT mode
//return true -> redraw; false -> don't redraw
func (e *Editor) findMatchForLeftBrace(leftBrace byte, back bool) bool {
	r := e.fr
	c := e.fc + 1
	count := 1
	max := len(e.bb)
	var b int
	if back {
		b = 1
	}

	m := map[byte]byte{'{': '}', '(': ')', '[': ']'}
	rightBrace := m[leftBrace]

	for {

		row := e.bb[r]

		// need >= because brace could be at end of line and in INSERT mode
		// fc could be row.size() [ie beyond the last char in the line
		// and so doing fc + 1 above leads to c > row.size()
		if c >= len(row) {
			r++
			if r == max {
				sess.showEdMessage("Couldn't find matching brace")
				return false
			}
			c = 0
			continue
		}

		if row[c] == rightBrace {
			count -= 1
			if count == 0 {
				break
			}
		} else if row[c] == leftBrace {
			count += 1
		}

		c++
	}
	y := e.getScreenYFromRowColWW(r, c) - e.lineOffset
	if y >= e.screenlines {
		return false
	}

	x := e.getScreenXFromRowColWW(r, c) + e.left_margin + e.left_margin_offset + 1
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s", y+e.top_margin, x, string(rightBrace))

	x = e.getScreenXFromRowColWW(e.fr, e.fc-b) + e.left_margin + e.left_margin_offset + 1
	y = e.getScreenYFromRowColWW(e.fr, e.fc-b) + e.top_margin - e.lineOffset // added line offset 12-25-2019
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s\x1b[0m", y, x, string(leftBrace))
	sess.showEdMessage("r = %d   c = %d", r, c)
	return true
}

//leaving this for the time being as another way to highlight braces
//would be tricky to do in INSERT mode since would have to leave
//INSERT to use % - but interesting
func (e *Editor) findMatchForBrace() bool {

	v.FeedKeys("%", "t", true)

	pos, _ := v.WindowCursor(w) //set screen cx and cy from pos
	r := pos[0] - 1
	c := pos[1]

	//m := map[byte]byte{'{': '}', '(': ')', '[': ']'}
	//right_brace := m[left_brace]

	//e.showMessage("left brace: {}", left_brace);
	y := e.getScreenYFromRowColWW(r, c) - e.lineOffset
	if y >= e.screenlines {
		return false
	}

	x := e.getScreenXFromRowColWW(r, c) + e.left_margin + e.left_margin_offset + 1
	//fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s", y+e.top_margin, x, string(right_brace))
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s", y+e.top_margin, x, "]")

	//x = e.getScreenXFromRowColWW(e.fr, e.fc-b) + e.left_margin + e.left_margin_offset + 1
	x = e.getScreenXFromRowColWW(e.fr, e.fc) + e.left_margin + e.left_margin_offset + 1
	//y = e.getScreenYFromRowColWW(e.fr, e.fc-b) + e.top_margin - e.line_offset // added line offset 12-25-2019
	y = e.getScreenYFromRowColWW(e.fr, e.fc) + e.top_margin - e.lineOffset // added line offset 12-25-2019
	//fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s\x1b[0m", y, x, string(left_brace))
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s\x1b[0m", y, x, "[")
	v.SetWindowCursor(w, [2]int{e.fr + 1, e.fc}) //set screen cx and cy from pos
	sess.showEdMessage("r = %d   c = %d; e.fr = %d e.fc = %d", r, c, e.fr, e.fc)
	return true
}

//'automatically' happens in NORMAL and INSERT mode
func (e *Editor) findMatchForRightBrace(rightBrace byte, back bool) bool {
	var b int
	if back {
		b = 1
	}
	r := e.fr
	c := e.fc - 1 - b
	count := 1

	row := e.bb[r]

	m := map[byte]byte{'}': '{', ')': '(', ']': '['}
	leftBrace := m[rightBrace]

	for {

		if c == -1 { //fc + 1 can be greater than row.size on first pass from INSERT if { at end of line
			r--
			if r == -1 {
				sess.showEdMessage("Couldn't find matching brace")
				return false
			}
			row = e.bb[r]
			c = len(row) - 1
			continue
		}

		if row[c] == leftBrace {
			count -= 1
			if count == 0 {
				break
			}
		} else if row[c] == rightBrace {
			count += 1
		}

		c--
	}

	y := e.getScreenYFromRowColWW(r, c) - e.lineOffset
	if y < 0 {
		return false
	}

	x := e.getScreenXFromRowColWW(r, c) + e.left_margin + e.left_margin_offset + 1
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s", y+e.top_margin, x, string(leftBrace))

	x = e.getScreenXFromRowColWW(e.fr, e.fc-b) + e.left_margin + e.left_margin_offset + 1
	y = e.getScreenYFromRowColWW(e.fr, e.fc-b) + e.top_margin - e.lineOffset // added line offset 12-25-2019
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s\x1b[0m", y, x, string(rightBrace))
	sess.showEdMessage("r = %d   c = %d", r, c)
	return true
}

func (e *Editor) drawHighlightedBraces() {

	// this guard is necessary
	if len(e.bb) == 0 || len(e.bb[e.fr]) == 0 {
		return
	}

	braces := "{}()" //? intentionally exclusing [] from auto drawing
	var c byte
	var back bool
	//if below handles case when in insert mode and brace is last char
	//in a row and cursor is beyond that last char (which is a brace)
	if e.fc == len(e.bb[e.fr]) {
		c = e.bb[e.fr][e.fc-1]
		back = true
	} else {
		c = e.bb[e.fr][e.fc]
		back = false
	}
	pos := strings.Index(braces, string(c))
	if pos != -1 {
		switch c {
		case '{', '(':
			e.findMatchForLeftBrace(c, back)
			return
		case '}', ')':
			e.findMatchForRightBrace(c, back)
			return
		//case '(':
		default: //should not need this
			return
		}
	} else if e.fc > 0 && e.mode == INSERT {
		c := e.bb[e.fr][e.fc-1]
		pos := strings.Index(braces, string(c))
		if pos != -1 {
			switch e.bb[e.fr][e.fc-1] {
			case '{', '(':
				e.findMatchForLeftBrace(c, true)
				return
			case '}', ')':
				e.findMatchForRightBrace(c, true)
				return
			//case '(':
			default: //should not need this
				return
			}
		}
	}
}

func (e *Editor) setLinesMargins() { //also sets top margin

	if e.output != nil {
		if e.output.is_below {
			e.screenlines = sess.textLines - LINKED_NOTE_HEIGHT - 1
			e.top_margin = TOP_MARGIN + 1
		} else {
			e.screenlines = sess.textLines
			e.top_margin = TOP_MARGIN + 1
		}
	} else {
		e.screenlines = sess.textLines
		e.top_margin = TOP_MARGIN + 1
	}
}

// used by updateNote
func (e *Editor) bufferToString() string {

	numRows := len(e.bb)
	if numRows == 0 {
		return ""
	}

	var sb strings.Builder
	for i := 0; i < numRows-1; i++ {
		sb.Write(e.bb[i])
		sb.Write([]byte("\n"))
	}
	sb.Write(e.bb[numRows-1])
	return sb.String()
}

func (e *Editor) getScreenXFromRowColWW(r, c int) int {
	row := e.bb[r]

	width := e.screencols - e.left_margin_offset

	if width >= len(row) {
		return c
	}

	pos := 0
	prev_pos := 0
	for {

		if width >= len(row[prev_pos:]) {
			break
		}

		pos = bytes.LastIndex(row[prev_pos:pos+width], []byte(" "))

		if pos == -1 {
			pos = prev_pos + width - 1
		} else {
			pos = pos + prev_pos
		}

		if pos >= c {
			break
		}
		prev_pos = pos + 1
	}
	return c - prev_pos
}

func (e *Editor) getScreenYFromRowColWW(r, c int) int {
	screenLine := 0

	for n := 0; n < r; n++ {
		screenLine += e.getLinesInRowWW(n)
	}

	screenLine = screenLine + e.getLineInRowWW(r, c) - 1
	return screenLine
}

func (e *Editor) getLinesInRowWW_old(r int) int {
	//row := e.rows[r]
	row := e.bb[r]

	if len(row) <= e.screencols-e.left_margin_offset {
		return 1
	}

	lines := 0
	pos := -1 //pos is the position of the last character in the line (zero-based)
	prev_pos := 0

	for {

		// we know the first time around this can't be true
		// could add if (line > 1 && row.substr(pos+1).size() ...);
		if len(row[pos+1:]) <= e.screencols-e.left_margin_offset {
			lines++
			break
		}

		prev_pos = pos
		//cpp find_last_of -the parameter defines the position from beginning to look at (inclusive)
		//need to add + 1 because slice :n includes chars up to the n - 1 char
		pos = bytes.LastIndex(row[:pos+e.screencols-e.left_margin_offset+1], []byte(" "))

		if pos == -1 {
			pos = prev_pos + e.screencols - e.left_margin_offset
		} else if pos == prev_pos {
			row = row[pos+1:]
			pos = e.screencols - e.left_margin_offset - 1
		}
		lines++
	}
	return lines
}

func (e *Editor) getLinesInRowWW(r int) int {
	row := e.bb[r]

	width := e.screencols - e.left_margin_offset

	if width >= len(row) {
		return 1
	}

	lines := 0
	pos := 0
	prev_pos := 0

	for {

		if width >= len(row[prev_pos:]) {
			lines++
			break
		}

		pos = bytes.LastIndex(row[prev_pos:pos+width], []byte(" "))

		if pos == -1 {
			pos = prev_pos + width - 1
		} else {
			pos = pos + prev_pos
		}

		lines++

		prev_pos = pos + 1
	}
	return lines
}

func (e *Editor) getLineInRowWW(r, c int) int {
	row := e.bb[r]

	width := e.screencols - e.left_margin_offset

	if width >= len(row) {
		return 1
	}

	lines := 0
	pos := 0
	prev_pos := 0

	for {

		if width >= len(row[prev_pos:]) {
			lines++
			break
		}

		pos = bytes.LastIndex(row[prev_pos:pos+width], []byte(" "))

		if pos == -1 {
			pos = prev_pos + width - 1
		} else {
			pos = pos + prev_pos
		}

		lines++

		if pos >= c {
			break
		}
		prev_pos = pos + 1
	}
	return lines
}

func (e *Editor) drawText() {
	var ab strings.Builder

	fmt.Fprintf(&ab, "\x1b[?25l\x1b[%d;%dH", e.top_margin, e.left_margin+1)
	// \x1b[NC moves cursor forward by n columns
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)
	erase_chars := fmt.Sprintf("\x1b[%dX", e.screencols)
	for i := 0; i < e.screenlines; i++ {
		ab.WriteString(erase_chars)
		ab.WriteString(lf_ret)
	}
	if e.highlightSyntax {
		e.drawCodeRows(&ab)
		fmt.Print(ab.String())
		//go e.drawHighlightedBraces() // this will produce data race
		e.drawHighlightedBraces() //has to come after drawing rows
	} else {
		//e.drawBuffer(&ab)
		e.drawPlainRows(&ab)
		fmt.Print(ab.String())
	}
	//e.drawStatusBar()
}

func (e *Editor) drawVisual(pab *strings.Builder) {

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin+e.left_margin_offset)

	if e.mode == VISUAL_LINE {
		startRow := e.vb_highlight[0][1] - 1 // i think better to subtract one here
		endRow := e.vb_highlight[1][1] - 1   //ditto - done differently for visual and v_block

		x := e.left_margin + e.left_margin_offset + 1
		y := e.getScreenYFromRowColWW(startRow, 0) - e.lineOffset

		if y >= 0 {
			fmt.Fprintf(pab, "\x1b[%d;%dH\x1b[48;5;244m", y+e.top_margin, x)
		} else {
			fmt.Fprintf(pab, "\x1b[%d;%dH\x1b[48;5;244m", e.top_margin, x)
		}

		for n := 0; n < (endRow - startRow + 1); n++ { //++n
			rowNum := startRow + n
			pos := 0
			for line := 1; line <= e.getLinesInRowWW(rowNum); line++ { //++line
				if y < 0 {
					y += 1
					continue
				}
				if y == e.screenlines {
					break //out for should be done (theoretically) - 1
				}
				line_char_count := e.getLineCharCountWW(rowNum, line)
				pab.Write(e.bb[rowNum][pos : pos+line_char_count])
				pab.WriteString(lf_ret)
				y += 1
				pos += line_char_count
			}
		}
	}

	if e.mode == VISUAL {
		startcol, endcol := e.vb_highlight[0][2], e.vb_highlight[1][2]

		// startRow always <= endRow and need to subtract 1 since counting starts at 1 not zero
		startRow, endRow := e.vb_highlight[0][1]-1, e.vb_highlight[1][1]-1 //startRow always <= endRow
		numrows := endRow - startRow + 1

		x := e.getScreenXFromRowColWW(startRow, startcol) + e.left_margin + e.left_margin_offset
		y := e.getScreenYFromRowColWW(startRow, startcol) + e.top_margin - e.lineOffset // - 1

		pab.WriteString("\x1b[48;5;244m")
		for n := 0; n < numrows; n++ {
			// i think would check here to see if a row has multiple lines (ie wraps)
			if n == 0 {
				fmt.Fprintf(pab, "\x1b[%d;%dH", y+n, x)
			} else {
				fmt.Fprintf(pab, "\x1b[%d;%dH", y+n, 1+e.left_margin+e.left_margin_offset)
			}
			row := e.bb[startRow+n]

			if len(row) == 0 {
				continue
			}
			if numrows == 1 {
				pab.Write(row[startcol-1 : endcol])
			} else if n == 0 {
				pab.Write(row[startcol-1:])
			} else if n < numrows-1 {
				pab.Write(row)
			} else {
				if len(row) < endcol {
					pab.Write(row)
				} else {
					pab.Write(row[:endcol])
				}
			}
			//(*pab).writestring(row[startcol-1:])
			//sess.showedmessage("%v; %v; %v; %v", startcol, endcol, startrow, endRow)
		}
	}

	if e.mode == VISUAL_BLOCK {

		var left, right int
		if e.vb_highlight[1][2] > e.vb_highlight[0][2] {
			right, left = e.vb_highlight[1][2], e.vb_highlight[0][2]
		} else {
			left, right = e.vb_highlight[1][2], e.vb_highlight[0][2]
		}

		x := e.getScreenXFromRowColWW(e.vb_highlight[0][1], left) + e.left_margin + e.left_margin_offset
		y := e.getScreenYFromRowColWW(e.vb_highlight[0][1], left) + e.top_margin - e.lineOffset - 1

		pab.WriteString("\x1b[48;5;244m")
		for n := 0; n < (e.vb_highlight[1][1] - e.vb_highlight[0][1] + 1); n++ {
			fmt.Fprintf(pab, "\x1b[%d;%dH", y+n, x)
			row := e.bb[e.vb_highlight[0][1]+n-1]
			rowLen := len(row)

			if rowLen == 0 || rowLen < left {
				continue
			}

			if rowLen < right {
				pab.Write(row[left-1 : rowLen])
			} else {
				pab.Write(row[left-1 : right])
			}
		}
	}

	pab.WriteString(RESET)
}

func (e *Editor) getLineCharCountWW(r, line int) int {
	row := e.bb[r]

	width := e.screencols - e.left_margin_offset

	if width >= len(row) {
		return len(row)
	}

	lines := 0
	pos := 0
	prev_pos := 0

	for {

		if width >= len(row[prev_pos:]) {
			return len(row[prev_pos:])
		}

		pos = bytes.LastIndex(row[prev_pos:pos+width], []byte(" "))

		if pos == -1 {
			pos = prev_pos + width - 1
		} else {
			pos = pos + prev_pos
		}

		lines++

		if lines == line {
			break
		}

		prev_pos = pos + 1
	}

	return pos - prev_pos + 1
}

func (e *Editor) getLineCharCountWW_old(r, line int) int {
	row := e.bb[r]

	if len(row) == 0 {
		return 0
	}

	if len(row) <= e.screencols-e.left_margin_offset {
		return len(row)
	}

	lines := 0
	pos := -1
	prev_pos := 0
	for {

		// we know the first time around this can't be true
		// could add if (line > 1 && row.substr(pos+1).size() ...);
		if len(row[pos+1:]) <= e.screencols-e.left_margin_offset {
			return len(row[pos+1:])
		}

		prev_pos = pos
		pos = bytes.LastIndex(row[:pos+e.screencols-e.left_margin_offset], []byte(" "))

		if pos == -1 {
			pos = prev_pos + e.screencols - e.left_margin_offset

			// only replace if you have enough characters without a space to trigger this
			// need to start at the beginning each time you hit this
			// unless you want to save the position which doesn't seem worth it
		} else if pos == prev_pos {
			row = bytes.ReplaceAll(row[:pos+1], []byte(" "), []byte("+")) // + row[pos+1:]
			row = append(row, row[pos+1:]...)
			pos = prev_pos + e.screencols - e.left_margin_offset
		}

		lines++
		if lines == line {
			break
		}
	}
	return pos - prev_pos
}
func (e *Editor) drawPlainRows(pab *strings.Builder) {
	note := e.generateWWStringFromBuffer() // need the \t for line num to be correct
	nnote := strings.Split(note, "\n")

	// for speed only looking at current row
	result := make(chan string)
	if e.checkSpelling {
		go highlightMispelledWords3(nnote[e.fr], result)
	}
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)
	fmt.Fprintf(pab, "\x1b[?25l\x1b[%d;%dH", e.top_margin, e.left_margin+1) //+1

	var s string
	if e.numberLines {
		var numCols strings.Builder
		// below draws the line number 'rectangle'
		// can be drawm to pab or &numCols
		fmt.Fprintf(&numCols, "\x1b[2*x\x1b[%d;%d;%d;%d;48;5;235$r\x1b[*x",
			e.top_margin,
			e.left_margin,
			e.top_margin+e.screenlines,
			e.left_margin+e.left_margin_offset)
		fmt.Fprintf(&numCols, "\x1b[?25l\x1b[%d;%dH", e.top_margin, e.left_margin+1)

		s = fmt.Sprintf("\x1b[%dC", e.left_margin_offset) + "%s" + lf_ret
		for n := e.first_visible_row; n < len(nnote); n++ {
			row := nnote[n]
			fmt.Fprintf(&numCols, "\x1b[48;5;235m\x1b[38;5;245m%3d \x1b[49m", n+1)
			line := strings.Split(row, "\t")
			for i := 0; i < len(line); i++ {
				fmt.Fprintf(pab, s, line[i])
				numCols.WriteString(lf_ret)
			}
		}
		pab.WriteString(numCols.String())
	} else {
		s = "%s" + lf_ret
		for n := e.first_visible_row; n < len(nnote); n++ {
			row := nnote[n]
			line := strings.Split(row, "\t")
			for i := 0; i < len(line); i++ {
				fmt.Fprintf(pab, s, line[i])
			}
		}
	}
	if e.checkSpelling {
		y := e.getScreenYFromRowColWW(e.fr, 0) + e.top_margin - e.lineOffset // - 1
		fmt.Fprintf(pab, "\x1b[%d;%dH\x1b[0m", y, e.left_margin+1)           //+1
		row := <-result
		line := strings.Split(row, "\t")
		for i := 0; i < len(line); i++ {
			fmt.Fprintf(pab, s, line[i])
		}
	}
	e.drawVisual(pab)
}

func (e *Editor) drawCodeRows(pab *strings.Builder) {
	tid := getFolderTid(e.id)
	note := e.generateWWStringFromBuffer()
	var lang string
	var buf bytes.Buffer
	switch tid {
	case 18:
		lang = "cpp"
	case 14:
		lang = "go"
	default:
		lang = "markdown"
	}
	_ = Highlight(&buf, note, lang, "terminal16m", sess.style[sess.styleIndex])
	note = buf.String()
	nnote := strings.Split(note, "\n")
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)
	fmt.Fprintf(pab, "\x1b[?25l\x1b[%d;%dH", e.top_margin, e.left_margin+1)

	if e.numberLines {
		var numCols strings.Builder
		// below draws the line number 'rectangle'
		// cam be drawm to pab or &numCols
		fmt.Fprintf(&numCols, "\x1b[2*x\x1b[%d;%d;%d;%d;48;5;235$r\x1b[*x",
			e.top_margin,
			e.left_margin,
			e.top_margin+e.screenlines,
			e.left_margin+e.left_margin_offset)
		fmt.Fprintf(&numCols, "\x1b[?25l\x1b[%d;%dH", e.top_margin, e.left_margin+1)

		s := fmt.Sprintf("\x1b[%dC", e.left_margin_offset) + "%s" + lf_ret
		for n := e.first_visible_row; n < len(nnote); n++ {
			row := nnote[n]
			fmt.Fprintf(&numCols, "\x1b[48;5;235m\x1b[38;5;245m%3d \x1b[49m", n+1)
			line := strings.Split(row, "\t")
			for i := 0; i < len(line); i++ {
				fmt.Fprintf(pab, s, line[i])
				numCols.WriteString(lf_ret)
			}
		}
		pab.WriteString(numCols.String())
	} else {
		s := "%s" + lf_ret
		for n := e.first_visible_row; n < len(nnote); n++ {
			row := nnote[n]
			line := strings.Split(row, "\t")
			for i := 0; i < len(line); i++ {
				fmt.Fprintf(pab, s, line[i])
			}
		}
	}
	e.drawVisual(pab)
}

/*
* simplified version of generateWWStringFromBuffer
* used by editor.showMarkdown and editor.spellCheck in editor_normal
* we know we want the whole buffer not just what is visible
* unlike the situation with syntax highlighting for code
* we don't have to handle word-wrapped lines in a special way
 */
func (e *Editor) generateWWStringFromBuffer2() string {
	numRows := len(e.bb)
	if numRows == 0 {
		return ""
	}

	var ab strings.Builder
	y := 0
	filerow := 0
	//width := e.screencols - e.left_margin_offset
	width := e.screencols //05042021

	for {
		if filerow == numRows {
			return ab.String()
		}

		row := e.bb[filerow]

		if len(row) == 0 {
			ab.Write([]byte("\n"))
			filerow++
			y++
			continue
		}

		pos := 0
		prev_pos := 0 //except for start -> pos + 1
		for {
			// if remainder of line is less than screen width
			if prev_pos+width > len(row)-1 {
				ab.Write(row[prev_pos:])
				ab.Write([]byte("\n"))
				y++
				filerow++
				break
			}

			//pos = bytes.LastIndex(row[:prev_pos+width], []byte(" "))
			pos = bytes.LastIndex(row[prev_pos:prev_pos+width], []byte(" "))
			//if pos == -1 || pos == prev_pos-1 {
			if pos == -1 {
				pos = prev_pos + width - 1
				/// else added 06/25/2021
			} else {
				pos = pos + prev_pos
			}

			ab.Write(row[prev_pos : pos+1]) //? pos+1
			ab.Write([]byte("\n"))
			y++
			prev_pos = pos + 1
		}
	}
}

/* below exists to create a string that has the proper
 * line breaks based on screen width for syntax highlighting
 * being done in drawcoderows
 * produces a text string that starts at the first line of the
 * file (need to deal with comments where start of comment might be scrolled
 * and ends on the last visible line. Word-wrapped rows are terminated by \t
 * so highlighter deals with them correctly and converted to \n in drawcoderows
 * very similar to dbfunc generateWWString except this uses buffer
 * and only returns as much file as fits the screen
 * and deals with the issue of multi-line comments
 */
func (e *Editor) generateWWStringFromBuffer() string {
	numRows := len(e.bb)
	if numRows == 0 {
		return ""
	}

	var ab strings.Builder
	y := 0
	filerow := 0
	width := e.screencols - e.left_margin_offset

	for {
		if filerow == numRows || y == e.screenlines+e.lineOffset-1 {
			e.last_visible_row = filerow - 1
			return ab.String()[:ab.Len()-1] // delete last \n
		}

		row := e.bb[filerow]

		if len(row) == 0 {
			ab.Write([]byte("\n"))
			filerow++
			y++
			continue
		}

		pos := 0
		prev_pos := 0
		for {
			// if remainder of line is less than screen width
			if prev_pos+width > len(row)-1 {
				ab.Write(row[prev_pos:])
				ab.Write([]byte("\n"))
				y++
				filerow++
				break
			}

			pos = bytes.LastIndex(row[prev_pos:prev_pos+width], []byte(" "))
			if pos == -1 {
				pos = prev_pos + width - 1
			} else {
				pos = pos + prev_pos
			}

			ab.Write(row[prev_pos : pos+1])
			if y == e.screenlines+e.lineOffset-1 {
				e.last_visible_row = filerow - 1
				return ab.String()
			}
			ab.Write([]byte("\t"))
			y++
			prev_pos = pos + 1
		}
	}
}

func (e *Editor) drawStatusBar() {
	var ab strings.Builder
	fmt.Fprintf(&ab, "\x1b[%d;%dH", e.screenlines+e.top_margin, e.left_margin+1)

	//erase from start of an Editor's status bar to the end of the Editor's status bar
	fmt.Fprintf(&ab, "\x1b[%dX", e.screencols)

	ab.WriteString("\x1b[7m ") //switches to inverted colors
	title := getTitle(e.id)
	if len(title) > 30 {
		title = title[:30]
	}
	if e.isModified() {
		title += "[+]"
	}
	status := fmt.Sprintf("%d - %s ...", e.id, title)

	if len(status) > e.screencols-1 {
		status = status[:e.screencols-1]
	}
	fmt.Fprintf(&ab, "%-*s", e.screencols, status)
	ab.WriteString("\x1b[0m") //switches back to normal formatting
	fmt.Print(ab.String())
}

func (e *Editor) drawFrame() {
	var ab strings.Builder
	ab.WriteString("\x1b(0") // Enter line drawing mode

	for j := 1; j < e.screenlines+1; j++ {
		fmt.Fprintf(&ab, "\x1b[%d;%dH", e.top_margin-1+j, e.left_margin+e.screencols+1)
		// below x = 0x78 vertical line (q = 0x71 is horizontal) 37 = white; 1m = bold (note
		// only need one 'm'
		ab.WriteString("\x1b[37;1mx")
	}

	//'T' corner = w or right top corner = k
	fmt.Fprintf(&ab, "\x1b[%d;%dH", e.top_margin-1, e.left_margin+e.screencols+1)

	if e.left_margin+e.screencols > sess.screenCols-4 {
		ab.WriteString("\x1b[37;1mk") //draw corner
	} else {
		ab.WriteString("\x1b[37;1mw")
	}

	//exit line drawing mode
	ab.WriteString("\x1b(B")
	ab.WriteString("\x1b[?25h") //shows the cursor
	ab.WriteString("\x1b[0m")   //or else subsequent editors are bold
	fmt.Print(ab.String())
}

func (e *Editor) scroll() {

	if e.fc == 0 && e.fr == 0 {
		e.cy, e.cx, e.lineOffset, e.first_visible_row, e.last_visible_row = 0, 0, 0, 0, 0
		return
	}

	e.cx = e.getScreenXFromRowColWW(e.fr, e.fc)
	cy := e.getScreenYFromRowColWW(e.fr, e.fc)

	//deal with scroll insufficient to include the current line
	if cy > e.screenlines+e.lineOffset-1 {
		e.lineOffset = cy - e.screenlines + 1 ////
		e.adjustFirstVisibleRow()             //can also change e.lineOffset
	}

	if cy < e.lineOffset {
		e.lineOffset = cy
		e.adjustFirstVisibleRow()
	}

	if e.lineOffset == 0 {
		e.first_visible_row = 0
	}

	e.cy = cy - e.lineOffset
}

// e.lineOffset determines the first
// visible row but we want the whole row
// visible so that can change e.lineOffset
func (e *Editor) adjustFirstVisibleRow() {

	if e.lineOffset == 0 {
		e.first_visible_row = 0
		return
	}

	rowNum := 0
	lines := 0

	for {
		lines += e.getLinesInRowWW(rowNum)
		rowNum++

		/*
			there is no need to adjust line_offset
			if it happens that we start
			on the first line of the first visible row
		*/
		if lines == e.lineOffset {
			break
		}

		/*
			need to adjust line_offset
			so we can start on the first
			line of the top row
		*/
		if lines > e.lineOffset {
			e.lineOffset = lines
			break
		}
	}
	e.first_visible_row = rowNum
}

func (e *Editor) readFileIntoNote(filename string) error {

	r, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("Error opening file %s: %w", filename, err)
	}
	defer r.Close()

	/*
		e.rows = nil
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			e.rows = append(e.rows, strings.ReplaceAll(scanner.Text(), "\t", " "))
		}
	*/
	e.bb = nil
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		e.bb = append(e.bb, scanner.Bytes()) // not dealing with tabs for the moment
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error reading file %q: %v", filename, err)
	}
	v.SetBufferLines(e.vbuf, 0, -1, true, e.bb)

	e.fr, e.fc, e.cy, e.cx, e.lineOffset, e.first_visible_row, e.last_visible_row = 0, 0, 0, 0, 0, 0, 0

	e.drawText()
	e.drawStatusBar() // not sure what state of isModified would be so not sure need to draw statubBar
	return nil
}

func (e *Editor) drawPreview() {
	fmt.Print("\x1b_Ga=d\x1b\\") //delete any images

	rows := strings.Split(e.renderedNote, "\n")
	fmt.Printf("\x1b[?25l\x1b[%d;%dH", e.top_margin, e.left_margin+1)
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)

	// erase specific editors 'window'
	erase_chars := fmt.Sprintf("\x1b[%dX", e.screencols)
	for i := 0; i < e.screenlines; i++ {
		fmt.Printf("%s%s", erase_chars, lf_ret)
	}

	fmt.Printf("\x1b[%d;%dH", e.top_margin, e.left_margin+1)

	fr := e.previewLineOffset - 1
	y := 0
	for {
		fr++
		if fr > len(rows)-1 || y > e.screenlines-1 {
			break
		}
		if strings.Contains(rows[fr], "Image") {
			fmt.Printf("Loading Image ... \x1b[%dG", e.left_margin+1)
			prevY := y
			path := getStringInBetween(rows[fr], "|", "|")
			var img image.Image
			var err error
			if strings.Contains(path, "http") {
				img, _, err = loadWebImage(path)
				if err != nil {
					fmt.Printf("%sError:%s %s%s", BOLD, RESET, rows[fr], lf_ret)
					y++
					continue
				}
			} else {
				maxWidth := e.screencols * int(sess.ws.Xpixel) / sess.screenCols
				maxHeight := e.screenlines * int(sess.ws.Ypixel) / sess.screenLines
				img, _, err = loadImage(path, maxWidth-5, maxHeight-150)
				if err != nil {
					fmt.Printf("%sError:%s %s%s", BOLD, RESET, rows[fr], lf_ret)
					y++
					continue
				}
			}
			height := img.Bounds().Max.Y / (int(sess.ws.Ypixel) / sess.screenLines)
			y += height
			if y > e.screenlines-1 {
				fmt.Printf("\x1b[3m\x1b[4mImage %s doesn't fit!\x1b[0m \x1b[%dG", path, e.left_margin+1)
				y = y - height + 1
				fmt.Printf("\x1b[%d;%dH", TOP_MARGIN+1+y, e.left_margin+1)
				continue
			}
			displayImage(img)
			// erases "Loading image ..."
			fmt.Printf("\x1b[%d;%dH\x1b[0K", e.top_margin+prevY, e.left_margin+1)
			fmt.Printf("\x1b[%d;%dH", e.top_margin+y, e.left_margin+1)
		} else {
			fmt.Printf("%s%s", rows[fr], lf_ret)
			y++
		}
	}
}

func (e *Editor) drawOverlay() {
	fmt.Print("\x1b_Ga=d\x1b\\") //delete any images

	//rows := strings.Split(s, "\n")
	fmt.Printf("\x1b[?25l\x1b[%d;%dH", e.top_margin, e.left_margin+1)
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)

	// erase specific editors 'window'
	erase_chars := fmt.Sprintf("\x1b[%dX", e.screencols)
	for i := 0; i < e.screenlines; i++ {
		fmt.Printf("%s%s", erase_chars, lf_ret)
	}

	fmt.Printf("\x1b[%d;%dH", e.top_margin, e.left_margin+1)

	fr := e.previewLineOffset - 1
	y := 0
	rows := e.overlay
	for {
		fr++
		if fr > len(rows)-1 || y > e.screenlines-1 {
			break
		}
		fmt.Printf("%s%s", rows[fr], lf_ret)
		y++
	}
}

// this func is reason that we are writing notes to file
// allows easy testing if a file is modified with BufferOption
func (e *Editor) isModified() bool {
	var result bool
	err := v.BufferOption(e.vbuf, "modified", &result)
	if err != nil {
		sess.showEdMessage("Error checking isModified: %v", err)
		return true
	}
	return result
}
