package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	//	_ "image/jpg"
	//	"github.com/disintegration/imaging"
	"image/png"
	"io"
	"os"
)

const (
	KITTY_IMG_HDR = "\x1b_G"
	KITTY_IMG_FTR = "\x1b\\"
)

// NOTE: uses $TERM, which is overwritten by tmux
func IsTermKitty() bool {

	V := GetEnvIdentifiers()
	return V["TERM"] == "xterm-kitty"
}

/*
Encode image using the Kitty terminal graphics protocol:
https://sw.kovidgoyal.net/kitty/graphics-protocol.html
*/
func KittyWriteImage(out io.Writer, iImg image.Image) error {

	pBuf := new(bytes.Buffer)
	if err := png.Encode(pBuf, iImg); err != nil {
		return err
	}

	return KittyCopyPNGInline(out, pBuf, int64(pBuf.Len()))
}

// NOTE: Encode raw PNG data into Kitty terminal format
func KittyCopyPNGInline(out io.Writer, in io.Reader, nLen int64) (E error) {

	OSC_OPEN, OSC_CLOSE := KITTY_IMG_HDR, KITTY_IMG_FTR

	// LAST CHUNK SIGNAL `m=0` TO KITTY
	defer func() {

		if E == nil {
			out.Write([]byte(OSC_OPEN))
			out.Write([]byte("m=0;"))
			_, E = out.Write([]byte(OSC_CLOSE))
		}
	}()

	// PIPELINE: PNG -> B64 -> CHUNKER -> out io.Writer
	// SEND IN 4K CHUNKS
	oWC := NewWriteChunker(out, 4096)
	defer oWC.Flush()
	//fmt.Printf("\x1b[%d;%dH", 10, 65) //y, x
	bsHdr := []byte(fmt.Sprintf("a=T,f=100,z=-1,S=%d,", nLen))
	//bsHdr := []byte(fmt.Sprintf("a=T,f=100,z=-1,w=300,h=200,x=300,y=200,S=%d,", nLen))
	oWC.CustomWriFunc = func(iWri io.Writer, bsDat []byte) (int, error) {

		parts := [][]byte{
			[]byte(OSC_OPEN),
			bsHdr,
			[]byte("m=1;"),
			bsDat,
			[]byte(OSC_CLOSE),
		}

		bsHdr = nil

		return iWri.Write(bytes.Join(parts, nil))
	}

	enc64 := base64.NewEncoder(base64.StdEncoding, &oWC)
	defer enc64.Close()

	_, E = io.Copy(enc64, in)
	return
}

func displayImage(path string) image.Rectangle {
	iImg, _, err := loadImage(path)
	/*
		if iImg.Bounds().Max.Y > 600 {
			iImg = imaging.Resize(iImg, 0, 600, imaging.Lanczos)
		}
	*/

	if err != nil {
		sess.showOrgMessage("Error loading image: %v", err)
	}
	err = KittyWriteImage(os.Stdout, iImg)
	if err != nil {
		sess.showOrgMessage("Error writing image: %v", err)
	}
	sess.showOrgMessage("bounds = %v", iImg.Bounds())
	return iImg.Bounds()
}

func displayImage2(img image.Image) {
	/*
		if iImg.Bounds().Max.Y > 600 {
			iImg = imaging.Resize(iImg, 0, 600, imaging.Lanczos)
		}
	*/

	err := KittyWriteImage(os.Stdout, img)
	if err != nil {
		sess.showOrgMessage("Error writing image: %v", err)
	}
	sess.showOrgMessage("bounds = %v", img.Bounds())
}
