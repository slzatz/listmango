package main

import "github.com/neovim/go-client/nvim"

type Editor struct {
	cx, cy              int //screen cursor x and y position
	fc, fr              int // file cursor x and y position
	lineOffset          int //first row based on user scroll
	screenlines         int //number of lines for this Editor
	screencols          int //number of columns for this Editor
	left_margin         int //can vary (so could TOP_MARGIN - will do that later
	left_margin_offset  int // 0 if no line numbers
	top_margin          int
	code                string //used by lsp thread and intended to avoid unnecessary calls to editorRowsToString
	dirty               int64  //file changes since last save
	vb_highlight        [2][4]int
	mode                Mode
	command_line        string //for commands on the command line; string doesn't include ':'
	command             string // right now includes normal mode commands and command line commands
	last_command        string
	first_visible_row   int
	last_visible_row    int
	spellcheck          bool
	highlight_syntax    bool
	redraw              bool
	pos_mispelled_words [][2]int
	search_string       string //word under cursor works with *, n, N etc.
	id                  int    //listmanager db id of the row
	linked_editor       *Editor
	is_subeditor        bool
	is_below            bool
	//nuspell::Dictionary dict;
	vbuf         nvim.Buffer
	bb           [][]byte
	searchPrefix string
	//coloff              int //first column based on user scroll (word wrap)
}

func NewEditor() *Editor {
	return &Editor{
		cx:                0, //actual cursor x position (takes into account any scroll/offset)
		cy:                0, //actual cursor y position ""
		fc:                0, //'file' x position as defined by reading sqlite text into rows vector
		fr:                0, //'file' y position ""
		lineOffset:        0, //the number of lines of text at the top scrolled off the screen
		dirty:             0, //has filed changed since last save
		mode:              0, //0=normal, 1=insert, 2=command line, 3=visual line, 4=visual, 5='r'
		command:           "",
		command_line:      "",
		first_visible_row: 0,
		spellcheck:        false,
		highlight_syntax:  true, // should only apply to code - not in use
		redraw:            false,
		//undo_mode:          false,
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
