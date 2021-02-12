package main

import (
        "fmt"
        "database/sql"
      _ "github.com/mattn/go-sqlite3"
)

var db, _ = sql.Open("sqlite3", "/home/slzatz/mylistmanager3/lmdb_s/mylistmanager_s.db")
var fts_db, _ = sql.Open("sqlite3", "/home/slzatz/listmanager_cpp/fts5.db")

func getId() int {
  return org.rows[org.fr].id
}

func toggleStar() {
  //orow& row = org.rows.at(org.fr);
  id := getId();

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
      sess.showOrgMessage("Not sure what you're trying to toggle");
      return
  }

  db, _ := sql.Open("sqlite3", "/home/slzatz/mylistmanager3/lmdb_s/mylistmanager_s.db")

  stmt, err := db.Prepare(fmt.Sprintf("UPDATE %s SET %s=?, modified=datetime('now') WHERE id=?;",
                                   table, column))

  if err != nil {
    log.Fatal(err)
  }

  defer stmt.Close()

  res, err := stmt.Exec(!org.rows[org.fr].star, id)
  if err != nil {
    log.Fatal(err)
  }

  numRows, err := res.RowsAffected()
  if err != nil {
    log.Fatal(err)
  }
  if numRows != 1 {
    log.Fatal("Toggle star numRows != 1")
  }
  //LastInsertId() (int64, error)

  org.rows[org.fr].star = !org.rows[org.fr].star;
  sess.showOrgMessage("Toggle star succeeded");
}

func toggleDeleted() {
  orow& row = org.rows.at(org.fr);
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
      sess.showOrgMessage("Somehow you are in a view I can't handle");
      return
  }

  stmt, err := db.Prepare(fmt.Sprintf("UPDATE %s SET deleted=?, modified=datetime('now') WHERE id=?;",
                                   table))

  if err != nil {
    log.Fatal(err)
  }

  defer stmt.Close()

  res, err := stmt.Exec(!org.rows[org.fr].deleted, id)
  if err != nil {
    log.Fatal(err)
  }

  numRows, err := res.RowsAffected()
  if err != nil {
    log.Fatal(err)
  }
  if numRows != 1 {
    log.Fatal("Toggle deleted numRows != 1")
  }
  //LastInsertId() (int64, error)

  org.rows[org.fr].star = !org.rows[org.fr].deleted
  sess.showOrgMessage("Toggle deleted succeeded");
}

func toggleCompleted() {
  //orow& row = org.rows.at(org.fr);
  id := getId()

  stmt, err := db.Prepare("UPDATE tasks SET completed=?, modified=datetime('now') WHERE id=?;")

  if err != nil {
    log.Fatal(err)
  }

  defer stmt.Close()

  var completed string
  if org.rows[org.fr].completed {
    completed = "NULL"
  } else
    completed = "date()"
  }

  res, err := stmt.Exec(completed, id)
  if err != nil {
    log.Fatal(err)
  }

  numRows, err := res.RowsAffected()
  if err != nil {
    log.Fatal(err)
  }

  if numRows != 1 {
    log.Fatal("Toggle completed numRows != 1")
  }
  //LastInsertId() (int64, error)

  org.rows[org.fr].completed = !org.rows[org.fr].completed
  sess.showOrgMessage("Toggle completed succeeded");
}

func updateTaskContext(new_context string, id int) {
  //id := getId()
  context_tid := org.context_map.at(new_context); ////////

  stmt, err := db.Prepare("UPDATE task SET context_tid=?, modified=datetime('now') WHERE id=?;")

  if err != nil {
    log.Fatal(err)
  }

  defer stmt.Close()

  res, err := stmt.Exec(context_tid, id)
  if err != nil {
    log.Fatal(err)
  }

  numRows, err := res.RowsAffected()
  if err != nil {
    log.Fatal(err)
  }

  if numRows != 1 {
    log.Fatal("updateTaskContext: numRows != 1")
  }
  //LastInsertId() (int64, error)

  org.rows[org.fr].completed = !org.rows[org.fr].completed
  sess.showOrgMessage("Update task context succeeded");
  }
  // doesn't get called
  //sess.showOrgMessage3("Update task context succeeded (new version)");
}

