package main

/** note that sqlite datetime('now') returns utc **/

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
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

func synchronize(reportOnly bool) {
	config, err := FromFile("/home/slzatz/listmango/config.json")
	if err != nil {
		sess.showOrgMessage("Problem reading postgres config file: %v", err)
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
		sess.showOrgMessage("postgres ping failure!: %w", err)
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
	rows, err = pdb.Query("SELECT id, title, star, created, modified, added, completed, context_tid, folder_tid, note FROM task WHERE modified > $1 AND deleted = $2;", server_t, false)
	var server_updated_entries []Entry
	for rows.Next() {
		var e Entry
		rows.Scan(
			&e.id,
			&e.title,
			&e.star,
			&e.created,
			&e.modified,
			&e.added,
			&e.completed,
			&e.context_tid,
			&e.folder_tid,
			&e.note,
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
		fmt.Fprintf(&lg, "id: %v; title: %v; star: %v, created: %v; modified; %v\n", e.id, e.title, e.star, e.created, e.modified)
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
	rows, err = db.Query("SELECT id, title, \"default\", created, modified FROM context WHERE substr(context.modified, 1, 19) > $1 AND context.deleted = $2;", client_t, false)

	//defer rows.Close()

	var client_updated_contexts []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
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
		fmt.Fprintf(&lg, "Deleted client Keywords since last sync: %d\n", len(client_updated_keywords))
	} else {
		lg.WriteString("No client Keywords deleted since last sync.\n")
	}

	//client updated entries
	//rows, err = db.Query("SELECT id, tid, title, star, created, modified, added, completed, context_tid, folder_tid FROM task WHERE task.modified > ? AND task.deleted = ?;", client_t, false)
	//rows, err = db.Query("SELECT id, tid, title, star, created, modified, added, completed, context_tid, folder_tid FROM task WHERE task.modified > ? AND task.deleted = ?;", client_t, "false")
	rows, err = db.Query("SELECT id, tid, title, star, created, modified, added, completed, context_tid, folder_tid FROM task WHERE substr(modified, 1, 19)  > ? AND deleted = ?;", client_t, false)
	var client_updated_entries []Entry
	for rows.Next() {
		var e Entry
		var tid sql.NullInt64
		rows.Scan(
			&e.id,
			&tid,
			&e.title,
			&e.star,
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
		fmt.Fprintf(&lg, "id: %v; tid: %v; title: %v; star: %v, created: %v; modified; %v\n", e.id, e.tid, e.title, e.star, e.created, e.modified)
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
		rows.Scan(
			&e.id,
			&e.tid,
			&e.title,
		)
		//fmt.Printf("%v\n", e)
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
		var id int
		err = row.Scan(&id)
		switch {
		// server context doesn't exist
		case err == sql.ErrNoRows:
			res, err1 := pdb.Exec("INSERT INTO context (title, \"default\", created, modified, deleted) VALUES ($1, $2, $3, now(), false);",
				c.title, c.star, c.created)
			if err1 != nil {
				fmt.Fprintf(&lg, "Problem inserting new context into postgres: %v", err1)
				break
			}
			lastId, err4 := res.LastInsertId()
			if err4 != nil {
				fmt.Fprintf(&lg, "Problem retrieving id/lastId from new server context to set client tid: %v\n", err4)
				break
			}
			fmt.Fprintf(&lg, "Created new server/postgres context: %v with id: %v\n", c.title, lastId)
			// need to update the new client context with the id/tid we got from the server
			res, err2 := db.Exec("UPDATE context SET context.tid=$1 WHERE context.id=$2;", lastId, c.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Problem setting new client context's tid: %v; id: %v\n", lastId, c.id, err2)
				break
			}
			fmt.Fprintf(&lg, "Set value of tid for client context with id: %v to tid = %v\n", c.id, lastId)
		case err != nil:
			fmt.Fprintf(&lg, "Problem querying postgres for a context with id: %v: %v\n", c.tid, err)
		default:
			_, err3 := pdb.Exec("UPDATE context SET title=$1, \"default\"=$2, modified=now() WHERE id=$3;", c.title, c.star, c.tid)
			if err3 != nil {
				fmt.Fprintf(&lg, "Problem updating postgres for a context with id: %v: %w\n", c.tid, err3)
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
		var id int
		err = row.Scan(&id)
		switch {
		// server folder doesn't exist
		case err == sql.ErrNoRows:
			res, err1 := pdb.Exec("INSERT INTO folder (title, private, created, modified, deleted) VALUES ($1, $2, $3, now(), false);",
				c.title, c.star, c.created)
			if err1 != nil {
				fmt.Fprintf(&lg, "Problem inserting new folder into postgres: %v", err1)
				break
			}
			lastId, err4 := res.LastInsertId()
			if err4 != nil {
				fmt.Fprintf(&lg, "Problem retrieving id/lastId from new server folder to set client tid: %v\n", err4)
				break
			}
			fmt.Fprintf(&lg, "Created new server/postgres folder: %v with id: %v\n", c.title, lastId)
			// need to update the new client folder with the id/tid we got from the server
			res, err2 := db.Exec("UPDATE folder SET folder.tid=$1 WHERE folder.id=$2;", lastId, c.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Problem setting new client folder's tid: %v; id: %v\n", lastId, c.id, err2)
				break
			}
			fmt.Fprintf(&lg, "Set value of tid for client folder with id: %v to tid = %v\n", c.id, lastId)
		case err != nil:
			fmt.Fprintf(&lg, "Problem querying postgres for a folder with id: %v: %v\n", c.tid, err)
		default:
			_, err3 := pdb.Exec("UPDATE folder SET title=$1, private=$2, modified=now() WHERE id=$3;", c.title, c.star, c.tid)
			if err3 != nil {
				fmt.Fprintf(&lg, "Problem updating postgres for a folder with id: %v: %w\n", c.tid, err3)
			} else {
				fmt.Fprintf(&lg, "Updated server/postgres folder: %v with id: %v\n", c.title, c.tid)
			}
		}
	}

	for _, c := range server_updated_keywords {
		row := db.QueryRow("SELECT id from keyword WHERE tid=?", c.id)
		var id int
		err = row.Scan(&id)
		switch {
		case err == sql.ErrNoRows:
			res, err1 := db.Exec("INSERT INTO keyword (tid, name, star, created, modified, deleted) VALUES (?,?,?,?, datetime('now'), false);",
				c.id, c.title, c.star, c.created)
			if err1 != nil {
				fmt.Fprintf(&lg, "Problem inserting new keyword into sqlite: %w", err1)
				break
			}
			lastId, _ := res.LastInsertId()
			fmt.Fprintf(&lg, "Created new local keyword: %v with local id: %v and tid: %v\n", c.title, lastId, c.id)
		case err != nil:
			fmt.Fprintf(&lg, "Problem querying sqlite for a keyword with tid: %v: %w\n", c.id, err)
		default:
			_, err2 := db.Exec("UPDATE keyword SET name=?, star=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Problem updating sqlite for a keyword with tid: %v: %w\n", c.id, err2)
			} else {
				fmt.Fprintf(&lg, "Updated local keyword: %v with tid: %v\n", c.title, c.id)
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
		var id int
		err = row.Scan(&id)
		switch {
		// server keyword doesn't exist
		case err == sql.ErrNoRows:
			res, err1 := pdb.Exec("INSERT INTO keyword (name, star, modified, deleted) VALUES ($1, $2, now(), false);",
				c.title, c.star)
			if err1 != nil {
				fmt.Fprintf(&lg, "Problem inserting new keyword into postgres: %v", err1)
				break
			}
			lastId, err4 := res.LastInsertId()
			if err4 != nil {
				fmt.Fprintf(&lg, "Problem retrieving id/lastId from new server keyword to set client tid: %v\n", err4)
				break
			}
			fmt.Fprintf(&lg, "Created new server/postgres keyword: %v with id: %v\n", c.title, lastId)
			// need to update the new client keyword with the id/tid we got from the server
			res, err2 := db.Exec("UPDATE keyword SET keyword.tid=$1 WHERE keyword.id=$2;", lastId, c.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Problem setting new client keyword's tid: %v; id: %v\n", lastId, c.id, err2)
				break
			}
			fmt.Fprintf(&lg, "Set value of tid for client keyword with id: %v to tid = %v\n", c.id, lastId)
		case err != nil:
			fmt.Fprintf(&lg, "Problem querying postgres for a keyword with id: %v: %v\n", c.tid, err)
		default:
			_, err3 := pdb.Exec("UPDATE keyword SET name=$1, star=$2, modified=now() WHERE id=$3;", c.title, c.star, c.tid)
			if err3 != nil {
				fmt.Fprintf(&lg, "Problem updating postgres for a keyword with id: %v: %w\n", c.tid, err3)
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
		var id int
		err = row.Scan(&id)
		switch {
		case err == sql.ErrNoRows:
			res, err1 := db.Exec("INSERT INTO task (tid, title, star, created, added, completed, context_tid, folder_tid, note, modified, deleted) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), false);",
				e.id, e.title, e.star, e.created, e.added, e.completed, e.context_tid, e.folder_tid, e.note)
			if err1 != nil {
				fmt.Fprintf(&lg, "Problem inserting new entry into sqlite: %w", err1)
				break
			}
			lastId, _ := res.LastInsertId()
			_, err2 := fts_db.Exec("INSERT INTO fts (title, note, lm_id) VALUES (?, ?, ?);", e.title, e.note, lastId)
			if err2 != nil {
				fmt.Fprintf(&lg, "Problem inserting into fts_db for entry with id: %v: %w\n", lastId, err2)
				break
			}
			fmt.Fprintf(&lg, "Created new local entry: %v with local id: %v and tid: %v\n", e.title, lastId, e.id)
		case err != nil:
			fmt.Fprintf(&lg, "Problem querying sqlite for a entry with tid: %v: %w\n", e.id, err)
		default:
			_, err3 := db.Exec("UPDATE task SET title=?, star=?, context_tid=?, folder_tid=?, note=?, modified=datetime('now') WHERE tid=?;",
				e.title, e.star, e.context_tid, e.folder_tid, e.note, e.id)
			if err3 != nil {
				fmt.Fprintf(&lg, "Problem updating sqlite for an entry with tid: %v: %w\n", e.id, err3)
			} else {
				row = db.QueryRow("SELECT id FROM task WHERE tid=?;", e.id)
				var lm_id int
				err4 := row.Scan(&lm_id)
				if err4 != nil {
					fmt.Fprintf(&lg, "Error trying to retrieve entry id to update fts_db: %w", err4)
				} else {
					_, err5 := fts_db.Exec("UPDATE fts SET title=?, note=? WHERE lm_id=?;", e.title, e.note, lm_id)
					if err5 != nil {
						fmt.Fprintf(&lg, "Problem updating fts_db for entry with id: %v: %w\n", lm_id, err5)
					} else {
						fmt.Fprintf(&lg, "fts_db updated for entry with id: %v\n", lm_id)
					}
				}
				fmt.Fprintf(&lg, "Updated local entry: %v with tid: %v\n", e.title, e.id)
			}
		}
	}

	for _, e := range client_updated_entries {
		// server wins if both client and server have updated an item
		if server_id, found := server_updated_entries_ids[e.tid]; found {
			fmt.Fprintf(&lg, "Server won updating server id/client tid: %v", server_id)
			continue
		}

		var exists bool
		err := pdb.QueryRow("SELECT EXISTS(SELECT 1 FROM task WHERE id=$1);", e.tid).Scan(&exists)
		switch {

		case err != nil:
			fmt.Fprintf(&lg, "Problem checking if postgres has an entry for '%s' with client tid/pg id: %d: %v\n", e.title[:15], e.tid, err)

		case exists:
			_, err3 := pdb.Exec("UPDATE task SET title=$1, star=$2, context_tid=$3, folder_tid=$4, note=$5, modified=now() WHERE id=$6;",
				e.title, e.star, e.context_tid, e.folder_tid, e.note, e.tid)
			if err3 != nil {
				fmt.Fprintf(&lg, "Problem updating server entry: %v with id: %v; %v", e.title, e.tid, err3)
			} else {
				fmt.Fprintf(&lg, "Updated server entry: %v with id: %v\n", e.title, e.tid)
			}

		case !exists:
			var id int
			err1 := pdb.QueryRow("INSERT INTO task (title, star, created, added, completed, context_tid, folder_tid, note, modified, deleted) "+
				"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), false) RETURNING id;",
				e.title, e.star, e.created, e.added, e.completed, e.context_tid, e.folder_tid, e.note).Scan(&id)
			if err1 != nil {
				fmt.Fprintf(&lg, "Problem inserting new entry %d: %s into postgres: %v\n", e.id, e.title[:15], err1)
				break
			}
			_, err2 := db.Exec("UPDATE task SET tid=? WHERE id=?;", id, e.id)
			if err2 != nil {
				fmt.Fprintf(&lg, "Problem setting new client entry's tid: %v; id: %v\n", id, e.id, err2)
				break
			}
			fmt.Fprintf(&lg, "Set value of tid for client task with id: %v to tid = %v\n", e.id, id)

		default:
			fmt.Fprintf(&lg, "Something went wrong in client_updated_entries for client entry id: %v\n", e.id)
		}
	}
	// server deleted entries
	for _, e := range server_deleted_entries {
		res, err := db.Exec("DELETE FROM task WHERE tid=?;", e.id)
		if err != nil {
			fmt.Fprintf(&lg, "Problem deleting local entry with tid = %v", e.id)
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
			fmt.Fprintf(&lg, "(rowsAffected != 1)Problem deleting local entry with tid = %v", e.id)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client entry %v with tid %v", e.title, e.id)
	}

	// client deleted entries
	for _, e := range client_deleted_entries {
		// since on server, we just set deleted to true
		// since may have to sync with other clients
		res, err := pdb.Exec("UPDATE task SET deleted=true, modified=now() WHERE id=$1", e.tid) /**************/
		if err != nil {
			fmt.Fprintf(&lg, "Problem (pdb.Exec) setting server entry with id = %v to deleted\n", e.tid)
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
			fmt.Fprintf(&lg, "(rowsAffected != 1) Problem setting server entry with id = %v to deleted; rowsAffected = %v\n", e.tid, rowsAffected)
			continue
		}
		fmt.Fprintf(&lg, "Updated server entry with id %v to deleted = true", e.tid)
	}

	//server_deleted_contexts
	//pdb.Exec("Update task SET context_tid=1 WHERE task.context_tid=c.id")
	//db.Exec("Update task SET context_tid=1 WHERE task.context_tid=c.id")
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

		res, err = db.Exec("DELETE FROM context WHERE tid=?", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Problem deleting local context with tid = %v", c.id)
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
			fmt.Fprintf(&lg, "(rowsAffected != 1) Problem deleting local context %v with tid = %v", c.id)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client context %v with tid %v", c.title, c.id)
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
		res, err = pdb.Exec("UPDATE context SET deleted=true, modified=now() WHERE context.id=$1", c.tid)
		if err != nil {
			fmt.Fprintf(&lg, "Problem (pdb.Exec) setting server context with id = %v to deleted\n", c.tid)
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
			fmt.Fprintf(&lg, "(rowsAffected != 1) Problem setting server context with id = %v to deleted; rowsAffected = %v\n", c.tid, rowsAffected)
			continue
		}
		fmt.Fprintf(&lg, "Updated server context with id %v to deleted = true", c.tid)
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

		res, err = db.Exec("DELETE FROM folder WHERE tid=?", c.id)
		if err != nil {
			fmt.Fprintf(&lg, "Problem deleting local folder with tid = %v", c.id)
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
			fmt.Fprintf(&lg, "(rowsAffected != 1) Problem deleting local folder %v with tid = %v", c.id)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client folder %v with tid %v", c.title, c.id)
	}

	// client deleted folders
	for _, c := range client_deleted_folders {
		res, err := pdb.Exec("Update task SET folder_tid=1, modified=now()  WHERE folder_tid=$1;", c.tid) //?modified=now()
		if err != nil {
			fmt.Fprintf(&lg, "Error trying to change server/postgres entry folders for a deleted folder: %v\n", err)
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
		res, err = pdb.Exec("UPDATE folder SET deleted=true, modified=now() WHERE folder.id=$1", c.tid)
		if err != nil {
			fmt.Fprintf(&lg, "Problem (pdb.Exec) setting server folder with id = %v to deleted\n", c.tid)
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
			fmt.Fprintf(&lg, "(rowsAffected != 1) Problem setting server folder with id = %v to deleted; rowsAffected = %v\n", c.tid, rowsAffected)
			continue
		}
		fmt.Fprintf(&lg, "Updated server folder with id %v to deleted = true", c.tid)
	}

	//server_deleted_keywords
	for _, c := range server_deleted_keywords {
		pdb.Exec("DELETE FROM task_keyword WHERE keyword_id=$1;", c.id)
		row = db.QueryRow("SELECT id FROM keyword WHERE keyword.tid=?", c.id)
		var id int
		err = row.Scan(&id)
		if err != nil {
			sess.showOrgMessage("Problem with getting current time from server: %w", err)
			return
		}
		db.Exec("DELETE FROM task_keyword WHERE keyword_id=?;", id)
		//res, err := db.Exec("DELETE FROM keyword WHERE tid=?", c.id)
		res, err := db.Exec("DELETE FROM keyword WHERE tid=?", id)
		if err != nil {
			fmt.Fprintf(&lg, "Problem deleting local keyword with tid = %v", c.id)
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
			fmt.Fprintf(&lg, "(rowsAffected != 1) Problem deleting local keyword %v with tid = %v", c.id)
			continue
		}
		fmt.Fprintf(&lg, "Deleted client keyword %v with tid %v", c.title, c.id)
	}

	// client deleted keywords
	for _, c := range client_deleted_keywords {
		pdb.Exec("DELETE FROM task_keyword WHERE keyword_id=$1;", c.tid)
		db.Exec("DELETE FROM task_keyword WHERE keyword_id=?;", c.id)
		// since on server, we just set deleted to true
		// since may have to sync with other clients
		res, err := pdb.Exec("UPDATE keyword SET deleted=true WHERE keyword.id=$1", c.tid)
		if err != nil {
			fmt.Fprintf(&lg, "Problem (pdb.Exec) setting server keyword with id = %v to deleted\n", c.tid)
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
			fmt.Fprintf(&lg, "(rowsAffected != 1) Problem setting server keyword with id = %v to deleted; rowsAffected = %v\n", c.tid, rowsAffected)
			continue
		}
		fmt.Fprintf(&lg, "Updated server keyword with id %v to deleted = true", c.tid)
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

	sess.drawPreviewText2(lg.String())
	sess.drawPreviewBox()
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
