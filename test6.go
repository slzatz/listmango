package main

import (
	"bytes"
	"fmt"
	//	"io"
	"log"
	//"os"
	"context"
	//"errors"
	//	"reflect"
	"runtime"
	"strings"
	//"sync/atomic"
	"time"

	"github.com/neovim/go-client/nvim"
)

func main() {

	ctx := context.Background()
	opts := []nvim.ChildProcessOption{

		// -u NONE is no vimrc and -n is no swap file
		nvim.ChildProcessArgs("-u", "NONE", "-n", "--embed", "--headless", "--noplugin"),

		//without headless nothing happens but should be OK once ui attached.
		//nvim.ChildProcessArgs("-u", "NONE", "-n", "--embed", "--noplugin"),

		nvim.ChildProcessContext(ctx),
		nvim.ChildProcessLogf(log.Printf),
	}

	if runtime.GOOS == "windows" {
		opts = append(opts, nvim.ChildProcessCommand("nvim.exe"))
	}

	v, err := nvim.NewChildProcess(opts...)
	if err != nil {
		log.Fatal(err)
	}

	// Cleanup on return.
	defer v.Close()

	wins, err := v.Windows()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	fmt.Printf("wins = %v\n", wins)
	w := wins[0]

	bufs, err := v.Buffers()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Number of buffers: %v\n", len(bufs))

	// Example using batch to get the names of buffers
	//using a single atomic call to Nvim.
	names := make([]string, len(bufs))
	b := v.NewBatch()
	for i, buf := range bufs {
		b.BufferName(buf, &names[i])
	}
	if err := b.Execute(); err != nil {
		log.Fatal(err)
	}

	// Print the names.
	fmt.Println("Names of buffers")
	for _, name := range names {
		fmt.Println(name)
	}

	targetTab := nvim.Tabpage(1)
	fmt.Println(targetTab)

	mode, err := v.Mode()
	if err != nil {
		log.Fatal()
	}

	fmt.Printf("mode = %v\n", *mode)

	buf, _ := v.CurrentBuffer() //_ => err
	fmt.Printf("current buffer = %v\n", buf)

	// I am planning to run embedded with no window dimensions
	// I am going to have to scroll etc I believe
	//func SetWindowHeight(window Window, height int) {
	//func SetWindowWidth(window Window, height int) {

	const wantWriteOut = `hello WriteOut`
	if err := v.WriteOut(wantWriteOut + "\n"); err != nil {
		fmt.Println("failed to WriteOut: %v", err)
	}

	var gotWriteOut string
	if err := v.VVar("statusmsg", &gotWriteOut); err != nil {
		fmt.Println("could not get v:statusmsg nvim variable: %v", err)
	}
	fmt.Printf("writeout = %v\n", gotWriteOut)

	var stringData = []string{
		"hello world. The rain in spain falls mainly on the plain.",
		"hello brave world. The rain in spain falls mainly on the plain.",
		"hello sweet world. The rain in spain falls \nmainly on the plain.",
	}

	var byteData = [][]byte{
		[]byte("Norm"),
		[]byte("hello world"),
		[]byte("blank line"),
	}

	for _, d := range stringData { //dealing with \n in strings
		if err := v.SetBufferLines(buf, 0, -1, true, bytes.Split([]byte(strings.TrimSuffix(d, "\n")), []byte{'\n'})); err != nil {
			fmt.Printf("%v\n", err)
		}
	}
	//v.SetBufferText(buffer Buffer, startRow int, startCol int, endRow int, endCol int, replacement [][]byte) error {
	//	func (b *Batch) SetBufferLines(buffer Buffer, start int, end int, strictIndexing bool, replacement [][]byte) {
	//err = v.SetBufferText(buf, 1, 1, 1, 1, byteData)
	err = v.SetBufferLines(buf, 0, 0, true, byteData)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	//func (v *Nvim) BufferLines(buffer Buffer, start int, end int, strictIndexing bool) (lines [][]byte, err error) {
	printBuffer(v, buf)
	/*
		fmt.Println()
		fmt.Println("Printout of buffer:")
		z, _ := v.BufferLines(buf, 0, -1, true)
		for _, vv := range z {
			fmt.Println(string(vv))
		}
		fmt.Println("--------------------------------")
	*/
	//v.Input("4OYoung Neil Young is the best\x1bdd")
	printBuffer(v, buf)
	inputString3(v, w, "5OYoung Neil Young is the best\x1b")
	//printBuffer(v, buf)
	//v.Input("gg")
	inputString3(v, w, "gg")
	inputString3(v, w, "i1234\x1bI9876\x1b")
	printBuffer(v, buf)
	inputString3(v, w, "v3l\x1b0V2j\x1b\x162j2k\x1b")
	//printBuffer(v, buf)

	//func (v *Nvim) Input(keys string) (written int, err error)
	_, err = v.Input("jcawSteve\x1b")
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	printBuffer(v, buf)
	//inputString(v, w, "jcawSuperBowl\x1b")
	v.Input("jcawSuperBowl\x1b")
	printBuffer(v, buf)

	_, err = v.Input("ggiJon\x1bggj^v4j")
	//written, err = v.Input("ggiJon\x1bggj^v4j\x1bgv")
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	//v.Input("\x1bgv") //I need to send this but may be a problem
	fmt.Println(highlightInfo(v))
	v.Input("3k")
	fmt.Println(highlightInfo(v))
	v.Input("2j")
	fmt.Println(highlightInfo(v))

	/* the idea is that by coming out of visual mode for a microsecond that the
	position of the last visual mode is available so imagine this:

	^v - note that my client goes into visual mode (can check but not sure necessary that buffer is in visual mode
	4 - don't do anything since its a number
	j - non-number issue v.Input("\x1bgv") and highlightInfo(v)
	j - non-number issue v.Input("\x1bgv") and highlightInfo(v)
	k - non-number issue v.Input("\x1bgv") and highlightInfo(v)
	need to recognize that x takes you out of visual mode
	just send x and get buffer
	*/

	//fmt.Printf("chars written = %v\n", written)

	/*
		z, _ = v.BufferLines(buf, 0, 2, true)
		for _, vv := range z {
			fmt.Println(string(vv))
		}
	*/
	/*
		//func (v *Nvim) WindowCursor(window Window) (pos [2]int, err error)
		pos, err := v.WindowCursor(w)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		fmt.Printf("pos = %v\n", pos)

		err = v.FeedKeys("GoNeil Young is the best\x1b", "t", true)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		_, err = v.Input("GoNeil Young is the best\x1b")
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		printBuffer(v, buf)
	*/

	for {
		time.Sleep(10 * time.Millisecond)
	}

}

