package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/slzatz/listmango/rawmode"
	"golang.org/x/sys/unix"
	"os"
	"strings"
)

type Session struct {
	screenCols       int
	screenLines      int
	textLines        int
	divider          int
	totaleditorcols  int
	initialFileRow   int
	temporaryTID     int
	lmBrowser        bool
	run              bool
	editors          []*Editor
	p                *Editor
	editorMode       bool
	fts_search_terms string
	//cfg config
	origTermCfg []byte //from GoKilo
	cfg         Config
}

type Config struct {
	user     string
	password string
	dbname   string
	hostaddr string
	port     int
	ed_pct   int
}

func contains(s []int, x int) bool {
	for _, y := range s {
		if x == y {
			return true
		}
	}
	return false
}

func (s *Session) eraseScreenRedrawLines() {
	fmt.Fprint(os.Stdout, "\x1b[2J") //Erase the screen
	fmt.Fprint(os.Stdout, "\x1b(0")  //Enter line drawing mode
	for j := 1; j < s.screenLines+1; j++ {
		fmt.Fprintf(os.Stdout, "\x1b[%d;%dH", TOP_MARGIN+j, s.divider)

		// x = 0x78 vertical line; q = 0x71 horizontal line
		// 37 = white; 1m = bold (note only need one 'm')
		fmt.Fprint(os.Stdout, "\x1b[37;1mx")
	}

	fmt.Fprint(os.Stdout, "\x1b[1;1H")
	for k := 1; k < s.screenCols; k++ {
		// cursor advances - same as char write
		fmt.Fprint(os.Stdout, "\x1b[37;1mq")
	}

	if s.divider > 10 {
		fmt.Fprintf(os.Stdout, "\x1b[%d;%dH", TOP_MARGIN, s.divider-TIME_COL_WIDTH+1)
		fmt.Fprint(os.Stdout, "\x1b[37;1mw") //'T' corner
	}

	// draw next column's 'T' corner - divider
	fmt.Fprintf(os.Stdout, "\x1b[%d;%dH", TOP_MARGIN, s.divider)
	fmt.Fprint(os.Stdout, "\x1b[37;1mw") //'T' corner

	fmt.Fprint(os.Stdout, "\x1b[0m") // return background to normal (? necessary)
	fmt.Fprint(os.Stdout, "\x1b(B")  //exit line drawing mode
}

func (s *Session) eraseRightScreen() {
	var ab strings.Builder

	ab.WriteString("\x1b[?25l") //hides the cursor

	//below positions cursor such that top line is erased the first time through
	//for loop although ? could really start on second line since need to redraw
	//horizontal line anyway
	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN, s.divider+1))

	//erase the screen
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", s.divider)
	for i := 0; i < s.screenLines-TOP_MARGIN; i++ {
		ab.WriteString("\x1b[K")
		ab.WriteString(lf_ret)
	}
	ab.WriteString("\x1b[K") //added 09302020 to erase the last line (message line)

	// redraw top horizontal line which has t's and was erased above
	// ? if the individual editors draw top lines do we need to just
	// erase but not draw
	ab.WriteString("\x1b(0")                   // Enter line drawing mode
	for j := 1; j < s.totaleditorcols+1; j++ { //added +1 0906/2020
		ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN, s.divider+j))
		// below x = 0x78 vertical line (q = 0x71 is horizontal) 37 = white;
		// 1m = bold (note only need one 'm'
		ab.WriteString("\x1b[37;1mq")
	}

	//exit line drawing mode
	ab.WriteString("\x1b(B")

	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+1, s.divider+2))
	ab.WriteString("\x1b[0m") // needed or else in bold mode from line drawing above

	fmt.Fprint(os.Stdout, ab.String())
}

