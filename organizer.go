package main

import (
	"fmt"
)

type oMode int

const (
	insert oMode = iota
	normal
	commandLine
)

type organizer struct {
	mode oMode
	rows entry
}

func (*organizer) drawPreview() {
}

type entry struct {
	id       int
	star     bool
	title    []rune
	ftsTitle []rune
	deleted  bool
	modified string //????
	dirty    bool
	mark     bool
	rowoff   int
	coloff   int
}
