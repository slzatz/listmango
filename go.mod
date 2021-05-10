module github.com/slzatz/listmango

//replace github.com/charmbracelet/glamour v0.3.0 => github.com/slzatz/glamour v0.3.1
replace github.com/charmbracelet/glamour v0.3.0 => /home/slzatz/glamour

go 1.16

require (
	github.com/alecthomas/chroma v0.9.1
	github.com/charmbracelet/glamour v0.3.0
	github.com/disintegration/imaging v1.6.2
	github.com/lib/pq v1.10.1
	github.com/mattn/go-runewidth v0.0.12 // indirect
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/microcosm-cc/bluemonday v1.0.9 // indirect
	github.com/neovim/go-client v1.1.7
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/yuin/goldmark v1.3.5 // indirect
	golang.org/x/image v0.0.0-20210504121937-7319ad40d33e // indirect
	golang.org/x/net v0.0.0-20210510120150-4163338589ed // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56
)
