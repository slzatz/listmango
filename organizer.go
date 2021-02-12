package main

import (
	"fmt"
)

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

type org struct {
	mode int
  lastMode int

  cx, cy int //cursor x and y position
  fc, fr int// file x and y position
  rowoff int //the number of rows scrolled (aka number of top rows now off-screen
  coloff int; //the number of columns scrolled (aka number of left rows now off-screen

  rows []entry
  context string
  folder string
  keywords string
  sort string
  cmdLine string

  normCmd string
  repeat int

  showDeleted bool
  showCompleted bool

  view int
  taskview int
  currentTaskID int
  stringBuffer string
  ftsTitles map[int]string

  contextMap map[string]int
  folderMap map[string]int
  sortMap map[string]int

  ftsIDs []int
  markedEntries []int

  titleSearchString string
}

func (*org)  outlineScroll() {
}