func updateTaskFolder(new_folder string, id int) {
  //id := getId()
  folder_tid := org.context_map.at(new_context); ////////

  stmt, err := db.Prepare("UPDATE task SET folder_tid=?, modified=datetime('now') WHERE id=?;")

  if err != nil {
    log.Fatal(err)
  }

  defer stmt.Close()

  res, err := stmt.Exec(context_tid, id)
  if err != nil {
    log.Fatal(err)
  }

  numRows, err := res.RowsAffected()
  if err != nil {
    log.Fatal(err)
  }

  if numRows != 1 {
    log.Fatal("updateTaskFolder: numRows != 1")
  }
  //LastInsertId() (int64, error)

  org.rows[org.fr].completed = !org.rows[org.fr].completed
  sess.showOrgMessage("Update task folder succeeded");
  }
  // doesn't get called
  //sess.showOrgMessage3("Update task context succeeded (new version)");
}

func updateNote() {

  text := sess.p.editorRowsToString()

  // need to escape single quotes with two single quotes

  stmt, err := db.Prepare("UPDATE task SET note=?, modified=datetime('now') WHERE id=?;")
  if err != nil {
    log.Fatal(err)
  }

  defer stmt.Close()

  res, err := stmt.Exec(text, sess.p.id)
  if err != nil {
    log.Fatal(err)
  }

  numRows, err := res.RowsAffected()
  if err != nil {
    log.Fatal(err)
  }

  if numRows != 1 {
    log.Fatal("updateTaskFolder: numRows != 1")
  }

  /***************fts virtual table update*********************/

  stmt2, err := fts_db.Prepare("UPDATE fts SET note=? WHERE lm_id=?;")
  if err != nil {
    log.Fatal(err)
  }

  defer stmt2.Close()

  res, err := stmt2.Exec(text, sess.p.id)
  if err != nil {
    log.Fatal(err)
  }

  numRows, err := res.RowsAffected()
  if err != nil {
    log.Fatal(err)
  }

  sess.showOrgMessage("Updated note and fts entry for item {} (new version)", sess.p->id);
}

func getItems(max int) {
  std::stringstream stmt;
  std::vector<std::string> keyword_vec;

  org.rows = nil
  org.fc = org.fr = org.rowoff = 0

  var s string
  if org.taskview == BY_CONTEXT {
    s = "SELECT * FROM task JOIN context ON context.tid=task.context_tid WHERE context.title=?"
    filter = org.context
  } else if org.taskview == BY_FOLDER {
    s = "SELECT * FROM task JOIN folder ON folder.tid = task.folder_tid WHERE folder.title=?"
    filter = org.folder
  } else if org.taskview == BY_KEYWORD {
    s = "SELECT * FROM task JOIN task_keyword ON task.id=task_keyword.task_id JOIN keyword ON keyword.id=task_keyword.keyword_id"
           " WHERE task.id = task_keyword.task_id AND task_keyword.keyword_id = keyword.id AND keyword.name=?";
    filter = org.keyword
  } else if org.taskview == BY_RECENT {
    s = "SELECT * FROM task WHERE 1=?";
    filter = "1"
  } else {
    sess.showOrgMessage("You asked for an unsupported db query");
    return;
  }

  if !org.show_deleted {
    s += " AND task.completed IS NULL AND task.deleted=false"
  }
  s += fmt.Sprintf(" ORDER BY task.star DESC, task.%s DESC LIMIT %d", org.sort, max)
  int sortcolnum = org.sort_map[org.sort]


  rows, err = db.Query(s, filter)
  if err != nil {
    log.Fatal(err)
  }
  defer rows.Close()

  for rows.Next() {
    var (
         id int64
         title string
         star bool
         deleted bool
       )

    err = rows.Scan(&row.id,
                    &row.title,
                    &row.star,
                    &row.deleted)
    if  err != nil {
      log.Fatal(err)
    }

    var row entry
    entry.id = id
    entry.title = title
    entry.star = star

  Query q(db, stmt.str()); 

  if (q.result != SQLITE_OK) {
    sess.showOrgMessage3("Problem in 'getItems'; result code: {}", q.result);
    return;
  }
  while (q.step() == SQLITE_ROW) {
    orow row;
    row.id = q.column_int(0);
    row.title = q.column_text(3);
    row.star = q.column_bool(8);
    row.deleted = q.column_bool(14);
    row.completed = (q.column_text(10) != "") ? true: false;

    if (q.column_text(sortcolnum) != "") row.modified = timeDelta(q.column_text(sortcolnum));
    else row.modified.assign(15, ' ');

    row.dirty = false;
    row.mark = false;

    org.rows.push_back(row);
  }

  org.view = TASK;

  if (org.rows.empty()) {
    sess.showOrgMessage("No results were returned");
    org.mode = NO_ROWS;
    sess.eraseRightScreen(); // in case there was a note displayed in previous view
  } else {
    org.mode = org.last_mode;
    sess.drawPreviewWindow(org.rows.at(org.fr).id); //if id == -1 does not try to retrieve note
  }
}
