package media

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math/rand/v2"
	"testing"
)

// CO-12: uploads are stored resized, upright, compressed and without
// their metadata.

func segment(marker byte, payload []byte) []byte {
	n := len(payload) + 2
	return append([]byte{0xFF, marker, byte(n >> 8), byte(n)}, payload...)
}

// exifSegment is an APP1 EXIF segment with an orientation and a text tag
// standing in for the location and camera details phones write.
func exifSegment(orientation int, secret string) []byte {
	t := []byte("MM\x00\x2a\x00\x00\x00\x08")
	t = append(t, 0, 2)
	t = append(t, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, byte(orientation), 0, 0)
	text := append([]byte(secret), 0)
	t = append(t, 0x01, 0x0E, 0, 2)
	t = binary.BigEndian.AppendUint32(t, uint32(len(text)))
	t = binary.BigEndian.AppendUint32(t, uint32(8+2+2*12+4))
	t = append(t, 0, 0, 0, 0)
	t = append(t, text...)
	return segment(0xE1, append([]byte("Exif\x00\x00"), t...))
}

// rgbProfile is the start of an RGB ICC profile (enough for its header).
func rgbProfile() []byte {
	p := make([]byte, 132)
	binary.BigEndian.PutUint32(p, 132)
	copy(p[12:], "mntr")
	copy(p[16:], "RGB ")
	copy(p[20:], "XYZ ")
	copy(p[36:], "acsp")
	return p
}

// withSegments puts segments right after a JPEG's start of image and data
// after its end.
func withSegments(jpg []byte, trailer string, segs ...[]byte) []byte {
	out := append([]byte{}, jpg[:2]...)
	for _, s := range segs {
		out = append(out, s...)
	}
	out = append(out, jpg[2:]...)
	return append(out, trailer...)
}

// halves is an image with its left half red and its right half blue.
func halves(w, h int) *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			c := color.RGBA{220, 30, 40, 255}
			if x >= w/2 {
				c = color.RGBA{30, 60, 220, 255}
			}
			m.SetRGBA(x, y, c)
		}
	}
	return m
}

// noise is an opaque image of random pixels.
func noise(w, h int) *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewPCG(1, 2))
	for i := range m.Pix {
		m.Pix[i] = uint8(r.IntN(256))
		if i%4 == 3 {
			m.Pix[i] = 255
		}
	}
	return m
}

