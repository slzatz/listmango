package main

var e_lookup = map[string]func(*Editor, int) {
                   "i":(*Editor).E_i,
                   "I":(*Editor).E_I,
                   "a":(*Editor).E_a,
                   "A":(*Editor).E_A,
                   "o":(*Editor).E_o,
                   "O":(*Editor).E_O,
                   "x":(*Editor).E_x,
                   "dw":(*Editor).E_dw,
                   "daw":(*Editor).E_daw,
                   "dd":(*Editor).E_dd,
                   "d$":(*Editor).E_deol,
                   "de":(*Editor).E_de,
                   "dG":(*Editor).E_dG,
                   "cw":(*Editor).E_cw,
                   "caw":(*Editor).E_caw,
                   "s":(*Editor).E_s,
                 }

func (e *Editor) E_i(repeat int) {
  switch repeat {
  case -1:
  }
}
func (e *Editor) E_I(repeat int) {
  e.moveCursorBOL();
  e.fc = e.indentAmount(e.fr);
}

func (e *Editor) E_a(repeat int) {
  e.moveCursor(ARROW_RIGHT)
}

func (e *Editor) E_A(repeat int) {
  e.moveCursorEOL();
  e.moveCursor(ARROW_RIGHT); //works even though not in INSERT mode
}

func (e *Editor) E_o(repeat int) {
  e.last_typed = ""
  e.insertNewline(1)
}

func (e *Editor) E_O(repeat int) {
  e.last_typed = ""
  e.insertNewline(0)
}

func (e *Editor) E_x(repeat int) {
  r = &e.rows[e.fr]
  if len(*r) == 0 {
    return
  }
  *r = (*r)[:fc] + (*r)[fc+1:]
  for i:=1; i < repeat; i++ {
    if fc == len(*r) - 1 {
      fc--
      break;
    }
    *r = (*r)[:fc] + (*r)[fc+1:]
  }
  dirty++
}

func (e *Editor) E_dw(repeat int) {
  for j := 0; j < repeat; j++ {
    start := e.fc
    e.moveEndWord2()
    end := e.fc
    e.fc = start
    r = &e.rows[e.fr]
    *r = (*r)[:e.fc] +(*r)[end+1:]
  }
}

func (e *Editor) E_daw(repeat int) {
}

func (e *Editor) E_dd(repeat int) {
}

func (e *Editor) E_deol(repeat int) {
}

func (e *Editor) E_de(repeat int) {
}

func (e *Editor) E_dG(repeat int) {
}

func (e *Editor) E_cw(repeat int) {
}

func (e *Editor) E_caw(repeat int) {
}

func (e *Editor) E_s(repeat int) {
}

