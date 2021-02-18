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

type Organizer struct {
	mode int
  last_mode int

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

  show_deleted bool
  show_completed bool

  view int
  taskview int
  current_task_id int
  string_buffer string
  fts_titles map[int]string

  context_map map[string]int
  folder_map map[string]int
  sort_map map[string]int

  ftsIDs []int
  marked_entries []int

  title_search_string string
}

var org Organizer
