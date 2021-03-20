package main

import (
	"strings"
	//"fmt"
	"unicode"
)

/*
var folder_map map[string]int = map[string]int{}
var context_map map[string]int = map[string]int{}
*/

func (o *Organizer) delWord() {
	// still needs to deal with possibility of utf8 multi-byte characters (see finding word under cursor)
	t := &o.rows[o.fr].title
	delimiters := " ,.;?:()[]{}&#"
	var beg int
	if o.fc != 0 {
		beg = strings.LastIndexAny((*t)[:o.fc], delimiters)
		if beg == -1 {
			beg = 0
		} else {
			beg++ //i think this is covered:  "#"
		}
	}

	end := strings.IndexAny((*t)[o.fc:], delimiters)
	if end == -1 {
		end = len(*t) - 1
	} else {
		//end = end + fc - 1
		end = end + o.fc + 1
	}

	*t = (*t)[:beg] + (*t)[end:]
	o.rows[o.fr].dirty = true
}

//Note: outlineMoveCursor worries about moving cursor beyond the size of the row
//OutlineScroll worries about moving cursor beyond the screen
func (o *Organizer) moveCursor(key int) {

	if len(o.rows) == 0 {
		return
	}

	switch key {
	case ARROW_LEFT, 'h':
		if o.fc > 0 {
			o.fc--
		}

	case ARROW_RIGHT, 'l':
		o.fc++

	case ARROW_UP, 'k':
		if o.fr > 0 {
			o.fr--
		}
		o.fc, o.coloff = 0, 0

		if o.view == TASK {
			sess.drawPreviewWindow(o.rows[o.fr].id) //if id == -1 does not try to retrieve note
		} else {
			c := getContainerInfo(o.rows[o.fr].id)
			if c.id != 0 {
				sess.displayContainerInfo(&c)
				sess.drawPreviewBox()
			}
		}

	case ARROW_DOWN, 'j':
		if o.fr < len(o.rows)-1 {
			o.fr++
		}
		o.fc, o.coloff = 0, 0
		if o.view == TASK {
			sess.drawPreviewWindow(o.rows[o.fr].id) //if id == -1 does not try to retrieve note
		} else {
			c := getContainerInfo(o.rows[o.fr].id)
			if c.id != 0 {
				sess.displayContainerInfo(&c)
				sess.drawPreviewBox()
			}
		}
	}

	t := &o.rows[o.fr].title
	if o.fc >= len(*t) {
		if o.mode != INSERT {
			o.fc = len(*t) - 1
		} else {
			o.fc = len(*t)
		}
	}
	if *t == "" {
		o.fc = 0
	}
}

func (o *Organizer) backspace() {
	t := &o.rows[o.fr].title
	if len(o.rows) == 0 || *t == "" || o.fc == 0 {
		return
	}
	// below handles when INSERT is one char beyond end of line
	fc := o.fc
	if o.fc == len(*t) {
		fc--
	}
	*t = (*t)[:fc] + (*t)[fc+1:] // should do with runes
	o.fc--
	o.rows[o.fr].dirty = true
}

func (o *Organizer) delChar() {
	t := &o.rows[o.fr].title
	if len(o.rows) == 0 || len(*t) == 0 {
		return
	}
	*t = (*t)[:o.fc] + (*t)[o.fc+1:]
	o.rows[o.fr].dirty = true
}

func (o *Organizer) deleteToEndOfLine() {
	t := &o.rows[o.fr].title
	*t = (*t)[:o.fc] // or row.chars.erase(row.chars.begin() + O.fc, row.chars.end())
	o.rows[o.fr].dirty = true
}

func (o *Organizer) pasteString() {
	t := &o.rows[o.fr].title

	if len(o.rows) == 0 || o.string_buffer == "" {
		return
	}

	*t = (*t)[:o.fc+1] + o.string_buffer + (*t)[o.fc+1:] // how about end of line - works fine
	o.fc += len(o.string_buffer)
	o.rows[o.fr].dirty = true
}

func (o *Organizer) yankString() {
	t := &o.rows[o.fr].title
	o.string_buffer = (*t)[o.highlight[0] : o.highlight[1]+1]
}

func (o *Organizer) moveCursorEOL() {
	o.fc = len(o.rows[o.fr].title) - 1 //if O.cx > O.titlecols will be adjusted in EditorScroll
}

func (o *Organizer) moveBeginningWord() {
	if o.fc == 0 {
		return
	}
	t := &o.rows[o.fr].title
	delimiters := " ,.;?:()[]{}&#"
	beg := strings.LastIndexAny((*t)[:o.fc], delimiters)
	if beg == -1 {
		o.fc = 0
	} else {
		o.fc = beg + 1 //i think this is covered:  "#"
	}
}

func (o *Organizer) moveEndWord() {
	t := &o.rows[o.fr].title
	delimiters := " ,.;?:()[]{}&#"
	end := strings.IndexAny((*t)[o.fc:], delimiters)
	if end == -1 {
		o.fc = len(*t) - 1
	} else {
		//end = end + fc - 1
		o.fc = end + o.fc + 1
	}
}

// needs to handle more corner cases (eg two spaces in a row)
func (o *Organizer) moveNextWord() {
	t := &o.rows[o.fr].title
	end := strings.Index((*t)[o.fc:], " ")
	if end == -1 {
		if o.fr < len(o.rows)-1 {
			o.fr++
			o.fc = 0
			return
		}
	} else {
		//end = end + fc - 1
		if o.fc < len(*t)-1 {
			o.fc = end + o.fc + 1
		}
	}
}