func (s *Session) positionEditors() {
	editorSlots := 0
	for _, z := range s.editors {
		if !z.is_below {
			editorSlots++
		}
	}

	cols := -1 + (s.screenCols-s.divider)/editorSlots
	i := -1 //i = number of columns of editors -1
	for _, e := range s.editors {
		if !e.is_below {
			i++
		}
		e.left_margin = s.divider + i*cols + i
		e.screencols = cols
		e.setLinesMargins()
	}
}

func (s *Session) drawOrgRows() {

	if len(org.rows) == 0 {
		return
	}

	var j, k int //to swap highlight if org.highlight[1] < org.highlight[0]
	var ab strings.Builder
	titlecols := s.divider - TIME_COL_WIDTH - LEFT_MARGIN

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", LEFT_MARGIN)

	for y := 0; y < s.textLines; y++ {
		frr := y + org.rowoff
		if frr > len(org.rows)-1 {
			break
		}

		// if a line is long you only draw what fits on the screen
		//below solves problem when deleting chars from a scrolled long line

		//can run into this problem when deleting chars from a scrolled log line
		var length int
		if frr == org.fr {
			length = len(org.rows[frr].title) - org.coloff
		} else {
			length = len(org.rows[frr].title)
		}

		if length > titlecols {
			length = titlecols
		}

		if org.rows[frr].star {
			ab.WriteString("\x1b[1m") //bold
			ab.WriteString("\x1b[1;36m")
		}

		if org.rows[frr].completed && org.rows[frr].deleted {
			ab.WriteString("\x1b[32m") //green foreground
		} else if org.rows[frr].completed {
			ab.WriteString("\x1b[33m") //yellow foreground
			//else if (row.deleted) ab.append("\x1b[31m", 5); //red foreground
		} else if org.rows[frr].deleted {
			ab.WriteString(COLOR_1)
		} //red (specific color depends on theme)

		if frr == org.fr {
			ab.WriteString("\x1b[48;5;236m") // 236 is a grey
		}
		if org.rows[frr].dirty {
			ab.WriteString("\x1b[41m") //red background
		}
		//if (row.mark) ab.append("\x1b[46m", 5); //cyan background
		if _, ok := org.marked_entries[org.rows[frr].id]; ok {
			ab.WriteString("\x1b[46m")
		}

		// below - only will get visual highlighting if it's the active
		// then also deals with column offset
		if org.mode == VISUAL && frr == org.fr {

			// below in case org.highlight[1] < org.highlight[0]
			if org.highlight[1] > org.highlight[0] {
				j, k = 0, 1
			} else {
				k, j = 0, 1
			}

			ab.WriteString(org.rows[frr].title[org.coloff : org.highlight[j]-org.coloff])
			ab.WriteString("\x1b[48;5;242m")
			ab.WriteString(org.rows[frr].title[org.highlight[j] : org.highlight[k]-org.coloff])

			ab.WriteString("\x1b[49m") // return background to normal
			ab.WriteString(org.rows[frr].title[:org.highlight[k]])

		} else {
			// current row is only row that is scrolled if org.coloff != 0
			var beg int
			if frr == org.fr {
				beg = org.coloff
			}
			if len(org.rows[frr].title[beg:]) > length {
				ab.WriteString(org.rows[frr].title[beg : beg+length])
			} else {
				ab.WriteString(org.rows[frr].title[beg:])
			}
		}
		// the spaces make it look like the whole row is highlighted
		//note len can't be greater than titlecols so always positive
		ab.WriteString(strings.Repeat(" ", titlecols-length+1))

		// believe the +2 is just to give some space from the end of long titles
		//ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", y+TOP_MARGIN+1, s.divider-TIME_COL_WIDTH+2))
		fmt.Fprintf(&ab, "\x1b[%d;%dH", y+TOP_MARGIN+1, s.divider-TIME_COL_WIDTH+2)
		ab.WriteString(org.rows[frr].modified)
		ab.WriteString("\x1b[0m") // return background to normal ////////////////////////////////
		ab.WriteString(lf_ret)
	}
	//fmt.Fprint(os.Stdout, ab.String())
	fmt.Print(ab.String())
}

