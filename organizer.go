package main

type Organizer struct {
	mode      Mode
	last_mode Mode

	cx, cy    int //cursor x and y position
	fc, fr    int // file x and y position
	rowoff    int //the number of rows scrolled (aka number of top rows now off-screen
	altRowoff int
	coloff    int //the number of columns scrolled (aka number of left rows now off-screen
	altR      int

	rows         []Row
	altRows      []AltRow
	context      string
	folder       string
	keyword      string // could be multiple (comma separated)
	sort         string
	command_line string
	message      string

	command string
	repeat  int

	show_deleted   bool
	show_completed bool

	view            int
	altView         int
	taskview        int
	current_task_id int
	string_buffer   string
	fts_titles      map[int]string

	context_map map[string]int
	idToContext map[int]string
	folder_map  map[string]int
	idToFolder  map[int]string
	sort_map    map[string]int

	fts_ids        []int
	marked_entries map[int]struct{} // map instead of list makes toggling a row easier

	title_search_string string
	highlight           [2]int

	*Session
}