// not same as 'e' but moves to end of word or stays put if already on end of word
func (o *Organizer) moveEndWord2() {
	var j int
	t := &o.rows[o.fr].title

	for j = o.fc + 1; j < len(*t); j++ {
		if (*t)[j] < 48 {
			break
		}
	}
	o.fc = j - 1
}

func (o *Organizer) getWordUnderCursor() {
	t := &o.rows[o.fr].title
	delimiters := " ,.;?:()[]{}&#"
	if strings.IndexAny(string((*t)[o.fc]), delimiters) != -1 {
		return
	}

	var beg int
	if o.fc != 0 {
		beg = strings.LastIndexAny((*t)[:o.fc], delimiters)
		if beg == -1 {
			beg = 0
		} else {
			beg++
		}
	}
	end := strings.IndexAny((*t)[o.fc:], delimiters)
	if end == -1 {
		end = len(*t) - 1
	} else {
		end = end + o.fc - 1
	}
	o.title_search_string = (*t)[beg : end+1]
}

func (o *Organizer) findNextWord() {
	var n int
	if o.fr < len(o.rows)-1 {
		n = o.fr + 1
	} else {
		n = 0
	}

	for {
		if n == len(o.rows) {
			n = 0
		}
		pos := strings.Index(o.rows[n].title, o.title_search_string)
		if pos == -1 {
			continue
		} else {
			o.fr = n
			o.fc = pos
			return
		}
		n++
	}
}

func (o *Organizer) changeCase() {
	t := &o.rows[o.fr].title
	char := rune((*t)[o.fc])
	if unicode.IsLower(char) {
		char = unicode.ToUpper(char)
	} else {
		char = unicode.ToLower(char)
	}
	*t = (*t)[:o.fc] + string(char) + (*t)[o.fc+1:]
	o.rows[o.fr].dirty = true
}

func (o *Organizer) insertRow(at int, s string, star bool, deleted bool, completed bool, modified string) {
	/* note since only inserting blank line at top, don't really need at, s and also don't need size_t*/

	var row Row
	o.rows = append(o.rows, row)     //make sure there is room to expand o.rows
	copy(o.rows[at+1:], o.rows[at:]) // move everything one over that will be to the right of new entry

	row.title = s
	row.id = -1
	row.star = star
	row.deleted = deleted
	row.completed = completed
	row.dirty = true
	row.modified = modified

	row.marked = false

	o.rows[at] = row
}

func (o *Organizer) scroll() {

	titlecols := sess.divider - TIME_COL_WIDTH - LEFT_MARGIN

	if len(o.rows) == 0 {
		o.fr, o.fc, o.coloff, o.cx, o.cy = 0, 0, 0, 0, 0
		return
	}

	if o.fr > sess.textLines+o.rowoff-1 {
		o.rowoff = o.fr - sess.textLines + 1
	}

	if o.fr < o.rowoff {
		o.rowoff = o.fr
	}

	if o.fc > titlecols+o.coloff-1 {
		o.coloff = o.fc - titlecols + 1
	}

	if o.fc < o.coloff {
		o.coloff = o.fc
	}

	o.cx = o.fc - o.coloff
	o.cy = o.fr - o.rowoff
}

func (o *Organizer) insertChar(c int) {
	if len(o.rows) == 0 {
		return
	}

	t := &o.rows[o.fr].title
	if *t == "" {
		*t = string(c)
	} else {
		//*t = (*t)[:o.fc+1] + string(c) + (*t)[o.fc+1:]
		*t = (*t)[:o.fc] + string(c) + (*t)[o.fc:]
	}
	o.fc++
	o.rows[o.fr].dirty = true
}

// corrected mistaken need to do (*row) when go able to figure out it's a pointer
func (o *Organizer) writeTitle() {
	row := &o.rows[o.fr]

	if !row.dirty {
		sess.showOrgMessage("Row has not been changed")
		return
	}

	if o.view == TASK {
		updateTitle()
		//msg = "Updated id {} to {} (+fts)";
		/*
		     if (sess.lm_browser) {
		       int folder_tid = getFolderTid(org.rows.at(org.fr).id);
		       if (!(folder_tid == 18 || folder_tid == 14)) sess.updateHTMLFile("assets/" + CURRENT_NOTE_FILE);
		     }

		   } else if (org.view == CONTEXT || org.view == FOLDER) {
		     updateContainerTitle();
		     msg = "Updated id {} to {}";
		   } else if (org.view == KEYWORD) {
		     updateKeywordTitle();
		     msg = "Updated id {} to {}";
		   }
		*/
	} else {
		updateContainerTitle()
	}

	o.command = ""
	o.mode = NORMAL
	row.dirty = false
	sess.showOrgMessage("Updated id %d to %s (+fts)", row.id, row.title)
	sess.refreshOrgScreen()
	if o.fc > 0 {
		o.fc--
	}
}

func (o *Organizer) clearMarkedEntries() {
	for k, _ := range o.marked_entries {
		delete(o.marked_entries, k)
	}
}

/*
std::string Organizer::outlineRowsToString(void) {
  std::string s = "";
  for (auto i: rows) {
      s += i.title;
      s += '\n';
  }
  s.pop_back(); //pop last return that we added
  return s;
}
*/
