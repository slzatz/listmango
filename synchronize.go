package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
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
		log.Fatalf("Problem reading config file: %w", err)
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
		log.Fatalf("Problem opening db: %w", err)
	}

	// Ping to connection
	err = pdb.Ping()
	if err != nil {
		sess.showOrgMessage("postgres ping failure!: %w", err)
		return
	} else {
		sess.showOrgMessage("postgres ping success!")
	}

	nn := 0
	var lg strings.Builder
	lg.WriteString("****************************** BEGIN SYNC *******************************************\n\n")

	row := db.QueryRow("SELECT timestamp FROM sync WHERE machine=$1;", "client")
	var client_t string
	err = row.Scan(&client_t)
	if err != nil {
		sess.showOrgMessage("Error retrieving last_client_sync: %w", err)
		return
	}
	last_client_sync, _ := time.Parse("2006-01-02T15:04:05Z", client_t)

	var server_t string
	row = db.QueryRow("SELECT timestamp FROM sync WHERE machine=$1;", "server")
	err = row.Scan(&server_t)
	if err != nil {
		sess.showOrgMessage("Error retrieving last_server_sync: %w", err)
		return
	}
	last_server_sync, _ := time.Parse("2006-01-02T15:04:05Z", server_t)
	sess.showOrgMessage("last_client_sync = %v; last_server_sync = %v\n", last_client_sync, last_server_sync)

	fmt.Fprintf(&lg, "Local time is %v\n", time.Now())
	fmt.Fprintf(&lg, "UTC time is %v\n", time.Now().UTC())
	sess.showEdMessage("local time = %v; UTC time = %v; since last sync = %v", time.Now(), time.Now().UTC(), timeDelta(client_t))

	//server updated contexts
	rows, err := pdb.Query("SELECT id, title, \"default\", created, deleted, modified FROM context WHERE context.modified > $1 AND context.deleted = $2;", server_t, false)

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
	rows, err = pdb.Query("SELECT id, title, \"default\", created, modified FROM context WHERE context.modified > $1 AND context.deleted = $2;", server_t, true)
	var server_deleted_contexts []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.title,
			&c.star,
			&c.created,
			&c.modified,
		)
		server_deleted_contexts = append(server_deleted_contexts, c)
	}
	if len(server_deleted_contexts) > 0 {
		nn += len(server_deleted_contexts)
		fmt.Fprintf(&lg, "Deleted server Contexts since last sync: %d\n", len(server_updated_contexts))
	} else {
		lg.WriteString("No server Contexts deleted since last sync.\n")
	}

	//server updated folders
	rows, err = pdb.Query("SELECT id, title, private, created, deleted, modified FROM folder WHERE folder.modified > $1 AND folder.deleted = $2;", server_t, false)

	defer rows.Close()

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
	rows, err = pdb.Query("SELECT id, title, private, created, modified FROM folder WHERE folder.modified > $1 AND folder.deleted = $2;", server_t, true)
	var server_deleted_folders []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.title,
			&c.star,
			&c.created,
			&c.modified,
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

	defer rows.Close()

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
	rows, err = pdb.Query("SELECT id, name, star, modified FROM keyword WHERE keyword.modified > $1 AND keyword.deleted = $2;", server_t, true)
	var server_deleted_keywords []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.title,
			&c.star,
			&c.modified,
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
	rows, err = pdb.Query("SELECT id, title, star, created, modified, added, completed, context_tid, folder_tid, note FROM task WHERE task.modified > $1 AND task.deleted = $2;", server_t, false)
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
		//fmt.Printf("%v\n", e)
		server_updated_entries = append(server_updated_entries, e)
	}
	if len(server_updated_entries) > 0 {
		nn += len(server_updated_entries)
		fmt.Fprintf(&lg, "Updated (new and modified) server Entries since last sync: %d\n", len(server_updated_entries))
	} else {
		lg.WriteString("No updated (new and modified) server Entries since last sync.\n")
	}
	sess.showEdMessage("Number of changes that server needs to transmit to client: %v", len(server_updated_entries))
	for _, e := range server_updated_entries {
		fmt.Fprintf(&lg, "id: %v; title: %v; star: %v, created: %v; modified; %v\n", e.id, e.title, e.star, e.created, e.modified)
	}
	//for _, et := range entries {
	// fmt.Printf("id = %v, title = %v, star = %v, created = %v, modified = %v\n", et.id, et.title, et.star, et.created, et.modified)

	// }
	//server deleted entries
	//rows, err = pdb.Query("SELECT id,title,star,created,modified FROM task WHERE task.modified > $1 AND task.deleted = $2;", server_t, true)
	rows, err = pdb.Query("SELECT id, title FROM task WHERE task.modified > $1 AND task.deleted = $2;", server_t, true)
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
	rows, err = db.Query("SELECT id, title, \"default\", created, deleted, modified FROM context WHERE context.modified > $1 AND context.deleted = $2;", client_t, false)

	defer rows.Close()

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
	rows, err = db.Query("SELECT id, title, \"default\", created, modified FROM context WHERE context.modified > $1 AND context.deleted = $2;", client_t, true)
	var client_deleted_contexts []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.title,
			&c.star,
			&c.created,
			&c.modified,
		)
		client_deleted_contexts = append(client_deleted_contexts, c)
	}
	if len(client_deleted_contexts) > 0 {
		nn += len(client_deleted_contexts)
		fmt.Fprintf(&lg, "Deleted client Contexts since last sync: %d\n", len(client_updated_contexts))
	} else {
		lg.WriteString("No client Contexts deleted since last sync.\n")
	}

	//client updated folders
	rows, err = db.Query("SELECT id, tid, title, private, created, deleted, modified FROM folder WHERE folder.modified > $1 AND folder.deleted = $2;", client_t, false)

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
		nn += len(client_updated_contexts)
		fmt.Fprintf(&lg, "Updated (new and modified) client Folders since last sync: %d\n", len(client_updated_folders))
	} else {
		lg.WriteString("No updated (new and modified) client Folders the last sync.\n")
	}

	//client deleted folders
	rows, err = db.Query("SELECT id, tid, title, private, created, modified FROM folder WHERE folder.modified > $1 AND folder.deleted = $2;", client_t, true)
	var client_deleted_folders []Container
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
		client_deleted_folders = append(client_deleted_folders, c)
	}
	if len(client_deleted_folders) > 0 {
		nn += len(client_deleted_folders)
		fmt.Fprintf(&lg, "Deleted client Folders since last sync: %d\n", len(client_updated_folders))
	} else {
		lg.WriteString("No client Folders deleted since last sync.\n")
	}

	//client updated keywords
	rows, err = db.Query("SELECT id, tid, name, star, modified FROM keyword WHERE keyword.modified > $1 AND keyword.deleted = $2;", client_t, false)

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
		nn += len(client_updated_contexts)
		fmt.Fprintf(&lg, "Updated (new and modified) client Keywords since last sync: %d\n", len(client_updated_keywords))
	} else {
		lg.WriteString("No updated (new and modified) client Keywords the last sync.\n")
	}

	//client deleted keywords
	rows, err = db.Query("SELECT id, tid, name, star, modified FROM keyword WHERE keyword.modified > $1 AND keyword.deleted = $2;", client_t, true)
	var client_deleted_keywords []Container
	for rows.Next() {
		var c Container
		rows.Scan(
			&c.id,
			&c.tid,
			&c.title,
			&c.star,
			&c.modified,
		)
		client_deleted_keywords = append(client_deleted_keywords, c)
	}
	if len(client_deleted_keywords) > 0 {
		nn += len(client_deleted_keywords)
		fmt.Fprintf(&lg, "Deleted client Keywords since last sync: %d\n", len(client_updated_keywords))
	} else {
		lg.WriteString("No client Keywords deleted since last sync.\n")
	}
	//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//client updated entries
	rows, err = db.Query("SELECT id, tid, title, star, created, modified, added, completed, context_tid, folder_tid FROM task WHERE task.modified > $1 AND task.deleted = $2;", client_t, false)
	var client_updated_entries []Entry
	for rows.Next() {
		var e Entry
		rows.Scan(
			&e.id,
			&e.tid,
			&e.title,
			&e.star,
			&e.created,
			&e.modified,
			&e.added,
			&e.completed,
			&e.context_tid,
			&e.folder_tid,
		)
		//fmt.Printf("%v\n", e)
		client_updated_entries = append(client_updated_entries, e)
	}
	if len(client_updated_entries) > 0 {
		nn += len(client_updated_entries)
		fmt.Fprintf(&lg, "Updated (new and modified) client Entries since last sync: %d\n", len(client_updated_entries))
	} else {
		lg.WriteString("No updated (new and modified) client Entries since last sync.\n")
	}
	sess.showEdMessage("Number of changes that client needs to transmit to client: %v", len(client_updated_entries))
	for _, e := range client_updated_entries {
		fmt.Fprintf(&lg, "id: %v; tid: %v; title: %v; star: %v, created: %v; modified; %v\n", e.id, e.tid, e.title, e.star, e.created, e.modified)
	}

	//client deleted entries
	rows, err = db.Query("SELECT id, tid, title WHERE task.modified > $1 AND task.deleted = $2;", client_t, true) //not sure need task.modified??
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

	sess.showEdMessage("Number of changes = %d", nn)
	sess.drawPreviewText2(lg.String())
	sess.drawPreviewBox()

	//updated server contexts -> client

	if len(server_updated_contexts) > 0 {
		for _, c := range server_updated_contexts {
			row := db.QueryRow("SELECT id from context WHERE tid=?", c.id)
			var id int
			err = row.Scan(&id)
			switch {
			case err == sql.ErrNoRows:
				//res, err := db.Exec("INSERT INTO context (tid, title, star, created, modified) VALUES (?,?,?,?, datetime('now'));",
				res, err := db.Exec("INSERT INTO context (tid, title, \"default\", created, modified, deleted) VALUES (?,?,?,?, datetime('now'), false);",
					c.id, c.title, c.star, c.created)
				if err != nil {
					log.Fatal(err)
				}
				lastId, _ := res.LastInsertId()
				fmt.Fprintf(&lg, "Created new local context: %v with local id: %v and tid: %v\n", c.title, lastId, c.id)
			case err != nil:
				log.Fatal(err)
			default:
				//res, err := db.Exec("UPDATE context SET title=?, star=?, created=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.created, c.id)
				_, err := db.Exec("UPDATE context SET title=?, \"default\"=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.id)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Fprintf(&lg, "Updated local context: %v with tid: %v\n", c.title, c.id)
			}
		}
	}

	if len(server_updated_folders) > 0 {
		for _, c := range server_updated_folders {
			row := db.QueryRow("SELECT id from folder WHERE tid=?", c.id)
			var id int
			err = row.Scan(&id)
			switch {
			case err == sql.ErrNoRows:
				//res, err := db.Exec("INSERT INTO folder (tid, title, star, created, modified) VALUES (?,?,?,?, datetime('now'));",
				res, err := db.Exec("INSERT INTO folder (tid, title, private, created, modified, deleted) VALUES (?,?,?,?, datetime('now'), false);",
					c.id, c.title, c.star, c.created)
				if err != nil {
					log.Fatal(err)
				}
				lastId, _ := res.LastInsertId()
				fmt.Fprintf(&lg, "Created new local folder: %v with local id: %v and tid: %v\n", c.title, lastId, c.id)
			case err != nil:
				log.Fatal(err)
			default:
				//res, err := db.Exec("UPDATE folder SET title=?, star=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.id)
				_, err := db.Exec("UPDATE folder SET title=?, private=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.id)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Fprintf(&lg, "Updated local folder: %v with tid: %v\n", c.title, c.id)
			}
		}
	}

	if len(server_updated_keywords) > 0 {
		for _, c := range server_updated_keywords {
			row := db.QueryRow("SELECT id from keyword WHERE tid=?", c.id)
			var id int
			err = row.Scan(&id)
			switch {
			case err == sql.ErrNoRows:
				//res, err := db.Exec("INSERT INTO keyword (tid, title, star, created, modified) VALUES (?,?,?,?, datetime('now'));",
				res, err := db.Exec("INSERT INTO keyword (tid, name, star, created, modified, deleted) VALUES (?,?,?,?, datetime('now'), false);",
					c.id, c.title, c.star, c.created)
				if err != nil {
					log.Fatal(err)
				}
				lastId, _ := res.LastInsertId()
				fmt.Fprintf(&lg, "Created new local keyword: %v with local id: %v and tid: %v\n", c.title, lastId, c.id)
			case err != nil:
				log.Fatal(err)
			default:
				//res, err := db.Exec("UPDATE keyword SET title=?, star=?, created=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.created, c.id)
				_, err := db.Exec("UPDATE keyword SET name=?, star=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.id)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Fprintf(&lg, "Updated local keyword: %v with tid: %v\n", c.title, c.id)
			}
		}
	}
	var server_updated_entries_ids map[int]struct{}
	for _, e := range server_updated_entries {
		server_updated_entries_ids[e.id] = struct{}{}
		row := db.QueryRow("SELECT id from task WHERE tid=?", e.id)
		var id int
		err = row.Scan(&id)
		switch {
		case err == sql.ErrNoRows:
			res, err := db.Exec("INSERT INTO task (tid, title, star, created, added, completed, context_tid, folder_tid, note, modified, deleted) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), false);",
				e.id, e.title, e.star, e.created, e.added, e.completed, e.context_tid, e.folder_tid, e.note)
			if err != nil {
				log.Fatal(err)
			}
			lastId, _ := res.LastInsertId()
			_, err = fts_db.Exec("INSERT INTO fts (title, note, lm_id) VALUES (?, ?, ?);", e.title, e.note, lastId)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(&lg, "Created new local entry: %v with local id: %v and tid: %v\n", e.title, lastId, e.id)
		case err != nil:
			log.Fatal(err)
		default:
			//res, err := db.Exec("UPDATE context SET title=?, star=?, created=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.created, c.id)
			_, err := db.Exec("UPDATE task SET title=?, star=?, context_tid=?, folder_tid=?, note=?, modified=datetime('now') WHERE tid=?;",
				e.title, e.star, e.context_tid, e.folder_tid, e.note, e.id)
			if err != nil {
				log.Fatal(err)
			}
			row = db.QueryRow("SELECT id FROM task WHERE task.tid=?;", e.id)
			var lm_id int
			row.Scan(&lm_id)
			if err != nil {
				log.Fatalf("Error retrieving last_client_sync: %w", err)
			}
			_, err = fts_db.Exec("UPDATE fts SET title=?, note=? WHERE lm_id=?;", e.title, e.note, lm_id)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(&lg, "Updated local entry: %v with tid: %v\n", e.title, e.id)
		}
	}

	for _, e := range client_updated_entries {
		// server wins
		if _, found := server_updated_entries_ids[e.tid]; found {
			continue
		}
		row := pdb.QueryRow("SELECT id from task WHERE id=?", e.tid)
		var id int
		err = row.Scan(&id)
		switch {
		case err == sql.ErrNoRows:
			res, err := pdb.Exec("INSERT INTO task (title, star, created, added, completed, context_tid, folder_tid, note, modified, deleted) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), false);",
				e.title, e.star, e.created, e.added, e.completed, e.context_tid, e.folder_tid, e.note)
			if err != nil {
				log.Fatal(err)
			}
			lastId, _ := res.LastInsertId()
			db.Exec("UPDATE task SET task.tid=$1 WHERE task.id=$2;", lastId, e.id)
			fmt.Fprintf(&lg, "Created new server entry: %v with server id: %v\n", e.title, lastId)
			fmt.Fprintf(&lg, "Set value of tid for client task with id: %v to tid = %v\n", e.id, lastId)
		case err != nil:
			log.Fatal(err)
		default:
			//res, err := db.Exec("UPDATE context SET title=?, star=?, created=?, modified=datetime('now') WHERE tid=?;", c.title, c.star, c.created, c.id)
			_, err := pdb.Exec("UPDATE task SET title=?, star=?, context_tid=?, folder_tid=?, note=?, modified=datetime('now') WHERE id=?;",
				e.title, e.star, e.context_tid, e.folder_tid, e.note, e.tid)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(&lg, "Updated server entry: %v with id: %v\n", e.title, e.tid)
		}
	}
	// server deleted entries
	for _, e := range server_deleted_entries {
		res, err := db.Exec("DELETE FROM task WHERE task.tid=?;", e.id)
		if err != nil {
			fmt.Fprintf(&lg, "Problem deleting local task with tid = %v", e.id)
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
			fmt.Fprintf(&lg, "Problem deleting local task with tid = %v", e.id)
		}
	}

	// client deleted entries
	for _, e := range client_deleted_entries {
		res, err := pdb.Exec("UPDATE task SET deleted=true WHERE task.tid=?", e.id)
		if err != nil {
			fmt.Fprintf(&lg, "Problem (pdb.Exec) setting server task with tid = %v to deleted\n", e.id)
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
			fmt.Fprintf(&lg, "Problem setting server task with tid = %v to deleted; rowsAffected = %v\n", e.id, rowsAffected)
			continue
		}
		fmt.Fprintf(&lg, "Updated server task with tid %v to deleted = true", e.id)
	}
	////////////////////////////////////////////////////
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
	fmt.Fprintf(&lg, "Client UTC timestamp: %s", client_ts)
	fmt.Fprint(&lg, "Server UTC timestamp: %s", server_ts)
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
