package create_databases

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func createDB() {
	filename = "config.json.test"
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("What do you want to name the database? ")
	res, _ := reader.ReadString('\n')
	config.Sqlite3.DB = res
	config.Sqlite3.FTS_DB = "fts5_" + res
	z = json.Marshall(config)
	f, err := os.Create(filename)
	if err != nil {
		sess.showEdMessage("Error creating file %s: %v", filename, err)
		return
	}
	defer f.Close()

	_, err = f.WriteString(z)
	if err != nil {
		sess.showEdMessage("Error writing file %s: %v", filename, err)
		return
	}
	sess.showEdMessage("Wrote configuration file %s", filename)
	// write this to config file

	db, _ = sql.Open("sqlite3", config.Sqlite3.DB)
	fts_db, _ = sql.Open("sqlite3", config.Sqlite3.FTS_DB)

}
