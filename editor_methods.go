package main

import (
       "bufio"
       "strings"
       "fmt"
       "os"
)

//var z0 = struct{}{}// in listmango
var line_commands = map[string]struct{} {
                                       "I":z0,
                                       "i":z0,
                                       "A":z0,
                                       "a":z0,
                                       "s":z0,
                                       "cw":z0,
                                       "caw":z0,
                                       "x":z0,
                                       "d$":z0,
                                       "daw":z0,
                                       "dw":z0,
                                       "r":z0,
                                       "~":z0,
}

func find_first_not_of(row *string, delimiters string, pos int) int {
  pos++
  for i,char := range (*row)[pos:] {
    if strings.Index(delimiters, string(char)) != -1 {
      continue
    } else {
      return pos + i
    }
  }
  return -1
}

func (e *Editor) showMessage(format string, a ...interface{}) {
  //fmt.Printf("\x1b[%d;%dH\x1b[0K\x1b[%d;%dH", sess.textLines + e.top_margin + 1, sess.divider, sess.textLines + e.top_margin + 1, sess.divider)
  fmt.Printf("\x1b[%d;%dH\x1b[K", sess.textLines + e.top_margin + 1, sess.divider + 1)
  str := fmt.Sprintf(format, a...)
  if len(str) > e.screencols {
    str = str[:e.screencols]
  }
  fmt.Print(str)
}

// a string is a sequence of uint8 == byte 
func (e *Editor) move_to_right_brace(left_brace byte) (int,int) {
  r := e.fr
  c := e.fc + 1
  count := 1
  max := len(e.rows)

  m := map[byte]byte{'{':'}', '(':')', '[':']'}
  right_brace := m[left_brace]

  for  {

    row := e.rows[r]

    // right now this function only called from NORMAL mode by typing '%'
    // note that function that deals with INSERT needs  c >= row.size() because
    // brace could be at end of line and fc could be row.size() before doing fc + 1
    if c == len(row) {
      r++
      if r == max {
        e.showMessage("Couldn't find matching brace");
        return e.fr,e.fc
      }
      c = 0
      continue;
    }

    if row[c] == right_brace {
      count -= 1
      if count == 0 {
        return r,c
      }
    } else if row[c] == left_brace {
      count += 1
    }
    c++
  }
}

func (e *Editor) move_to_left_brace(right_brace byte) (int,int) {
  r := e.fr
  c := e.fc - 1
  count := 1

  m := map[byte]byte{'}':'{', ')':'(', ']':'['}
  left_brace := m[right_brace]

  row := e.rows[r]

  for {

    if (c == -1) { //fc + 1 can be greater than row.size on first pass from INSERT if { at end of line
      r--;
      if (r == -1) {
        e.showMessage("Couldn't find matching brace");
        return e.fr,e.fc
      }
      row = e.rows[r]
      c = len(row) - 1
      continue;
    }

    if row[c] == left_brace {
      count -= 1
      if count == 0 {
      return r,c
    }
    } else if row[c] == right_brace {
      count += 1;
   }
    c--;
  }
}

