package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var db, _ = sql.Open("sqlite3", "/home/slzatz/listmango/mylistmanager_s.db")
var fts_db, _ = sql.Open("sqlite3", "/home/slzatz/listmango/fts5.db")

func getId() int {
	return org.rows[org.fr].id
}

func timeDelta(t string) string {
	t0 := time.Now()
	t1, _ := time.Parse("2006-01-02T15:04:05Z", t)
	diff := t0.Sub(t1)
	//diff2 := time.Since(t1)

	/*
	  fmt.Println(t1)
	  fmt.Println(diff)
	  fmt.Printf("%#v\n", diff)
	*/

	diff = diff / 1000000000
	if diff <= 120 {
		return fmt.Sprintf("%d seconds ago", diff)
	} else if diff <= 60*120 {
		return fmt.Sprintf("%d minutes ago", diff/60) // <120 minutes we report minute
	} else if diff <= 48*60*60 {
		return fmt.Sprintf("%d hours ago", diff/3600) // <48 hours report hours
	} else if diff <= 24*60*60*60 {
		return fmt.Sprintf("%d days ago", diff/3600/24) // <60 days report days
	} else if diff <= 24*30*24*60*60 {
		return fmt.Sprintf("%d months ago", diff/3600/24/30) // <24 months rep
	} else {
		return fmt.Sprintf("%d years ago", diff/3600/24/30/12)
	}
}

func keywordExists(name string) int {
	row := db.QueryRow("SELECT keyword.id FROM keyword WHERE keyword.name=?;", name)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return -1
	}
	return id
}

func generateContextMap() {
	rows, err := db.Query("SELECT tid, title FROM context;")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var tid int
		var title string

		err = rows.Scan(&tid, &title)
		org.context_map[title] = tid
		org.idToContext[tid] = title
	}
}

func generateFolderMap() {
	rows, err := db.Query("SELECT tid, title FROM folder;")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var tid int
		var title string

		err = rows.Scan(&tid, &title)
		org.folder_map[title] = tid
		org.idToFolder[tid] = title
	}
}

func toggleStar() {
	//orow& row = org.rows.at(org.fr);
	id := getId()

	var table string
	var column string

	switch org.view {

	case TASK:
		table = "task"
		column = "star"

	case CONTEXT:
		table = "context"
		column = "\"default\""

	case FOLDER:
		table = "folder"
		column = "private"

	case KEYWORD:
		table = "keyword"
		column = "star"

	default:
		sess.showOrgMessage("Not sure what you're trying to toggle")
		return
	}

	/*
		stmt, err := db.Prepare(fmt.Sprintf("UPDATE %s SET %s=?, modified=datetime('now') WHERE id=?;",
			table, column))
	*/

	s := fmt.Sprintf("UPDATE %s SET %s=?, modified=datetime('now') WHERE id=?;",
		table, column)
	res, err := db.Exec(s, !org.rows[org.fr].star, id)

	if err != nil {
		log.Fatal(err)
	}

	//defer stmt.Close()

	/*
		res, err := stmt.Exec(!org.rows[org.fr].star, id)
		if err != nil {
			log.Fatal(err)
		}
	*/

	numRows, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	if numRows != 1 {
		log.Fatal("Toggle star numRows != 1")
	}
	//LastInsertId() (int64, error)

	org.rows[org.fr].star = !org.rows[org.fr].star
	sess.showOrgMessage("Toggle star succeeded")
}

func toggleDeleted() {
	id := getId()
	var table string

	switch org.view {
	case TASK:
		table = "task"
	case CONTEXT:
		table = "context"
	case FOLDER:
		table = "folder"
	case KEYWORD:
		table = "keyword"
	default:
		sess.showOrgMessage("Somehow you are in a view I can't handle")
		return
	}

	s := fmt.Sprintf("UPDATE %s SET deleted=?, modified=datetime('now') WHERE id=?;", table)
	_, err := db.Exec(s, !org.rows[org.fr].deleted, id)
	if err != nil {
		sess.showOrgMessage("Error toggling %s id %d to deleted: %v", table, id, err)
		return
	}

	org.rows[org.fr].deleted = !org.rows[org.fr].deleted
	sess.showOrgMessage("Toggle deleted for %s id %d succeeded", table, id)
}

