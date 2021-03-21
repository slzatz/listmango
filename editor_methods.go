package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
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

// want to transition to Session showEdMessage
func (e *Editor) showMessage(format string, a ...interface{}) {
	fmt.Printf("\x1b[%d;%dH\x1b[K", sess.textLines+e.top_margin+1, sess.divider+1)
	str := fmt.Sprintf(format, a...)
	if len(str) > e.screencols {
		str = str[:e.screencols]
	}
	fmt.Print(str)
}

//'automatically' happens in NORMAL and INSERT mode
//return true -> redraw; false -> don't redraw
func (e *Editor) find_match_for_left_brace(left_brace byte, back bool) bool {
	r := e.fr
	c := e.fc + 1
	count := 1
	max := len(e.rows)
	var b int
	if back {
		b = 1
	}

	m := map[byte]byte{'{': '}', '(': ')', '[': ']'}
	right_brace := m[left_brace]

	for {

		row := e.rows[r]

		// need >= because brace could be at end of line and in INSERT mode
		// fc could be row.size() [ie beyond the last char in the line
		// and so doing fc + 1 above leads to c > row.size()
		if c >= len(row) {
			r++
			if r == max {
				e.showMessage("Couldn't find matching brace")
				return false
			}
			c = 0
			continue
		}

		if row[c] == right_brace {
			count -= 1
			if count == 0 {
				break
			}
		} else if row[c] == left_brace {
			count += 1
		}

		c++
	}
	y := e.getScreenYFromRowColWW(r, c) - e.line_offset
	if y >= e.screenlines {
		return false
	}

	x := e.getScreenXFromRowColWW(r, c) + e.left_margin + e.left_margin_offset + 1
	//fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%d", y+e.top_margin, x, right_brace)
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s", y+e.top_margin, x, string(right_brace))

	x = e.getScreenXFromRowColWW(e.fr, e.fc-b) + e.left_margin + e.left_margin_offset + 1
	y = e.getScreenYFromRowColWW(e.fr, e.fc-b) + e.top_margin - e.line_offset // added line offset 12-25-2019
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s\x1b[0m", y, x, string(left_brace))
	e.showMessage("r = %d   c = %d", r, c)
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
	y := e.getScreenYFromRowColWW(r, c) - e.line_offset
	if y >= e.screenlines {
		return false
	}

	x := e.getScreenXFromRowColWW(r, c) + e.left_margin + e.left_margin_offset + 1
	//fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s", y+e.top_margin, x, string(right_brace))
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s", y+e.top_margin, x, "]")

	//x = e.getScreenXFromRowColWW(e.fr, e.fc-b) + e.left_margin + e.left_margin_offset + 1
	x = e.getScreenXFromRowColWW(e.fr, e.fc) + e.left_margin + e.left_margin_offset + 1
	//y = e.getScreenYFromRowColWW(e.fr, e.fc-b) + e.top_margin - e.line_offset // added line offset 12-25-2019
	y = e.getScreenYFromRowColWW(e.fr, e.fc) + e.top_margin - e.line_offset // added line offset 12-25-2019
	//fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s\x1b[0m", y, x, string(left_brace))
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s\x1b[0m", y, x, "[")
	v.SetWindowCursor(w, [2]int{e.fr + 1, e.fc}) //set screen cx and cy from pos
	sess.showEdMessage("r = %d   c = %d; e.fr = %d e.fc = %d", r, c, e.fr, e.fc)
	return true
}

//'automatically' happens in NORMAL and INSERT mode
func (e *Editor) find_match_for_right_brace(right_brace byte, back bool) bool {
	var b int
	if back {
		b = 1
	}
	r := e.fr
	c := e.fc - 1 - b
	count := 1

	row := e.rows[r]

	m := map[byte]byte{'}': '{', ')': '(', ']': '['}
	left_brace := m[right_brace]

	for {

		if c == -1 { //fc + 1 can be greater than row.size on first pass from INSERT if { at end of line
			r--
			if r == -1 {
				e.showMessage("Couldn't find matching brace")
				return false
			}
			row = e.rows[r]
			c = len(row) - 1
			continue
		}

		if row[c] == left_brace {
			count -= 1
			if count == 0 {
				break
			}
		} else if row[c] == right_brace {
			count += 1
		}

		c--
	}

	y := e.getScreenYFromRowColWW(r, c) - e.line_offset
	if y < 0 {
		return false
	}

	x := e.getScreenXFromRowColWW(r, c) + e.left_margin + e.left_margin_offset + 1
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s", y+e.top_margin, x, string(left_brace))

	x = e.getScreenXFromRowColWW(e.fr, e.fc-b) + e.left_margin + e.left_margin_offset + 1
	y = e.getScreenYFromRowColWW(e.fr, e.fc-b) + e.top_margin - e.line_offset // added line offset 12-25-2019
	fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%s\x1b[0m", y, x, string(right_brace))
	e.showMessage("r = %d   c = %d", r, c)
	return true
}

func (e *Editor) draw_highlighted_braces() {

	braces := "{}()" //? intentionally exclusing [] from auto drawing
	var c byte
	var back bool
	//if below handles case when in insert mode and brace is last char
	//in a row and cursor is beyond that last char (which is a brace)
	if e.fc == len(e.rows[e.fr]) {
		c = e.rows[e.fr][e.fc-1]
		back = true
	} else {
		c = e.rows[e.fr][e.fc]
		back = false
	}
	pos := strings.Index(braces, string(c))
	if pos != -1 {
		switch c {
		case '{', '(':
			e.redraw = e.find_match_for_left_brace(c, back)
			return
		case '}', ')':
			e.redraw = e.find_match_for_right_brace(c, back)
			return
		//case '(':
		default: //should not need this
			return
		}
	} else if e.fc > 0 && e.mode == INSERT {
		c := e.rows[e.fr][e.fc-1]
		pos := strings.Index(braces, string(c))
		if pos != -1 {
			switch e.rows[e.fr][e.fc-1] {
			case '{', '(':
				e.redraw = e.find_match_for_left_brace(c, true)
				return
			case '}', ')':
				e.redraw = e.find_match_for_right_brace(c, true)
				return
			//case '(':
			default: //should not need this
				return
			}
		} else {
			e.redraw = false
		}
	} else {
		e.redraw = false
	}
}

func (e *Editor) setLinesMargins() { //also sets top margin

	if e.linked_editor != nil {
		if e.is_subeditor {
			if e.is_below {
				e.screenlines = LINKED_NOTE_HEIGHT
				e.top_margin = sess.textLines - LINKED_NOTE_HEIGHT + 2
			} else {
				e.screenlines = sess.textLines
				e.top_margin = TOP_MARGIN + 1
			}
		} else {
			if e.linked_editor.is_below {
				e.screenlines = sess.textLines - LINKED_NOTE_HEIGHT - 1
				e.top_margin = TOP_MARGIN + 1
			} else {
				e.screenlines = sess.textLines
				e.top_margin = TOP_MARGIN + 1
			}
		}
	} else {
		e.screenlines = sess.textLines
		e.top_margin = TOP_MARGIN + 1
	}
}

// normal mode 'e'
func (e *Editor) moveEndWord_() {

	if len(e.rows) == 0 {
		return
	}

	if len(e.rows[e.fr]) == 0 || e.fc == len(e.rows[e.fr])-1 {
		if e.fr+1 > len(e.rows)-1 {
			return
		}
		e.fr++
		e.fc = 0
	} else {
		e.fc++
	}

	r := e.fr
	c := e.fc
	var pos int
	delimiters := " *%!^<>,.;?:()[]{}&#~'\""
	delimiters_without_space := "*%!^<>,.;?:()[]{}&#~'\""

	for {

		if r > len(e.rows)-1 {
			return
		}

		row := &e.rows[r]

		if len(*row) == 0 {
			r++
			c = 0
			continue
		}

		if strings.Index(delimiters, string((*row)[c])) == -1 {
			if c == len(*row)-1 || strings.Index(delimiters, string((*row)[c+1])) != -1 {
				e.fc = c
				e.fr = r
				return
			}

			pos = strings.IndexAny(string((*row)[c]), delimiters)
			if pos == -1 {
				e.fc = len(*row) - 1
				return
			} else {
				e.fr = r
				e.fc = pos - 1
				return
			}

			// we started on punct or space
		} else {
			if (*row)[c] == ' ' {
				if c == len(*row)-1 {
					r++
					c = 0
					continue
				} else {
					c++
					continue
				}
			} else {
				pos = find_first_not_of(row, delimiters_without_space, c)
				if pos != -1 {
					e.fc = pos - 1
					return
				} else {
					e.fc = len(*row) - 1
					return
				}
			}
		}
	}
}

func (e *Editor) rowsToString() string {

	numRows := len(e.rows)
	if numRows == 0 {
		return ""
	}

	var sb strings.Builder
	for i := 0; i < numRows-1; i++ {
		sb.WriteString(e.rows[i] + "\n")
	}
	sb.WriteString(e.rows[numRows-1])
	return sb.String()
}

func (e *Editor) getScreenXFromRowColWW(r, c int) int {
	// can't use reference to row because replacing blanks to handle corner case
	//bb, _ := v.BufferLines(0, 0, -1, true)
	//row := string(bb[r])
	row := e.rows[r]

	/* pos is the position of the last char in the line
	 * and pos+1 is the position of first character of the next row
	 */

	if len(row) <= e.screencols-e.left_margin_offset {
		return c
	}

	pos := -1
	prev_pos := 0
	for {

		if len(row[pos+1:]) <= e.screencols-e.left_margin_offset {
			prev_pos = pos
			break
		}

		prev_pos = pos
		//cpp find_last_of -the parameter defines the position from beginning to look at (inclusive)
		//need to add + 1 because slice :n includes chars up to the n - 1 char
		pos = strings.LastIndex(row[:pos+e.screencols-e.left_margin_offset+1], " ")

		if pos == -1 {
			pos = prev_pos + e.screencols - e.left_margin_offset
		} else if pos == prev_pos {
			row = strings.Replace(row[:pos+1], " ", "+", -1) + row[pos+1:]
			pos = prev_pos + e.screencols - e.left_margin_offset
		}
		/*
		   else
		     replace(row.begin()+prev_pos+1, row.begin()+pos+1, ' ', '+');
		*/

		if pos >= c {
			break
		}
	}
	return c - prev_pos - 1
}

func (e *Editor) getScreenYFromRowColWW(r, c int) int {
	screenline := 0

	for n := 0; n < r; n++ {
		screenline += e.getLinesInRowWW(n)
	}

	screenline = screenline + e.getLineInRowWW(r, c) - 1
	return screenline
}

func (e *Editor) getLinesInRowWW(r int) int {
	row := e.rows[r]

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
		pos = strings.LastIndex(row[:pos+e.screencols-e.left_margin_offset+1], " ")

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

func (e *Editor) getLineInRowWW(r, c int) int {
	// can't use reference to row because replacing blanks to handle corner case
	row := e.rows[r]

	if len(row) <= e.screencols-e.left_margin_offset {
		return 1
	}

	/* pos is the position of the last char in the line
	 * and pos+1 is the position of first character of the next row
	 */

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
		pos = strings.LastIndex(row[:pos+e.screencols-e.left_margin_offset+1], " ")

		if pos == -1 {
			pos = prev_pos + e.screencols - e.left_margin_offset

			// only replace if you have enough characters without a space to trigger this
			// need to start at the beginning each time you hit this
			// unless you want to save the position which doesn't seem worth it
		} else if pos == prev_pos {
			row = strings.Replace(row[:pos+1], " ", "+", -1) + row[pos+1:]
			pos = prev_pos + e.screencols - e.left_margin_offset
		}

		lines++
		if pos >= c {
			break
		}
	}
	return lines
}

// if draw is false this only draws cursor
func (e *Editor) refreshScreen(draw bool) {
	var ab strings.Builder
	var tid int

	if draw {
		// \x1b[?25l hides cursor
		fmt.Fprintf(&ab, "\x1b[?25l\x1b[%d;%dH", e.top_margin, e.left_margin+1)
		// \x1b[NC moves cursor forward by N columns
		lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)
		erase_chars := fmt.Sprintf("\x1b[%dX", e.screencols)
		for i := 0; i < e.screenlines; i++ {
			ab.WriteString(erase_chars)
			ab.WriteString(lf_ret)
		}

		tid = getFolderTid(e.id)
		if tid == 18 || tid == 14 { //&& !e.is_subeditor {
			e.drawCodeRows(&ab) //uaing pointer so drawing is smoother
		} else {
			e.drawRows2(&ab) //2 -> uses nvim buffer
		}
		fmt.Print(ab.String())
		e.drawStatusBar()
	}

	// the lines below position the cursor where it should go
	// ? if we're every in COMMAND_LINE when we are drawing rows??
	//if e.mode != COMMAND_LINE {
	if true {
		fmt.Fprintf(&ab, "\x1b[%d;%dH", e.cy+e.top_margin, e.cx+e.left_margin+e.left_margin_offset+1)
	}

	if len(e.rows) == 0 || len(e.rows[e.fr]) == 0 {
		return
	}

	if tid == 18 || tid == 14 { //&& !e.is_subeditor {
		e.draw_highlighted_braces()
	}
	//fmt.Print(ab.String())
	//if draw {
	//	e.drawStatusBar()
	//	}

	/*
	  // can't do the below until ab is written or will just overwite highlights
	  if (draw && spellcheck) editorSpellCheck();
	*/
}

func (e *Editor) drawRows(pab *strings.Builder) {

	if len(e.rows) == 0 {
		return
	}
	//var ab strings.Builder

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)
	(*pab).WriteString("\x1b[?25l") //hides the cursor

	// format for positioning cursor is "\x1b[%d;%dH"
	fmt.Fprintf(pab, "\x1b[%d;%dH", e.top_margin, e.left_margin+1)

	y := 0
	filerow := e.first_visible_row
	flag := false

	for {

		if flag {
			break
		}

		if filerow == len(e.rows) {
			e.last_visible_row = filerow - 1
			break
		}

		row := e.rows[filerow]

		if len(row) == 0 {
			if y == e.screenlines-1 {
				break
			}
			(*pab).WriteString(lf_ret)
			filerow++
			y++
			continue
		}

		pos := 0
		prev_pos := 0 //except for start -> pos + 1
		for {
			/* this is needed because it deals where the end of the line doesn't have a space*/
			if prev_pos+e.screencols-e.left_margin_offset > len(row)-1 { //? if need -1;cpp generateWWString had it
				(*pab).WriteString(row[prev_pos:])
				if y == e.screenlines-1 {
					flag = true
					break
				}
				(*pab).WriteString(lf_ret)
				y++
				filerow++
				break
			}

			pos = strings.LastIndex(row[:prev_pos+e.screencols-e.left_margin_offset], " ")

			//note npos when signed = -1 and order of if/else may matter
			if pos == -1 || pos == prev_pos-1 {
				pos = prev_pos + e.screencols - e.left_margin_offset - 1
			}

			(*pab).WriteString(row[prev_pos : pos+1]) //? pos+1
			if y == e.screenlines-1 {
				flag = true
				break
			}
			(*pab).WriteString(lf_ret)
			prev_pos = pos + 1
			y++
		}
	}
	// ? only used so spellcheck stops at end of visible note
	e.last_visible_row = filerow - 1 // note that this is not exactly true - could be the whole last row is visible

	e.draw_visual(pab)
}

func (e *Editor) draw_visual(pab *strings.Builder) {

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin+e.left_margin_offset)

	if e.mode == VISUAL_LINE {
		startRow := e.vb_highlight[0][1] - 1 // I think better to subtract one here
		endRow := e.vb_highlight[1][1] - 1   //ditto - done differently for VISUAL and V_BLOCK

		// \x1b[NC moves cursor forward by N columns
		// snprintf(lf_ret, sizeof(lf_ret), "\r\n\x1b[%dC", left_margin + left_margin_offset);

		x := e.left_margin + e.left_margin_offset + 1
		//int y = editorGetScreenYFromRowColWW(h_light[0], 0) + top_margin - line_offset;
		y := e.getScreenYFromRowColWW(startRow, 0) - e.line_offset

		if y >= 0 {
			fmt.Fprintf(pab, "\x1b[%d;%dH\x1b[48;5;244m", y+e.top_margin, x)
		} else {
			fmt.Fprintf(pab, "\x1b[%d;%dH\x1b[48;5;244m", e.top_margin, x)
		}

		for n := 0; n < (endRow - startRow + 1); n++ { //++n
			row_num := startRow + n
			pos := 0
			for line := 1; line <= e.getLinesInRowWW(row_num); line++ { //++line
				if y < 0 {
					y += 1
					continue
				}
				if y == e.screenlines {
					break //out for should be done (theoretically) - 1
				}
				line_char_count := e.getLineCharCountWW(row_num, line)
				(*pab).WriteString(e.rows[row_num][pos : pos+line_char_count])
				(*pab).WriteString(lf_ret)
				y += 1
				pos += line_char_count
			}
		}
	}

	if e.mode == VISUAL {
		startCol, endCol := e.vb_highlight[0][2], e.vb_highlight[1][2]
		startRow, endRow := e.vb_highlight[0][1], e.vb_highlight[1][1] //startRow always <= endRow
		numRows := endRow - startRow + 1

		x := e.getScreenXFromRowColWW(startRow, startCol) + e.left_margin + e.left_margin_offset
		y := e.getScreenYFromRowColWW(startRow, startCol) + e.top_margin - e.line_offset - 1

		(*pab).WriteString("\x1b[48;5;244m")
		for n := 0; n < numRows; n++ {
			// I think would check here to see if a row has multiple lines (ie wraps)
			if n == 0 {
				fmt.Fprintf(pab, "\x1b[%d;%dH", y+n, x)
			} else {
				fmt.Fprintf(pab, "\x1b[%d;%dH", y+n, 1+e.left_margin+e.left_margin_offset)
			}
			//row := e.rows[startRow+n-1]
			row := e.rows[startRow+n-1]
			row_len := len(row)

			if row_len == 0 { //|| row_len < left {
				continue
			}
			if numRows == 1 {
				(*pab).WriteString(row[startCol-1 : endCol])
			} else if n == 0 {
				(*pab).WriteString(row[startCol-1:])
			} else if n < numRows-1 {
				(*pab).WriteString(row)
			} else {
				if len(row) < endCol {
					(*pab).WriteString(row)
				} else {
					(*pab).WriteString(row[:endCol])
				}
			}
			//(*pab).WriteString(row[startCol-1:])
			sess.showOrgMessage("%v; %v; %v; %v", startCol, endCol, startRow, endRow)
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
		y := e.getScreenYFromRowColWW(e.vb_highlight[0][1], left) + e.top_margin - e.line_offset - 1

		(*pab).WriteString("\x1b[48;5;244m")
		for n := 0; n < (e.vb_highlight[1][1] - e.vb_highlight[0][1] + 1); n++ {
			fmt.Fprintf(pab, "\x1b[%d;%dH", y+n, x)
			row := e.rows[e.vb_highlight[0][1]+n-1]
			row_len := len(row)

			if row_len == 0 || row_len < left {
				continue
			}

			if row_len < right {
				(*pab).WriteString(row[left-1 : row_len])
			} else {
				(*pab).WriteString(row[left-1 : right])
			}
		}
	}

	(*pab).WriteString("\x1b[0m")
}

func (e *Editor) getLineCharCountWW(r, line int) int {
	b, _ := v.BufferLines(0, r, r+1, true)
	row := string(b[0])
	//row := e.rows[r]

	if len(row) == 0 {
		return 0
	}

	if len(row) <= e.screencols-e.left_margin_offset {
		return len(row)
	}

	lines := 0 //1
	pos := -1
	prev_pos := 0
	for {

		// we know the first time around this can't be true
		// could add if (line > 1 && row.substr(pos+1).size() ...);
		if len(row[pos+1:]) <= e.screencols-e.left_margin_offset {
			return len(row[pos+1:])
		}

		prev_pos = pos
		pos = strings.LastIndex(row[:pos+e.screencols-e.left_margin_offset], " ")

		if pos == -1 {
			pos = prev_pos + e.screencols - e.left_margin_offset

			// only replace if you have enough characters without a space to trigger this
			// need to start at the beginning each time you hit this
			// unless you want to save the position which doesn't seem worth it
		} else if pos == prev_pos {
			row = strings.ReplaceAll(row[:pos+1], " ", "+") + row[pos+1:]
			pos = prev_pos + e.screencols - e.left_margin_offset
		}

		lines++
		if lines == line {
			break
		}
	}
	return pos - prev_pos
}

// not in use -- was attempt to draw rows without e.rows just nvim buffer
func (e *Editor) drawRows2(pab *strings.Builder) {
	// v.BufferLines appears to die if in blocking mode
	// so probably should protect it directly and not rely on redraw bool
	// Also v.Bufferlines doesn't return an err when in blocking mode - just dies so checking for err not useful
	bb, _ := v.BufferLines(0, 0, -1, true)

	if len(bb) == 0 {
		return
	}

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)
	// hides the cursor
	(*pab).WriteString("\x1b[?25l") //hides the cursor

	// position the cursor"
	fmt.Fprintf(pab, "\x1b[%d;%dH", e.top_margin, e.left_margin+1)

	y := 0
	filerow := e.first_visible_row
	flag := false

	for {

		if flag {
			break
		}

		if filerow == len(bb) {
			e.last_visible_row = filerow - 1
			break
		}

		row := string(bb[filerow])

		if len(row) == 0 {
			if y == e.screenlines-1 {
				break
			}
			(*pab).WriteString(lf_ret)
			filerow++
			y++
			continue
		}

		pos := 0
		prev_pos := 0 //except for start -> pos + 1
		for {
			/* this is needed because it deals where the end of the line doesn't have a space*/
			if prev_pos+e.screencols-e.left_margin_offset > len(row)-1 { //? if need -1;cpp
				(*pab).WriteString(row[prev_pos:])
				if y == e.screenlines-1 {
					flag = true
					break
				}
				(*pab).WriteString(lf_ret)
				y++
				filerow++
				break
			}

			pos = strings.LastIndex(row[:prev_pos+e.screencols-e.left_margin_offset], " ")

			if pos == -1 || pos == prev_pos-1 {
				pos = prev_pos + e.screencols - e.left_margin_offset - 1
			}

			(*pab).WriteString(row[prev_pos : pos+1]) //? pos+1
			if y == e.screenlines-1 {
				flag = true
				break
			}
			(*pab).WriteString(lf_ret)
			prev_pos = pos + 1
			y++
		}
	}
	// ? only used so spellcheck stops at end of visible note
	e.last_visible_row = filerow - 1 // note that this is not exactly true - could be the whole last row is visible

	e.draw_visual(pab)
}