//triggered by % in NORMAL mode
func (e *Editor) E_move_to_matching_brace(repeat int) {
  c := e.rows[e.fr][e.fc]
  left := "{([";
  i := strings.Index(left, string(c))
  if i != -1 {
    e.fr, e.fc = e.move_to_right_brace(c);
  } else {
    right := "})]";
    i = strings.Index(right, string(c))
    if i != -1 {
      e.fr,e.fc = e.move_to_left_brace(c);
    }
  }
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

  m := map[byte]byte{'{':'}', '(':')', '[':']'}
  right_brace := m[left_brace]

  //e.showMessage("left brace: {}", left_brace);
  for  {

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
      continue;
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
  fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%d", y + e.top_margin, x, right_brace)

  x = e.getScreenXFromRowColWW(e.fr, e.fc - b) + e.left_margin + e.left_margin_offset + 1
  y = e.getScreenYFromRowColWW(e.fr, e.fc - b) + e.top_margin - e.line_offset; // added line offset 12-25-2019
  fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%d\x1b[0m", y, x, left_brace)
  e.showMessage("r = %d   c = %d", r, c)
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

  m := map[byte]byte{'}':'{', ')':'(', ']':'['}
  left_brace := m[right_brace]

  for {

    if c == -1 { //fc + 1 can be greater than row.size on first pass from INSERT if { at end of line
      r--
      if r == -1 {
        e.showMessage("Couldn't find matching brace");
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
  fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%d", y + e.top_margin, x, right_brace)

  x = e.getScreenXFromRowColWW(e.fr, e.fc - b) + e.left_margin + e.left_margin_offset + 1
  y = e.getScreenYFromRowColWW(e.fr, e.fc - b) + e.top_margin - e.line_offset; // added line offset 12-25-2019
  fmt.Printf("\x1b[%d;%dH\x1b[48;5;244m%d\x1b[0m", y, x, right_brace)
  e.showMessage("r = %d   c = %d", r, c)
  return true
}

func (e *Editor) draw_highlighted_braces() {

  // below is code to automatically find matching brace - should be in separate member function
  braces := "{}()"
  var c byte
  var back bool
  //if below handles case when in insert mode and brace is last char
  //in a row and cursor is beyond that last char (which is a brace)
  if e.fc == len(e.rows[e.fr]) {
    c = e.rows[e.fr][e.fc-1]
    back = true;
  } else {
    c = e.rows[e.fr][e.fc]
    back = false;
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
      default://should not need this
        return
    }
  } else if ( e.fc > 0 && e.mode == INSERT ) {
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
          default://should not need this
            return
      }
    } else {e.redraw = false}
  } else {e.redraw = false}
}

func (e *Editor) setLinesMargins() { //also sets top margin

  if(e.linked_editor != nil) {
    if (e.is_subeditor) {
      if (e.is_below) {
        e.screenlines = LINKED_NOTE_HEIGHT;
        e.top_margin = sess.textLines - LINKED_NOTE_HEIGHT + 2;
      } else {
        e.screenlines = sess.textLines;
        e.top_margin =  TOP_MARGIN + 1;
      }
    } else {
      if (e.linked_editor.is_below) {
        e.screenlines = sess.textLines - LINKED_NOTE_HEIGHT - 1;
        e.top_margin =  TOP_MARGIN + 1;
      } else {
        e.screenlines = sess.textLines;
        e.top_margin =  TOP_MARGIN + 1;
      }
    }
  } else {
    e.screenlines = sess.textLines;
    e.top_margin =  TOP_MARGIN + 1;
  }
}

// normal mode 'e'
func (e *Editor) moveEndWord() {

if len(e.rows) == 0 {
  return
}

if ( len(e.rows[e.fr]) == 0 || e.fc == len(e.rows[e.fr]) - 1 ) {
  if e.fr + 1 > len(e.rows) - 1 {
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

    if r > len(e.rows) - 1 {
      return
    }

    row := &e.rows[r]

    if len(*row) == 0 {
      r++
      c = 0
      continue
    }

    if strings.Index(delimiters, string((*row)[c])) == -1 {
      if ( c == len(*row) - 1 || strings.Index(delimiters, string((*row)[c+1])) != -1 ) {
        e.fc = c
        e.fr = r
        return
      }

      pos = strings.IndexAny(string((*row)[c]), delimiters)
      if pos == -1 {
        e.fc = len(*row) - 1
        return
      } else {
        e.fr = r;
        e.fc = pos - 1
        return
      }

    // we started on punct or space
    } else {
      if (*row)[c] == ' ' {
        if c == len(*row) - 1 {
          r++
          c = 0
          continue
        } else {
          c++
          continue
        }
      } else {
        pos = find_first_not_of(row, delimiters_without_space, c);
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

func (e *Editor) moveCursor(key int) {

  switch key {
    case ARROW_LEFT, 'h':
      if e.fc > 0 {
        e.fc--
      }

    case ARROW_RIGHT, 'l':
      e.fc++

    case ARROW_UP, 'k':
      if e.fr > 0 {
        e.fr--
      }

    case ARROW_DOWN, 'j':
      if e.fr < len(e.rows) - 1 {
        e.fr++;
      }
  }
}

func (e *Editor) moveCursorEOL() {
  if len(e.rows[e.fr]) > 0 {
    e.fc = len(e.rows[e.fr]) - 1
  }
}

func (e *Editor) moveCursorBOL() {
  e.fc = 0
}

func (e *Editor) insertNewLine(direction int) {
  /* note this func does position fc and fr*/
  if len(e.rows) == 0 { // creation of NO_ROWS may make this unnecessary
    e.insertRow(0, "")
    return
  }

  if ( e.fr == 0 && direction == 0 ){ // this is for 'O'
    e.insertRow(0, "")
    e.fc = 0
    return
  }

  //int indent = (smartindent) ? editorIndentAmount(fr) : 0;
  indent := e.indentAmount(e.fr)
  spaces := strings.Repeat(" ", indent)

  e.fr += direction;
  e.insertRow(e.fr, spaces)
  e.fc = indent
}

func (e *Editor) insertRow(r int, s string) {
  e.rows = append(e.rows, "")
  copy(e.rows[r:], e.rows[r+1:])
  e.rows[r] = s
  e.dirty++
}

func (e *Editor) rowsToString() string {

  numRows := len(e.rows)
  if numRows == 0 {
    return ""
  }

  var sb strings.Builder
  for i := 0; i < numRows - 1; i++ {
      sb.WriteString(e.rows[i] + "\n")
  }
  sb.WriteString(e.rows[numRows - 1])
  return sb.String()
}

func (e *Editor) getScreenXFromRowColWW(r, c int) int {
  // can't use reference to row because replacing blanks to handle corner case
  row := e.rows[r]

  /* pos is the position of the last char in the line
   * and pos+1 is the position of first character of the next row
   */

  if len(row) <= e.screencols - e.left_margin_offset {
    return c
  }

  pos := -1;
  prev_pos := 0
  for  {

    if len(row[pos+1:]) <= e.screencols - e.left_margin_offset {
      prev_pos = pos
      break
  }

  prev_pos = pos
  //cpp find_last_of -the parameter defines the position from beginning to look at (inclusive)
  //need to add + 1 because slice :n includes chars up to the n - 1 char
  pos = strings.LastIndex(row[:pos + e.screencols - e.left_margin_offset + 1], " ")

  if pos == -1 {
      pos = prev_pos + e.screencols - e.left_margin_offset;
  } else if pos == prev_pos {
      row = strings.Replace(row[:pos+1], " ", "+", -1) + row[pos+1:]
      pos = prev_pos + e.screencols - e.left_margin_offset;
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

  if len(row) <= e.screencols - e.left_margin_offset {
    return 1
  }

  lines := 0;
  pos := -1; //pos is the position of the last character in the line (zero-based)
  prev_pos := 0

  for {

    // we know the first time around this can't be true
    // could add if (line > 1 && row.substr(pos+1).size() ...);
    if len(row[pos+1:]) <= e.screencols - e.left_margin_offset {
      lines++
      break
    }

    prev_pos = pos
   //cpp find_last_of -the parameter defines the position from beginning to look at (inclusive)
   //need to add + 1 because slice :n includes chars up to the n - 1 char
    pos = strings.LastIndex(row[:pos + e.screencols - e.left_margin_offset + 1], " ")

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

  if len(row) <= e.screencols - e.left_margin_offset {
    return 1
  }

  /* pos is the position of the last char in the line
   * and pos+1 is the position of first character of the next row
   */

  lines := 0;
  pos := -1; //pos is the position of the last character in the line (zero-based)
  prev_pos := 0
  for  {

    // we know the first time around this can't be true
    // could add if (line > 1 && row.substr(pos+1).size() ...);
    if len(row[pos+1:]) <= e.screencols - e.left_margin_offset {
      lines++
      break
    }

    prev_pos = pos;
    //cpp find_last_of -the parameter defines the position from beginning to look at (inclusive)
    //need to add + 1 because slice :n includes chars up to the n - 1 char
    pos = strings.LastIndex(row[:pos + e.screencols - e.left_margin_offset + 1], " ")

    if (pos == -1) {
        pos = prev_pos + e.screencols - e.left_margin_offset;

   // only replace if you have enough characters without a space to trigger this
   // need to start at the beginning each time you hit this
   // unless you want to save the position which doesn't seem worth it
    } else if pos == prev_pos {
      row = strings.Replace(row[:pos+1], " ", "+", -1) + row[pos+1:]
      pos = prev_pos + e.screencols - e.left_margin_offset;
    }

    lines++
    if pos >= c {
    break
    }
  }
  return lines
}

func (e *Editor) indentAmount(r int) int {
  if len(e.rows) == 0 {
    return 0
  }
  var i int
  row := e.rows[r]

  for i = 0; i < len(row); i++ {
    if row[i] != ' ' {
      break
    }
  }

  return i
}

func (e *Editor) insertReturn() { // right now only used for editor->INSERT mode->'\r'
  r := &e.rows[e.fr]
  r1 := (*r)[:e.fc] //(current_row.begin(), current_row.begin() + fc);
  r2 := (*r)[e.fc:] //(current_row.begin() + fc, current_row.end());

  //int indent = (e.smartindent) ? editorIndentAmount(fr) : 0;
  indent := 0
  if e.smartindent > 0 {
    indent = e.indentAmount(e.fr)
  }

  *r = r1

  e.rows = append(e.rows, "")
  copy(e.rows[e.fr + 1:], e.rows[e.fr:])
  e.rows[e.fr] = r2
  e.fr++


  if e.fc==0 {
    return
  }

  e.fc = 0
  for i := 0; i < indent; i++ {
    e.insertChar(' ')
  }
}

func (e *Editor) insertChar(chr int) {
  // does not handle returns which must be intercepted before calling this function
  // necessary even with NO_ROWS because putting new entries into insert mode
  if len(e.rows) == 0 {
    e.insertRow(0, "")
  }
  r := &e.rows[e.fr]
  //row.insert(row.begin() + fc, chr); // works if row is empty
  *r = (*r)[:e.fc] + string(chr) + (*r)[e.fc:]
  e.dirty++
  e.fc++
}

func (e *Editor) backspace() {

  if ( e.fc == 0 && e.fr == 0 ) {
    return
  }

  r := &e.rows[e.fr]
  if e.fc > 0 {
    *r = (*r)[:e.fc] + (*r)[e.fc + 1:]
    e.fc--
  } else if len(*r) > 1 {
    e.rows[e.fr-1] = e.rows[e.fr-1] + *r
    e.delRow(e.fr)
    e.fr--
    e.fc = len(e.rows[e.fr])
  } else {
    e.delRow(e.fr)
    e.fr--
    e.fc = len(e.rows[e.fr])
}
  e.dirty++;
}

func (e *Editor) delRow(r int) {
  if len(e.rows) == 0 {
    return // creation of NO_ROWS may make this unnecessary
  }

  copy(e.rows[r:], e.rows[r+1:])
  if len(e.rows) == 0 {
    e.fr, e.fc, e.cy, e.cx, e.line_offset, e.prev_line_offset, e.first_visible_row, e.last_visible_row = 0,0,0,0,0,0,0,0
    e.mode = NO_ROWS
  }

  e.dirty++
  //editorSetMessage("Row deleted = %d; numrows after deletion = %d cx = %d row[fr].size = %d", fr,
  //numrows, cx, row[fr].size); 
}

func (e *Editor) delChar() {
  if len(e.rows) == 0 {
    return // creation of NO_ROWS may make this unnecessary
  }
  r := &e.rows[e.fr]
  if ( len(*r) == 0 || e.fc > len(*r) - 1 ) {
    return
  }
  *r = (*r)[:e.fc] + (*r)[e.fc + 1:]
  e.dirty++
}

func (e *Editor) refreshScreen(draw bool) {
  var ab strings.Builder
  var tid int

  ab.WriteString("\x1b[?25l") //hides the cursor
  //ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", e.top_margin, e.left_margin + 1))
  fmt.Fprintf(&ab, "\x1b[%d;%dH", e.top_margin, e.left_margin + 1)

  if draw { //draw
    // \x1b[NC moves cursor forward by N columns
    lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)
    erase_chars := fmt.Sprintf("\x1b[%dX", e.screencols)
    for i := 0; i < e.screenlines; i++ {
      ab.WriteString(erase_chars)
      ab.WriteString(lf_ret)
    }

    // this must be here -- if at end it erases the rows that are drawn by the calls to drawRows and drawCodeRows below
    fmt.Print(ab.String())

    tid = getFolderTid(e.id)
    if ( (tid == 18 || tid == 14) && !e.is_subeditor ) {
      //e.drawCodeRows(ab) ///////////////////////////////////////////////////////////////////////////
      e.drawRows()
    } else {
      e.drawRows()
    }
  }

  e.drawStatusBar()
  //e.drawMessageBar()

  // the lines below position the cursor where it should go
  if e.mode != COMMAND_LINE {
    //ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", e.cy + e.top_margin, e.cx + e.left_margin + e.left_margin_offset + 1))
    fmt.Fprintf(&ab, "\x1b[%d;%dH", e.cy + e.top_margin, e.cx + e.left_margin + e.left_margin_offset + 1)
  }
  //fmt.Print(ab.String())

  /*
  // can't do the below until ab is written or will just overwite highlights
  if (draw && spellcheck) editorSpellCheck();

  if (rows.empty() || rows.at(fr).empty()) return;

  if (!tid) tid = getFolderTid(id);
  if ((tid == 18 || tid == 14) && !(is_subeditor)) draw_highlighted_braces();;
  */
}

func (e *Editor) drawRows() {

  if len(e.rows) == 0 {
    return
  }
  var ab strings.Builder

  lf_ret := fmt.Sprintf("\r\n\x1b[%dC", e.left_margin)
  ab.WriteString("\x1b[?25l") //hides the cursor

  // format for positioning cursor is "\x1b[%d;%dH"
  fmt.Fprintf(&ab, "\x1b[%d;%dH", e.top_margin, e.left_margin + 1)

  y := 0
  filerow := e.first_visible_row
  flag := false;

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
      if y == e.screenlines - 1 {
      break
    }
      ab.WriteString(lf_ret)
      filerow++
      y++
      continue
    }

    pos := 0
    prev_pos := 0 //except for start -> pos + 1
    for  {
      /* this is needed because it deals where the end of the line doesn't have a space*/
      if prev_pos + e.screencols - e.left_margin_offset > len(row) {
        ab.WriteString(row[prev_pos:])
        if y == e.screenlines - 1 {
          flag = true
          break
        }
        ab.WriteString(lf_ret)
        y++
        filerow++
        break
      }

      pos = strings.LastIndex(row[:prev_pos + e.screencols - e.left_margin_offset], " ")

      //note npos when signed = -1 and order of if/else may matter
      if pos == -1 || pos == prev_pos - 1 {
        pos = prev_pos + e.screencols - e.left_margin_offset - 1
      }

      ab.WriteString(row[prev_pos:pos+1]) //? pos+1
      if y == e.screenlines - 1 {
        flag = true
        break
      }
      ab.WriteString(lf_ret)
      prev_pos = pos + 1
      y++
    }
  }
  // ? only used so spellcheck stops at end of visible note
  e.last_visible_row = filerow - 1 // note that this is not exactly true - could be the whole last row is visible
  fmt.Print(ab.String())

  //draw_visual(ab)
}

func (e *Editor) drawStatusBar() {
  var ab strings.Builder
  fmt.Fprintf(&ab, "\x1b[%d;%dH", e.screenlines + e.top_margin, e.left_margin + 1)

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

  if len(status) > e.screencols - 1 {
    status = status[:e.screencols - 1]
  }
  ab.WriteString(status)
  spaces := strings.Repeat(" ", e.screencols - len(status))
  ab.WriteString(spaces)
  ab.WriteString("\x1b[0m") //switches back to normal formatting
  fmt.Print(ab.String())
}

func (e *Editor) drawMessageBar() {
  var ab strings.Builder
  fmt.Fprintf(&ab, "\x1b[%d;%dH", sess.textLines + e.top_margin + 1, sess.divider + 1)

  ab.WriteString("\x1b[K") // will erase midscreen -> R; cursor doesn't move after erase
  if len(e.message) > e.screencols {
   e.message = e.message[:e.screencols]
  }
  ab.WriteString(e.message)
  fmt.Print(ab.String())
}

func (e *Editor) scroll() bool {

  if len(e.rows) == 0  {
    e.fr, e.fc, e.cy, e.cx, e.line_offset, e.prev_line_offset, e.first_visible_row, e.last_visible_row = 0,0,0,0,0,0,0,0
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

  e.cx = e.getScreenXFromRowColWW(e.fr, e.fc)
  cy_ := e.getScreenYFromRowColWW(e.fr, e.fc);

  //my guess is that if you wanted to adjust line_offset to take into account that you wanted
  // to only have full rows at the top (easier for drawing code) you would do it here.
  // something like screenlines goes from 4 to 5 so that adjusts cy
  // it's complicated and may not be worth it.

  //deal with scroll insufficient to include the current line
  if cy_ > e.screenlines + e.line_offset - 1 {
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

  e.cy = cy_ - e.line_offset;

  // vim seems to want full rows to be displayed although I am not sure
  // it's either helpful or worth it but this is a placeholder for the idea

  // returns true if display needs to scroll and false if it doesn't
  // could just be redraw = true or do nothing since don't want to override if already true.
  if e.line_offset == e.prev_line_offset {
    return false
  } else {
    e.prev_line_offset = e.line_offset; return true
  }
}

func (e *Editor) getInitialRow(line_offset int) (int, int) {

  if line_offset == 0 {
    return 0,0
  }

  initial_row := 0
  lines := 0

  for  {
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

  e.fr, e.fc, e.cy, e.cx, e.line_offset, e.prev_line_offset, e.first_visible_row, e.last_visible_row = 0,0,0,0,0,0,0,0

  e.dirty++
  //sess.editor_mode = true;
  e.refreshScreen(true)
  return nil
}