func (s *Session) drawOrgSearchRows() {

	if len(org.rows) == 0 {
		return
	}

	var ab strings.Builder
	titlecols := s.divider - TIME_COL_WIDTH - LEFT_MARGIN

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", LEFT_MARGIN)

	for y := 0; y < s.textLines; y++ {
		frr := y + org.rowoff
		if frr > len(org.rows)-1 {
			break
		}
		//orow& row = org.rows[frr];
		var length int

		//if (row.star) ab.append("\x1b[1m"); //bold
		if org.rows[frr].star {
			ab.WriteString("\x1b[1m") //bold
			ab.WriteString("\x1b[1;36m")
		}

		if org.rows[frr].completed && org.rows[frr].deleted {
			ab.WriteString("\x1b[32m") //green foreground
		} else if org.rows[frr].completed {
			ab.WriteString("\x1b[33m") //yellow foreground
		} else if org.rows[frr].deleted {
			ab.WriteString("\x1b[31m")
		} //red foreground

		if len(org.rows[frr].title) <= titlecols { // we know it fits
			ab.WriteString(org.rows[frr].fts_title)
		} else {
			pos := strings.Index(org.rows[frr].fts_title, "\x1b[49m")
			if pos < titlecols+10 { //length of highlight escape
				ab.WriteString(org.rows[frr].fts_title) //titlecols + 15); // length of highlight escape + remove formatting escape
			} else {
				ab.WriteString(org.rows[frr].title[:titlecols])
			}
		}
		if len(org.rows[frr].title) <= titlecols {
			length = len(org.rows[frr].title)
		} else {
			length = titlecols
		}
		spaces := titlecols - length
		ab.WriteString(strings.Repeat(" ", spaces))

		//snprintf(buf, sizeof(buf), "\x1b[%d;%dH", y + 2, screencols/2 - TIME_COL_WIDTH + 2); //wouldn't need offset
		ab.WriteString("\x1b[0m") // return background to normal
		ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", y+2, s.divider-TIME_COL_WIDTH+2))
		ab.WriteString(org.rows[frr].modified)
		ab.WriteString(lf_ret)
	}
	fmt.Print(ab.String())
}

func (s *Session) drawEditors() {
	var ab strings.Builder
	for _, e := range s.editors {
		//for (size_t i=0, max=editors.size(); i!=max; ++i) {
		//Editor *&e = editors.at(i);
		e.refreshScreen(true)
		ab.WriteString("\x1b(0") // Enter line drawing mode

		for j := 1; j < e.screenlines+1; j++ {
			ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", e.top_margin-1+j, e.left_margin+e.screencols+1))
			// below x = 0x78 vertical line (q = 0x71 is horizontal) 37 = white; 1m = bold (note
			// only need one 'm'
			ab.WriteString("\x1b[37;1mx")
		}

		if !e.is_below {
			//'T' corner = w or right top corner = k
			ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", e.top_margin-1, e.left_margin+e.screencols+1))

			if e.left_margin+e.screencols > s.screenCols-4 {
				ab.WriteString("\x1b[37;1mk") //draw corner
			} else {
				ab.WriteString("\x1b[37;1mw")
			}
		}

		//exit line drawing mode
		ab.WriteString("\x1b(B")
	}
	ab.WriteString("\x1b[?25h") //shows the cursor
	ab.WriteString("\x1b[0m")   //or else subsequent editors are bold

	fmt.Fprint(os.Stdout, ab.String())
}

//not in use at moment - using rawmode.GetWindowSize
func (s *Session) GetWindowSize() error {

	ws, err := unix.IoctlGetWinsize(unix.Stdout, unix.TIOCGWINSZ)
	if err != nil {
		//return 0, 0, fmt.Errorf("error fetching window size: %w", err)
		return fmt.Errorf("error in getWindowSize: %w", err)
	}
	if ws.Row == 0 || ws.Col == 0 {
		//return 0, 0, fmt.Errorf("Got a zero size column or row")
		return fmt.Errorf("Got a zero size column or row")
	}

	//return int(ws.Row), int(ws.Col), nil
	s.screenCols = int(ws.Col)
	s.screenLines = int(ws.Row)

	return nil
}