func (e *Editor) drawCodeRows(pab *strings.Builder) {
	//save the current file to code_file with correct extension
	f, err := os.Create("code_file")
	if err != nil {
		sess.showEdMessage("Error creating code_file: %v", err)
		return
	}
	defer f.Close()

	_, err = f.WriteString(e.generateWWStringFromBuffer())
	if err != nil {
		sess.showEdMessage("Error writing code_file: %v", err)
		return
	}
	//f.Close()

	//var ab strings.Builder

	var syntax string
	if getFolderTid(e.id) == 18 {
		syntax = "--syntax=cpp"
	} else {
		syntax = "--syntax=go"
	}
	cmd := exec.Command("highlight", "code_file", "--out-format=xterm256", "--style=gruvbox-dark-hard-slz", syntax)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	buffer := bufio.NewReader(stdout)

	/* alternative is to use a Scanner
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
	 z = scanner.Text()
	*/

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)
	fmt.Fprintf(pab, "\x1b[?25l\x1b[%d;%dH", e.top_margin, e.left_margin+1)

	// below draws the line number 'rectangle' only matters for the word-wrapped lines
	fmt.Fprintf(pab, "\x1b[2*x\x1b[%d;%d;%d;%d;48;5;235$r\x1b[*x",
		e.top_margin, e.left_margin, e.top_margin+e.screenlines, e.left_margin+e.left_margin_offset)
	n := 0
	//func (b *Reader) ReadLine() (line []byte, isPrefix bool, err)
	for {
		bytes, _, err := buffer.ReadLine()
		if err == io.EOF {
			break
		}

		if n >= e.first_visible_row { //substituted for above on 12312020
			line := string(bytes)
			fmt.Fprintf(pab, "\x1b[48;5;235m\x1b[38;5;245m%3d \x1b[0m", n)
			ll := strings.Split(line, "\t")
			for i := 0; i < len(ll)-1; i++ {
				fmt.Fprintf(pab, "%s%s\x1b[%dC", ll[i], lf_ret, e.left_margin_offset)
			}
			fmt.Fprintf(pab, "%s%s", ll[len(ll)-1], lf_ret)
		}
		n++
	}
	e.draw_visual(pab)
}