func encJPEG(t *testing.T, m image.Image, q int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, m, &jpeg.Options{Quality: q}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func encPNG(t *testing.T, m image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, m); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func pngChunk(typ string, body []byte) []byte {
	c := binary.BigEndian.AppendUint32(nil, uint32(len(body)))
	c = append(c, typ...)
	c = append(c, body...)
	return binary.BigEndian.AppendUint32(c, crc32.ChecksumIEEE(c[4:]))
}

// afterIHDR inserts chunks after a PNG's header.
func afterIHDR(p []byte, chunks ...[]byte) []byte {
	out := append([]byte{}, p[:33]...)
	for _, c := range chunks {
		out = append(out, c...)
	}
	return append(out, p[33:]...)
}

func isRed(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r>>8 > 180 && g>>8 < 90 && b>>8 < 90
}

func isBlue(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return b>>8 > 180 && r>>8 < 90 && g>>8 < 110
}

func TestStripJPEG(t *testing.T) {
	icc := segment(0xE2, append([]byte("ICC_PROFILE\x00\x01\x01"), rgbProfile()...))
	jfif := segment(0xE0, append([]byte("JFIF\x00\x01\x02\x01\x00\x48\x00\x48\x02\x01"), bytes.Repeat([]byte{0x7F}, 6)...))
	src := withSegments(encJPEG(t, halves(64, 32), 90), "MPF second picture: SECRET-TRAILER",
		jfif,
		exifSegment(6, "GPS 10.7769 N 106.7009 E SECRET-EXIF"),
		segment(0xE1, []byte("http://ns.adobe.com/xap/1.0/\x00<x:xmpmeta>SECRET-XMP</x:xmpmeta>")),
		icc,
		segment(0xE2, []byte("MPF\x00SECRET-MPF")),
		segment(0xED, []byte("Photoshop 3.0\x00SECRET-IPTC")),
		segment(0xFE, []byte("SECRET-COMMENT")),
	)
	out, orientation, profile, ok := stripJPEG(src)
	if !ok || orientation != 6 {
		t.Fatalf("strip: ok %v, orientation %d", ok, orientation)
	}
	if bytes.Contains(out, []byte("SECRET")) {
		t.Fatalf("metadata left: %q", out[:200])
	}
	if !bytes.Contains(out, []byte("ICC_PROFILE")) || len(profile) != 1 {
		t.Fatalf("the colour profile must stay: %d segments", len(profile))
	}
	if !bytes.Contains(out, []byte("JFIF\x00")) || bytes.Contains(out, []byte{0x7F, 0x7F, 0x7F}) {
		t.Fatal("JFIF stays, without its thumbnail")
	}
	if !bytes.HasSuffix(out, []byte{0xFF, 0xD9}) {
		t.Fatal("nothing after the end of the image")
	}
	m, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil || m.Bounds().Dx() != 64 || m.Bounds().Dy() != 32 {
		t.Fatalf("stripped JPEG must decode: %v", err)
	}
	if want, _ := jpeg.Decode(bytes.NewReader(src)); !bytes.Equal(want.(*image.YCbCr).Y, m.(*image.YCbCr).Y) {
		t.Fatal("stripping must not touch the pixels")
	}
}

func TestOptimizeJPEGTurnsUprightAndResizes(t *testing.T) {
	// A phone photo held upright: stored sideways (3000×1500) with
	// orientation 6, so it shows as 1500×3000.
	src := withSegments(encJPEG(t, halves(3000, 1500), 92), "", exifSegment(6, "SECRET-GPS"))
	o, err := optimize("image/jpeg", src, 3000, 1500)
	if err != nil {
		t.Fatal(err)
	}
	if o.contentType != "image/jpeg" || o.width != 1200 || o.height != 2400 {
		t.Fatalf("stored %s %d×%d, want an upright 1200×2400 JPEG", o.contentType, o.width, o.height)
	}
	if bytes.Contains(o.data, []byte("Exif")) || bytes.Contains(o.data, []byte("SECRET")) {
		t.Fatal("EXIF must be gone once the photo is upright")
	}
	m, err := jpeg.Decode(bytes.NewReader(o.data))
	if err != nil || m.Bounds().Dx() != 1200 || m.Bounds().Dy() != 2400 {
		t.Fatalf("decode: %v %v", err, m.Bounds())
	}
	// Turned right: the left (red) half is now on top.
	if !isRed(m.At(600, 400)) || !isBlue(m.At(600, 2000)) {
		t.Fatalf("not upright: top %v, bottom %v", m.At(600, 400), m.At(600, 2000))
	}
	if len(o.data) >= len(src) {
		t.Fatalf("not smaller: %d ≥ %d", len(o.data), len(src))
	}
}

func TestOptimizeJPEGKeepsWhatIsAlreadySmall(t *testing.T) {
	// A small, already compressed JPEG is only stripped: its pixels stay.
	src := withSegments(encJPEG(t, halves(400, 200), 70), "", exifSegment(1, "SECRET"))
	o, err := optimize("image/jpeg", src, 400, 200)
	if err != nil || o.width != 400 || o.height != 200 {
		t.Fatalf("%v %d×%d", err, o.width, o.height)
	}
	stripped, _, _, _ := stripJPEG(src)
	if !bytes.Equal(o.data, stripped) {
		t.Fatal("a small JPEG must only lose its metadata")
	}
	// A large, lightly compressed one is compressed.
	big := encJPEG(t, noise(1000, 800), 100)
	o, err = optimize("image/jpeg", big, 1000, 800)
	if err != nil || len(o.data) >= len(big)*9/10 || o.width != 1000 {
		t.Fatalf("compress: %v %d of %d", err, len(o.data), len(big))
	}
}

func TestOptimizeJPEGThatCantBeDecodedKeepsOrientation(t *testing.T) {
	// Image data Go can't decode (cut short) still loses its metadata and
	// keeps only its orientation, so clients still show it upright.
	full := encJPEG(t, noise(256, 256), 90)
	cut := withSegments(full[:len(full)/2], "", exifSegment(8, "SECRET"))
	o, err := optimize("image/jpeg", cut, 256, 256)
	if err != nil {
		t.Fatal(err)
	}
	// The APP1 right after the start of image holds the orientation only.
	seg := o.data[2:]
	if seg[0] != 0xFF || seg[1] != 0xE1 || exifOrientation(seg[4:4+(int(seg[2])<<8|int(seg[3]))-2]) != 8 {
		t.Fatalf("orientation segment: % x", seg[:16])
	}
	if bytes.Contains(o.data, []byte("SECRET")) {
		t.Fatal("metadata left")
	}
}

func TestOptimizePNG(t *testing.T) {
	t.Run("metadata", func(t *testing.T) {
		logo := image.NewNRGBA(image.Rect(0, 0, 200, 80))
		for i := 3; i < len(logo.Pix); i += 4 {
			logo.Pix[i-3], logo.Pix[i] = 15, 255
		}
		src := afterIHDR(encPNG(t, logo),
			pngChunk("tEXt", []byte("Author\x00SECRET-AUTHOR")),
			pngChunk("eXIf", []byte("MM\x00\x2aSECRET-EXIF")),
			pngChunk("tIME", []byte{0x07, 0xEA, 10, 25, 9, 40, 0}),
			pngChunk("sRGB", []byte{0}),
			pngChunk("iDOT", []byte("SECRET-APPLE")),
		)
		o, err := optimize("image/png", src, 200, 80)
		if err != nil || o.contentType != "image/png" || o.width != 200 {
			t.Fatalf("%v %s %d", err, o.contentType, o.width)
		}
		if bytes.Contains(o.data, []byte("SECRET")) || bytes.Contains(o.data, []byte("tIME")) {
			t.Fatal("metadata left")
		}
		if !bytes.Contains(o.data, []byte("sRGB")) {
			t.Fatal("colour chunks stay")
		}
		if _, err := png.Decode(bytes.NewReader(o.data)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("wide logo keeps clean transparent edges", func(t *testing.T) {
		// Transparent (black, alpha 0) on the left, white on the right.
		m := image.NewNRGBA(image.Rect(0, 0, 2400, 40))
		for y := range 40 {
			for x := 1201; x < 2400; x++ {
				m.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
		o, err := optimize("image/png", encPNG(t, m), 2400, 40)
		if err != nil || o.contentType != "image/png" || o.width != 1200 || o.height != 20 {
			t.Fatalf("%v %s %d×%d", err, o.contentType, o.width, o.height)
		}
		d, err := png.Decode(bytes.NewReader(o.data))
		if err != nil {
			t.Fatal(err)
		}
		edge := color.NRGBAModel.Convert(d.At(600, 10)).(color.NRGBA)
		if edge.A < 100 || edge.A > 160 || edge.R < 250 {
			t.Fatalf("edge pixel %v: half transparent white, not grey", edge)
		}
		if !hasAlpha(o.contentType, o.data, image.Config{}) {
			t.Fatal("transparency stays")
		}
	})

	t.Run("palette stays a palette", func(t *testing.T) {
		m := image.NewPaletted(image.Rect(0, 0, 1600, 400), palette.WebSafe)
		for i := range m.Pix {
			m.Pix[i] = uint8((i / 400) % 4 * 30)
		}
		o, err := optimize("image/png", encPNG(t, m), 1600, 400)
		if err != nil || o.width != 1200 || o.height != 300 {
			t.Fatalf("%v %d×%d", err, o.width, o.height)
		}
		d, err := png.Decode(bytes.NewReader(o.data))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := d.(*image.Paletted); !ok {
			t.Fatalf("stored as %T", d)
		}
	})

	t.Run("photo saved as PNG becomes a JPEG", func(t *testing.T) {
		m := image.NewNRGBA(image.Rect(0, 0, 1600, 1000))
		r := rand.New(rand.NewPCG(3, 4))
		for y := range 1000 {
			for x := range 1600 {
				m.SetNRGBA(x, y, color.NRGBA{uint8(x / 7), uint8(y / 4), uint8(r.IntN(60) + x%97), 255})
			}
		}
		src := encPNG(t, m)
		o, err := optimize("image/png", src, 1600, 1000)
		if err != nil || o.contentType != "image/jpeg" || o.width != 1200 || o.height != 750 {
			t.Fatalf("%v %s %d×%d", err, o.contentType, o.width, o.height)
		}
		if len(o.data) > len(src)/4 {
			t.Fatalf("%d of %d", len(o.data), len(src))
		}
	})

	t.Run("animated PNG keeps its frames", func(t *testing.T) {
		src := afterIHDR(encPNG(t, halves(1600, 100)), pngChunk("acTL", []byte{0, 0, 0, 1, 0, 0, 0, 0}))
		o, err := optimize("image/png", src, 1600, 100)
		if err != nil || o.width != 1600 || !bytes.Contains(o.data, []byte("acTL")) {
			t.Fatalf("%v %d", err, o.width)
		}
	})

	t.Run("damaged", func(t *testing.T) {
		src := encPNG(t, halves(1600, 100))
		if _, err := optimize("image/png", src[:len(src)/2], 1600, 100); err == nil {
			t.Fatal("a cut-short PNG can't be stored safely")
		}
	})
}

func TestStripGIF(t *testing.T) {
	frames := &gif.GIF{LoopCount: 0}
	for i := range 2 {
		p := image.NewPaletted(image.Rect(0, 0, 20, 10), palette.Plan9)
		for j := range p.Pix {
			p.Pix[j] = uint8(i*40 + j%3)
		}
		frames.Image = append(frames.Image, p)
		frames.Delay = append(frames.Delay, 50)
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, frames); err != nil {
		t.Fatal(err)
	}
	src := b.Bytes()
	// A comment and XMP before the first frame, junk after the trailer.
	at := bytes.Index(src, []byte{0x21, 0xF9})
	extra := []byte{0x21, 0xFE, 14}
	extra = append(extra, "SECRET-COMMENT"...)
	extra = append(extra, 0, 0x21, 0xFF, 11)
	extra = append(extra, "XMP DataXMP"...)
	extra = append(extra, 10)
	extra = append(extra, "SECRET-XMP"...)
	extra = append(extra, 0)
	src = append(append(append(append([]byte{}, src[:at]...), extra...), src[at:]...), "SECRET-TRAILER"...)

	out := stripGIF(src)
	if bytes.Contains(out, []byte("SECRET")) {
		t.Fatalf("metadata left")
	}
	if !bytes.Contains(out, []byte("NETSCAPE2.0")) {
		t.Fatal("looping stays")
	}
	g, err := gif.DecodeAll(bytes.NewReader(out))
	if err != nil || len(g.Image) != 2 || g.Delay[1] != 50 {
		t.Fatalf("frames: %v", err)
	}
	if got := stripGIF([]byte("GIF89a\x01")); !bytes.Equal(got, []byte("GIF89a\x01")) {
		t.Fatal("unreadable GIFs stay as they are")
	}
}

func TestScaleAveragesArea(t *testing.T) {
	// 3 → 2 pixels: each output pixel covers one and a half source pixels.
	m := image.NewGray(image.Rect(0, 0, 3, 1))
	m.Pix = []uint8{0, 255, 0}
	d := scale(m, 2, 1)
	if d.Pix[0] != 85 || d.Pix[4] != 85 {
		t.Fatalf("got %v", d.Pix)
	}
	// Never enlarged.
	if d := scale(m, 6, 2); d.Rect.Dx() != 3 || d.Rect.Dy() != 1 {
		t.Fatalf("enlarged to %v", d.Rect)
	}
	if w, h := fit(4032, 3024); w != 1200 || h != 900 {
		t.Fatalf("fit %d×%d", w, h)
	}
	if w, h := fit(600, 3000); w != 600 || h != 3000 {
		t.Fatalf("fit %d×%d", w, h)
	}
}

func TestOrient(t *testing.T) {
	// A B C
	// D E F
	m := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for i := range 6 {
		m.Pix[i*4] = byte('A' + i)
	}
	read := func(o *image.RGBA) string {
		s := ""
		for y := range o.Rect.Dy() {
			if y > 0 {
				s += "/"
			}
			for x := range o.Rect.Dx() {
				s += string(rune(o.Pix[y*o.Stride+x*4]))
			}
		}
		return s
	}
	want := map[int]string{1: "ABC/DEF", 2: "CBA/FED", 3: "FED/CBA", 4: "DEF/ABC", 5: "AD/BE/CF", 6: "DA/EB/FC", 7: "FC/EB/DA", 8: "CF/BE/AD"}
	for o, w := range want {
		if got := read(orient(m, o)); got != w {
			t.Errorf("orientation %d: %s, want %s", o, got, w)
		}
	}
}

// The parsers read untrusted bytes: they must never panic or loop.
func FuzzStrip(f *testing.F) {
	f.Add([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x02, 0xFF, 0xD9})
	f.Add(append([]byte{0xFF, 0xD8}, exifSegment(6, "x")...))
	f.Add([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x00IEND\xaeB`\x82"))
	f.Add([]byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff\x21\xfe\x01x\x00\x3b"))
	f.Add([]byte("Exif\x00\x00MM\x00\x2a\xff\xff\xff\xf0"))
	f.Fuzz(func(t *testing.T, b []byte) {
		stripJPEG(b)
		stripPNG(b)
		stripGIF(b)
		exifOrientation(b)
	})
}
