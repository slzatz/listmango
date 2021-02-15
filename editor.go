package main

import (
	"fmt"

	"github.com/slzatz/listmango/runes"
)

type eMode int

const SMARTINDENT = 4; //should be in config

type Editor struct {
    cx, cy  int//cursor x and y position
    fc, fr int// file x and y position
    line_offset int//row the user is currently scrolled to
    prev_line_offset int
    coloff  int//column user is currently scrolled to
    screenlines int //number of lines for this Editor
    screencols int  //number of columns for this Editor
    left_margin int //can vary (so could TOP_MARGIN - will do that later
    left_margin_offset int // 0 if no line numbers
    top_margin int
    rows []string
    code  string//used by lsp thread and intended to avoid unnecessary calls to editorRowsToString
    dirty int //file changes since last save
    highlight [2]int
    vb0 [3]int
    mode int
    command_line string //for commands on the command line; string doesn't include ':'
    command string // right now includes normal mode commands and command line commands
    last_command string
    repeat int
    last_repeat int
    prev_fr, prev_fc int
    //what's typed between going into INSERT mode and leaving INSERT mode
    last_typed string
    indent int
    smartindent int
    first_visible_row int
    last_visible_row int
    spellcheck bool
    highlight_syntax bool
    redraw bool
    pos_mispelled_words [][2]int
    message string//status msg is a character array max 80 char
    string_buffer string//yanking chars
    line_buffer []string
    //static int total_screenlines; //total screenlines available to Editors vertically
    //static int origin; //x column of Editor section
    search_string string //word under cursor works with *, n, N etc.
    id int//listmanager db id of the row
    undo_deque []Diff//if neg it was a delete
    d_index int//undo_deque index
    undo_mode bool
    snapshot []string
    linked_editor *Editor
    is_subeditor bool
    is_below bool
    //nuspell::Dictionary dict;
  }

func NewEditor() *Editor {
	return &Editor{
      cx: 0, //actual cursor x position (takes into account any scroll/offset)
      cy: 0, //actual cursor y position ""
      fc: 0, //'file' x position as defined by reading sqlite text into rows vector
      fr: 0, //'file' y position ""
      line_offset: 0,  //the number of lines of text at the top scrolled off the screen
      prev_line_offset: 0,  //the prev number of lines of text at the top scrolled off the screen
      //E.coloff: 0,  //always zero because currently only word wrap supported
      dirty: 0, //has filed changed since last save
      message[0]: '\0', //very bottom of screen, ex. -- INSERT --
      highlight = [2]{-1,-1},
      mode: 0, //0=normal, 1=insert, 2=command line, 3=visual line, 4=visual, 5='r' 
      command: "",
      command_line: "",
      repeat: 0, //number of times to repeat commands like x,s,yy also used for visual line mode x,y
      indent: 4,
      smartindent: 1, //CTRL-z toggles - don't want on what pasting from outside source
      first_visible_row: 0,
      spellcheck: false,
      highlight_syntax: true, // should only apply to code
      redraw: false,
      undo_mode: false,
      linked_editor: nil,
      is_subeditor: false,
      is_below: false,
      left_margin_offset: 0, // 0 if no line numbers

      /*
      auto dict_list: std::vector<std::pair<std::string, std::string>>{},
      nuspell::search_default_dirs_for_dicts(dict_list),
      auto dict_name_and_path: nuspell::find_dictionary(dict_list, "en_US"),
      auto & dict_path: dict_name_and_path->second,
      dict: nuspell::Dictionary::load_from_path(dict_path),
      */
    }
}

// cmd1_map = make(map[string]func(*Editor, int),4)
cmd_map1 := map[string]func(*Editor, int){
                   "i":(*Editor).E_i,
                   "I":(*Editor).E_a,
                   "a":(*Editor).E_a,
                   "A":(*Editor).E_A,
                 }
// to call it's cmd1_map["i"](e, repeat)

