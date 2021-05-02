module github.com/slzatz/listmango

//replace github.com/charmbracelet/glamour v0.3.0 => github.com/slzatz/glamour v0.3.1
replace github.com/charmbracelet/glamour v0.3.0 => /home/slzatz/glamour

go 1.16

require (
	github.com/alecthomas/chroma v0.8.2
	github.com/charmbracelet/glamour v0.3.0
	github.com/lib/pq v1.10.1
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/neovim/go-client v1.1.7
	golang.org/x/sys v0.0.0-20210330210617-4fbd30eecc44
	golang.org/x/term v0.0.0-20210429154555-c04ba851c2a4 // indirect
)
