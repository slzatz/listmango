package main

import (
	"bufio"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

func highlightMispelledWords(rows []string) []string {

	cmd := exec.Command("nuspell", "-d", "en_US")
	var highlighted_rows []string

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sess.showEdMessage("Problem in highlightMispelled stdout: %v", err)
		return highlighted_rows
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		sess.showEdMessage("Problem in highlightMispelled stdin: %v", err)
		return highlighted_rows
	}
	err = cmd.Start()
	if err != nil {
		sess.showEdMessage("Problem in highlightMispelled stdin: %v", err)
		return highlighted_rows
	}
	buf_out := bufio.NewReader(stdout)
	// just sees tab as any other character

	for _, row := range rows {
		io.WriteString(stdin, row+"\n")
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

		var positions [][2]int
		for _, np_row := range np_rows {
			if np_row == "*" {
			} else if np_row == "" {
			} else {
				z := strings.SplitN(np_row, ":", 2)
				data := strings.Split(z[0], " ")
				pos, _ := strconv.Atoi(data[3])
				length := len(data[1])
				//suggestions := strings.Split(strings.ReplaceAll(z[1], " ", ""), ",")
				positions = append(positions, [2]int{pos, length})
			}
		}
		highlighted_row := row
		for j := len(positions) - 1; j >= 0; j-- {
			pos := positions[j][0]
			length := positions[j][1]
			//fmt.Println(j, positions[j])
			//highlighted_row = highlighted_row[:pos] + "\x1b[1m" + highlighted_row[pos:pos+length] + "\x1b[0m" + highlighted_row[pos+length:]
			highlighted_row = highlighted_row[:pos] + "\x1b[48;5;31m" + highlighted_row[pos:pos+length] + "\x1b[0m" + highlighted_row[pos+length:]
		}
		//fmt.Printf("%d: %s\n", k, highlighted_row)
		highlighted_rows = append(highlighted_rows, highlighted_row)
	}
	stdin.Close()
	return highlighted_rows
}
