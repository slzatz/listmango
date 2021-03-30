package main

import (
	"fmt"
	"strings"
)

func (o *Organizer) refreshScreen() {
	var ab strings.Builder
	titlecols := o.divider - TIME_COL_WIDTH - LEFT_MARGIN

	ab.WriteString("\x1b[?25l") //hides the cursor

	//Below erase screen from middle to left - `1K` below is cursor to left erasing
	//Now erases time/sort column (+ 17 in line below)
	//if (org.view != KEYWORD) {
	if o.mode != ADD_CHANGE_FILTER {
		for j := TOP_MARGIN; j < o.textLines+1; j++ {
			fmt.Fprintf(&ab, "\x1b[%d;%dH\x1b[1K", j+TOP_MARGIN, titlecols+LEFT_MARGIN+17)
		}
	}
	// put cursor at upper left after erasing
	ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", TOP_MARGIN+1, LEFT_MARGIN+1))

	//fmt.Fprint(os.Stdout, ab.String())
	fmt.Print(ab.String())

	if o.mode == FIND {
		o.drawSearchRows()
		//} else if org.mode == ADD_CHANGE_FILTER {
		//  s.drawOrgFilters()
	} else if o.mode == ADD_CHANGE_FILTER {
		o.drawAltRows()
	} else {
		o.drawRows()
	}
}

func (o *Organizer) drawRows() {

	if len(o.rows) == 0 {
		return
	}

	var j, k int //to swap highlight if org.highlight[1] < org.highlight[0]
	var ab strings.Builder
	titlecols := o.divider - TIME_COL_WIDTH - LEFT_MARGIN

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", LEFT_MARGIN)

	for y := 0; y < o.textLines; y++ {
		fr := y + o.rowoff
		if fr > len(o.rows)-1 {
			break
		}

		// if a line is long you only draw what fits on the screen
		//below solves problem when deleting chars from a scrolled long line

		//can run into this problem when deleting chars from a scrolled log line
		var length int
		if fr == o.fr {
			length = len(o.rows[fr].title) - o.coloff
		} else {
			length = len(o.rows[fr].title)
		}

		if length > titlecols {
			length = titlecols
		}

		if o.rows[fr].star {
			ab.WriteString("\x1b[1m")    //bold
			ab.WriteString("\x1b[1;36m") //light cyan
		}

		if o.rows[fr].completed && o.rows[fr].deleted {
			ab.WriteString("\x1b[32m") //green foreground
		} else if o.rows[fr].completed {
			ab.WriteString("\x1b[33m") //yellow foreground
			//else if (row.deleted) ab.append("\x1b[31m", 5); //red foreground
		} else if o.rows[fr].deleted {
			ab.WriteString(RED_FG)
		} //red (specific color depends on theme)

		if fr == o.fr {
			ab.WriteString("\x1b[48;5;236m") // 236 is a grey
		}
		if o.rows[fr].dirty {
			ab.WriteString("\x1b[30;47m") //black letters on white bg
			//ab.WriteString(BLACK_FG + WHITE_BG) //this unbolded for star for some reason
		}
		//if (row.mark) ab.append("\x1b[46m", 5); //cyan background
		if _, ok := o.marked_entries[o.rows[fr].id]; ok {
			//ab.WriteString("\x1b[46m")
			//ab.WriteString(YELLOW_BG)
			ab.WriteString(YELLOW_BG)
			ab.WriteString("\x1b[30;43m") //black letters on yellow bg
		}

		// below - only will get visual highlighting if it's the active
		// then also deals with column offset
		if o.mode == VISUAL && fr == o.fr {

			// below in case org.highlight[1] < org.highlight[0]
			if o.highlight[1] > o.highlight[0] {
				j, k = 0, 1
			} else {
				k, j = 0, 1
			}

			ab.WriteString(o.rows[fr].title[o.coloff : o.highlight[j]-o.coloff])
			ab.WriteString("\x1b[48;5;242m")
			ab.WriteString(o.rows[fr].title[o.highlight[j] : o.highlight[k]-o.coloff])

			ab.WriteString("\x1b[49m") // return background to normal
			ab.WriteString(o.rows[fr].title[:o.highlight[k]])

		} else {
			// current row is only row that is scrolled if org.coloff != 0
			var beg int
			if fr == o.fr {
				beg = o.coloff
			}
			if len(o.rows[fr].title[beg:]) > length {
				ab.WriteString(o.rows[fr].title[beg : beg+length])
			} else {
				ab.WriteString(o.rows[fr].title[beg:])
			}
		}
		// the spaces make it look like the whole row is highlighted
		//note len can't be greater than titlecols so always positive
		ab.WriteString(strings.Repeat(" ", titlecols-length+1))

		// believe the +2 is just to give some space from the end of long titles
		//ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", y+TOP_MARGIN+1, s.divider-TIME_COL_WIDTH+2))
		fmt.Fprintf(&ab, "\x1b[%d;%dH", y+TOP_MARGIN+1, o.divider-TIME_COL_WIDTH+2)
		ab.WriteString(o.rows[fr].modified)
		ab.WriteString("\x1b[0m") // return background to normal ////////////////////////////////
		ab.WriteString(lf_ret)
	}
	//fmt.Fprint(os.Stdout, ab.String())
	fmt.Print(ab.String())
}

