package main

/*
struct orow {  //Entry
  std::string title;
  std::string fts_title;
  int id; //listmanager db id of the row
  bool star;
  bool deleted;
  bool completed;
  std::string modified;

  // note the members below are temporary editing flags
  // and don't need to be reflected in database
  bool dirty;
  bool mark;
};
*/

type Entry struct {
	id        int
	title     string
	fts_title string
	star      bool
	deleted   bool
	completed string
	modified  string

	// below not in db
	dirty  bool
	marked bool
}

//type outlineKey int

const (
	BACKSPACE  = iota + 127
	ARROW_LEFT = iota + 999 //would have to be < 127 to be chars
	ARROW_RIGHT
	ARROW_UP
	ARROW_DOWN
	DEL_KEY
	HOME_KEY
	END_KEY
	PAGE_UP
	PAGE_DOWN
	SHIFT_TAB
)

type Mode int

const (
	NORMAL Mode = iota
	INSERT
	COMMAND_LINE
	VISUAL_LINE // only editor mode
	VISUAL
	REPLACE
	FILE_DISPLAY // only outline mode
	NO_ROWS
	VISUAL_BLOCK      // only editor mode
	SEARCH            // only editor mode
	FIND              // only outline mode
	ADD_CHANGE_FILTER // only outline mode
)

var mode_text [12]string = [12]string{
	"NORMAL",
	"INSERT",
	"COMMAND LINE",
	"VISUAL LINE",
	"VISUAL",
	"REPLACE",
	"FILE DISPLAY",
	"NO ROWS",
	"VISUAL BLOCK",
	"SEARCH",
	"FIND",
	"ADD/CHANGE FILTER",
}

func (m Mode) String() string {
  return [...]string{
                   	"NORMAL",
                   	"INSERT",
                   	"COMMAND LINE",
                   	"VISUAL LINE",
                   	"VISUAL",
                   	"REPLACE",
                   	"FILE DISPLAY",
                   	"NO ROWS",
                   	"VISUAL BLOCK",
                   	"SEARCH",
                   	"FIND",
                   	"ADD/CHANGE FILTER",
                  }[m]
}
//var m Mode = NORMAL
//fmt.Print(m)
/*
enum DB {
  SQLITE,
  POSTGRES
};
*/

//type View int

const (
	TASK = iota
	CONTEXT
	FOLDER
	KEYWORD
)

//type TaskView int

const (
	BY_CONTEXT = iota
	BY_FOLDER
	BY_KEYWORD
	BY_JOIN
	BY_RECENT
	BY_FIND
)

type Container struct {
	id       int
	tid      int
	title    string
	star     bool
	created  string
	deleted  bool
	modified string
	count    int
}

/*
struct Lsp {
  std::jthread thred;
  std::string name{};
  std::string file_name{};
  std::string client_uri{};
  std::string language{};
  std::atomic<bool> code_changed = false;
  std::atomic<bool> closed = true;
};
*/

/* Task
0: id = 1
1: tid = 1
2: priority = 3
3: title = Parents refrigerator broken.
4: tag =
5: folder_tid = 1
6: context_tid = 1
7: duetime = NULL
8: star = 0
9: added = 2009-07-04
10: completed = 2009-12-20
11: duedate = NULL
12: note = new one coming on Monday, June 6, 2009.
13: repeat = NULL
14: deleted = 0
15: created = 2016-08-05 23:05:16.256135
16: modified = 2016-08-05 23:05:16.256135
17: startdate = 2009-07-04
18: remind = NULL

I thought I should be using tid as the "id" for sqlite version but realized
that would work and mean you could always compare the tid to the pg id
but for new items created with sqlite, there would be no tid so
the right thing to use is the id.  At some point might also want to
store the tid in orow row
*/
/* Context
0: id => int in use
1: tid => int in use
2: title = string 32 in use
3: "default" = Boolean ? -> star in use
4: created = 2016-08-05 23:05:16.256135 in use
5: deleted => bool in use
6: icon => string 32
7: textcolor, Integer
8: image, largebinary
9: modified in use
*/
/* Folder
0: id => int
1: tid => int
2: title = string 32
3: private = Boolean -> star
4: archived = Boolean ? what this is
5: "order" = integer
6: created = 2016-08-05 23:05:16.256135
7: deleted => bool
8: icon => string 32
9: textcolor, Integer
10: image, largebinary
11: modified
*/
/* Keyword
0: id => int
1: name = string 25
2: tid => int
3: star = Boolean
4: modified
5: deleted
*/