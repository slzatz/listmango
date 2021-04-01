package main

/** note that sqlite datetime('now') returns utc **/

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/ioutil"
	//	"log"
	//"os"
	"strings"
	"time"
)

type dbConfig struct {
	Server struct {
		Host string `json:"host"`
		Port string `json:"port"`
	} `json:"server"`
	Postgres struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		DB       string `json:"db"`
		Test     string `json:"test"`
	} `json:"postgres"`

	Options struct {
		Prefix string `json:"prefix"`
	} `json:"options"`
}

// FromFile returns a dbConfig struct parsed from a file.
func FromFile(path string) (*dbConfig, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg dbConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func keywordExistsS(dbase *sql.DB, plg io.Writer, name string) int {
	row := dbase.QueryRow("SELECT keyword.id FROM keyword WHERE keyword.name=$1;", name)
	var id int
	err := row.Scan(&id)
	if err != nil {
		fmt.Fprintf(plg, "Error in keywordExistsS: %v\n", err)
		return -1
	}
	return id
}

func addTaskKeywordS(dbase *sql.DB, plg io.Writer, keyword_id, entry_id int) {
	_, err := dbase.Exec("INSERT INTO task_keyword (task_id, keyword_id) VALUES ($1, $2);",
		entry_id, keyword_id)
	if err != nil {
		fmt.Fprintf(plg, "Error in addTaskKeywordS = INSERT INTO task_keyword: %v\n", err)
		return
	} else {
		fmt.Fprintf(plg, "Inserted into task_keyword entry id %d and keyword_id %d\n", entry_id, keyword_id)
	}
}

func getTaskKeywordsS(dbase *sql.DB, plg io.Writer, id int) []string {

	rows, err := dbase.Query("SELECT keyword.name FROM task_keyword LEFT OUTER JOIN keyword ON "+
		"keyword.id=task_keyword.keyword_id WHERE task_keyword.task_id=$1;", id)
	if err != nil {
		fmt.Fprintf(plg, "Error in getTaskKeywordsS: %v\n", err)
	}
	defer rows.Close()

	kk := []string{}
	for rows.Next() {
		var name string

		err = rows.Scan(&name)
		kk = append(kk, name)
	}
	return kk
}

func synchronize(reportOnly bool) {
	config, err := FromFile("/home/slzatz/listmango/config.json")
	if err != nil {
		sess.showOrgMessage("Error reading postgres config file: %v", err)
		return
	}

	connect := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Postgres.Host,
		config.Postgres.Port,
		config.Postgres.User,
		config.Postgres.Password,
		config.Postgres.DB,
	)

	pdb, err := sql.Open("postgres", connect)
	if err != nil {
		sess.showOrgMessage("Problem opening postgres db: %w", err)
		return
	}

	// Ping to connection
	err = pdb.Ping()
	if err != nil {
		sess.showOrgMessage("postgres ping failure!: %v", err)
		return
	}

	nn := 0
	var lg strings.Builder
	lg.WriteString("****************************** BEGIN SYNC *******************************************\n\n")

	row := db.QueryRow("SELECT timestamp FROM sync WHERE machine=$1;", "client")
	var raw_client_t string
	err = row.Scan(&raw_client_t)
	if err != nil {
		sess.showOrgMessage("Error retrieving last client sync: %w", err)
		return
	}
	//last_client_sync, _ := time.Parse("2006-01-02T15:04:05Z", client_t)
	// note postscript doesn't seem to require the below and seems to be really doing a date comparison
	//client_t = client_t[0:10] + " " + client_t[11:16]
	client_t := raw_client_t[0:10] + " " + raw_client_t[11:19]

	var server_t string
	row = db.QueryRow("SELECT timestamp FROM sync WHERE machine=$1;", "server")
	err = row.Scan(&server_t)
	if err != nil {
		sess.showOrgMessage("Error retrieving last server sync: %w", err)
		return
	}
	//last_server_sync, _ := time.Parse("2006-01-02T15:04:05Z", server_t)
	//sess.showOrgMessage("last_client_sync = %v; last_server_sync = %v\n", last_client_sync, last_server_sync)

	fmt.Fprintf(&lg, "Local time is %v\n", time.Now())
	fmt.Fprintf(&lg, "UTC time is %v\n", time.Now().UTC())
	fmt.Fprintf(&lg, "Server last sync: %v\n", server_t)
	fmt.Fprintf(&lg, "(raw) Client last sync: %v\n", raw_client_t)
	fmt.Fprintf(&lg, "Client last sync: %v\n", client_t)
	//sess.showEdMessage("local time = %v; UTC time = %v; since last sync = %v", time.Now(), time.Now().UTC(), timeDelta(client_t))

	//server updated contexts
	rows, err := pdb.Query("SELECT id, title, \"default\", created, modified FROM context WHERE context.modified > $1 AND context.deleted = $2;", server_t, false)

	defer rows.Close()

	var server_updated_contexts []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.title,
			&c.star,
			&c.created,
			&c.modified,
		)
		server_updated_contexts = append(server_updated_contexts, c)
	}
	if len(server_updated_contexts) > 0 {
		nn += len(server_updated_contexts)
		fmt.Fprintf(&lg, "Updated (new and modified) server Contexts since last sync: %d\n", len(server_updated_contexts))
	} else {
		lg.WriteString("No updated (new and modified) server Contexts the last sync.\n")
	}

	//server deleted contexts
	//rows, err = pdb.Query("SELECT id, title, \"default\", created, modified FROM context WHERE context.modified > $1 AND context.deleted = $2;", server_t, true)
	rows, err = pdb.Query("SELECT id, title FROM context WHERE context.modified > $1 AND context.deleted = $2;", server_t, true)
	var server_deleted_contexts []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.title,
		//	&c.star,
		//	&c.created,
		//	&c.modified,
		)
		server_deleted_contexts = append(server_deleted_contexts, c)
	}
	if len(server_deleted_contexts) > 0 {
		nn += len(server_deleted_contexts)
		fmt.Fprintf(&lg, "Deleted server Contexts since last sync: %d\n", len(server_deleted_contexts))
	} else {
		lg.WriteString("No server Contexts deleted since last sync.\n")
	}

	//server updated folders
	rows, err = pdb.Query("SELECT id, title, private, created, modified FROM folder WHERE folder.modified > $1 AND folder.deleted = $2;", server_t, false)

	//defer rows.Close()

	var server_updated_folders []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.title,
			&c.star,
			&c.created,
			&c.modified,
		)
		server_updated_folders = append(server_updated_folders, c)
	}
	if len(server_updated_folders) > 0 {
		nn += len(server_updated_contexts)
		fmt.Fprintf(&lg, "Updated (new and modified) server Folders since last sync: %d\n", len(server_updated_folders))
	} else {
		lg.WriteString("No updated (new and modified) server Folders the last sync.\n")
	}

	//server deleted folders
	//rows, err = pdb.Query("SELECT id, title, private, created, modified FROM folder WHERE folder.modified > $1 AND folder.deleted = $2;", server_t, true)
	rows, err = pdb.Query("SELECT id, title FROM folder WHERE folder.modified > $1 AND folder.deleted = $2;", server_t, true)
	var server_deleted_folders []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.title,
			//&c.star,
			//&c.created,
			//&c.modified,
		)
		server_deleted_folders = append(server_deleted_folders, c)
	}
	if len(server_deleted_folders) > 0 {
		nn += len(server_deleted_folders)
		fmt.Fprintf(&lg, "Deleted server Folders since last sync: %d\n", len(server_updated_folders))
	} else {
		lg.WriteString("No server Folders deleted since last sync.\n")
	}

	//server updated keywords
	rows, err = pdb.Query("SELECT id, name, star, modified FROM keyword WHERE keyword.modified > $1 AND keyword.deleted = $2;", server_t, false)

	//defer rows.Close()

	var server_updated_keywords []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.title,
			&c.star,
			&c.modified,
		)
		server_updated_keywords = append(server_updated_keywords, c)
	}
	if len(server_updated_keywords) > 0 {
		nn += len(server_updated_contexts)
		fmt.Fprintf(&lg, "Updated (new and modified) server Keywords since last sync: %d\n", len(server_updated_keywords))
	} else {
		lg.WriteString("No updated (new and modified) server Keywords the last sync.\n")
	}

	//server deleted keywords
	//rows, err = pdb.Query("SELECT id, name, star, modified FROM keyword WHERE keyword.modified > $1 AND keyword.deleted = $2;", server_t, true)
	rows, err = pdb.Query("SELECT id, name FROM keyword WHERE keyword.modified > $1 AND keyword.deleted = $2;", server_t, true)
	var server_deleted_keywords []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.title,
			//&c.star,
			//&c.modified,
		)
		server_deleted_keywords = append(server_deleted_keywords, c)
	}
	if len(server_deleted_keywords) > 0 {
		nn += len(server_deleted_keywords)
		fmt.Fprintf(&lg, "Deleted server Keywords since last sync: %d\n", len(server_updated_keywords))
	} else {
		lg.WriteString("No server Keywords deleted since last sync.\n")
	}
	//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//server updated entries
	rows, err = pdb.Query("SELECT id, title, star, note, created, modified, added, completed, context_tid, folder_tid FROM task WHERE modified > $1 AND deleted = $2;", server_t, false)
	var server_updated_entries []Entry
	for rows.Next() {
		var e Entry
		rows.Scan(
			&e.id,
			&e.title,
			&e.star,
			&e.note,
			&e.created,
			&e.modified,
			&e.added,
			&e.completed,
			&e.context_tid,
			&e.folder_tid,
		)
		server_updated_entries = append(server_updated_entries, e)
	}
	if len(server_updated_entries) > 0 {
		nn += len(server_updated_entries)
		fmt.Fprintf(&lg, "Updated (new and modified) server Entries since last sync: %d\n", len(server_updated_entries))
	} else {
		lg.WriteString("No updated (new and modified) server Entries since last sync.\n")
	}
	//sess.showEdMessage("Number of changes that server needs to transmit to client: %v", len(server_updated_entries))
	for _, e := range server_updated_entries {
		fmt.Fprintf(&lg, "id: %d %q; star: %t; modified: %v\n", e.id, truncate(e.title, 15), e.star, tc(e.modified, 18, false))
	}

	//server deleted entries
	//rows, err = pdb.Query("SELECT id,title,star,created,modified FROM task WHERE task.modified > $1 AND task.deleted = $2;", server_t, true)
	rows, err = pdb.Query("SELECT id, title FROM task WHERE modified > $1 AND deleted = $2;", server_t, true)
	var server_deleted_entries []Entry
	for rows.Next() {
		var e Entry
		rows.Scan(
			&e.id,
			&e.title,
			//&e.star,
			//&e.created,
			//&e.modified,
		)
		server_deleted_entries = append(server_deleted_entries, e)
	}
	if len(server_deleted_entries) > 0 {
		nn += len(server_deleted_entries)
		fmt.Fprintf(&lg, "Deleted server Entries since last sync: %d\n", len(server_deleted_entries))
	} else {
		lg.WriteString("No server Entries deleted since last sync.\n")
	}

	//Client changes

	//client updated contexts
	rows, err = db.Query("SELECT id, tid, title, \"default\", created, modified FROM context WHERE substr(context.modified, 1, 19) > $1 AND context.deleted = $2;", client_t, false)

	//defer rows.Close()

	var client_updated_contexts []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.tid,
			&c.title,
			&c.star,
			&c.created,
			&c.modified,
		)
		client_updated_contexts = append(client_updated_contexts, c)
	}
	if len(client_updated_contexts) > 0 {
		nn += len(client_updated_contexts)
		fmt.Fprintf(&lg, "Updated (new and modified) client Contexts since last sync: %d\n", len(client_updated_contexts))
	} else {
		lg.WriteString("No updated (new and modified) client Contexts the last sync.\n")
	}

	//client deleted contexts
	//rows, err = db.Query("SELECT id, title, \"default\", created, modified FROM context WHERE context.modified > $1 AND context.deleted = $2;", client_t, true)
	rows, err = db.Query("SELECT id, tid, title FROM context WHERE substr(context.modified, 1, 19) > $1 AND context.deleted = $2;", client_t, true)
	var client_deleted_contexts []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.tid,
			&c.title,
			//&c.star,
			//&c.created,
			//&c.modified,
		)
		client_deleted_contexts = append(client_deleted_contexts, c)
	}
	if len(client_deleted_contexts) > 0 {
		nn += len(client_deleted_contexts)
		fmt.Fprintf(&lg, "Deleted client Contexts since last sync: %d\n", len(client_deleted_contexts))
	} else {
		lg.WriteString("No client Contexts deleted since last sync.\n")
	}

	//client updated folders
	rows, err = db.Query("SELECT id, tid, title, private, created, modified FROM folder WHERE substr(folder.modified, 1, 19) > $1 AND folder.deleted = $2;", client_t, false)

	defer rows.Close()

	var client_updated_folders []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.tid,
			&c.title,
			&c.star,
			&c.created,
			&c.modified,
		)
		client_updated_folders = append(client_updated_folders, c)
	}
	if len(client_updated_folders) > 0 {
		nn += len(client_updated_folders)
		fmt.Fprintf(&lg, "Updated (new and modified) client Folders since last sync: %d\n", len(client_updated_folders))
	} else {
		lg.WriteString("No updated (new and modified) client Folders the last sync.\n")
	}

	//client deleted folders
	//rows, err = db.Query("SELECT id, tid, title, private, created, modified FROM folder WHERE folder.modified > $1 AND folder.deleted = $2;", client_t, true)
	rows, err = db.Query("SELECT id, tid, title FROM folder WHERE substr(folder.modified, 1, 19) > $1 AND folder.deleted = $2;", client_t, true)
	var client_deleted_folders []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.tid,
			&c.title,
			//&c.star,
			//&c.created,
			//&c.modified,
		)
		client_deleted_folders = append(client_deleted_folders, c)
	}
	if len(client_deleted_folders) > 0 {
		nn += len(client_deleted_folders)
		fmt.Fprintf(&lg, "Deleted client Folders since last sync: %d\n", len(client_updated_folders))
	} else {
		lg.WriteString("No client Folders deleted since last sync.\n")
	}

	//client updated keywords
	rows, err = db.Query("SELECT id, tid, name, star, modified FROM keyword WHERE substr(keyword.modified, 1, 19)  > $1 AND keyword.deleted = $2;", client_t, false)

	defer rows.Close()

	var client_updated_keywords []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.tid,
			&c.title,
			&c.star,
			&c.modified,
		)
		client_updated_keywords = append(client_updated_keywords, c)
	}
	if len(client_updated_keywords) > 0 {
		nn += len(client_updated_keywords)
		fmt.Fprintf(&lg, "Updated (new and modified) client Keywords since last sync: %d\n", len(client_updated_keywords))
	} else {
		lg.WriteString("No updated (new and modified) client Keywords the last sync.\n")
	}

	//client deleted keywords
	//rows, err = db.Query("SELECT id, tid, name, star, modified FROM keyword WHERE keyword.modified > $1 AND keyword.deleted = $2;", client_t, true)
	rows, err = db.Query("SELECT id, tid, name FROM keyword WHERE substr(keyword.modified, 1, 19) > $1 AND keyword.deleted = $2;", client_t, true)
	var client_deleted_keywords []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.tid,
			&c.title,
			//&c.star,
			//&c.modified,
		)
		client_deleted_keywords = append(client_deleted_keywords, c)
	}
	if len(client_deleted_keywords) > 0 {
		nn += len(client_deleted_keywords)
		fmt.Fprintf(&lg, "Deleted client Keywords since last sync: %d\n", len(client_deleted_keywords))
	} else {
		lg.WriteString("No client Keywords deleted since last sync.\n")
	}

	//client updated entries
	//rows, err = db.Query("SELECT id, tid, title, star, created, modified, added, completed, context_tid, folder_tid FROM task WHERE task.modified > ? AND task.deleted = ?;", client_t, false)
	//rows, err = db.Query("SELECT id, tid, title, star, created, modified, added, completed, context_tid, folder_tid FROM task WHERE task.modified > ? AND task.deleted = ?;", client_t, "false")
	rows, err = db.Query("SELECT id, tid, title, star, note, created, modified, added, completed, context_tid, folder_tid FROM task WHERE substr(modified, 1, 19)  > ? AND deleted = ?;", client_t, false)
	var client_updated_entries []Entry
	for rows.Next() {
		var e Entry
		var tid sql.NullInt64
		rows.Scan(
			&e.id,
			&tid,
			&e.title,
			&e.star,
			&e.note,
			&e.created,
			&e.modified,
			&e.added,
			&e.completed,
			&e.context_tid,
			&e.folder_tid,
		)
		if tid.Valid {
			e.tid = int(tid.Int64)
		} else {
			e.tid = 0
		}

		client_updated_entries = append(client_updated_entries, e)
	}
	if len(client_updated_entries) > 0 {
		nn += len(client_updated_entries)
		fmt.Fprintf(&lg, "Updated (new and modified) client Entries since last sync: %d\n", len(client_updated_entries))
	} else {
		lg.WriteString("No updated (new and modified) client Entries since last sync.\n")
	}
	//sess.showEdMessage("Number of changes that client needs to transmit to client: %v", len(client_updated_entries))
	for _, e := range client_updated_entries {
		fmt.Fprintf(&lg, "id: %d; tid: %d %q; star: %t; modified: %v\n", e.id, e.tid, tc(e.title, 15, true), e.star, tc(e.modified, 18, false))
	}

	//client deleted entries
	rows, err = db.Query("SELECT id, tid, title FROM task WHERE substr(modified, 1, 19) > $1 AND deleted = $2;", client_t, true) //not sure need task.modified??
	if err != nil {
		sess.showOrgMessage("Problem with retrieving client deleted entries: %v", err)
		return
	}
	var client_deleted_entries []Entry
	for rows.Next() {
		var e Entry
		var tid sql.NullInt64
		rows.Scan(
			&e.id,
			&tid,
			&e.title,
		)
		if tid.Valid {
			e.tid = int(tid.Int64)
		} else {
			e.tid = 0
		}
		client_deleted_entries = append(client_deleted_entries, e)
	}
	if len(client_deleted_entries) > 0 {
		nn += len(client_deleted_entries)
		fmt.Fprintf(&lg, "Deleted client Entries since last sync: %d\n", len(client_deleted_entries))
	} else {
		lg.WriteString("No client Entries deleted since last sync.\n")
	}

	fmt.Fprintf(&lg, "Number of changes (before accounting for server/client conflicts) is: %d\n\n", nn)
	if reportOnly {
		sess.drawPreviewText2(lg.String())
		sess.drawPreviewBox()
		return
	}

	/****************below is where changes start***********************************/

	//updated server contexts -> client

	for _, c := range server_updated_contexts {
		row := db.QueryRow("SELECT id from context WHERE tid=?", c.id)
		var id int
		err = row.Scan(&id)
		switch {
		case err == sql.ErrNoRows:
			res, err1 := db.Exec("INSERT INTO context (tid, title, \"default\", created, modified, deleted) VALUES (?,?,?,?, datetime('now'), false);",
				c.id, c.title, c.star, c.created)
			if err1 != nil {
				fmt.Fprintf(&lg, "Problem inserting new context into sqlite: %w", err1)
				break
			}
			lastId, _ := res.LastInsertId()
			fmt.Fprintf(&lg, "Created new local context: %v with local id: %v and tid: %v\n", c.title, lastId, c.id)
		case err != nil:
			fmt.Fprintf(&lg, "Problem querying sqlite for a context with tid: %v: %w\n", c.id, err)
		default:
			_, err2 := db.Exec("UPDATE context SET title=?, \"default\"=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Problem updating sqlite for a context with tid: %v: %w\n", c.id, err2)
			} else {
				fmt.Fprintf(&lg, "Updated local context: %v with tid: %v\n", c.title, c.id)
			}
		}
	}

	for _, c := range client_updated_contexts {
		/*
			// server wins
			if server_id, found := server_updated_contexts_ids[c.tid]; found {
				fmt.Fprintf(&lg, "Server won updating server id/client tid: %v", server_id)
				continue
			}
		*/

		row := pdb.QueryRow("SELECT id from context WHERE id=$1", c.tid)
		var tid int
		err = row.Scan(&tid)
		switch {
		// server context doesn't exist
		case err == sql.ErrNoRows:
			err1 := pdb.QueryRow("INSERT INTO context (title, \"default\", created, modified, deleted) VALUES ($1, $2, $3, now(), false) RETURNING id;",
				c.title, c.star, c.created).Scan(&tid)
			if err1 != nil {
				fmt.Fprintf(&lg, "Error inserting new context %q with id %d into postgres: %v", truncate(c.title, 15), c.id, err1)
				break
			}
			_, err2 := db.Exec("UPDATE context SET tid=$1 WHERE id=$2;", tid, c.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Error setting tid for new client context %q with id %d to %d: %v\n", truncate(c.title, 15), c.id, tid, err2)
				break
			}
			fmt.Fprintf(&lg, "Set value of tid for client context with id: %v to tid = %v\n", c.id, tid)
		case err != nil:
			fmt.Fprintf(&lg, "Error querying postgres for a context with id: %v: %v\n", c.tid, err)
		default:
			_, err3 := pdb.Exec("UPDATE context SET title=$1, \"default\"=$2, modified=now() WHERE id=$3;", c.title, c.star, c.tid)
			if err3 != nil {
				fmt.Fprintf(&lg, "Error updating postgres for context %q with id %d: %v\n", truncate(c.title, 15), c.tid, err3)
			} else {
				fmt.Fprintf(&lg, "Updated server/postgres context: %v with id: %v\n", c.title, c.tid)
			}
		}
	}

	for _, c := range server_updated_folders {
		row := db.QueryRow("SELECT id from folder WHERE tid=?", c.id)
		var id int
		err = row.Scan(&id)
		switch {
		case err == sql.ErrNoRows:
			res, err1 := db.Exec("INSERT INTO folder (tid, title, private, created, modified, deleted) VALUES (?,?,?,?, datetime('now'), false);",
				c.id, c.title, c.star, c.created)
			if err1 != nil {
				fmt.Fprintf(&lg, "Problem inserting new folder into sqlite: %w", err1)
				break
			}
			lastId, _ := res.LastInsertId()
			fmt.Fprintf(&lg, "Created new local folder: %v with local id: %v and tid: %v\n", c.title, lastId, c.id)
		case err != nil:
			fmt.Fprintf(&lg, "Problem querying sqlite for a folder with tid: %v: %w\n", c.id, err)
		default:
			_, err2 := db.Exec("UPDATE folder SET title=?, private=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Problem updating sqlite for a folder with tid: %v: %w\n", c.id, err2)
			} else {
				fmt.Fprintf(&lg, "Updated local folder: %v with tid: %v\n", c.title, c.id)
			}
		}
	}

	for _, c := range client_updated_folders {
		/*
			// server wins
			if server_id, found := server_updated_contexts_ids[c.tid]; found {
				fmt.Fprintf(&lg, "Server won updating server id/client tid: %v", server_id)
				continue
			}
		*/

		row := pdb.QueryRow("SELECT id from folder WHERE id=$1", c.tid)
		var tid int
		err = row.Scan(&tid)
		switch {
		// server folder doesn't exist
		case err == sql.ErrNoRows:
			err1 := pdb.QueryRow("INSERT INTO folder (title, private, created, modified, deleted) VALUES ($1, $2, $3, now(), false) RETURNING id;",
				c.title, c.star, c.created).Scan(&tid)
			if err1 != nil {
				fmt.Fprintf(&lg, "Problem inserting new folder %d: %s into postgres: %v", c.id, c.title, err1)
				break
			}
			_, err2 := db.Exec("UPDATE folder SET tid=$1 WHERE id=$2;", tid, c.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Error setting tid to %d for new client folder %q with id %d: %v\n", tid, truncate(c.title, 15), c.id, err2)
				break
			}
			fmt.Fprintf(&lg, "Set value of tid for client folder with id: %v to tid = %v\n", c.id, tid)
		case err != nil:
			fmt.Fprintf(&lg, "Error querying postgres for a folder with id: %v: %v\n", c.tid, err)
		default:
			_, err3 := pdb.Exec("UPDATE folder SET title=$1, private=$2, modified=now() WHERE id=$3;", c.title, c.star, c.tid)
			if err3 != nil {
				fmt.Fprintf(&lg, "Error updating postgres for folder %q with id %d: %v\n", truncate(c.title, 15), c.tid, err3)
			} else {
				fmt.Fprintf(&lg, "Updated server/postgres folder %q with id %d\n", truncate(c.title, 15), c.tid)
			}
		}
	}

	for _, c := range server_updated_keywords {
		row := db.QueryRow("SELECT id from keyword WHERE tid=?", c.id)
		var id int
		err = row.Scan(&id)
		switch {
		case err == sql.ErrNoRows:
			res, err1 := db.Exec("INSERT INTO keyword (tid, name, star, modified, deleted) VALUES (?,?,?, datetime('now'), false);",
				c.id, c.title, c.star)
			if err1 != nil {
				fmt.Fprintf(&lg, "Error inserting new keyword %q into sqlite: %v", truncate(c.title, 15), err1)
				break
			}
			lastId, _ := res.LastInsertId()
			fmt.Fprintf(&lg, "Created new local keyword %q with local id %d and tid: %d\n", truncate(c.title, 15), lastId, c.id)
		case err != nil:
			fmt.Fprintf(&lg, "Error querying sqlite for a keyword with tid: %v: %w\n", c.id, err)
		default:
			_, err2 := db.Exec("UPDATE keyword SET name=?, star=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Error updating local keyword %q with tid %d: %v\n", truncate(c.title, 15), c.id, err2)
			} else {
				fmt.Fprintf(&lg, "Updated local keyword %q with tid %d\n", truncate(c.title, 15), c.id)
			}
		}
	}

	for _, c := range client_updated_keywords {
		/*
			// server wins
			if server_id, found := server_updated_contexts_ids[c.tid]; found {
				fmt.Fprintf(&lg, "Server won updating server id/client tid: %v", server_id)
				continue
			}
		*/

		row := pdb.QueryRow("SELECT id from keyword WHERE id=$1", c.tid)
		var tid int
		err = row.Scan(&tid)
		switch {
		// server keyword doesn't exist
		case err == sql.ErrNoRows:
			err1 := pdb.QueryRow("INSERT INTO keyword (name, star, modified, deleted) VALUES ($1, $2, now(), false) RETURNING id;",
				c.title, c.star).Scan(&tid)
			if err1 != nil {
				fmt.Fprintf(&lg, "Problem inserting new keyword: %d: %s into postgres: %v", c.id, c.title, err1)
				break
			}
			_, err2 := db.Exec("UPDATE keyword SET tid=$1 WHERE id=$2;", tid, c.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Error setting new client keyword's tid: %v; id: %v\n", tid, c.id, err2)
				break
			}
			fmt.Fprintf(&lg, "Set value of tid for client keyword with id: %v to tid = %v\n", c.id, tid)
		case err != nil:
			fmt.Fprintf(&lg, "Error querying postgres for a keyword with id: %v: %v\n", c.tid, err)
		default:
			_, err3 := pdb.Exec("UPDATE keyword SET name=$1, star=$2, modified=now() WHERE id=$3;", c.title, c.star, c.tid)
			if err3 != nil {
				fmt.Fprintf(&lg, "Error updating postgres for a keyword with id: %v: %w\n", c.tid, err3)
			} else {
				fmt.Fprintf(&lg, "Updated server/postgres keyword: %v with id: %v\n", c.title, c.tid)
			}
		}
	}

	/**********should come before container deletes to change tasks here*****************/
	server_updated_entries_ids := make(map[int]struct{})
	for _, e := range server_updated_entries {
		// below is for server always wins
		server_updated_entries_ids[e.id] = struct{}{}
		row := db.QueryRow("SELECT id from task WHERE tid=?", e.id)
		var client_id int
		err := row.Scan(&client_id)
		switch {
		case err == sql.ErrNoRows:
			res, err1 := db.Exec("INSERT INTO task (tid, title, star, created, added, completed, context_tid, folder_tid, note, modified, deleted) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), false);",
				e.id, e.title, e.star, e.created, e.added, e.completed, e.context_tid, e.folder_tid, e.note)
			if err1 != nil {
				fmt.Fprintf(&lg, "Error inserting new entry for %q into sqlite: %v\n", truncate(e.title, 15), err1)
				break
			}
			id, _ := res.LastInsertId()
			client_id = int(id)
			_, err2 := fts_db.Exec("INSERT INTO fts (title, note, lm_id) VALUES (?, ?, ?);", e.title, e.note, client_id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Error inserting into fts_db for entry with id %d: %v\n", client_id, err2)
				break
			}
			fmt.Fprintf(&lg, "Created new local entry %q with id %d and tid %d\n", truncate(e.title, 15), client_id, e.id)
		case err != nil:
			fmt.Fprintf(&lg, "Error querying sqlite for a entry with tid: %v: %w\n", e.id, err)
			continue
		default:
			_, err3 := db.Exec("UPDATE task SET title=?, star=?, context_tid=?, folder_tid=?, note=?, completed=?,  modified=datetime('now') WHERE tid=?;",
				e.title, e.star, e.context_tid, e.folder_tid, e.note, e.completed, e.id)
			if err3 != nil {
				fmt.Fprintf(&lg, "Error updating sqlite for an entry with tid: %d: %v\n", e.id, err3)
				continue
			}
			_, err4 := fts_db.Exec("UPDATE fts SET title=?, note=? WHERE lm_id=?;", e.title, e.note, client_id)
			if err4 != nil {
				fmt.Fprintf(&lg, "Error updating fts_db for entry %s; id: %d; %v\n", truncate(e.title, 15), client_id, err4)
			} else {
				fmt.Fprintf(&lg, "fts_db updated for entry %q with id %d and tid %d\n", truncate(e.title, 15), client_id, e.id)
			}
			fmt.Fprintf(&lg, "Updated local entry: %s with id %d and tid: %d\n", truncate(e.title, 15), client_id, e.id)
		}
		// Update the client entry's keywords
		_, err5 := db.Exec("DELETE FROM task_keyword WHERE task_id=?;", client_id)
		if err5 != nil {
			fmt.Fprintf(&lg, "Error deleting from task_keyword from server id %d: %v\n", client_id, err5)
			continue
		}
		kwns := getTaskKeywordsS(pdb, &lg, e.id) // returns []string
		for _, kwn := range kwns {
			keyword_id := keywordExistsS(db, &lg, kwn)
			if keyword_id != -1 { // ? should create the keyword if it doesn't exist
				addTaskKeywordS(db, &lg, keyword_id, client_id)
			}
		}
		tag := strings.Join(kwns, ",")
		_, err = fts_db.Exec("UPDATE fts SET tag=$1 WHERE lm_id=$2;", tag, client_id)
		if err != nil {
			fmt.Fprintf(&lg, "Error in Update tag in fts: %v\n", err)
		} else {
			fmt.Fprintf(&lg, "Keywords %q updated in fts_db for client id %d\n", tag, client_id)
		}
	}

	for _, e := range client_updated_entries {
		// server wins if both client and server have updated an item
		if _, found := server_updated_entries_ids[e.tid]; found {
			fmt.Fprintf(&lg, "Server won: client entry %q with id %d and tid %d was updated by server\n", truncate(e.title, 15), e.id, e.tid)
			continue
		}

		var exists bool
		var server_id int
		err := pdb.QueryRow("SELECT EXISTS(SELECT 1 FROM task WHERE id=$1);", e.tid).Scan(&exists)
		switch {

		case err != nil:
			fmt.Fprintf(&lg, "Problem checking if postgres has an entry for %q with client tid/pg id: %d: %v\n", truncate(e.title, 15), e.tid, err)

		case exists:
			_, err3 := pdb.Exec("UPDATE task SET title=$1, star=$2, context_tid=$3, folder_tid=$4, note=$5, completed=$6, modified=now() WHERE id=$7;",
				e.title, e.star, e.context_tid, e.folder_tid, e.note, e.completed, e.tid)
			if err3 != nil {
				fmt.Fprintf(&lg, "Error updating server entry %q with id %d: %v", truncate(e.title, 15), e.tid, err3)
			} else {
				fmt.Fprintf(&lg, "Updated server entry %q with id %d\n", truncate(e.title, 15), e.tid)
			}
			server_id = e.tid // needed below for keywords
		case !exists:
			err1 := pdb.QueryRow("INSERT INTO task (title, star, created, added, completed, context_tid, folder_tid, note, modified, deleted) "+
				"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), false) RETURNING id;",
				e.title, e.star, e.created, e.added, e.completed, e.context_tid, e.folder_tid, e.note).Scan(&server_id)
			if err1 != nil {
				fmt.Fprintf(&lg, "Error inserting new server entry for client entry %s with id %d into postgres: %v\n", truncate(e.title, 15), e.id, err1)
				break
			}
			_, err2 := db.Exec("UPDATE task SET tid=? WHERE id=?;", server_id, e.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Error setting tid for client entry %q with id %d to tid %d: %v\n", truncate(e.title, 15), e.id, server_id, err2)
				break
			}
			fmt.Fprintf(&lg, "Created new server entry %q with id %d\n", truncate(e.title, 15), server_id)
			fmt.Fprintf(&lg, "and set tid for client entry (id %d) to tid %d\n", e.id, server_id)

		default:
			fmt.Fprintf(&lg, "Error in SELECT EXISTS in client_updated_entries for client entry id: %d\n", e.id)
			continue
		}
		// Update the server entry's keywords
		_, err4 := pdb.Exec("DELETE FROM task_keyword WHERE task_id=$1;", server_id)
		if err4 != nil {
			fmt.Fprintf(&lg, "Error deleting from task_keyword from server id %d: %v\n", server_id, err4)
			continue
		}
		kwns := getTaskKeywordsS(db, &lg, e.id) // returns []string
		for _, kwn := range kwns {
			keyword_id := keywordExistsS(pdb, &lg, kwn)
			if keyword_id != -1 { // ? should create the keyword if it doesn't exist
				addTaskKeywordS(pdb, &lg, keyword_id, server_id)
			}
		}
	}

	// server deleted entries
	for _, e := range server_deleted_entries {
		_, err := db.Exec("DELETE FROM task WHERE tid=?;", e.id)
		if err != nil {
			fmt.Fprintf(&lg, "Error deleting client entry %q with tid %d: %v\n", tc(e.title, 15, true), e.id, err)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client entry %q with tid %d\n", truncate(e.title, 15), e.id)
	}

	// client deleted entries
	for _, e := range client_deleted_entries {
		// since on server, we just set deleted to true
		// since may have to sync with other clients
		if e.tid == 0 {
			fmt.Fprintf(&lg, "There is no server entry to delete for client id %d\n", e.id)
			continue
		}

		_, err := pdb.Exec("UPDATE task SET deleted=true, modified=now() WHERE id=$1", e.tid) /**************/
		if err != nil {
			fmt.Fprintf(&lg, "Error setting server entry with id %d to deleted: %v\n", e.tid, err)
			continue
		}
		fmt.Fprintf(&lg, "Updated server entry %q with id %d to deleted = true\n", truncate(e.title, 15), e.tid)
		_, err = db.Exec("DELETE FROM task WHERE id=?", e.id)
		if err != nil {
			fmt.Fprintf(&lg, "Error deleting client entry %q with id %d: %v\n", tc(e.title, 15, true), e.id, err)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client entry %q with id %d\n", tc(e.title, 15, true), e.id)
	}

	//server_deleted_contexts
	for _, c := range server_deleted_contexts {
		// I think the task changes may not be necessary because only a previous client sync can delete server context
		res, err := pdb.Exec("Update task SET context_tid=1, modified=now() WHERE context_tid=$1;", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to change server/postgres entry context to 'No Context' for a deleted context: %v\n", err)
		} else {
			rowsAffected, _ := res.RowsAffected()
			fmt.Fprintf(&lg, "The number of server entries that were changed to 'No Context' (might be zero): %d\n", rowsAffected)
		}
		res, err = db.Exec("Update task SET context_tid=1, modified=datetime('now') WHERE context_tid=?;", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to change client/sqlite entry contexts for a deleted context: %v\n", err)
		} else {
			rowsAffected, _ := res.RowsAffected()
			fmt.Fprintf(&lg, "The number of client entries that were changed to 'No Context' (might be zero): %d\n", rowsAffected)
		}

		_, err = db.Exec("DELETE FROM context WHERE tid=?", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Problem deleting local context with tid = %v", c.id)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client context %s with tid %d", c.title, c.id)
	}

	// client deleted contexts
	for _, c := range client_deleted_contexts {
		res, err := pdb.Exec("Update task SET context_tid=1, modified=now() WHERE context_tid=$1;", c.tid) //?modified=now()
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to change server/postgres entry contexts for a deleted context: %v\n", err)
		} else {
			rowsAffected, _ := res.RowsAffected()
			fmt.Fprintf(&lg, "The number of server entries that were changed to No Context: %d\n", rowsAffected)
		}
		res, err = db.Exec("Update task SET context_tid=1, modified=now() WHERE context_tid=?;", c.tid)
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to change client/sqlite entry contexts for a deleted context: %v\n", err)
		} else {
			rowsAffected, _ := res.RowsAffected()
			fmt.Fprintf(&lg, "The number of client entries that were changed to No Context: %d\n", rowsAffected)
		}
		// since on server, we just set deleted to true
		// since may have to sync with other clients
		_, err = pdb.Exec("UPDATE context SET deleted=true, modified=now() WHERE context.id=$1", c.tid)
		if err != nil {
			fmt.Fprintf(&lg, "Error (pdb.Exec) setting server context %s with id = %d to deleted: %v\n", c.title, c.tid, err)
			continue
		}
		_, err = db.Exec("DELETE FROM folder WHERE id=?", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Error deleting local context %s with id %d: %v", c.title, c.id, err)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client context %s: id %d and updated server context with id %d to deleted = true", c.title, c.id, c.tid)
	}

	//server_deleted_folders
	for _, c := range server_deleted_folders {
		// I think the task changes may not be necessary because only a previous client sync can delete server context
		// and that previous client sync should have changed the relevant tasks to 'No Folder'
		res, err := pdb.Exec("Update task SET folder_tid=1, modified=now() WHERE folder_tid=$1;", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to change server/postgres entry folder to 'No Folder' for a deleted folder: %v\n", err)
		} else {
			rowsAffected, _ := res.RowsAffected()
			fmt.Fprintf(&lg, "The number of server entries that were changed to 'No Folder' (might be zero): %d\n", rowsAffected)
		}
		res, err = db.Exec("Update task SET folder_tid=1, modified=datetime('now') WHERE folder_tid=?;", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to change client/sqlite entry folders for a deleted folder: %v\n", err)
		} else {
			rowsAffected, _ := res.RowsAffected()
			fmt.Fprintf(&lg, "The number of client entries that were changed to 'No Folder' (might be zero): %d\n", rowsAffected)
		}

		_, err = db.Exec("DELETE FROM folder WHERE tid=?", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Problem deleting local folder with tid = %v", c.id)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client folder %v with tid %v", c.title, c.id)
	}

	// client deleted folders
	for _, c := range client_deleted_folders {
		res, err := pdb.Exec("Update task SET folder_tid=1, modified=now() WHERE folder_tid=$1;", c.tid) //?modified=now()
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to change server/postgres entry folders for a deleted folder: %v\n", err)
			continue
		} else {
			rowsAffected, _ := res.RowsAffected()
			fmt.Fprintf(&lg, "The number of server entries that were changed to No Folder: %d\n", rowsAffected)
		}
		res, err = db.Exec("Update task SET folder_tid=1, modified=now() WHERE folder_tid=?;", c.tid)
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to change client/sqlite entry folders for a deleted folder: %v\n", err)
		} else {
			rowsAffected, _ := res.RowsAffected()
			fmt.Fprintf(&lg, "The number of client entries that were changed to No Folder: %d\n", rowsAffected)
		}
		// since on server, we just set deleted to true
		// since may have to sync with other clients
		_, err = pdb.Exec("UPDATE folder SET deleted=true, modified=now() WHERE folder.id=$1", c.tid)
		if err != nil {
			fmt.Fprintf(&lg, "Error (pdb.Exec) setting server folder %s with id = %v to deleted: %v\n", c.title, c.tid, err)
			continue
		}
		_, err = db.Exec("DELETE FROM folder WHERE id=?", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Error deleting local folder %s with id %d: %v", c.title, c.id, err)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client folder %s: id %d and updated server folder with id %d to deleted = true", c.title, c.id, c.tid)
	}

	//server_deleted_keywords
	for _, c := range server_deleted_keywords {
		pdb.Exec("DELETE FROM task_keyword WHERE keyword_id=$1;", c.id)
		row = db.QueryRow("SELECT id FROM keyword WHERE keyword.tid=?", c.id)
		var id int
		err = row.Scan(&id)
		if err != nil {
			sess.showOrgMessage("Error SELECTING id FROM client keyword with tid %d: %v", c.id, err)
			continue
		}
		db.Exec("DELETE FROM task_keyword WHERE keyword_id=?;", id)
		//res, err := db.Exec("DELETE FROM keyword WHERE tid=?", c.id)
		_, err := db.Exec("DELETE FROM keyword WHERE tid=?", id)
		if err != nil {
			fmt.Fprintf(&lg, "Error deleting local keyword with tid = %v", c.id)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client keyword %q with tid %d", truncate(c.title, 15), c.id)
	}

	// client deleted keywords
	for _, c := range client_deleted_keywords {
		_, err = pdb.Exec("DELETE FROM task_keyword WHERE keyword_id=$1;", c.tid)
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to delete server task_keyword with id %d", c.tid)
		}
		_, err = db.Exec("DELETE FROM task_keyword WHERE keyword_id=?;", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to delete client task_keyword with id %d", c.id)
		}
		// since on server, we just set deleted to true
		// since may have to sync with other clients
		_, err := pdb.Exec("UPDATE keyword SET deleted=true WHERE keyword.id=$1", c.tid)
		if err != nil {
			fmt.Fprintf(&lg, "Error (pdb.Exec) setting server keyword %s with id %d to deleted:%v\n", c.title, c.tid, err)
			continue
		}
		_, err = db.Exec("DELETE FROM keyword WHERE id=?", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Error deleting local keyword %s with id %d: %v\n", c.title, c.id, err)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client keyword %s: id %d and updated server keyword with id %d to deleted = true", c.title, c.id, c.tid)
	}
	/*********************end of sync changes*************************/

	var server_ts string
	row = pdb.QueryRow("SELECT now();")
	err = row.Scan(&server_ts)
	if err != nil {
		sess.showOrgMessage("Problem with getting current time from server: %w", err)
		return
	}
	_, err = db.Exec("UPDATE sync SET timestamp=$1 WHERE machine='server';", server_ts)
	if err != nil {
		sess.showOrgMessage("Problem updating client with server timestamp: %w", err)
		return
	}
	_, err = db.Exec("UPDATE sync SET timestamp=datetime('now') WHERE machine='client';")
	if err != nil {
		sess.showOrgMessage("Problem updating client with client timestamp: %w", err)
		return
	}
	var client_ts string
	row = db.QueryRow("SELECT datetime('now');")
	err = row.Scan(&client_ts)
	if err != nil {
		sess.showOrgMessage("Problem with getting current time from client: %w", err)
		return
	}
	fmt.Fprintf(&lg, "\nClient UTC timestamp: %s\n", client_ts)
	fmt.Fprintf(&lg, "Server UTC timestamp: %s", server_ts)

	log := lg.String()
	sess.drawPreviewText2(log)
	sess.drawPreviewBox()
	log_title := fmt.Sprintf("%v - %d change(s)", time.Now().Format("Mon Jan 2 15:04:05"), nn)
	insertSyncEntry(log_title, log)
}

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
*/