/* below exists to create a text file that has the proper
 * line breaks based on screen width for syntax highlighters
 * that are utilized by drawCodeRows
 * Produces a text string that starts at the first line of the
 * file (need to deal with comments where start of comment might be scrolled
 * and ends on the last visible linei. Also multilines are indicated by \t
 * so highlighter deals with them correctly and converted to \n in drawCodeRows
 * Only used by editorDrawCodeRows
 */
func (e *Editor) generateWWString() string {
	if len(e.rows) == 0 {
		return ""
	}

	var ab strings.Builder
	y := -e.line_offset
	filerow := 0

	for {
		if filerow == len(e.rows) {
			e.last_visible_row = filerow - 1
			return ab.String()
		}

		//char ret = '\n';
		ret := "\t"
		row := e.rows[filerow]
		// if you put a \n in the middle of a comment the wrapped portion won't be italic
		//if (row.find("//") != std::string::npos) ret = '\t';
		//ret = '\t';

		if len(row) == 0 {
			if y == e.screenlines-1 {
				return ab.String()
			}
			ab.WriteString("\n")
			filerow++
			y++
			continue
		}

		pos := 0
		prev_pos := 0 //except for start -> pos + 1
		for {
			// if remainder of line is less than screen width
			if prev_pos+e.screencols-e.left_margin_offset > len(row)-1 {
				ab.WriteString(row[prev_pos:])
				if y == e.screenlines-1 {
					e.last_visible_row = filerow - 1
					return ab.String()
				}
				ab.WriteString("\n")
				y++
				filerow++
				break
			}

			pos = strings.LastIndex(row[:prev_pos+e.screencols-e.left_margin_offset], " ")

			//note npos when signed = -1 and order of if/else may matter
			if pos == -1 || pos == prev_pos-1 {
				pos = prev_pos + e.screencols - e.left_margin_offset - 1
			}

			ab.WriteString(row[prev_pos : pos+1]) //? pos+1
			if y == e.screenlines-1 {
				e.last_visible_row = filerow - 1
				return ab.String()
			}
			ab.WriteString(ret)
			prev_pos = pos + 1
			y++
		}
	}
}

