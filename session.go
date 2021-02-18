package main

import (
  "fmt"
  "os"
	"bytes"
  "strings"
	"encoding/gob"
	"golang.org/x/sys/unix"
)

type Session struct {
  screenCols int
  screenLines int
  textLines int
  divider int
  totalEditorCols int
  initialFileRow int
  temporaryTID int
  lmBrowser bool
  run bool
  editors []*Editors
  p *Editor
  editor_mode bool
  ftsSearchTerms string
  cfg config
}

func contains(s []int, x int) bool {
  for _, y := range s {
    if x == y {
      return true
    }
  }
    return false
}

func (s Session) eraseScreenRedrawLines() {
  fmt.Fprint(os.Stdout, "\x1b[2J") //Erase the screen
  fmt.Fprint(os.Stdout, "\x1b(0") //Enter line drawing mode
  for j := 1; j < s.screenLines + 1; j++ {
    fmt.Fprintf(os.Stdout, "\x1b[%d;%dH", topMargin + j, s.divider)

    // x = 0x78 vertical line; q = 0x71 horizontal line
    // 37 = white; 1m = bold (note only need one 'm')
    fmt.Fprint(os.Stdout, "\x1b[37;1mx")
  }

  fmt.Fprint(os.Stdout, "\x1b[1;1H")
  for k := 1; k < s.screencols; k++ {
    // cursor advances - same as char write 
    fmt.Fprint(os.Stdout, "\x1b[37;1mq")
  }

  if divider > 10 {
    fmt.Fprintf(os.Stdout, "\x1b[%d;%dH", topMargin , s.divider - timeColWidth + 1)
    fmt.Fprint(os.Stdout, "\x1b[37;1mw") //'T' corner
  }

  // draw next column's 'T' corner - divider
  fmt.Fprintf(os.Stdout, "\x1b[%d;%dH", topMargin , s.divider)
  fmt.Fprint(os.Stdout, "\x1b[37;1mw") //'T' corner

  fmt.Fprint(os.Stdout, "\x1b[0m") // return background to normal (? necessary)
  fmt.Fprint(os.Stdout, "\x1b(B") //exit line drawing mode
}

func (s Session) eraseRightScreen() {
  var ab strings.Builder

  ab.WriteString("\x1b[?25l") //hides the cursor

  //below positions cursor such that top line is erased the first time through
  //for loop although ? could really start on second line since need to redraw
  //horizontal line anyway
  ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN, divider + 1))

  //erase the screen
  lf_ret := fmt.Sprintf("\r\n\x1b[%dC", divider)
  for i := 0; i < screenLines - topMargin; i++ {
    ab.WriteString("\x1b[K")
    ab.WriteString(lf_ret)
  }
    ab.WriteString("\x1b[K"); //added 09302020 to erase the last line (message line)

  // redraw top horizontal line which has t's and was erased above
  // ? if the individual editors draw top lines do we need to just
  // erase but not draw
  ab.WriteString("\x1b(0") // Enter line drawing mode
  for j := 1; j < totaleditorcols + 1; j++ { //added +1 0906/2020
    ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN, divider + j))
    // below x = 0x78 vertical line (q = 0x71 is horizontal) 37 = white;
    // 1m = bold (note only need one 'm'
    ab.WriteString("\x1b[37;1mq")
  }

  //exit line drawing mode
  ab.WriteString("\x1b(B")

  ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN + 1, divider + 2))
  ab.WriteString("\x1b[0m") // needed or else in bold mode from line drawing above

  fmt.Fprint(os.Stdout, ab.String())
}

func (s Session) positionEditors() {
  editorSlots := 0
  for _, z := range s.editors {
    if !z.is_below {editorSlots++}
  }

  cols := -1 + (s.screencols - s.divider)/editorSlots;
  i := -1; //i = number of columns of editors -1
  for _, e := range s.editors {
    if !e.is_below {i++}
    e.left_margin = s.divider + i*cols + i
    e.screencols = cols;
    e.setLinesMargins();
  }
}

