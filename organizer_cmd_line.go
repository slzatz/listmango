package main


func F_open(pos int) { //C_open - by context
  if pos == 0 {
    sess.showOrgMessage("You did not provide a context!")
    org.mode = NORMAL
    return
  }

  cl := org.command_line
  var success bool
  success = false
  for k, _ := range org.context_map {
    if strings.HasPrefix(k, cl[pos+1:]) {
      org.context = k
      success = true
      break
    }
  }

  if (!success) {
    sess.showOrgMessage(fmt.Sprintf("%s is not a valid  context!", cl[:pos]))
    org.mode = NORMAL
    return
  }

  sess.showOrgMessage("'%s' will be opened", org.context);

  org.marked_entries = nil
  org.folder = ""
  org.taskview = BY_CONTEXT
  getItems(MAX)
  org.mode = NORMAL
  return
}

func F_openfolder(pos int) { //C_open - by context
  if pos == 0 {
    sess.showOrgMessage("You did not provide a folder!")
    org.mode = NORMAL
    return
  }

  cl := &org.command_line
  var success bool
  success = false;
  for k, _ := range org.folder {
    if strings.HasPrefix(k, (*cl)[pos+1:]) {
      org.folder = k
      success = true
      break
    }
  }

  if (!success) {
    sess.showOrgMessage("%s is not a valid  folder!", (*cl)[:pos])
    org.mode = NORMAL
    return
  }

  sess.showOrgMessage("'%s' will be opened", org.folder);

  org.marked_entries = nil
  org.context = ""
  org.taskview = BY_FOLDER
  getItems(MAX)
  org.mode = NORMAL
  return
}

func F_openkeyword(pos int) {
  if pos == 0 {
    sess.showOrgMessage("You did not provide a keyword!")
    org.mode = NORMAL
    return
  }
  //O.keyword = O.command_line.substr(pos+1);
  keyword := org.command_line[pos+1:]
  if keywordExists(keyword) == -1 {
    org.mode = org.last_mode
    sess.showOrgMessage("keyword '%s' does not exist!", keyword)
    return
  }

  org.keyword = keyword
  sess.showOrgMessage("'%s' will be opened", org.keyword)
  org.marked_entries = nil
  org.context = ""
  org.folder = ""
  org.taskview = BY_KEYWORD
  getItems(MAX)
  org.mode = NORMAL
  return
}

func F_write(pos int) {
  if org.view == TASK {
    updateRows()
  }
  org.mode = org.last_mode
  org.command_line = ""
}