func (e *Editor) generateWWStringFromBuffer() string {

	bb, _ := v.BufferLines(0, 0, -1, true)
	numLines := len(bb)
	if numLines == 0 {
		return ""
	}
	var ab strings.Builder
	y := -e.line_offset
	filerow := 0

	for {
		if filerow == numLines {
			e.last_visible_row = filerow - 1
			return ab.String()
		}

		//char ret = '\n';
		ret := "\t"
		row := string(bb[filerow])
		// if you put a \n in the middle of a comment the wrapped portion won't be italic
		//if (row.find("//") != std::string::npos) ret = '\t';
		//ret = '\t';

		if len(row) == 0 {
			if y == e.screenlines-1 {
				return ab.String()
			}
			ab.WriteString("\n")
			filerow++
			y++
			continue
		}

		pos := 0
		prev_pos := 0 //except for start -> pos + 1
		for {
			// if remainder of line is less than screen width
			if prev_pos+e.screencols-e.left_margin_offset > len(row)-1 {
				ab.WriteString(row[prev_pos:])
				if y == e.screenlines-1 {
					e.last_visible_row = filerow - 1
					return ab.String()
				}
				ab.WriteString("\n")
				y++
				filerow++
				break
			}

			pos = strings.LastIndex(row[:prev_pos+e.screencols-e.left_margin_offset], " ")

			//note npos when signed = -1 and order of if/else may matter
			if pos == -1 || pos == prev_pos-1 {
				pos = prev_pos + e.screencols - e.left_margin_offset - 1
			}

			ab.WriteString(row[prev_pos : pos+1]) //? pos+1
			if y == e.screenlines-1 {
				e.last_visible_row = filerow - 1
				return ab.String()
			}
			ab.WriteString(ret)
			prev_pos = pos + 1
			y++
		}
	}
}