func (s Session) drawOrgRows() {

  if len(org.rows) == 0  {return}

  var j, k int //to swap highlight if org.highlight[1] < org.highlight[0]
  var ab strings.Builder
  titlecols := s.divider - TIME_COL_WIDTH - LEFT_MARGIN;

  lf_ret := fmt.Sprintf("\r\n\x1b[%dC", LEFT_MARGIN)

  for y := 0; y < s.textlines; y++ {
    frr := y + org.rowoff
    if frr > len(org.rows - 1) {break}

    // if a line is long you only draw what fits on the screen
    //below solves problem when deleting chars from a scrolled long line

    //can run into this problem when deleting chars from a scrolled log line
    if frr == org.fr {
      length = len(org.rows[frr].title) - org.coloff
    } else {
     length = len(org.rows[frr].title)
    }

    if length > titlecols {length = titlecols}

    if org.rows[frr].star {
      ab.WriteString("\x1b[1m"); //bold
      ab.WriteString("\x1b[1;36m");
    }

    if org.rows[frr].completed && org.rows[frr].deleted {
      ab.WriteString("\x1b[32m") //green foreground
    } else if org.rows[frr].completed {
      ab.WriteString("\x1b[33m") //yellow foreground
    //else if (row.deleted) ab.append("\x1b[31m", 5); //red foreground
    } else if org.rows[frr].deleted {
      ab.WriteString(COLOR_1)
    } //red (specific color depends on theme)

    if frr == org.fr ab.WriteString("\x1b[48;5;236m"); // 236 is a grey
    if org.rows[frr].dirty {ab.WriteString("\x1b[41m")} //red background
    //if (row.mark) ab.append("\x1b[46m", 5); //cyan background
    if contains(org.marked_entries, org.rows[frr].id)  {ab.WriteString("\x1b[46m")}

    // below - only will get visual highlighting if it's the active
    // then also deals with column offset
    if (org.mode == VISUAL && frr == org.fr) {

       // below in case org.highlight[1] < org.highlight[0]
      k = (org.highlight[1] > org.highlight[0]) ? 1 : 0;
      j =!k;
      ab.WriteString(org.rows[frr].title[:org.coloff:org.highlight[j] - org.coloff])
      ab.WriteString("\x1b[48;5;242m")
      ab.WriteString(org.rows[frr].title[:org.highlight[j]:org.highlight[k] - org.coloff])

      ab.WriteString("\x1b[49m") // return background to normal
      ab.WriteString(row.title[:org.highlight[k]])

    } else {
        // current row is only row that is scrolled if org.coloff != 0
        ab.WriteString(&org.rows[frr].title[((frr == org.fr) ? org.coloff : 0)], len);
    }

    // the spaces make it look like the whole row is highlighted
    //note len can't be greater than titlecols so always positive
    ab.WriteString(strings.Repeat(" ", titlecols - len + 1))

    //snprintf(buf, sizeof(buf), "\x1b[%d;%dH", y + 2, org.divider - TIME_COL_WIDTH + 2); // + offset
    // believe the +2 is just to give some space from the end of long titles
    ab.WriteString("\x1b[%d;%dH", y + TOP_MARGIN + 1, s.divider - TIME_COL_WIDTH + 2)
    ab.WriteString(org.rows[frr].modified);
    ab.WriteString("\x1b[0m") // return background to normal ////////////////////////////////
    ab.WriteString(lf_ret)
  }
  fmt.Fprint(os.Stdout, ab.String())
}

func (s Session) drawOrgSearchRows() {

  if len(org.rows) == 0 { return}

  var ab strings.Builder
  titlecols := s.divider - TIME_COL_WIDTH - LEFT_MARGIN;

  lf_ret := fmt.Sprintf("\r\n\x1b[%dC", LEFT_MARGIN)

  for y := 0; y < textlines; y++ {
    frr := y + org.rowoff;
    if frr > len(org.rows - 1) {break}
    //orow& row = org.rows[frr];
    length int

    //if (row.star) ab.append("\x1b[1m"); //bold
    if (org.rows[frr].star) {
      ab.WriteString("\x1b[1m") //bold
      ab.WriteString("\x1b[1;36m")
    }

    if (org.rows[frr].completed && org.rows[frr].deleted) {ab.WriteString("\x1b[32m"}) //green foreground
    else if (org.rows[frr].completed) {ab.WriteString("\x1b[33m")} //yellow foreground
    else if (org.rows[frr].deleted) {ab.WriteString("\x1b[31m")} //red foreground

    if (len(org.rows[frr].title) <= titlecols) {// we know it fits
      ab.WriteString(org.rows[frr].fts_title.c_str(), org.rows[frr].fts_title.size());
    } else {
      size_t pos = org.rows[frr].fts_title.find("\x1b[49m");
      if (pos < titlecols + 10) {//length of highlight escape
        ab.WriteString(org.rows[frr].fts_title.c_str(), titlecols + 15); // length of highlight escape + remove formatting escape
      } else {
        ab.WriteString(org.rows[frr].title[:titlecols]);
      }
    }
    if len(org.rows[frr].title) <= titlecols {
      length = len(org.rows.[frr].title)
    } else {
      length = titlecols
    }
    spaces := titlecols - length
    for i := 0; i < spaces; i++  {ab.WriteString(" ", 1)}
    //snprintf(buf, sizeof(buf), "\x1b[%d;%dH", y + 2, screencols/2 - TIME_COL_WIDTH + 2); //wouldn't need offset
    ab.WriteString("\x1b[0m") // return background to normal
    ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", y + 2, divider - TIME_COL_WIDTH + 2))
    ab.WriteString(org.rows[frr].modified)
    ab.WriteString(lf_ret)
  }
  fmt.Fprint(os.Stdout, ab.String())
}

