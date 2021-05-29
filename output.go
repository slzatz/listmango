package main

import (
	"fmt"
	"strings"
)

type Output struct {
	lineOffset         int //first row based on user scroll
	screenlines        int //number of lines for this Editor
	screencols         int //number of columns for this Editor
	left_margin        int //can vary (so could TOP_MARGIN - will do that later
	left_margin_offset int
	top_margin         int
	first_visible_row  int
	last_visible_row   int
	is_below           bool
	rows               []string
}

func NewOutput() *Output {
	return &Output{
		lineOffset:        0, //the number of lines of text at the top scrolled off the screen
		first_visible_row: 0,
		is_below:          false,
	}
}

func (o *Output) drawText() {
	// probably unnecessary
	if len(o.rows) == 0 {
		o.drawStatusBar()
		return
	}
	var ab strings.Builder

	fmt.Fprintf(&ab, "\x1b[?25l\x1b[%d;%dH", o.top_margin, o.left_margin+1)
	// \x1b[NC moves cursor forward by n columns
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", o.left_margin)
	erase_chars := fmt.Sprintf("\x1b[%dX", o.screencols)
	for i := 0; i < o.screenlines; i++ {
		ab.WriteString(erase_chars)
		ab.WriteString(lf_ret)
	}

	// format for positioning cursor is "\x1b[%d;%dh"
	fmt.Fprintf(&ab, "\x1b[%d;%dH", o.top_margin, o.left_margin+1)

	y := 0
	filerow := o.first_visible_row
	flag := false

	for {

		if flag {
			break
		}

		if filerow == len(o.rows) {
			break
		}

		row := o.rows[filerow]

		if len(row) == 0 {
			if y == o.screenlines-1 {
				break
			}
			ab.WriteString(lf_ret)
			filerow++
			y++
			continue
		}

		pos := 0
		prev_pos := 0 //except for start -> pos + 1
		for {
			/* this is needed because it deals where the end of the line doesn't have a space*/
			if prev_pos+o.screencols-o.left_margin_offset > len(row)-1 { //? if need -1;cpp generatewwstring had it
				ab.WriteString(row[prev_pos:])
				if y == o.screenlines-1 {
					flag = true
					break
				}
				ab.WriteString(lf_ret)
				y++
				filerow++
				break
			}

			pos = strings.LastIndex(row[:prev_pos+o.screencols-o.left_margin_offset], " ")

			if pos == -1 || pos == prev_pos-1 {
				pos = prev_pos + o.screencols - o.left_margin_offset - 1
			}

			ab.WriteString(row[prev_pos : pos+1]) //? pos+1
			if y == o.screenlines-1 {
				flag = true
				break
			}
			ab.WriteString(lf_ret)
			prev_pos = pos + 1
			y++
		}
	}
	fmt.Print(ab.String())
	//o.drawStatusBar()
}

func (o *Output) drawStatusBar() {
	var ab strings.Builder
	fmt.Fprintf(&ab, "\x1b[%d;%dH", o.screenlines+o.top_margin, o.left_margin+1)

	//erase from start of an Editor's status bar to the end of the Editor's status bar
	fmt.Fprintf(&ab, "\x1b[%dX", o.screencols)

	ab.WriteString("\x1b[7m ") //switches to inverted colors

	/*
		title := getTitle(e.id)
		if len(title) > 30 {
			title = title[:30]
		}
		status := fmt.Sprintf("%d - %s ... %s", e.id, title, sub)
	*/

	status := "Output"

	if len(status) > o.screencols-1 {
		status = status[:o.screencols-1]
	}
	fmt.Fprintf(&ab, "%-*s", o.screencols, status)
	ab.WriteString("\x1b[0m") //switches back to normal formatting
	fmt.Print(ab.String())
}

func (o *Output) setLinesMargins() { //also sets top margin

	if o.is_below {
		o.screenlines = LINKED_NOTE_HEIGHT
		o.top_margin = sess.textLines - LINKED_NOTE_HEIGHT + 2
	} else {
		o.screenlines = sess.textLines
		o.top_margin = TOP_MARGIN + 1
	}
}

func (o *Output) drawFrame() {
	var ab strings.Builder
	ab.WriteString("\x1b(0") // Enter line drawing mode

	for j := 1; j < o.screenlines+1; j++ {
		fmt.Fprintf(&ab, "\x1b[%d;%dH", o.top_margin-1+j, o.left_margin+o.screencols+1)
		// below x = 0x78 vertical line (q = 0x71 is horizontal) 37 = white; 1m = bold (note
		// only need one 'm'
		ab.WriteString("\x1b[37;1mx")
	}

	//'T' corner = w or right top corner = k
	fmt.Fprintf(&ab, "\x1b[%d;%dH", o.top_margin-1, o.left_margin+o.screencols+1)

	if o.left_margin+o.screencols > sess.screenCols-4 {
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