func (e *Editor) drawStatusBar() {
	var ab strings.Builder
	fmt.Fprintf(&ab, "\x1b[%d;%dH", e.screenlines+e.top_margin, e.left_margin+1)

	//erase from start of an Editor's status bar to the end of the Editor's status bar
	//ab.append("\x1b[K"); //erases from cursor to end of screen on right - not what we want
	fmt.Fprintf(&ab, "\x1b[%dX", e.screencols)

	ab.WriteString("\x1b[7m ") //switches to inverted colors
	title := getTitle(e.id)
	if len(title) > 30 {
		title = title[:30]
	}
	if e.dirty > 0 {
		title += "[+]"
	}
	var sub string
	if e.is_subeditor {
		sub = "subeditor"
	}
	status := fmt.Sprintf("%d - %s ... %s", e.id, title, sub)

	if len(status) > e.screencols-1 {
		status = status[:e.screencols-1]
	}
	fmt.Fprintf(&ab, "%-*s", e.screencols, status)
	ab.WriteString("\x1b[0m") //switches back to normal formatting
	fmt.Print(ab.String())
}

func (e *Editor) drawMessageBar() {
	var ab strings.Builder
	fmt.Fprintf(&ab, "\x1b[%d;%dH", sess.textLines+e.top_margin+1, sess.divider+1)

	ab.WriteString("\x1b[K") // will erase midscreen -> R; cursor doesn't move after erase
	if len(e.message) > e.screencols {
		e.message = e.message[:e.screencols]
	}
	ab.WriteString(e.message)
	fmt.Print(ab.String())
}