func (s *Session) enableRawMode() ([]byte, error) {

	// Gets TermIOS data structure. From glibc, we find the cmd should be TCGETS
	// https://code.woboq.org/userspace/glibc/sysdeps/unix/sysv/linux/tcgetattr.c.html
	termios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		return nil, fmt.Errorf("error fetching existing console settings: %w", err)
	}

	buf := bytes.Buffer{}
	if err := gob.NewEncoder(&buf).Encode(termios); err != nil {
		return nil, fmt.Errorf("error serializing existing console settings: %w", err)
	}

	// turn off echo & canonical mode by using a bitwise clear operator &^
	termios.Lflag = termios.Lflag &^ (unix.ECHO | unix.ICANON | unix.ISIG | unix.IEXTEN)
	termios.Iflag = termios.Iflag &^ (unix.IXON | unix.ICRNL | unix.BRKINT | unix.INPCK | unix.ISTRIP)
	termios.Oflag = termios.Oflag &^ (unix.OPOST)
	termios.Cflag = termios.Cflag | unix.CS8
	//termios.Cc[unix.VMIN] = 0
	//termios.Cc[unix.VTIME] = 1
	// from the code of tcsetattr in glibc, we find that for TCSAFLUSH,
	// the corresponding command is TCSETSF
	// https://code.woboq.org/userspace/glibc/sysdeps/unix/sysv/linux/tcsetattr.c.html
	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETSF, termios); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}

func Restore(original []byte) error {

	var termios unix.Termios

	if err := gob.NewDecoder(bytes.NewReader(original)).Decode(&termios); err != nil {
		return fmt.Errorf("error decoding terminal settings: %w", err)
	}

	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETSF, &termios); err != nil {
		return fmt.Errorf("error restoring original console settings: %w", err)
	}
	return nil
}

func (s *Session) refreshOrgScreen() {
	var ab strings.Builder
	titlecols := s.divider - TIME_COL_WIDTH - LEFT_MARGIN

	ab.WriteString("\x1b[?25l") //hides the cursor

	//char buf[20];

	//Below erase screen from middle to left - `1K` below is cursor to left erasing
	//Now erases time/sort column (+ 17 in line below)
	//if (org.view != KEYWORD) {
	if org.mode != ADD_CHANGE_FILTER {
		for j := TOP_MARGIN; j < s.textLines+1; j++ {
			ab.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[1K", j+TOP_MARGIN, titlecols+LEFT_MARGIN+17))
		}
	}
	// put cursor at upper left after erasing
	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+1, LEFT_MARGIN+1))

	//fmt.Fprint(os.Stdout, ab.String())
	fmt.Print(ab.String())

	if org.mode == FIND {
		s.drawOrgSearchRows()
		//} else if org.mode == ADD_CHANGE_FILTER {
		//  s.drawOrgFilters()
	} else {
		s.drawOrgRows()
	}
}

func (s *Session) showOrgMessage(format string, a ...interface{}) {
	fmt.Printf("\x1b[%d;%dH\x1b[1K\x1b[%d;1H", s.textLines+2+TOP_MARGIN, s.divider, s.textLines+2+TOP_MARGIN)
	str := fmt.Sprintf(format, a...)
	if len(str) > s.divider {
		str = str[:s.divider]
	}
	fmt.Print(str)
}

func (s *Session) showEdMessage(format string, a ...interface{}) {
	fmt.Printf("\x1b[%d;%dH\x1b[K", s.textLines+2+TOP_MARGIN, s.divider+1)
	str := fmt.Sprintf(format, a...)

	cols := s.screenCols - s.divider
	if len(str) > cols {
		str = str[:cols]
	}
	fmt.Print(str)
}

