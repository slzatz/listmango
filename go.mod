module github.com/slzatz/listmango

//replace github.com/charmbracelet/glamour v0.3.0 => github.com/slzatz/glamour v0.3.1
replace github.com/charmbracelet/glamour v0.3.0 => /home/slzatz/glamour

replace github.com/neovim/go-client v1.1.7 => /home/slzatz/go-client

go 1.16

require (
	github.com/alecthomas/chroma v0.9.2
	github.com/charmbracelet/glamour v0.3.0
	github.com/disintegration/imaging v1.6.2
	github.com/lib/pq v1.10.2
	github.com/mattn/go-sqlite3 v1.14.8
	github.com/microcosm-cc/bluemonday v1.0.15 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.9.0 // indirect
	github.com/neovim/go-client v1.1.7
	github.com/yuin/goldmark v1.4.0 // indirect
	golang.org/x/image v0.0.0-20210628002857-a66eb6448b8d // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b
)
