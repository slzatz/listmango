package main

func organzerProcessKey(k rune) {

  o := &s.organizers[s.organizerIndex]

	switch o.mode {

  case norows:
  switch k {
  case ':':
    o.command = ""
    o.commandLine = ""
    orgShowMessage(":")
    o.mode = commandLine
    return
  case '\x1b':
    o.command = ""
    o.commandLine = ""
    o.repeat = 0
    return
  case 'i', 'I', 'a', 'A', 's':
    o.insertRow()
    o.mode = insert
    o.command = ""
    o.repeat = 0
    orgShowMessage("\x1b[1m-- INSERT --\x1b[0m")
  }
	case insert:
    switch k {
    case '\r':
      if o.view == task {
        o.updateTitle()
      }
    case '\b':
      o.backspace()
      return
    case KeyEnd:
      entry = o.entries[o.fr]
      if entry.title > 0 {
        o.fc = len(entry.title) //rem you're in insert mode
      {
    case KeyArrowUp, KeyArrowDown, KeyArrowRight, KeyArrowLeft:
      o.moveCursor(k)
    case 'x1b':
      o.command = ""
      o.mode = normal
      if o.fc > 0 {
        o.fc--
      }
      orgShowMessage("")
    default:
      o.insertChar(k)
      return
    }

	case normalMode:

    switch k {

    case '\x1b':
      e.command = ""
      e.repeat = 0
      return false

    case ':':
      o.mode = commandLine
      o.commandLine = ""
      o.command = ""
      o.setMessage(":")
      return false

    case KeyArrowUp, KeyArrowDown, KeyArrowRight, KeyArrowLeft:
      o.moveCursor(k)
      command = ""
      o.repeat = 0
      return

    default:
      o.command += k
      if v, found := n_lookup[o.command]; found {
        n_lookup[o.command]()
        o.command = ""
        o.repeat = 0
        return
    }
  } //end switch
	case commandLineMode:

    switch k {

    case '\x1b':
      o.mode = normal
      orgShowMessage("")
      return false
    case '\r':
      if commandLine == 'e' {
        o.editEntry()
        orgShowMessage("edit entry")
      } else {
        orgShowMessage(commandLine)
      }
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