func (s Session) drawEditors() {
  var ab strings.Builder
  for _, e := range s.editors {
  //for (size_t i=0, max=editors.size(); i!=max; ++i) {
    //Editor *&e = editors.at(i);
    e.editorRefreshScreen(true)
    ab.WriteString("\x1b(0") // Enter line drawing mode

    for j := 1; j < e.screenLines+1; j++ {
      ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", e.top_margin - 1 + j, e.left_margin + e.screencols+1))
      // below x = 0x78 vertical line (q = 0x71 is horizontal) 37 = white; 1m = bold (note
      // only need one 'm'
      ab.WriteString("\x1b[37;1mx")
    }

    if !e.is_below {
      //'T' corner = w or right top corner = k
      ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", e.top_margin - 1, e.left_margin + e.screencols+1))

      if e.left_margin + e.screencols > screencols - 4 {ab.WriteString("\x1b[37;1mk")} //draw corner
      else ab.WriteString("\x1b[37;1mw")
    }

    //exit line drawing mode
    ab.WriteString("\x1b(B")
  }
  ab.WriteString("\x1b[?25h") //shows the cursor
  ab.WriteString("\x1b[0m") //or else subsequent editors are bold

  fmt.Fprint(os.Stdout, ab.String())
}

func (s Session) GetWindowSize() error {

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
  s.screenCols  = int(ws.Col)
  s.screenLines = int(ws.Row)

  return nil
}

func (s Session) enableRawMode() ([]byte, error) {

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

func (s Session) refreshOrgScreen {
  var ab strings.Builder
  titlecols := s.divider - TIME_COL_WIDTH - LEFT_MARGIN;

  ab.WriteString("\x1b[?25l"); //hides the cursor

  //char buf[20];

  //Below erase screen from middle to left - `1K` below is cursor to left erasing
  //Now erases time/sort column (+ 17 in line below)
  //if (org.view != KEYWORD) {
  if (org.mode != ADD_CHANGE_FILTER) {
    for (unsigned int j=TOP_MARGIN; j < textlines + 1; j++) {
      ab.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[1K", j + TOP_MARGIN, titlecols + LEFT_MARGIN + 17))
    }
  }
  // put cursor at upper left after erasing
  ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN + 1 , LEFT_MARGIN + 1))

  //fmt.Fprint(os.Stdout, ab.String())
  fmt.Print(ab.String())

  if org.mode == FIND {
    drawOrgSearchRows()
  } else if org.mode == ADD_CHANGE_FILTER {
    drawOrgFilters()
  } else {
    drawOrgRows();
  }
}

func (s Session) showOrgMessage(format string, a ...interface{}) {
  fmt.Printf("\x1b[%d;%dH\x1b[1K\x1b[%d;1H", s.textlines + 2 + TOP_MARGIN, s.divider, s.textlines + 2 + TOP_MARGIN)
  s := fmt.Sprintf(format, a...)
  if len(s) > divider {
    s = s[:divider]
  }
  fmt.Print(s)
}