func (s *Session) drawOrgStatusBar() {

	var ab strings.Builder
	//position cursor and erase - and yes you do have to reposition cursor after erase
	fmt.Fprintf(&ab, "\x1b[%d;%dH\x1b[1K\x1b[%d;1H", s.textLines+TOP_MARGIN+1, s.divider, s.textLines+TOP_MARGIN+1)
	ab.WriteString("\x1b[7m") //switches to inverted colors

	var str string

	switch org.view {
	case TASK:
		switch org.taskview {
		case BY_FIND:
			str = "search - " + s.fts_search_terms
		case BY_FOLDER:
			str = org.folder + "[f]"
		case BY_CONTEXT:
			str = org.context + "[c]"
		case BY_RECENT:
			str = "recent"
		case BY_JOIN:
			str = org.context + "[c] + " + org.folder + "[f]"
		case BY_KEYWORD:
			str = org.keyword + "[k]"
		}
	case CONTEXT:
		str = "Contexts"
	case FOLDER:
		str = "Folders"
	case KEYWORD:
		str = "Keywords"
	}

	var length int
	var status string
	if len(org.rows) > 0 {

		r := &org.rows[org.fr]

		var title string
		if len((*r).title) > 12 {
			title = (*r).title[:12]
		} else {
			title = (*r).title
		}
		//if (p->dirty) truncated_title.append( "[+]"); /****this needs to be in editor class*******/

		// needs to be here because org.rows could be empty
		var keywords string
		if org.view == TASK {
			keywords = getTaskKeywords((*r).id)
		}

		// because video is reversted [42 sets text to green and 49 undoes it
		// also [0;35;7m -> because of 7m it reverses background and foreground
		// I think the [0;7m is revert to normal and reverse video
		status = fmt.Sprintf("\x1b[1m%s\x1b[0;7m %s...\x1b[0;35;7m %s \x1b[0;7m %d %d/%d \x1b[1;42m%s\x1b[49m",
			str, title, keywords, (*r).id, org.fr+1, len(org.rows), mode_text[org.mode])

		// klugy way of finding length of string without the escape characters
		length = len(fmt.Sprintf("%s %s... %s  %d %d/%d %s",
			str, title, keywords, (*r).id, org.fr+1, len(org.rows), mode_text[org.mode]))
	} else {

		status = fmt.Sprintf("\x1b[1m%s\x1b[0;7m %.15s...\x1b[0;35;7m %s \x1b[0;7m %d %d/%d \x1b[1;42m%s\x1b[49m",
			str, "   No Results   ", -1, 0, 0, mode_text[org.mode])
		length = len(fmt.Sprintf("%s %.15s... %d %d/%zu %s",
			str, "   No Results   ", -1, 0, 0, mode_text[org.mode]))
	}

	if length < s.divider {
		// need to do the below because the escapes make string
		// longer than it actually prints so pad separately
		fmt.Fprintf(&ab, "%s%-*s", status, s.divider-length, " ")
	} else {
		ab.WriteString(status[:s.divider])
	}
	ab.WriteString("\x1b[0m") //switches back to normal formatting
	fmt.Print(ab.String())
}

