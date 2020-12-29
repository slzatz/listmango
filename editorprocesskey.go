package main

func editorProcessKey(k) {

  e := &s.editors[s.editorIndex]

	switch e.mode {

	case insert:
    switch k {
    case '\r':
      e.insertReturn()
      e.lastTyped += k
    case '\b':
      e.backspace()
      e.lastTyped += k
    case KeyEnd:
      e.moveCursorEOL()
      e.moveCursor(KeyArrowRight)
    case KeyArrowUp, KeyArrowDown, KeyArrowRight, KeyArrowLeft:
      e.moveCursor(k)
    case 'x1b':
      if cmd, found : = cmdMap1[e.lastCommand]; found {
        e.pushCurrent()
        for n int = 0; n < e.lastRepeat - 1; n++ {
          for _, c := range e.lastTyped {
            e.insertChar(c)
          }
        }
      }
      default:
        e.insertChar(k)
        e.lastTyped += k
    }
	case normal:
    switch k {
    case '\x1b':
      e.command = ""
      e.repeat = 0
      return false
    case ':':
      e.mode = commandLine
      e.commandLine = ""
      e.command = ""
      e.setMessage(":")
      return false

	case commandLine:
    switch k {
    case '\x1b':
      e.mode = normal
      e.command = ""
      e.repeat = e.lastRepeat = 0
      editorSetMessage("")
      return false
    case '\r':
      editorSetMessage("you executed a command")
    default:
      if k == KeyDelete || k == '\b') {
        if !e.commandLine != "" {
          e.commandLine = e.commandLine[0:len(e.commandLine - 1)]
      } else {
          e.commandLine += k
      }
      editorSetMessage(":%s", commandLine);
      return false; //end of case COMMAND_LINE
    }
	}
}
