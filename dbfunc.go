package main

import (
	"database/sql"
	"fmt"
  "log"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

var db, _ = sql.Open("sqlite3", "/home/slzatz/mylistmanager3/lmdb_s/mylistmanager_s.db")
var fts_db, _ = sql.Open("sqlite3", "/home/slzatz/listmanager_cpp/fts5.db")

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
	res, err = db.Exec(s, !org.rows[org.fr].deleted, id)

	/*
		stmt, err := db.Prepare(fmt.Sprintf("UPDATE %s SET deleted=?, modified=datetime('now') WHERE id=?;",
			table))
	*/

	if err != nil {
		log.Fatal(err)
	}

	//defer stmt.Close()

	/*
		res, err := stmt.Exec(!org.rows[org.fr].deleted, id)
		if err != nil {
			log.Fatal(err)
		}
	*/

	numRows, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	if numRows != 1 {
		log.Fatal("Toggle deleted numRows != 1")
	}
	//LastInsertId() (int64, error)

	org.rows[org.fr].star = !org.rows[org.fr].deleted
	sess.showOrgMessage("Toggle deleted succeeded")
}

func toggleCompleted() {
	//orow& row = org.rows.at(org.fr);
	id := getId()

	var completed string
	if org.rows[org.fr].completed {
		completed = "NULL"
	} else {
		completed = "date()"
	}

	res, err := db.Exec("UPDATE tasks SET completed=?, "+
		"modified=datetime('now') WHERE id=?;",
		completed, id)

	//stmt, err := db.Prepare("UPDATE tasks SET completed=?, modified=datetime('now') WHERE id=?;")

	if err != nil {
		log.Fatal(err)
	}

	//defer stmt.Close()
	/*
		res, err := stmt.Exec(completed, id)
		if err != nil {
			log.Fatal(err)
		}
	*/

	numRows, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	if numRows != 1 {
		log.Fatal("Toggle completed numRows != 1")
	}
	//LastInsertId() (int64, error)

	org.rows[org.fr].completed = !org.rows[org.fr].completed
	sess.showOrgMessage("Toggle completed succeeded")
}

func updateTaskContext(new_context string, id int) {
	//id := getId()
	context_tid := org.context_map.at(new_context) ////////

	res, err := db.exec("UPDATE task SET context_tid=?, modified=datetime('now') "+
		"WHERE id=?;", context_tid, id)

	//stmt, err := db.Prepare("UPDATE task SET context_tid=?, modified=datetime('now') WHERE id=?;")

	if err != nil {
		log.Fatal(err)
	}

	//defer stmt.Close()

	/*
		res, err := stmt.Exec(context_tid, id)
		if err != nil {
			log.Fatal(err)
		}
	*/

	numRows, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	if numRows != 1 {
		log.Fatal("updateTaskContext: numRows != 1")
	}
	//LastInsertId() (int64, error)

	org.rows[org.fr].completed = !org.rows[org.fr].completed
	sess.showOrgMessage("Update task context succeeded")
	// doesn't get called
	//sess.showOrgMessage3("Update task context succeeded (new version)");
}

func updateTaskFolder(new_folder string, id int) {
	//id := getId()
	folder_tid := org.context_map.at(new_context) ////////

	//stmt, err := db.Prepare("UPDATE task SET folder_tid=?, modified=datetime('now') WHERE id=?;")

	res, err := db.exec("UPDATE task SET folder_tid=?, modified=datetime('now') "+
		"WHERE id=?;", folder_tid, id)

	if err != nil {
		log.Fatal(err)
	}

	//defer stmt.Close()

	/*
		res, err := stmt.Exec(context_tid, id)
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
	//LastInsertId() (int64, error)

	org.rows[org.fr].completed = !org.rows[org.fr].completed
	sess.showOrgMessage("Update task folder succeeded")
	// doesn't get called
	//sess.showOrgMessage3("Update task context succeeded (new version)");
}

func updateNote() {

	text := sess.p.editorRowsToString()

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

func getItems(max int) {

	org.rows = nil
	org.fc, org.fr, org.rowoff = 0, 0, 0

	var arg string

	s := "SELECT task.id, task.title, task.star, task.deleted, task.completed, task.modified FROM task "

	if org.taskview == BY_CONTEXT {
		s = "JOIN context ON context.tid=task.context_tid WHERE context.title=?"
		arg = org.context
	} else if org.taskview == BY_FOLDER {
		s = "JOIN folder ON folder.tid = task.folder_tid WHERE folder.title=?"
		arg = org.folder
	} else if org.taskview == BY_KEYWORD {
		s = "JOIN task_keyword ON task.id=task_keyword.task_id " +
			"JOIN keyword ON keyword.id=task_keyword.keyword_id " +
			"WHERE task.id = task_keyword.task_id AND " +
			"task_keyword.keyword_id = keyword.id AND keyword.name=?"
		arg = org.keyword
	} else if org.taskview == BY_RECENT {
		s = "WHERE 1=1"
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

	if arg == "" { //Recent
		rows, err = db.Query(s)
	} else {
		rows, err = db.Query(s, filter)
	}
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	for rows.Next() {
		var row Entry
		//var completed string
		var completed sql.NullString
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
		sess.drawPreviewWindow(org.rows[org.fr].id) //if id == -1 does not try to retrieve note
	}
}

func updateRows() {
  var updated_rows []int

  for _, row := range org.rows {
    if !row.dirty {
      continue
    }

    if row.id == -1 {
      id = insertRow(row)
      append(updated_rows, id)
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
    append(updated_rows, row.id)
  }

  if (len(updated_rows) == 0) {
    sess.showOrgMessage("There were no rows to update")
    return
  }
  sess.showOrgMessage("These ids were updated: %v",  updated_rows);
}

func insertRow(row Entry) int {

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
  res, err = db.Exec("INSERT INTO task (priority, title, folder_tid, context_tid, " +
              "star, added, note, deleted, created, modified) " +
              "VALUES (3, ?, ?, ?, True, date(), '', False, " +
              fmt.Sprintf("datetime('now', '-%s hours'), ", TZ_OFFSET) +
              "datetime('now'));",
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

  row.id =  res.LastInsertId()
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

  sess.showOrgMessage("Successfully inserted new row with id {} and indexed it (new vesrsion)", row.id);

  return row.id
}
