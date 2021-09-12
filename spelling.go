package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

func (e *Editor) highlightMispelledWords() {
	e.highlightPositions = nil
	var rowNum int
	var start int
	var end int
	var ln interface{}
	curPos, _ := v.WindowCursor(w)
	v.Command("set spell")
	v.Input("gg")
	v.Input("]s")
	first, _ := v.WindowCursor(w)
	rowNum = first[0] - 1
	start = first[1]
	err := v.Command("let length = strlen(expand('<cword>'))")
	if err != nil {
		sess.showEdMessage("Error in test/cword =: %v", err)
	}
	v.Var("length", &ln)
	end = start + int(ln.(int64))
	e.highlightPositions = append(e.highlightPositions, Position{rowNum, start, end})
	var pos [2]int
	for {
		v.Input("]s")
		pos, _ = v.WindowCursor(w)
		if pos == first {
			break
		}
		rowNum = pos[0] - 1
		//start = utf8.RuneCount(p.bb[rowNum][:pos[1]])
		start = pos[1]
		err := v.Command("let length = strlen(expand('<cword>'))")
		if err != nil {
			sess.showEdMessage("Error in test/cword =: %v", err)
		}
		var ln interface{}
		v.Var("length", &ln)
		end = start + int(ln.(int64))

		e.highlightPositions = append(e.highlightPositions, Position{rowNum, start, end})
	}
	v.SetWindowCursor(w, curPos) //return cursor to where it was

	// done here because no need to redraw text
	/*
		var ab strings.Builder
		e.drawHighlights(&ab)
		fmt.Print(ab.String())
		sess.showEdMessage("e.highlightPositions = %=v", e.highlightPositions)
	*/
}

func (e *Editor) highlightMispelledWordsold() {
	e.highlightPositions = nil

	cmd := exec.Command("nuspell", "-d", "en_US")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sess.showEdMessage("Problem in highlightMispelled stdout: %v", err)
		return
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		sess.showEdMessage("Problem in highlightMispelled stdin: %v", err)
		return
	}
	defer stdin.Close()
	err = cmd.Start()
	if err != nil {
		sess.showEdMessage("Problem in cmd.Start (nuspell) stdin: %v", err)
		return
	}
	buf_out := bufio.NewReader(stdout)
	// just sees tab as any other character

	for rowNum, row := range p.bb {
		io.WriteString(stdin, string(row)+"\n")
		var np_rows []string
		for {
			//bytes, _, err := buf_out.ReadLine()
			bytes, _, _ := buf_out.ReadLine()

			// Don't think this is needed
			/*
				if err == io.EOF {
					break
				}
			*/

			if len(bytes) == 0 {
				break
			}

			np_rows = append(np_rows, string(bytes))
		}

		for _, np_row := range np_rows {
			switch np_row {
			case "*", "":
			default:
				z := strings.SplitN(np_row, ":", 2)
				data := strings.Split(z[0], " ")
				start, _ := strconv.Atoi(data[len(data)-1])
				end := start + len(data[1])
				e.highlightPositions = append(e.highlightPositions, Position{rowNum, start, end})
			}
		}
	}

	// done here because no need to redraw text
	var ab strings.Builder
	e.drawHighlights(&ab)
	fmt.Print(ab.String())
}