cmd_map2 := map[string]func(*Editor, int){
                   "o":(*Editor).E_o_escape,
                   "O":(*Editor).E_O_escape,
                 }

cmd_map3 := map[string]func(*Editor, int){
                   "x":(*Editor).E_x,
                   "dw":(*Editor).E_dw,
                   "daw":(*Editor).E_daw,
                   "dd":(*Editor).E_dd,
                   "d$":(*Editor).E_d$,
                   "de":(*Editor).E_de,
                   "dG":(*Editor).E_dG,
                 }

cmd_map4 := map[string]func(*Editor, int){
                   "cw":(*Editor).E_cw,
                   "caw":(*Editor).E_caw,
                   "s":(*Editor).E_s,
                   "A":(*Editor).E_A,
                 }

    void setLinesMargins(void);
    bool find_match_for_left_brace(char, bool back=false);
    std::pair<int,int> move_to_right_brace(char);
    bool find_match_for_right_brace(char, bool back=false);
    std::pair<int,int> move_to_left_brace(char);
    void draw_highlighted_braces(void);
    //void position_editors(void); in session struct
    












	mode eMode
	rows []Erow
}

type Diff struct {
  fr int
  fc int
  repeat int
  command string
  rows []string
  num_rows int//the row where insert occurs counts 1 and then any rows added with returns
  inserted_text string
  deleted_text string //deleted chars - being recorded by not used right now or perhaps ever!
  diff []struct{byte, int}
  changed_rows []struct{int, string}
  undo_method int//CHANGE_ROW< REPLACE_NOTE< ADD_ROWS, DELETE_ROWS
  mode int
};

// ERow represents a line of text in a file
type ERow []rune

// cmd1_map = make(map[string]func(*Editor, int),4)
cmd1_map1 := map[string]func(*Editor, int){
                   "i":(*Editor).E_i,
                   "I":(*Editor).E_a,
                   "a":(*Editor).E_a,
                   "A":(*Editor).E_A,
                 }
// to call it's cmd1_map["i"](e, repeat)

cmd1_map2 := map[string]func(*Editor, int){
                   "o":(*Editor).E_o_escape,
                   "O":(*Editor).E_O_escape,
                 }

cmd1_map3 := map[string]func(*Editor, int){
                   "x":(*Editor).E_x,
                   "dw":(*Editor).E_dw,
                   "daw":(*Editor).E_daw,
                   "dd":(*Editor).E_dd,
                   "d$":(*Editor).E_d$,
                   "de":(*Editor).E_de,
                   "dG":(*Editor).E_dG,
                 }

cmd1_map4 := map[string]func(*Editor, int){
                   "cw":(*Editor).E_cw,
                   "caw":(*Editor).E_caw,
                   "s":(*Editor).E_s,
                   "A":(*Editor).E_A,
                 }

// Text expands tabs in an eRow to spaces
func (row ERow) Text() ERow {
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
func (row ERow) CxToRx(cx int) int {
	rx := 0
	for j := 0; j < cx; j++ {
		if row[j] == '\t' {
			rx = (rx + kiloTabStop - 1) - (rx % kiloTabStop)
		}
		rx++
	}
	return rx
}

// Editor represents the data in the being edited in memory
type Editor struct {
	Cx, Cy           int // Cx and Cy represent current cursor position
	Fc, Fr           int
	PrevFc, PrevFr   int
	Cursor           Point
	Rows             []ERow // Rows represent the textual data
	Dirty            bool   // has the file been edited
	FileName         string // the path to the file being edited. Could be empty string
	LineOffset       int
	ScreenLines      int
	LeftMargin       int
	LeftMarginOffset int
	TopMargin        int
	Mode             int
	Redraw           bool
}

// NewEditor returns a new blank editor
func NewEditor() *Editor {
	return &Editor{
		FileName: "",
		Dirty:    false,
		Cursor:   Point{0, 0},
		Cx:       0,
		Cy:       0,
		Fc:       0,
		Fr:       0,
		Rows:     []ERow{},
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