func toggleCompleted() {
	//orow& row = org.rows.at(org.fr);
	id := getId()

	var completed sql.NullTime
	if org.rows[org.fr].completed {
		completed = sql.NullTime{}
	} else {
		completed = sql.NullTime{Time: time.Now(), Valid: true}
	}

	_, err := db.Exec("UPDATE task SET completed=?, modified=datetime('now') WHERE id=?;",
		completed, id)

	if err != nil {
		sess.showOrgMessage("Error toggling entry id %d to completed: %v", id, err)
		return
	}

	org.rows[org.fr].completed = !org.rows[org.fr].completed
	sess.showOrgMessage("Toggle completed for entry %d succeeded", id)
}

func updateTaskContext(new_context string, id int) {
	context_tid := org.context_map[new_context]

	_, err := db.Exec("UPDATE task SET context_tid=?, modified=datetime('now') WHERE id=?;",
		context_tid, id)

	if err != nil {
		sess.showOrgMessage("Error updating context for entry %d to %s: %v", id, new_context, err)
		return
	}
}

func updateTaskFolder(new_folder string, id int) {
	folder_tid := org.folder_map[new_folder]

	_, err := db.Exec("UPDATE task SET folder_tid=?, modified=datetime('now') WHERE id=?;",
		folder_tid, id)

	if err != nil {
		sess.showOrgMessage("Error updating folder for entry %d to %s: %v", id, new_folder, err)
		return
	}
}

func updateNote() {

	text := sess.p.rowsToString()

	// need to escape single quotes with two single quotes

	//stmt, err := db.Prepare("UPDATE task SET note=?, modified=datetime('now') WHERE id=?;")

	res, err := db.Exec("UPDATE task SET note=?, modified=datetime('now') WHERE id=?;",
		text, sess.p.id)
	if err != nil {
		log.Fatal(err)
	}

	//defer stmt.Close()

	/*
		res, err := stmt.Exec(text, sess.p.id)
		if err != nil {
			log.Fatal(err)
		}
	*/

	numRows, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	if numRows != 1 {
		log.Fatal("updateTaskFolder: numRows != 1")
	}

	/***************fts virtual table update*********************/

	_, err = fts_db.Exec("UPDATE fts SET note=? WHERE lm_id=?;", text, sess.p.id)
	if err != nil {
		log.Fatal(err)
	}

	sess.showOrgMessage("Updated note and fts entry for item %d", sess.p.id)
}

func getSyncItems(max int) {
	rows, err := db.Query(fmt.Sprintf("SELECT id, title, modified FROM sync_log ORDER BY modified DESC LIMIT %d", max))
	if err != nil {
		sess.showOrgMessage("Error in getSyncItems: %v", err)
		return
	}

	defer rows.Close()

	org.rows = nil
	for rows.Next() {
		var row Row
		var modified string

		err = rows.Scan(&row.id,
			&row.title,
			&modified,
		)

		if err != nil {
			sess.showOrgMessage("Error in getSyncItems: %v", err)
			return
		}

		row.modified = timeDelta(modified)
		org.rows = append(org.rows, row)

	}
}