func highlightInfo(v *nvim.Nvim) [2][4]int {
	var bufnum, lnum, col, off int
	var z [2][4]int
	v.Input("\x1bgv") //I need to send this but may be a problem

	err := v.Eval("getpos(\"'<\")", []*int{&bufnum, &lnum, &col, &off})
	if err != nil {
		fmt.Printf("getpos error: %v", err)
	}
	//fmt.Printf("beginning: bufnum = %v; lnum = %v; col = %v; off = %v\n", bufnum, lnum, col, off)
	z[0] = [4]int{bufnum, lnum, col, off}

	err = v.Eval("getpos(\"'>\")", []*int{&bufnum, &lnum, &col, &off})
	if err != nil {
		fmt.Printf("getpos error: %v\n", err)
	}
	//fmt.Printf("end: bufnum = %v; lnum = %v; col = %v; off = %v\n", bufnum, lnum, col, off)
	z[1] = [4]int{bufnum, lnum, col, off}

	return z
}

func printBuffer(v *nvim.Nvim, b nvim.Buffer) {
	fmt.Println()
	fmt.Println("--------------------------------")
	//func (v *Nvim) BufferLines(buffer Buffer, start int, end int, strictIndexing bool)
	//                      (lines [][]byte, err error)
	z, _ := v.BufferLines(b, 0, -1, true)
	for _, vv := range z {
		fmt.Println(string(vv))
	}
	fmt.Println("--------------------------------")
}

func inputString(v *nvim.Nvim, w nvim.Window, s string) {
	for _, c := range s {
		v.Input(string(c))
		mode, _ := v.Mode() //status msg and branch if v
		if mode.Mode == string('v') || mode.Mode == string('V') || mode.Mode == string('\x16') {
			fmt.Printf("visual mode %v: %v\n", mode.Mode, highlightInfo(v))
		}
		pos, _ := v.WindowCursor(w) //set screen cx and cy from pos
		fmt.Printf("char = %v => mode = %v => cursor pos = %v\n", string(c), mode.Mode, pos)
	}
}

func inputString2(v *nvim.Nvim, w nvim.Window, s string) {
	for _, c := range s {
		v.Input(string(c))
		mode, _ := v.Mode() //status msg and branch if v
		if mode.Mode == string('v') || mode.Mode == string('V') || mode.Mode == string('\x16') {
			fmt.Printf("visual mode %v: %v\n", mode.Mode, highlightInfo(v))
		}
		fmt.Printf("char = %v => mode = %v; blocking = %v", string(c), mode.Mode, mode.Blocking)
		if mode.Blocking == false {
			pos, _ := v.WindowCursor(w) //set screen cx and cy from pos
			fmt.Printf(" => position = %v\n", pos)
		} else {
			fmt.Printf("\n")
		}
	}
}

func inputString3(v *nvim.Nvim, w nvim.Window, s string) {
	var pos [2]int
	for _, c := range s {
		v.Input(string(c))
		mode, _ := v.Mode() //status msg and branch if v
		if mode.Blocking == false {
			pos, _ = v.WindowCursor(w) //set screen cx and cy from pos
			if mode.Mode == string('v') || mode.Mode == string('V') || mode.Mode == string('\x16') {
				fmt.Printf("visual mode %v: %v\n", mode.Mode, highlightInfo(v))
			}
		} else {
			pos = [2]int{-1, -1}
		}
		fmt.Printf("char = %v => mode = %v; blocking = %v => pos = %v\n", string(c), mode.Mode, mode.Blocking, pos)
	}

	/* text of the current line - not sure why
	line, _ := v.CurrentLine()
	fmt.Printf("line = %v", string(line))
	*/
}

/*
type Mode struct {
	// Mode is the current mode.
	Mode string `msgpack:"mode"`

	// Blocking is true if Nvim is waiting for input.
	Blocking bool `msgpack:"blocking"`
}
*/

/*
	for n := 1; n < 20; n++ {
		var buf bytes.Buffer
		r := NewBufferReader(v, b)
		_, err := io.CopyBuffer(struct{ io.Writer }{&buf}, r, make([]byte, n))
		if err != nil {
			fmt.Printf("copy %q with buffer size %d returned error %v\n", d, n, err)
			continue
		}
		fmt.Printf(buf.String())
		continue
	}
*/
