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
	/*
	  Redis struct {
	    Host     string `json:"host"`
	    Password string `json:"password"`
	    DB       string `json:"db"`
	  } `json:"redis"`
	*/
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

// FromFile returns a configuration parsed from the given file.
func FromFile(path string) (*dbConfig, error) {
	b, err := ioutil.ReadFile(path)
	//fmt.Printf("Result of ioutil.ReadFile is %T\n", b)
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
		fmt.Printf("ping error: %v", err)
	} else {
		sess.showOrgMessage("ping success!")
	}

	nn := 0
	var lg strings.Builder
	lg.WriteString("****************************** BEGIN SYNC *******************************************\n\n")

	row := db.QueryRow("SELECT timestamp FROM sync WHERE machine=$1;", "client")
	var client_t string
	err = row.Scan(&client_t)
	if err != nil {
		log.Fatalf("Error retrieving last_client_sync: %w", err)
	}
	last_client_sync, _ := time.Parse("2006-01-02T15:04:05Z", client_t)

	var server_t string
	row = db.QueryRow("SELECT timestamp FROM sync WHERE machine=$1;", "server")
	err = row.Scan(&server_t)
	if err != nil {
		log.Fatalf("Error retrieving last_server_sync: %w", err)
	}
	last_server_sync, _ := time.Parse("2006-01-02T15:04:05Z", server_t)
	sess.showOrgMessage("last_client_sync = %v; last_server_sync = %v", last_client_sync, last_server_sync)
	sess.showOrgMessage("last_server_sync = %v | %T; server_t = %v | %T", last_server_sync, last_server_sync, server_t, server_t)

	fmt.Fprintf(&lg, "Local time is %v", time.Now())
	fmt.Fprintf(&lg, "UTC time is %v", time.Now().UTC())
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
	rows, err = pdb.Query("SELECT id,title,star,created,modified FROM task WHERE task.modified > $1 AND task.deleted = $2;", server_t, false)
	var server_updated_entries []Entry
	for rows.Next() {
		var e Entry
		rows.Scan(
			&e.id,
			&e.title,
			&e.star,
			&e.created,
			&e.modified,
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
	rows, err = pdb.Query("SELECT id,title,star,created,modified FROM task WHERE task.modified > $1 AND task.deleted = $2;", server_t, true)
	var server_deleted_entries []Entry
	for rows.Next() {
		var e Entry
		rows.Scan(
			&e.id,
			&e.title,
			&e.star,
			&e.created,
			&e.modified,
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
	rows, err = db.Query("SELECT id, tid, title, star, created, modified FROM task WHERE task.modified > $1 AND task.deleted = $2;", client_t, false)
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
	rows, err = db.Query("SELECT id, tid, title, star, created, modified FROM task WHERE task.modified > $1 AND task.deleted = $2;", client_t, true)
	var client_deleted_entries []Entry
	for rows.Next() {
		var e Entry
		rows.Scan(
			&e.id,
			&e.tid,
			&e.title,
			&e.star,
			&e.created,
			&e.modified,
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
}
