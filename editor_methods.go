package main

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

// a string is a sequence of uint8 == byte 
func (e *Editor) move_to_right_brace(left_brace byte) (int,int) {
  r := e.fr
  c := e.fc + 1
  count := 1
  max := len(e.rows)

  m := map[byte]byte{{'{','}'}, {'(',')'}, {'[',']'}}
  right_brace := m[left_brace]

  for  {

    row := e.rows[r]

    // right now this function only called from NORMAL mode by typing '%'
    // note that function that deals with INSERT needs  c >= row.size() because
    // brace could be at end of line and fc could be row.size() before doing fc + 1
    if c == len(row) {
      r++
      if r == max {
        editorSetMessage("Couldn't find matching brace");
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
  count = 1

  m := map[byte]byte{{'}','{'}, {')','('}, {']','['}}
  left_brace = m[right_brace]

  row := e.rows[r]

  for {

    if (c == -1) { //fc + 1 can be greater than row.size on first pass from INSERT if { at end of line
      r--;
      if (r == -1) {
        editorSetMessage("Couldn't find matching brace");
        return e.fr,e.fc
      }
      row = rows[r]
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
  i = strings.Index(left, c)
  if i != -1 {
    e.fr, e.fc = move_to_right_brace(c);
  } else {
    right := "})]";
    i = strings.Index(right, c)
    if i != -1 {
      e.fr,e.fc = move_to_left_brace(c);
    }
  }
}
//'automatically' happens in NORMAL and INSERT mode
//return true -> redraw; false -> don't redraw
func (e *Editor) find_match_for_left_brace(left_brace byte, back bool) bool {
  r := e.fr
  c := e.fc + 1
  count := 1
  max = len(e.rows)

  m := map[byte]byte{{'{','}'}, {'(',')'}, {'[',']'}}
  right_brace := m[left_brace]

  //editorSetMessage("left brace: {}", left_brace);
  for  {

    row := e.rows[r]

    // need >= because brace could be at end of line and in INSERT mode
    // fc could be row.size() [ie beyond the last char in the line
    // and so doing fc + 1 above leads to c > row.size()
    if c >= len(row) {
      r++
      if r == max {
        editorSetMessage("Couldn't find matching brace")
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
  fmt.Printf(os.Stdout, "\x1b[%d;%dH\x1b[48;5;244m%d", y + e.top_margin, x, right_brace)

  x = editorGetScreenXFromRowColWW(fr, fc-back) + e.left_margin + e.left_margin_offset + 1
  y = editorGetScreenYFromRowColWW(fr, fc-back) + e.top_margin - e.line_offset; // added line offset 12-25-2019
  fmt.Printf(os.Stdout, "\x1b[%d;%dH\x1b[48;5;244m%d\x1b[0m", y, x, left_brace)
  editorSetMessage("r = %d   c = %d", r, c)
  return true
}

//'automatically' happens in NORMAL and INSERT mode
func (e *Editor) find_match_for_right_brace(right_brace byte, back bool) bool {
  r = e.fr
  c = e.fc - 1 - back
  count := 1

  row := e.rows[r]

  m := map[byte]byte{{'}','{'}, {')','('}, {']','['}}
  left_brace := m[right_brace]

  for {

    if c == -1 { //fc + 1 can be greater than row.size on first pass from INSERT if { at end of line
      r--
      if r == -1 {
        editorSetMessage("Couldn't find matching brace");
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
  fmt.Printf(os.Stdout, "\x1b[%d;%dH\x1b[48;5;244m%d", y + e.top_margin, x, right_brace)

  x = editorGetScreenXFromRowColWW(fr, fc-back) + e.left_margin + e.left_margin_offset + 1
  y = editorGetScreenYFromRowColWW(fr, fc-back) + e.top_margin - e.line_offset; // added line offset 12-25-2019
  fmt.Printf(os.Stdout, "\x1b[%d;%dH\x1b[48;5;244m%d\x1b[0m", y, x, right_brace)
  editorSetMessage("r = %d   c = %d", r, c)
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
  pos := strings.Index(braces, c)
  if pos != -1 {
    switch c {
      case '{', '(':
        e.redraw = find_match_for_left_brace(c, back)
        return
      case '}', ')':
        e.redraw = find_match_for_right_brace(c, back)
        return
      //case '(':  
      default://should not need this
        return
    }
  } else if ( e.fc > 0 && e.mode == INSERT ) {
      c := e.rows[fr][fc-1]
      pos := strings.Index(braces, c)
      if pos != -1 {
        switch e.rows[fr][fc-1] {
          case '{', '(':
            e.redraw = find_match_for_left_brace(c, true)
            return
          case '}', ')':
            e.redraw = find_match_for_right_brace(c, true)
            return
          //case '(':  
          default://should not need this
            return
      }
    } else {e.redraw = false}
  } else {e.redraw = false}
}

func (e *Editor) setLinesMargins() { //also sets top margin

  if(e.linked_editor) {
    if (e.is_subeditor) {
      if (e.is_below) {
        e.screenlines = LINKED_NOTE_HEIGHT;
        e.top_margin = sess.textlines - LINKED_NOTE_HEIGHT + 2;
      } else {
        e.screenlines = sess.textlines;
        e.top_margin =  TOP_MARGIN + 1;
      }
    } else {
      if (e.linked_editor.is_below) {
        e.screenlines = sess.textlines - LINKED_NOTE_HEIGHT - 1;
        e.top_margin =  TOP_MARGIN + 1;
      } else {
        e.screenlines = sess.textlines;
        e.top_margin =  TOP_MARGIN + 1;
      }
    }
  } else {
    e.screenlines = sess.textlines;
    e.top_margin =  TOP_MARGIN + 1;
  }
}

// normal mode 'e'
func moveEndWord() {

if len(rows) == o {
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

    row = &rows[r]

    if len(*row) == 0 {
      r++
      c = 0
      continue
    }

    if strings.Index(delimiters, string((*row)[c])) == -1 {
      if ( c == len(*row) - 1 || strings.Index(delimiters, string((*row)[c+1])) ) != -1 {
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
          fc = len(*row) - 1
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
        fr++;
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
  indent := 2

  var spaces string
  for j := 0; j < indent; j++ {
      spaces += ' '
  }
  e.fc = indent

  e.fr += direction;
  e.insertRow(fr, spaces)
}

func (e *Editor) insertRow(r int, s string) {
  append(e.rows, "")
  copy(e.rows[:r+1], e.rows[r:])
  e.rows[r] = s
  dirty++;
}