func getItems(max int) {

	org.rows = nil
	org.fc, org.fr, org.rowoff = 0, 0, 0

	var arg string

	s := "SELECT task.id, task.title, task.star, task.deleted, task.completed, task.modified FROM task "

	if org.taskview == BY_CONTEXT {
		s += "JOIN context ON context.tid=task.context_tid WHERE context.title=?"
		arg = org.context
	} else if org.taskview == BY_FOLDER {
		s += "JOIN folder ON folder.tid = task.folder_tid WHERE folder.title=?"
		arg = org.folder
	} else if org.taskview == BY_KEYWORD {
		s += "JOIN task_keyword ON task.id=task_keyword.task_id " +
			"JOIN keyword ON keyword.id=task_keyword.keyword_id " +
			"WHERE task.id = task_keyword.task_id AND " +
			"task_keyword.keyword_id = keyword.id AND keyword.name=?"
		arg = org.keyword
	} else if org.taskview == BY_RECENT {
		s += "WHERE 1=1"
		arg = ""
	} else {
		sess.showOrgMessage("You asked for an unsupported db query")
		return
	}

	if !org.show_deleted {
		s += " AND task.completed IS NULL AND task.deleted=false"
	}
	s += fmt.Sprintf(" ORDER BY task.star DESC, task.%s DESC LIMIT %d;", org.sort, max)
	//int sortcolnum = org.sort_map[org.sort] //cpp
	var rows *sql.Rows
	var err error
	if arg == "" { //Recent
		rows, err = db.Query(s)
	} else {
		rows, err = db.Query(s, arg)
	}
	if err != nil {
		sess.showOrgMessage("Error in getItems: %v", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var row Row
		var completed sql.NullTime
		var modified string

		err = rows.Scan(&row.id,
			&row.title,
			&row.star,
			&row.deleted,
			&completed,
			&modified,
		)

		if err != nil {
			log.Fatal(err)
		}

		if completed.Valid {
			row.completed = true
		} else {
			row.completed = false
		}

		row.modified = timeDelta(modified)

		org.rows = append(org.rows, row)

	}

	org.view = TASK

	if len(org.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		org.mode = NO_ROWS
		sess.eraseRightScreen() // in case there was a note displayed in previous view
	} else {
		org.mode = org.last_mode
		sess.drawPreviewWindow(org.rows[0].id) //if id == -1 does not try to retrieve note
	}
}

func updateTitle() {

	// needs to be a pointer because may send to insertRow
	row := &org.rows[org.fr]

	/* check is in calling method writeTitle
	if !row.dirty {
		sess.showOrgMessage("Row has not been changed")
		return
	}
	*/

	if row.id == -1 {
		// want to send pointer to insertRow
		insertRow(row)
		return
	}

	res, err := db.Exec("UPDATE task SET title=?, modified=datetime('now') WHERE id=?", row.title, row.id)
	if err != nil {
		log.Fatal(err)
	}

	_, err = res.RowsAffected()
	if err != nil {
		log.Fatal(err)
		return
	}

	//row.dirty = false // done in caller
	/***************fts virtual table update*********************/
	//_, err = fts_db.Exec("INSERT INTO fts (title, lm_id) VALUES (?, ?);", row.title, row.id)
	_, err = fts_db.Exec("UPDATE fts SET title=? WHERE lm_id=?;", row.title, row.id)
	if err != nil {
		log.Fatal(err)
		return
	}
	//sess.showOrgMessage("Updated title for %v and indexed it", row.id)
}

func updateRows() {
	var updated_rows []int

	for _, row := range org.rows {
		if !row.dirty {
			continue
		}

		if row.id == -1 {
			id := insertRow(&row)
			updated_rows = append(updated_rows, id)
			row.dirty = false
			continue
		}

		res, err := db.Exec("UPDATE task SET title=?, modified=datetime('now') WHERE id=?", row.title, row.id)
		if err != nil {
			log.Fatal(err)
		}

		_, err = res.RowsAffected()
		if err != nil {
			log.Fatal(err)
			return
		}

		row.dirty = false
		updated_rows = append(updated_rows, row.id)
	}

	if len(updated_rows) == 0 {
		sess.showOrgMessage("There were no rows to update")
		return
	}
	sess.showOrgMessage("These ids were updated: %v", updated_rows)
}

func insertRow(row *Row) int {

	var folder_tid int
	var context_tid int

	if org.context == "" {
		context_tid = 1
	} else {
		context_tid = org.context_map[org.context]
	}

	if org.folder == "" {
		folder_tid = 1
	} else {
		folder_tid = org.folder_map[org.folder]
	}
	res, err := db.Exec("INSERT INTO task (priority, title, folder_tid, context_tid, "+
		"star, added, note, deleted, created, modified) "+
		"VALUES (3, ?, ?, ?, True, date(), '', False, "+
		//fmt.Sprintf("datetime('now', '-%s hours'), ", TZ_OFFSET)+
		"date(), datetime('now'));",
		row.title, folder_tid, context_tid)

	/*
	   not used:
	   tid,
	   tag,
	   duetime,
	   completed,
	   duedate,
	   repeat,
	   remind
	*/
	if err != nil {
		return -1
	}

	row_id, err := res.LastInsertId()
	if err != nil {
		log.Fatal(err)
		return -1
	}
	row.id = int(row_id)
	row.dirty = false

	/***************fts virtual table update*********************/

	//should probably create a separate function that is a klugy
	//way of making up for fact that pg created tasks don't appear in fts db
	//"INSERT OR IGNORE INTO fts (title, lm_id) VALUES ('" << title << row.id << ");";
	/***************fts virtual table update*********************/
	_, err = fts_db.Exec("INSERT INTO fts (title, lm_id) VALUES (?, ?);", row.title, row.id)
	if err != nil {
		log.Fatal(err)
		return -1
	}

	sess.showOrgMessage("Successfully inserted new row with id {} and indexed it (new vesrsion)", row.id)

	return row.id
}

func insertSyncEntry(title, note string) {
	_, err := db.Exec("INSERT INTO sync_log (title, note, modified) VALUES (?, ?, datetime('now'));",
		title, note)
	if err != nil {
		sess.showOrgMessage("Error inserting sync log into db: %v", err)
	} else {
		sess.showOrgMessage("Wrote sync log to db")
	}
}

func readNoteIntoString(id int) string {
	if id == -1 {
		return "" // id given to new and unsaved entries
	}

	row := db.QueryRow("SELECT note FROM task WHERE id=?;", id)
	var note string
	err := row.Scan(&note)
	if err != nil {
		return ""
	}
	return note
}

func readNoteIntoEditor(id int) {
	if id == -1 {
		return // id given to new and unsaved entries
	}

	row := db.QueryRow("SELECT note FROM task WHERE id=?;", id)
	var note string
	err := row.Scan(&note)
	if err != nil {
		return
	}

	//? use scan which will catch /r/n
	note = strings.ReplaceAll(note, "\r", "")
	sess.p.rows = strings.Split(note, "\n")
	//rows := strings.Split(note, "\n")
	// send note to nvim
	var bb [][]byte
	for _, s := range sess.p.rows {
		bb = append(bb, []byte(s))
	}
	//func (v *Nvim) CreateBuffer(listed bool, scratch bool) (buffer Buffer, err error) {
	//sess.p.vbuf, err = v.CreateBuffer(true, false)
	sess.p.vbuf, err = v.CreateBuffer(true, true)
	if err != nil {
		sess.showOrgMessage("%v", err)
	}
	err = v.SetCurrentBuffer(sess.p.vbuf)
	if err != nil {
		sess.showOrgMessage("%v", err)
	} else {
		sess.showOrgMessage("%v", sess.p.vbuf)
	}
	v.SetBufferLines(sess.p.vbuf, 0, -1, true, bb)

}

func readSyncLogIntoAltRows(id int) {
	row := db.QueryRow("SELECT note FROM sync_log WHERE id=?;", id)
	var note string
	err := row.Scan(&note)
	if err != nil {
		return
	}
	org.altRows = nil
	for _, line := range strings.Split(note, "\n") {
		var r AltRow
		r.title = line
		org.altRows = append(org.altRows, r)
	}

}

func readSyncLog(id int) string {
	row := db.QueryRow("SELECT note FROM sync_log WHERE id=?;", id)
	var note string
	err := row.Scan(&note)
	if err != nil {
		return ""
	}
	return note
}

func getEntryInfo(id int) Entry {
	if id == -1 {
		return Entry{}
	}
	row := db.QueryRow("SELECT id, tid, title, created, folder_tid, context_tid, star, added, completed, deleted, modified FROM task WHERE id=?;", id)

	var e Entry
	var tid sql.NullInt64
	err := row.Scan(
		&e.id,
		&tid,
		&e.title,
		&e.created,
		&e.folder_tid,
		&e.context_tid,
		&e.star,
		&e.added,
		&e.completed,
		&e.deleted,
		&e.modified,
	)
	if err != nil {
		log.Fatal(err)
		return Entry{}
	}
	if tid.Valid {
		e.tid = int(tid.Int64)
	} else {
		e.tid = 0
	}
	return e
}

func getFolderTid(id int) int {
	row := db.QueryRow("SELECT folder_tid FROM task WHERE id=?;", id)
	var tid int
	err := row.Scan(&tid)
	if err != nil {
		return -1
	}
	return tid
}

// used in Editor.cpp
func getTitle(id int) string {
	row := db.QueryRow("SELECT title FROM task WHERE id=?;", id)
	var title string
	err := row.Scan(&title)
	if err != nil {
		return ""
	}
	return title
}

func getTaskKeywords(id int) string {

	rows, err := db.Query("SELECT keyword.name FROM task_keyword LEFT OUTER JOIN keyword ON "+
		"keyword.id=task_keyword.keyword_id WHERE task_keyword.task_id=?;",
		id)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	kk := []string{}
	for rows.Next() {
		var name string

		err = rows.Scan(&name)
		kk = append(kk, name)
	}
	if len(kk) == 0 {
		return ""
	}
	return strings.Join(kk, ",")
}

func (o *Organizer) searchDB(st string, help bool) {

	o.rows = nil
	o.fc, o.fr, o.rowoff = 0, 0, 0

	rows, err := fts_db.Query("SELECT lm_id, highlight(fts, 0, '\x1b[48;5;31m', '\x1b[49m') "+
		"FROM fts WHERE fts MATCH ? ORDER BY bm25(fts, 2.0, 1.0, 5.0);",
		st)

	defer rows.Close()

	o.fts_ids = nil

	for k := range o.fts_titles {
		delete(o.fts_titles, k)
	}

	for rows.Next() {
		var fts_id int
		var fts_title string

		err = rows.Scan(
			&fts_id,
			&fts_title,
		)

		if err != nil {
			sess.showOrgMessage("Error trying to retrieve search info from fts_db - term: %s; %v", st, err)
			return
		}
		o.fts_ids = append(o.fts_ids, fts_id)
		o.fts_titles[fts_id] = fts_title
	}

	if len(o.fts_ids) == 0 {
		sess.showOrgMessage("No results were returned")
		sess.eraseRightScreen() //note can still return no rows from get_items_by_id if we found rows above that were deleted
		org.mode = NO_ROWS
		return
	}

	var stmt string

	// As noted above, if the item is deleted (gone) from the db it's id will not be found if it's still in fts
	if help {
		stmt = "SELECT task.id, task.title, task.star, task.deleted, task.completed, task.modified FROM task WHERE task.context_tid = 16 and task.id IN ("
	} else {
		stmt = "SELECT task.id, task.title, task.star, task.deleted, task.completed, task.modified FROM task WHERE task.id IN ("
	}

	max := len(o.fts_ids) - 1
	for i := 0; i < max; i++ {
		stmt += strconv.Itoa(o.fts_ids[i]) + ", "
	}

	stmt += strconv.Itoa(o.fts_ids[max]) + ")"
	//stmt +=      << ((!org.show_deleted) ? " AND task.completed IS NULL AND task.deleted = False" : "")
	stmt += " AND task.completed IS NULL AND task.deleted = False ORDER BY "

	for i := 0; i < max; i++ {
		stmt += "task.id = " + strconv.Itoa(o.fts_ids[i]) + " DESC, "
	}
	stmt += "task.id = " + strconv.Itoa(o.fts_ids[max]) + " DESC"

	rows, err = db.Query(stmt)
	for rows.Next() {
		var row Row
		var completed sql.NullString
		var modified string

		err = rows.Scan(
			&row.id,
			&row.title,
			&row.star,
			&row.deleted,
			&completed,
			&modified,
		)

		if err != nil {
			sess.showOrgMessage("Error in searchDB()")
			return
		}

		if completed.Valid {
			row.completed = true
		} else {
			row.completed = false
		}

		row.modified = timeDelta(modified)
		row.fts_title = o.fts_titles[row.id]

		o.rows = append(o.rows, row)
	}

	// think these are the initialized values
	//row.dirty = false;
	//row.mark = false;

	if len(o.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		o.mode = NO_ROWS
		sess.eraseRightScreen() // in case there was a note displayed in previous view
	} else {
		o.mode = FIND
		sess.drawPreviewWindow(o.rows[0].id) //if id == -1 does not try to retrieve note
	}
}

func getContainers() {
	org.rows = nil

	var table string
	var columns string
	var orderBy string //only needs to be change for keyword

	switch org.view {
	case CONTEXT:
		table = "context"
		columns = "id, title, \"default\", deleted, modified"
		orderBy = "title"
	case FOLDER:
		table = "folder"
		columns = "id, title, private, deleted, modified"
		orderBy = "title"
	case KEYWORD:
		table = "keyword"
		columns = "id, name, star, deleted, modified"
		orderBy = "name"
	default:
		sess.showOrgMessage("Somehow you are in a view I can't handle")
		return
	}

	stmt := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s COLLATE NOCASE ASC;", columns, table, orderBy)
	rows, err := db.Query(stmt)
	if err != nil {
		sess.showOrgMessage("Error SELECTING %s FROM %s", columns, table)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var r Row
		var modified string
		rows.Scan(
			&r.id,
			&r.title,
			&r.star,
			&r.deleted,
			&modified,
		)

		r.modified = timeDelta(modified)
		org.rows = append(org.rows, r)
	}
	if len(org.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		org.mode = NO_ROWS
	}

	// below should be somewhere else
	org.fc, org.fr, org.rowoff = 0, 0, 0
	org.context, org.folder, org.keyword = "", "", "" // this makes sense if you are not in an O.view == TASK

}

func getAltContainers() {
	org.altRows = nil

	var table string
	var columns string
	var orderBy string //only needs to be change for keyword

	switch org.altView {
	case CONTEXT:
		table = "context"
		columns = "id, title, \"default\""
		orderBy = "title"
	case FOLDER:
		table = "folder"
		columns = "id, title, private"
		orderBy = "title"
	case KEYWORD:
		table = "keyword"
		columns = "id, name, star"
		orderBy = "name"
	default:
		sess.showOrgMessage("Somehow you are in a view I can't handle")
		return
	}

	stmt := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s COLLATE NOCASE ASC;", columns, table, orderBy)
	rows, err := db.Query(stmt)
	if err != nil {
		sess.showOrgMessage("Error SELECTING %s FROM %s", columns, table)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var r AltRow
		rows.Scan(
			&r.id,
			&r.title,
			&r.star,
		)

		org.altRows = append(org.altRows, r)
	}
	/*
		if len(org.altRows) == 0 {
			sess.showOrgMessage("No results were returned")
		}
	*/

	// below should ? be somewhere else
	org.altR = 0

}

func getContainerInfo(id int) Container {

	/*
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
	*/

	if id == -1 {
		return Container{}
	}

	var table string
	var countQuery string
	var columns string
	switch org.view {
	case CONTEXT:
		table = "context"
		countQuery = "SELECT COUNT(*) FROM task JOIN context ON context.tid = task.context_tid WHERE context.id=?;"
		columns = "id, tid, title, \"default\", created, deleted, modified"
	case FOLDER:
		table = "folder"
		countQuery = "SELECT COUNT(*) FROM task JOIN folder ON folder.tid = task.folder_tid WHERE folder.id=?;"
		columns = "id, tid, title, private, created, deleted, modified"
	case KEYWORD:
		table = "keyword"
		countQuery = "SELECT COUNT(*) FROM task_keyword WHERE keyword_id=?;"
		columns = "id, tid, name, star, deleted, modified"
	default:
		sess.showOrgMessage("Somehow you are in a view I can't handle")
		return Container{}
	}

	var c Container

	row := db.QueryRow(countQuery, id)
	err := row.Scan(&c.count)
	if err != nil {
		sess.showOrgMessage("Error in getContainerInfo: %v", err)
		return Container{}
	}

	stmt := fmt.Sprintf("SELECT %s FROM %s WHERE id=?;", columns, table)
	row = db.QueryRow(stmt, id)
	var tid sql.NullInt64
	if org.view == KEYWORD {
		err = row.Scan(
			&c.id,
			&tid,
			&c.title,
			&c.star,
			&c.deleted,
			&c.modified,
		)
	} else {
		err = row.Scan(
			&c.id,
			&tid,
			&c.title,
			&c.star,
			&c.created,
			&c.deleted,
			&c.modified,
		)
	}
	if err != nil {
		sess.showOrgMessage("Error in getContainerInfo: %v", err)
		return Container{}
	}

	if tid.Valid {
		c.tid = int(tid.Int64)
	} else {
		c.tid = 0
	}

	return c
}

func addTaskKeyword(keyword_id, entry_id int, update_fts bool) {

	_, err := db.Exec("INSERT OR IGNORE INTO task_keyword (task_id, keyword_id) VALUES (?, ?);",
		entry_id, keyword_id)

	if err != nil {
		sess.showOrgMessage("Error in addTaskKeyword = INSERT or IGNORE INTO task_keyword: %v", err)
		return
	}

	_, err = db.Exec("UPDATE task SET modified = datetime('now') WHERE id=?;", entry_id)
	if err != nil {
		sess.showOrgMessage("Error in addTaskKeyword - Update task modified: %v", err)
		return
	}

	// *************fts virtual table update**********************
	if !update_fts {
		return
	}
	s := getTaskKeywords(entry_id)
	_, err = fts_db.Exec("UPDATE fts SET tag=? WHERE lm_id=?;", s, entry_id)
	if err != nil {
		sess.showOrgMessage("Error in addTaskKeyword - fts Update: %v", err)
	}
}

func getNoteSearchPositions(id int) [][]int {
	row := fts_db.QueryRow("SELECT rowid FROM fts WHERE lm_id=?;", id)
	var rowid int
	err := row.Scan(&rowid)
	if err != nil {
		return [][]int{}
	}
	var word_positions [][]int
	for i, term := range strings.Split(sess.fts_search_terms, " ") {
		word_positions = append(word_positions, []int{})
		rows, err := fts_db.Query("SELECT offset FROM fts_v WHERE doc=? AND term=? AND col='note';", rowid, term)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		for rows.Next() {
			var offset int
			err = rows.Scan(&offset)
			if err != nil {
				log.Fatal(err)
			}
			word_positions[i] = append(word_positions[i], offset)
		}
	}
	return word_positions
}

func updateContainerTitle() {

	row := &org.rows[org.fr]

	if !row.dirty {
		sess.showOrgMessage("Row has not been changed")
		return
	}

	if row.id == -1 {
		insertContainer(row)
		return
	}

	var table string
	var column string
	switch org.view {
	case CONTEXT:
		table = "context"
		column = "title"
	case FOLDER:
		table = "folder"
		column = "title"
	case KEYWORD:
		table = "keyword"
		column = "name"
	default:
		sess.showOrgMessage("Somehow that's a container I don't recognize")
		return
	}

	stmt := fmt.Sprintf("UPDATE %s SET %s=?, modified=datetime('now') WHERE id=?",
		table, column)
	_, err := db.Exec(stmt, row.title, row.id)
	if err != nil {
		sess.showOrgMessage("Error updating %s title for %d", table, row.id)
	}
}

func insertContainer(row *Row) int {

	var stmt string
	if org.view != KEYWORD {
		var table string
		var star string
		switch org.view {
		case CONTEXT:
			table = "context"
			star = "\"default\""
		case FOLDER:
			table = "folder"
			star = "private"
		default:
			sess.showOrgMessage("Somehow that's a container I don't recognize")
			return -1
		}

		stmt = fmt.Sprintf("INSERT INTO %s (title, %s, deleted, created, modified, tid, textcolor) ",
			table, star)

		stmt += "VALUES (?, ?, False, datetime('now'), datetime('now'), 0, 10);"
	} else {

		stmt = "INSERT INTO keyword (name, star, deleted, modified, tid) " +
			"VALUES (?, ?, False, datetime('now'), 0);"
	}

	res, err := db.Exec(stmt, row.title, row.star)
	if err != nil {
		sess.showOrgMessage("Error in insertContainer: %v", err)
		return -1
	}

	id, _ := res.LastInsertId()
	row.id = int(id)
	row.dirty = false

	return row.id
}

func deleteKeywords(id int) int {
	res, err := db.Exec("DELETE FROM task_keyword WHERE task_id=?;", id)
	if err != nil {
		sess.showOrgMessage("Error deleting from task_keyword: %v", err)
		return -1
	}
	rowsAffected, _ := res.RowsAffected()
	_, err = db.Exec("UPDATE task SET modified=datetime('now') WHERE id=?;", id)
	if err != nil {
		sess.showOrgMessage("Error updating entry modified column in deleteKeywords: %v", err)
		return -1
	}

	_, err = fts_db.Exec("UPDATE fts SET tag='' WHERE lm_id=?", id)
	if err != nil {
		sess.showOrgMessage("Error updating fts in deleteKeywords: %v", err)
		return -1
	}
	return int(rowsAffected)
}

func highlight_terms_string(text string, word_positions [][]int) string {

	delimiters := " |,.;?:()[]{}&#/`-'\"—_<>$~@=&*^%+!\t\n\\" //must have \f if using it as placeholder

	for _, v := range word_positions {
		sess.showEdMessage("%v", word_positions)

		// start and end are positions in the text
		// word_num is what word number we are at in the text
		//wp is the position that we are currently looking for to highlight

		word_num := -1 //word position in text
		end := -1
		var start int

		for _, wp := range v {

			for {
				// I don't think the check below is necessary but we'll see
				if end >= len(text)-1 {
					break
				}

				start = start + end + 1
				end = strings.IndexAny(text[start:], delimiters)
				if end == -1 {
					end = len(text) - 1
				}

				if end != 0 { //if end = 0 we were sitting on a delimiter like a space
					word_num++
				}

				if wp == word_num {
					text = text[:start+end] + "\x1b[48;5;235m" + text[start+end:]
					text = text[:start] + "\x1b[48;5;31m" + text[start:]
					end += 21
					break // this breaks out of loop that was looking for the current highlighted word position
				}
			}
		}
	}
	return text
}

func generateWWString(text string, width int, length int, ret string) string {

	if text == "" {
		return ""
	}
	ss := strings.Split(text, "\n")
	var ab strings.Builder

	y := 0
	filerow := 0

	for _, s := range ss {
		if filerow == len(ss) {
			return ab.String()
		}

		if s == "" {
			if y == length-1 {
				return ab.String()
			}
			ab.WriteString(ret)
			filerow++
			y++
			continue
		}

		pos := 0
		prev_pos := 0

		for {
			if prev_pos+width > len(s)-1 {
				ab.WriteString(s[prev_pos:])
				if y == length-1 {
					return ab.String()
				}
				ab.WriteString(ret)
				y++
				filerow++
				break
			}
			pos = strings.LastIndex(s[:prev_pos+width], " ")
			if pos == -1 || pos == prev_pos-1 {
				pos = prev_pos + width - 1
			}

			ab.WriteString(s[prev_pos : pos+1])

			if y == length-1 {
				return ab.String()
			}
			ab.WriteString(ret)
			y++
			prev_pos = pos + 1
		}
	}
	return ab.String()
}

func updateCodeFile() {
	sess.showOrgMessage("got here")

	var filePath string
	if tid := getFolderTid(sess.p.id); tid == 18 {
		filePath = "/home/slzatz/clangd_examples/test.cpp"
		//lsp_name = "clangd";
	} else {
		filePath = "/home/slzatz/go_fragments/main.go"
		//lsp_name = "gopls";
	}

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		sess.showEdMessage("error opening file %s: %w", filePath, err)
		return
	}
	defer f.Close()

	f.Truncate(0)

	//n, err := f.WriteString(sess.p.code)
	f.WriteString(sess.p.code)

	f.Sync()

	/*
	  std::string lsp_name;

	  if (tid == 18) {
	    file_path  = "/home/slzatz/clangd_examples/test.cpp";
	    lsp_name = "clangd";
	  } else {
	    file_path = "/home/slzatz/go/src/example/main.go";
	    lsp_name = "gopls";
	  }

	  if (!sess.lsp_v.empty()) {
	    auto it = std::ranges::find_if(sess.lsp_v, [&lsp_name](auto & lsp){return lsp->name == lsp_name;});
	    if (it != sess.lsp_v.end()) (*it)->code_changed = true;
	  }
	*/
}