func (s *Session) returnCursor() {
	var ab strings.Builder
	if s.editorMode {
		// the lines below position the cursor where it should go
		if s.p.mode != COMMAND_LINE {
			ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", s.p.cy+s.p.top_margin, s.p.cx+s.p.left_margin+s.p.left_margin_offset+1))
		} else { //E.mode == COMMAND_LINE
			ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", s.textLines+TOP_MARGIN+2, len(s.p.command_line)+s.divider+2))
			ab.WriteString("\x1b[?25h") // show cursor
		}
	} else {
		if org.mode == ADD_CHANGE_FILTER {
			ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", org.cy+TOP_MARGIN+1, s.divider+1))
		} else if org.mode == FIND {
			ab.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[1;34m>", org.cy+TOP_MARGIN+1, LEFT_MARGIN)) //blue
		} else if org.mode != COMMAND_LINE {
			ab.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[1;31m>", org.cy+TOP_MARGIN+1, LEFT_MARGIN))
			// below restores the cursor position based on org.cx and org.cy + margin
			ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", org.cy+TOP_MARGIN+1, org.cx+LEFT_MARGIN+1))
		} else { //org.mode == COMMAND_LINE
			ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", s.textLines+2+TOP_MARGIN, len(org.command_line)+LEFT_MARGIN+1))
		}
	}
	ab.WriteString("\x1b[0m")   //return background to normal
	ab.WriteString("\x1b[?25h") //shows the cursor
	fmt.Print(ab.String())
}

func (s *Session) drawPreviewWindow(id int) { //get_preview

	if org.taskview != BY_FIND {
		s.drawPreviewText()
	} else {
		s.drawSearchPreview()
	}
	s.drawPreviewBox()

	/*
	  if (lm_browser) {
	    int folder_tid = getFolderTid(org.rows.at(org.fr).id);
	    if (!(folder_tid == 18 || folder_tid == 14)) updateHTMLFile("assets/" + CURRENT_NOTE_FILE);
	    else updateHTMLCodeFile("assets/" + CURRENT_NOTE_FILE);
	  }
	*/
}
func (s *Session) drawSearchPreview() {
	var ab strings.Builder
	width := s.totaleditorcols - 10
	length := s.textLines - 10
	//hide the cursor
	ab.WriteString("\x1b[?25l")
	fmt.Fprintf(&ab, "\x1b[%d;%dH", TOP_MARGIN+6, s.divider+6)
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", s.divider+6)

	erase_chars := fmt.Sprintf("\x1b[%dX", s.totaleditorcols-10)

	for i := 0; i < length-1; i++ {
		ab.WriteString(erase_chars)
		ab.WriteString(lf_ret)
	}

	fmt.Fprintf(&ab, "\x1b[%d;%dH", TOP_MARGIN+6, s.divider+7)
	fmt.Fprintf(&ab, "\x1b[2*x\x1b[%d;%d;%d;%d;48;5;235$r\x1b[*x",
		TOP_MARGIN+6, s.divider+7, TOP_MARGIN+4+length, s.divider+7+width)
	ab.WriteString("\x1b[48;5;235m")
	note := readNoteIntoString(org.rows[org.fr].id)
	var t string
	if note != "" {
		t = generateWWString(note, width, length, "\f")
		wp := getNoteSearchPositions(org.rows[org.fr].id)
		t = highlight_terms_string(t, wp)
	}
	t = strings.ReplaceAll(t, "\f", lf_ret)
	ab.WriteString(t)
	fmt.Print(ab.String())
}

func (s *Session) drawPreviewText() { //draw_preview

	var ab strings.Builder

	width := s.totaleditorcols - 10
	length := s.textLines - 10
	//hide the cursor
	ab.WriteString("\x1b[?25l")
	fmt.Fprintf(&ab, "\x1b[%d;%dH", TOP_MARGIN+6, s.divider+6)

	//ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+6, s.divider+7))

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", s.divider+6)
	//erase set number of chars on each line
	erase_chars := fmt.Sprintf("\x1b[%dX", s.totaleditorcols-10)

	for i := 0; i < length-1; i++ {
		ab.WriteString(erase_chars)
		ab.WriteString(lf_ret)
	}

	fmt.Fprintf(&ab, "\x1b[%d;%dH", TOP_MARGIN+6, s.divider+7)
	fmt.Fprintf(&ab, "\x1b[2*x\x1b[%d;%d;%d;%d;48;5;235$r\x1b[*x",
		TOP_MARGIN+6, s.divider+7, TOP_MARGIN+4+length, s.divider+7+width)

	ab.WriteString("\x1b[48;5;235m")
	note := readNoteIntoString(org.rows[org.fr].id)
	if note != "" {
		ab.WriteString(generateWWString(note, width, length, lf_ret))
	}
	fmt.Print(ab.String())
}

// being used for synchronize right now
func (s *Session) drawPreviewText2(text string) { //draw_preview

	var ab strings.Builder

	width := s.totaleditorcols - 10
	length := s.textLines - 10
	//hide the cursor
	ab.WriteString("\x1b[?25l")
	fmt.Fprintf(&ab, "\x1b[%d;%dH", TOP_MARGIN+6, s.divider+6)

	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+6, s.divider+7))

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", s.divider+6)
	//erase set number of chars on each line
	erase_chars := fmt.Sprintf("\x1b[%dX", s.totaleditorcols-10)

	for i := 0; i < length-1; i++ {
		ab.WriteString(erase_chars)
		ab.WriteString(lf_ret)
	}

	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+6, s.divider+7))

	ab.WriteString(fmt.Sprintf("\x1b[2*x\x1b[%d;%d;%d;%d;48;5;235$r\x1b[*x",
		TOP_MARGIN+6, s.divider+7, TOP_MARGIN+4+length, s.divider+7+width))

	ab.WriteString("\x1b[48;5;235m")
	//note := readNoteIntoString(org.rows[org.fr].id)
	if text != "" {
		ab.WriteString(generateWWString(text, width, length, lf_ret))
	}
	fmt.Print(ab.String())
}
func (s *Session) displayEntryInfo(e *Entry) {
	var ab strings.Builder
	width := s.totaleditorcols - 10
	length := s.textLines - 10

	// \x1b[NC moves cursor forward by N columns
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", s.divider+6)

	//hide the cursor
	ab.WriteString("\x1b[?25l")
	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+6, s.divider+7))

	//erase set number of chars on each line
	erase_chars := fmt.Sprintf("\x1b[%dX", s.totaleditorcols-10)
	for i := 0; i < length-1; i++ {
		ab.WriteString(erase_chars)
		ab.WriteString(lf_ret)
	}

	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+6, s.divider+7))

	ab.WriteString(fmt.Sprintf("\x1b[2*x\x1b[%d;%d;%d;%d;48;5;235$r\x1b[*x",
		TOP_MARGIN+6, s.divider+7, TOP_MARGIN+4+length, s.divider+7+width))
	ab.WriteString("\x1b[48;5;235m") //draws the box lines with same background as above rectangle

	ab.WriteString(fmt.Sprintf("id: %d%s", e.id, lf_ret))
	ab.WriteString(fmt.Sprintf("tid: %d%s", e.tid, lf_ret))

	title := fmt.Sprintf("title: %s", e.title)
	if len(title) > width {
		title = title[:width-3] + "..."
	}
	//coloring labels will take some work b/o gray background
	//s.append(fmt::format("{}title:{} {}{}", COLOR_1, "\x1b[m", title, lf_ret));
	ab.WriteString(fmt.Sprintf("%s%s", title, lf_ret))

	var context string
	for k, v := range org.context_map {
		if v == e.context_tid {
			context = k
			break
		}
	}
	ab.WriteString(fmt.Sprintf("context: %s%s", context, lf_ret))

	var folder string
	for k, v := range org.folder_map {
		if v == e.folder_tid {
			folder = k
			break
		}
	}
	ab.WriteString(fmt.Sprintf("folder: %s%s", folder, lf_ret))

	ab.WriteString(fmt.Sprintf("star: %t%s", e.star, lf_ret))
	ab.WriteString(fmt.Sprintf("deleted: %t%s", e.deleted, lf_ret))

	var completed bool
	// may be NULL
	if e.completed.Valid {
		completed = true
	} else {
		completed = false
	}

	ab.WriteString(fmt.Sprintf("completed: %t%s", completed, lf_ret))
	ab.WriteString(fmt.Sprintf("modified: %s%s", e.modified, lf_ret))
	ab.WriteString(fmt.Sprintf("added: %s%s", e.added, lf_ret))

	ab.WriteString(fmt.Sprintf("keywords: %s%s", getTaskKeywords(getId()), lf_ret))

	fmt.Print(ab.String())
	// display_item_info_pg needs to be updated if it is going to be used
	//if (tid) display_item_info_pg(tid); //// ***** remember to remove this guard
}