// for drawing containers when making a selection
func (o *Organizer) drawAltRows() {

	if len(o.altRows) == 0 {
		return
	}

	var ab strings.Builder
	fmt.Fprintf(&ab, "\x1b[%d;%dH", TOP_MARGIN+1, o.divider+2)
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", o.divider+1)

	for y := 0; y < o.textLines; y++ {

		fr := y + o.altRowoff
		if fr > len(o.altRows)-1 {
			break
		}

		length := len(o.altRows[fr].title)
		if length > o.totaleditorcols {
			length = o.totaleditorcols
		}

		if o.altRows[fr].star {
			ab.WriteString("\x1b[1m") //bold
			ab.WriteString("\x1b[1;36m")
		}

		if fr == o.altR {
			ab.WriteString("\x1b[48;5;236m") // 236 is a grey
		}

		ab.WriteString(o.altRows[fr].title[:length])
		ab.WriteString("\x1b[0m") // return background to normal
		ab.WriteString(lf_ret)
	}
	fmt.Print(ab.String())
}

// for drawing sync log (note)
func (o *Organizer) drawAltRows2() {

	if len(o.altRows) == 0 {
		return
	}
	//scroll
	if o.altR > o.textLines+o.altRowoff-1 {
		o.altRowoff = o.altR - o.textLines + 1
	}
	if o.altR < o.altRowoff {
		o.altRowoff = o.altR
	}
	// end scroll

	var ab strings.Builder
	fmt.Fprintf(&ab, "\x1b[%d;%dH", TOP_MARGIN+1, o.divider+2)
	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", o.divider+1)

	for y := 0; y < o.textLines; y++ {

		fr := y + o.altRowoff
		if fr > len(o.altRows)-1 {
			break
		}

		length := len(o.altRows[fr].title)
		if length > o.totaleditorcols {
			length = o.totaleditorcols
		}

		ab.WriteString(o.altRows[fr].title[:length])
		ab.WriteString("\x1b[0m") // return background to normal
		ab.WriteString(lf_ret)
	}
	fmt.Print(ab.String())
	o.showOrgMessage("altR = %d; altRowoff = %d", o.altR, o.altRowoff)
}

func (o *Organizer) drawStatusBar() {

	var ab strings.Builder
	//position cursor and erase - and yes you do have to reposition cursor after erase
	fmt.Fprintf(&ab, "\x1b[%d;%dH\x1b[1K\x1b[%d;1H", o.textLines+TOP_MARGIN+1, o.divider, o.textLines+TOP_MARGIN+1)
	ab.WriteString("\x1b[7m") //switches to reversed colors

	var str string
	var id int
	var title string
	var keywords string
	if len(o.rows) > 0 {
		switch o.view {
		case TASK:
			e := getEntryInfo(getId())
			switch o.taskview {
			case BY_FIND:
				str = "search - " + o.fts_search_terms
			case BY_FOLDER:
				str = fmt.Sprintf("%s[f] (%s[c])", o.folder, o.idToContext[e.context_tid])
				//str = org.folder + "[f]" + " (" + org.context
			case BY_CONTEXT:
				//str = org.context + "[c]"
				str = fmt.Sprintf("%s[c] (%s[f])", o.context, o.idToFolder[e.folder_tid])
			case BY_RECENT:
				str = fmt.Sprintf("Recent: %s[c] %s[f]",
					o.idToContext[e.context_tid], o.idToFolder[e.folder_tid])
				//str = "recent"
			//case BY_JOIN:
			//	str = org.context + "[c] + " + org.folder + "[f]"
			case BY_KEYWORD:
				str = o.keyword + "[k]"
			}
		case CONTEXT:
			str = "Contexts"
		case FOLDER:
			str = "Folders"
		case KEYWORD:
			str = "Keywords"
		case SYNC_LOG_VIEW:
			str = "Sync Log"
		}

		/*
			var id int
			var title string
			var keywords string
			if len(o.rows) > 0 {
		*/

		row := &o.rows[o.fr]

		if len(row.title) > 16 {
			title = row.title[:12] + "..."
		} else {
			title = row.title
		}

		id = row.id

		if o.view == TASK {
			keywords = getTaskKeywords(row.id)
		}
	} else {
		title = "   No Results   "
		id = -1

	}

	// [49m - revert background to normal
	// 7m - reverses video
	// because video is reversted [42 sets text to green and 49 undoes it
	// also [0;35;7m -> because of 7m it reverses background and foreground
	// I think the [0;7m is revert text to normal and reverse video
	status := fmt.Sprintf("\x1b[1m%s\x1b[0;7m %s \x1b[0;35;7m%s\x1b[0;7m %d %d/%d \x1b[1;42m%s\x1b[49m",
		str, title, keywords, id, o.fr+1, len(o.rows), o.mode)

	// klugy way of finding length of string without the escape characters
	plain := fmt.Sprintf("%s %s %s %d %d/%d %s",
		str, title, keywords, id, o.fr+1, len(o.rows), o.mode)
	length := len(plain)

	if length < o.divider {
		// need to do the below because the escapes make string
		// longer than it actually prints so pad separately
		fmt.Fprintf(&ab, "%s%-*s", status, o.divider-length, " ")
	} else {
		status = fmt.Sprintf("\x1b[1m%s\x1b[0;7m %s \x1b[0;35;7m%s\x1b[0;7m %d %d/%d\x1b[49m",
			str, title, keywords, id, o.fr+1, len(o.rows))
		plain = fmt.Sprintf("%s %s %s %d %d/%d",
			str, title, keywords, id, o.fr+1, len(o.rows))
		length := len(plain)
		if length < o.divider {
			fmt.Fprintf(&ab, "%s%-*s", status, o.divider-length, " ")
		} else {
			status = fmt.Sprintf("\x1b[1m%s\x1b[0;7m %s %s %d %d/%d",
				str, title, keywords, id, o.fr+1, len(o.rows))
			ab.WriteString(status[:o.divider+10])
		}
	}
	ab.WriteString("\x1b[0m") //switches back to normal formatting
	fmt.Print(ab.String())
}