func (e *Editor) scroll() bool {

	if e.fc == 0 && e.fr == 0 {
		e.cy, e.cx, e.line_offset, e.prev_line_offset, e.first_visible_row, e.last_visible_row = 0, 0, 0, 0, 0, 0
		return false // blocking issue with bb, err := v.BufferLines(0, 0, -1, true) in drawRows2
	}

	/*
		if len(e.rows) == 0 {
			e.fr, e.fc, e.cy, e.cx, e.line_offset, e.prev_line_offset, e.first_visible_row, e.last_visible_row = 0, 0, 0, 0, 0, 0, 0, 0
			return true
		}

		if e.fr >= len(e.rows) {
			e.fr = len(e.rows) - 1
		}

		row_size := len(e.rows[e.fr])
		if e.fc >= row_size {
			if e.mode != INSERT {
				e.fc = row_size - 1
			} else {
				e.fc = row_size
			}
		}

		if e.fc < 0 {
			e.fc = 0
		}
	*/

	e.cx = e.getScreenXFromRowColWW(e.fr, e.fc)
	cy_ := e.getScreenYFromRowColWW(e.fr, e.fc)

	//my guess is that if you wanted to adjust line_offset to take into account that you wanted
	// to only have full rows at the top (easier for drawing code) you would do it here.
	// something like screenlines goes from 4 to 5 so that adjusts cy
	// it's complicated and may not be worth it.

	//deal with scroll insufficient to include the current line
	if cy_ > e.screenlines+e.line_offset-1 {
		e.line_offset = cy_ - e.screenlines + 1 ////
		e.first_visible_row, e.line_offset = e.getInitialRow(e.line_offset)
	}

	//let's check if the current line_offset is causing there to be an incomplete row at the top

	// this may further increase line_offset so we can start
	// at the top with the first line of some row
	// and not start mid-row which complicates drawing the rows

	//deal with scrol where current line wouldn't be visible because we're scrolled too far
	if cy_ < e.line_offset {
		e.line_offset = cy_
		//e.first_visible_row = e.getInitialRow(e.line_offset, SCROLL_UP); ????????????????????????????? 2 getInitialRow
		e.first_visible_row, e.line_offset = e.getInitialRow(e.line_offset)
	}

	if e.line_offset == 0 {
		e.first_visible_row = 0
	}

	e.cy = cy_ - e.line_offset

	// vim seems to want full rows to be displayed although I am not sure
	// it's either helpful or worth it but this is a placeholder for the idea

	// returns true if display needs to scroll and false if it doesn't
	// could just be redraw = true or do nothing since don't want to override if already true.
	if e.line_offset == e.prev_line_offset {
		return false
	} else {
		e.prev_line_offset = e.line_offset
		return true
	}
}

func (e *Editor) getInitialRow(line_offset int) (int, int) {

	if line_offset == 0 {
		return 0, 0
	}

	initial_row := 0
	lines := 0

	for {
		lines += e.getLinesInRowWW(initial_row)
		initial_row++

		// there is no need to adjust line_offset
		// if it happens that we start
		// on the first line of row r
		if lines == line_offset {
			break
		}

		// need to adjust line_offset
		// so we can start on the first
		// line of row r
		if lines > line_offset {
			line_offset = lines
			break
		}
	}
	return initial_row, line_offset
}

func (e *Editor) readFileIntoNote(filename string) error {

	r, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file %s: %w", filename, err)
	}
	defer r.Close()

	e.rows = nil
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		e.rows = append(e.rows, strings.ReplaceAll(scanner.Text(), "\t", " "))
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %s: %w", filename, err)
	}

	e.fr, e.fc, e.cy, e.cx, e.line_offset, e.prev_line_offset, e.first_visible_row, e.last_visible_row = 0, 0, 0, 0, 0, 0, 0, 0

	e.dirty++
	//sess.editor_mode = true;
	e.refreshScreen(true)
	return nil
}