func (s *Session) drawPreviewBox() {
	width := s.totaleditorcols - 10
	length := s.textLines - 10
	var ab strings.Builder
	move_cursor := fmt.Sprintf("\x1b[%dC", width)

	ab.WriteString("\x1b(0") // Enter line drawing mode
	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+5, s.divider+6))
	ab.WriteString("\x1b[37;1ml") //upper left corner

	for i := 1; i < length; i++ {
		ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+5+i, s.divider+6))
		// x=0x78 vertical line (q=0x71 is horizontal) 37=white; 1m=bold (only need 1 m)
		ab.WriteString("\x1b[37;1mx")
		ab.WriteString(move_cursor)
		ab.WriteString("\x1b[37;1mx")
	}

	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+4+length, s.divider+6))
	ab.WriteString("\x1b[1B")
	ab.WriteString("\x1b[37;1mm") //lower left corner

	move_cursor = fmt.Sprintf("\x1b[1D\x1b[%dB", length)

	for i := 1; i < width+1; i++ {
		ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+5, s.divider+6+i))
		ab.WriteString("\x1b[37;1mq")
		ab.WriteString(move_cursor)
		ab.WriteString("\x1b[37;1mq")
	}

	ab.WriteString("\x1b[37;1mj") //lower right corner
	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+5, s.divider+7+width))
	ab.WriteString("\x1b[37;1mk") //upper right corner

	//exit line drawing mode
	ab.WriteString("\x1b(B")
	ab.WriteString("\x1b[0m")
	ab.WriteString("\x1b[?25h")
	fmt.Print(ab.String())
}

