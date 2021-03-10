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
	sess.showEdMessage("last_client_sync = %v; last_server_sync = %v", last_client_sync, last_server_sync)

	fmt.Fprintf(&lg, "Local time is %v", time.Now())
	fmt.Fprintf(&lg, "UTC time is %v", time.Now().UTC())
	sess.showEdMessage("local time = %v; UTC time = %v; since last sync = %v", time.Now(), time.Now().UTC(), timeDelta(client_t))

	//server updated contexts
	rows, err := pdb.Query("SELECT id, title, \"default\", created, deleted, modified FROM context WHERE context.modified > $1 AND context.deleted = $2;", last_server_sync, false)

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

	rows, err = pdb.Query("SELECT id, title, \"default\", created, modified FROM context WHERE context.modified > $1 AND context.deleted = $2;", last_server_sync, true)
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

	rows, err = pdb.Query("SELECT id,title,star,created,modified FROM task WHERE task.modified > $1;", server_t)
	var entries []Entry
	for rows.Next() {
		var e Entry
		rows.Scan(
			&e.id,
			//&e.tid,
			&e.title,
			&e.star,
			&e.created,
			&e.modified,
		)
		//fmt.Printf("%v\n", e)
		entries = append(entries, e)
	}
	sess.showEdMessage("Number of changes that server needs to transmit to client: %v", len(entries))
	//for _, et := range entries {
	// fmt.Printf("id = %v, title = %v, star = %v, created = %v, modified = %v\n", et.id, et.title, et.star, et.created, et.modified)

	// }
}
