package countdown

import (
	"bytes"
	"compress/lzw"
	"image"
	"image/color"
	"sync"
)

// A small GIF89a writer for the countdown's frames. image/gif needs every
// frame as its own image and starts a new LZW coder for each one; this one
// writes the frames straight from one canvas with a reused coder, so an
// image costs its bytes and little else. It writes a global palette, no
// local ones, no transparency and no loop extension: the animation plays
// once and stays on its last frame.

// gifWriter writes one GIF; its scratch (the canvas, the frames' plan) is
// kept for the next one.
type gifWriter struct {
	buf     bytes.Buffer
	blocks  blockWriter
	lzw     *lzw.Writer
	litBits int
	canvas  image.Paletted
	steps   []step
}

// canvasOf is the writer's canvas, a copy of base.
func (g *gifWriter) canvasOf(base *image.Paletted) *image.Paletted {
	if cap(g.canvas.Pix) < len(base.Pix) {
		g.canvas.Pix = make([]uint8, len(base.Pix))
	}
	g.canvas.Pix = g.canvas.Pix[:len(base.Pix)]
	copy(g.canvas.Pix, base.Pix)
	g.canvas.Stride, g.canvas.Rect, g.canvas.Palette = base.Stride, base.Rect, base.Palette
	return &g.canvas
}

var writers = sync.Pool{New: func() any { return &gifWriter{} }}

// WriteByte and Flush make blockWriter what compress/lzw writes to without
// a buffer of its own.
func (w *blockWriter) WriteByte(c byte) error {
	w.b[w.n] = c
	w.n++
	if w.n == len(w.b) {
		w.flush()
	}
	return nil
}

// Flush writes the pending sub-block.
func (w *blockWriter) Flush() error {
	w.flush()
	return nil
}

// blockWriter cuts LZW output into GIF data sub-blocks of 255 bytes.
type blockWriter struct {
	out *bytes.Buffer
	n   int
	b   [255]byte
}

func (w *blockWriter) Write(p []byte) (int, error) {
	total := len(p)
	for len(p) > 0 {
		c := copy(w.b[w.n:], p)
		w.n += c
		p = p[c:]
		if w.n == len(w.b) {
			w.flush()
		}
	}
	return total, nil
}

func (w *blockWriter) flush() {
	if w.n == 0 {
		return
	}
	w.out.WriteByte(byte(w.n))
	w.out.Write(w.b[:w.n])
	w.n = 0
}

func le16(b *bytes.Buffer, v int) { b.WriteByte(byte(v)); b.WriteByte(byte(v >> 8)) }

// start writes the header, the screen and the global palette.
func (g *gifWriter) start(w, h int, pal color.Palette) {
	g.buf.Reset()
	g.blocks.out = &g.buf
	// The table holds 2^(bits) colours, at least four.
	bits := 2
	for 1<<bits < len(pal) {
		bits++
	}
	g.litBits = bits
	g.buf.WriteString("GIF89a")
	le16(&g.buf, w)
	le16(&g.buf, h)
	g.buf.WriteByte(0x80 | 0x70 | byte(bits-1)) // global table, 8-bit colour resolution
	g.buf.WriteByte(0)                          // background: the first colour
	g.buf.WriteByte(0)                          // square pixels
	for i := range 1 << bits {
		var r, gr, b uint32
		if i < len(pal) {
			r, gr, b, _ = pal[i].RGBA()
		}
		g.buf.WriteByte(byte(r >> 8))
		g.buf.WriteByte(byte(gr >> 8))
		g.buf.WriteByte(byte(b >> 8))
	}
}

// frame writes a rectangle of src shown for delay hundredths of a second,
// kept under the next frame (disposal: none).
func (g *gifWriter) frame(src *image.Paletted, r image.Rectangle, delay int) {
	g.buf.Write([]byte{0x21, 0xf9, 0x04, 0x04}) // graphic control: disposal none
	le16(&g.buf, delay)
	g.buf.Write([]byte{0x00, 0x00})
	g.buf.WriteByte(0x2c) // image descriptor
	le16(&g.buf, r.Min.X)
	le16(&g.buf, r.Min.Y)
	le16(&g.buf, r.Dx())
	le16(&g.buf, r.Dy())
	g.buf.WriteByte(0x00) // no local table, not interlaced
	g.buf.WriteByte(byte(g.litBits))
	if g.lzw == nil {
		g.lzw = lzw.NewWriter(&g.blocks, lzw.LSB, g.litBits).(*lzw.Writer)
	} else {
		g.lzw.Reset(&g.blocks, lzw.LSB, g.litBits)
	}
	if r == src.Rect && r.Dx() == src.Stride {
		_, _ = g.lzw.Write(src.Pix)
	} else {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			o := (y-src.Rect.Min.Y)*src.Stride + r.Min.X - src.Rect.Min.X
			_, _ = g.lzw.Write(src.Pix[o : o+r.Dx()])
		}
	}
	_ = g.lzw.Close()
	g.blocks.flush()
	g.buf.WriteByte(0x00) // end of the image data
}

// finish ends the GIF and returns a copy of its bytes.
func (g *gifWriter) finish() []byte {
	g.buf.WriteByte(0x3b)
	return bytes.Clone(g.buf.Bytes())
}
