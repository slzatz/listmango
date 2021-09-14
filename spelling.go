package main

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
		// adjustment is made in drawHighlights
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