func (o *Organizer) drawSearchRows() {

	if len(o.rows) == 0 {
		return
	}

	var ab strings.Builder
	titlecols := o.divider - TIME_COL_WIDTH - LEFT_MARGIN

	lf_ret := fmt.Sprintf("\r\n\x1b[%dC", LEFT_MARGIN)

	for y := 0; y < o.textLines; y++ {
		fr := y + o.rowoff
		if fr > len(o.rows)-1 {
			break
		}
		//orow& row = org.rows[fr];
		var length int

		if o.rows[fr].star {
			ab.WriteString("\x1b[1m") //bold
			ab.WriteString("\x1b[1;36m")
		}

		if o.rows[fr].completed && o.rows[fr].deleted {
			ab.WriteString("\x1b[32m") //green foreground
		} else if o.rows[fr].completed {
			ab.WriteString("\x1b[33m") //yellow foreground
		} else if o.rows[fr].deleted {
			ab.WriteString("\x1b[31m") //red foreground
		}

		if len(o.rows[fr].title) <= titlecols { // we know it fits
			ab.WriteString(o.rows[fr].fts_title)
			// note below doesn't handle two highlighted terms in same line
			// and it might cause display issues if second highlight isn't fully escaped
			// need to come back and deal with this
			// coud check if LastIndex"\x1b[49m" or Index(fts_title[pos+1:titlecols+15] contained another escape
		} else {
			pos := strings.Index(o.rows[fr].fts_title, "\x1b[49m") //\x1b[48;5;31m', '\x1b[49m'
			if pos > 0 && pos < titlecols+11 {                     //length of highlight escape
				ab.WriteString(o.rows[fr].fts_title[:titlecols+15]) //titlecols + 15); // length of highlight escape + remove formatting escape
			} else {
				ab.WriteString(o.rows[fr].title[:titlecols])
			}
		}
		if len(o.rows[fr].title) <= titlecols {
			length = len(o.rows[fr].title)
		} else {
			length = titlecols
		}
		spaces := titlecols - length
		ab.WriteString(strings.Repeat(" ", spaces))

		//snprintf(buf, sizeof(buf), "\x1b[%d;%dH", y + 2, screencols/2 - TIME_COL_WIDTH + 2); //wouldn't need offset
		ab.WriteString("\x1b[0m") // return background to normal
		//ab.WriteString(fmt.Sprintf("\x1b[%d;%dH", y+2, s.divider-TIME_COL_WIDTH+2))
		fmt.Fprintf(&ab, "\x1b[%d;%dH", y+2, o.divider-TIME_COL_WIDTH+2)
		ab.WriteString(o.rows[fr].modified)
		ab.WriteString(lf_ret)
	}
	fmt.Print(ab.String())
}

func (o *Organizer) drawPreviewWindow() { //get_preview
	id := o.rows[o.fr].id

	if o.taskview != BY_FIND {
		sess.drawPreviewText(id)
	} else {
		sess.drawSearchPreview()
	}
	sess.drawPreviewBox()

	/*
	  if (lm_browser) {
	    int folder_tid = getFolderTid(org.rows.at(org.fr).id);
	    if (!(folder_tid == 18 || folder_tid == 14)) updateHTMLFile("assets/" + CURRENT_NOTE_FILE);
	    else updateHTMLCodeFile("assets/" + CURRENT_NOTE_FILE);
	  }
	*/
}
