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