func (s Session) drawOrgStatusBar() {

  /*
  so the below should 1) position the cursor on the status
  bar row and midscreen and 2) erase previous statusbar
  r -> l and then put the cursor back where it should be
  at LEFT_MARGIN
  */

  var ab strings.Builder
  ab.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[1K\x1b[%d;1H", s.textlines + TOP_MARGIN + 1, s.divider, s.textlines + TOP_MARGIN + 1))
  ab.WriteString("\x1b[7m"); //switches to inverted colors
  char status[300], status0[300], rstatus[80];

  var s string

  switch org.view {
    case TASK:
      switch org.taskview {
        case BY_FIND:
          s =  "search - " + fts_search_terms
        case BY_FOLDER:
          s = org.folder + "[f]"
        case BY_CONTEXT:
          s = org.context + "[c]"
        case BY_RECENT:
          s = "recent"
        case BY_JOIN:
          s = org.context + "[c] + " + org.folder + "[f]"
        case BY_KEYWORD:
          s = org.keyword + "[k]"
      }
    case CONTEXT:
      s = "Contexts"
    case FOLDER:
      s = "Folders"
    case KEYWORD:
      s = "Keywords"
  }

  if len(org.rows) > 0 {

    r = &org.rows[org.fr]
    // note the format is for 15 chars - 12 from substring below and "[+]" when needed
    var title string
    if len((*r).title) > 12 {
      title = (*r).title[:12]
    } else {
      title = (*r).title
    }
    //if (p->dirty) truncated_title.append( "[+]"); /****this needs to be in editor class*******/

    // needs to be here because org.rows could be empty
    var keywords string
    if org.view == Task {
      keywords = getTaskKeywords((*r).id)
    }

    // because video is reversted [42 sets text to green and 49 undoes it
    // also [0;35;7m -> because of 7m it reverses background and foreground
    // I think the [0;7m is revert to normal and reverse video
    status := fmt.Sprintf( "\x1b[1m%s\x1b[0;7m %.15s...\x1b[0;35;7m %s \x1b[0;7m %d %d/%zu \x1b[1;42m%s\x1b[49m",
                              s, title, keywords, (*r).id, org.fr+1, len(org.rows), mode_text[org.mode])

    // klugy way of finding length of string without the escape characters
    length := len(fmt.Sprintf("%s %.15s... %s  %d %d/%zu %s",
                              s, title, keywords, (*r).id, org.fr+1, len(org.rows), mode_text[org.mode]))
  } else {

    status := fmt.Sprintf( "\x1b[1m%s\x1b[0;7m %.15s...\x1b[0;35;7m %s \x1b[0;7m %d %d/%zu \x1b[1;42m%s\x1b[49m",
                              s, "   No Results   ", -1, 0, 0, mode_text[org.mode])
    length := len(fmt.Sprintf( "%s %.15s... %d %d/%zu %s",
                              s, "   No Results   ", -1, 0, 0, mode_text[org.mode]))
  }

  if (length < s.divider) {
    ab.WriteString(status)
  } else {
    ab.WriteString(status[:s.divider]
  }
  ab.WriteString("\x1b[0m") //switches back to normal formatting
  fmt.Print(ab)
}

func (s Session) returnCursor(){
  var ab strings.Builder
  if s.editor_mode {
  // the lines below position the cursor where it should go
    if p->mode != COMMAND_LINE)
      ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", p->cy + p->top_margin, p->cx + p->left_margin + p->left_margin_offset + 1))
    } else { //E.mode == COMMAND_LINE
      ab.WriteString(fmt.Sprintf("\x1b[%d;%ldH", textlines + TOP_MARGIN + 2, p->command_line.size() + divider + 2))
      ab.WriteString("\x1b[?25h"); // show cursor
    }
  } else {
    if org.mode == ADD_CHANGE_FILTER {
      ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", org.cy + TOP_MARGIN + 1, divider + 1))
    } else if org.mode == FIND {
      ab.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[1;34m>", org.cy + TOP_MARGIN + 1, LEFT_MARGIN)) //blue
    } else if org.mode != COMMAND_LINE {
      ab.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[1;31m>", org.cy + TOP_MARGIN + 1, LEFT_MARGIN))
      // below restores the cursor position based on org.cx and org.cy + margin
      ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", org.cy + TOP_MARGIN + 1, org.cx + LEFT_MARGIN + 1))
    } else { //org.mode == COMMAND_LINE
      ab.WriteString(fmt.Sprintf("\x1b[%d;%ldH", textlines + 2 + TOP_MARGIN, org.command_line.size() + LEFT_MARGIN + 1))
    }
  }
  ab.WriteString("\x1b[0m"); //return background to normal
  ab.WriteString("\x1b[?25h"); //shows the cursor
  fmt.Print(ab)
}

func (s Session) moveDivider()

func (s Session) drawOrgFilters

func (s Session) displayContainerInfo

func (s Session) showOrgMessage()

func (s Session) updateCodeFile()

func (s Session) loadMeta()

func (s Session) quitApp()
