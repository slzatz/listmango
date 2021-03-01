package main

import "github.com/neovim/go-client/nvim"

const SMARTINDENT = 4 //should be in config

type Editor struct {
	cx, cy             int //cursor x and y position
	fc, fr             int // file x and y position
	line_offset        int //row the user is currently scrolled to
	prev_line_offset   int
	coloff             int //column user is currently scrolled to
	screenlines        int //number of lines for this Editor
	screencols         int //number of columns for this Editor
	left_margin        int //can vary (so could TOP_MARGIN - will do that later
	left_margin_offset int // 0 if no line numbers
	top_margin         int
	rows               []string
	code               string //used by lsp thread and intended to avoid unnecessary calls to editorRowsToString
	dirty              int    //file changes since last save
	//highlight          [2]int
	vb0              [3]int
	vb_highlight     [2][4]int
	mode             Mode
	command_line     string //for commands on the command line; string doesn't include ':'
	command          string // right now includes normal mode commands and command line commands
	last_command     string
	repeat           int
	last_repeat      int
	prev_fr, prev_fc int
	//what's typed between going into INSERT mode and leaving INSERT mode
	last_typed          string
	indent              int
	smartindent         int
	first_visible_row   int
	last_visible_row    int
	spellcheck          bool
	highlight_syntax    bool
	redraw              bool
	pos_mispelled_words [][2]int
	message             string //status msg is a character array max 80 char
	string_buffer       string //yanking chars
	line_buffer         []string
	//static int total_screenlines; //total screenlines available to Editors vertically
	//static int origin; //x column of Editor section
	search_string string //word under cursor works with *, n, N etc.
	id            int    //listmanager db id of the row
	undo_deque    []Diff //if neg it was a delete
	d_index       int    //undo_deque index
	undo_mode     bool
	snapshot      []string
	linked_editor *Editor
	is_subeditor  bool
	is_below      bool
	//nuspell::Dictionary dict;
	vbuf nvim.Buffer
}

func NewEditor() *Editor {
	return &Editor{
		cx:                 0,  //actual cursor x position (takes into account any scroll/offset)
		cy:                 0,  //actual cursor y position ""
		fc:                 0,  //'file' x position as defined by reading sqlite text into rows vector
		fr:                 0,  //'file' y position ""
		line_offset:        0,  //the number of lines of text at the top scrolled off the screen
		prev_line_offset:   0,  //the prev number of lines of text at the top scrolled off the screen
		dirty:              0,  //has filed changed since last save
		message:            "", //very bottom of screen, ex. -- INSERT --
		mode:               0,  //0=normal, 1=insert, 2=command line, 3=visual line, 4=visual, 5='r'
		command:            "",
		command_line:       "",
		repeat:             0, //number of times to repeat commands like x,s,yy also used for visual line mode x,y
		indent:             4,
		smartindent:        1, //CTRL-z toggles - don't want on what pasting from outside source
		first_visible_row:  0,
		spellcheck:         false,
		highlight_syntax:   true, // should only apply to code
		redraw:             false,
		undo_mode:          false,
		linked_editor:      nil,
		is_subeditor:       false,
		is_below:           false,
		left_margin_offset: 0, // 0 if no line numbers
		//E.coloff: 0,  //always zero because currently only word wrap supported

		/*
		   auto dict_list: std::vector<std::pair<std::string, std::string>>{},
		   nuspell::search_default_dirs_for_dicts(dict_list),
		   auto dict_name_and_path: nuspell::find_dictionary(dict_list, "en_US"),
		   auto & dict_path: dict_name_and_path->second,
		   dict: nuspell::Dictionary::load_from_path(dict_path),
		*/
	}
}

type pair1 struct {
	char byte
	num  int
}

type pair2 struct {
	num int
	str string
}

type Diff struct {
	fr            int
	fc            int
	repeat        int
	command       string
	rows          []string
	num_rows      int //the row where insert occurs counts 1 and then any rows added with returns
	inserted_text string
	deleted_text  string //deleted chars - being recorded by not used right now or perhaps ever!
	diff          pair1
	changed_rows  pair2
	undo_method   int //CHANGE_ROW< REPLACE_NOTE< ADD_ROWS, DELETE_ROWS
	mode          int
}

// ERow represents a line of text in a file
//type ERow []rune

/*
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
*/
