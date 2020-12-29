package main

import (
	"fmt"
)

type org struct {
	mode int
	rows []entry
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

func (*org) drawPreview() {
}