func (s *Session) quitApp() {
	fmt.Print("\x1b[2J\x1b[H") //clears the screen and sends cursor home
	//Py_FinalizeEx();
	//sqlite3_close(S.db); //something should probably be done here
	//PQfinish(conn);
	//lsp_shutdown("all");

	if err := rawmode.Restore(sess.origTermCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: disabling raw mode: %s\r\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func (s *Session) moveDivider(pct int) {
	// note below only necessary if window resized or font size changed
	s.textLines = s.screenLines - 2 - TOP_MARGIN

	if pct == 100 {
		s.divider = 1
	} else {
		s.divider = s.screenCols - pct*s.screenCols/100
	}
	s.totaleditorcols = s.screenCols - s.divider - 2 //? OUTLINE MARGINS?

	s.eraseScreenRedrawLines()

	if s.divider > 10 { //////////////////////////////////////////////////////
		s.refreshOrgScreen()
		s.drawOrgStatusBar()
	}

	if s.editorMode {
		s.positionEditors()
		s.eraseRightScreen() //erases editor area + statusbar + msg
		s.drawEditors()
	} else if org.view == TASK && org.mode != NO_ROWS {
		s.drawPreviewWindow(org.rows[org.fr].id) //get_preview
	}
	s.showOrgMessage("rows: %d  cols: %d  divider: %d", s.screenLines, s.screenCols, s.divider)

	s.returnCursor()
}

func (s *Session) signalHandler() {
	//s.GetWindowSize()
	var err error
	s.screenLines, s.screenCols, err = rawmode.GetWindowSize()
	if err != nil {
		//SafeExit(fmt.Errorf("couldn't get window size: %v", err))
		os.Exit(1)
	}
	//that percentage should be in session
	// so right now this reverts back if it was changed during session
	//s.moveDivider(s.cfg.ed_pct)
	s.moveDivider(60)
}

/*
func (s Session) moveDivider()
func (s Session) drawOrgFilters
func (s Session) displayContainerInfo
func (s Session) updateCodeFile()
func (s Session) loadMeta()
func (s Session) quitApp()
*/
