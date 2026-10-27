package media

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
)

// Uploads are stored the way email needs them (review 2026-10, CO-12): at
// most MaxWidth pixels wide (keeping their proportions, never enlarged),
// photos turned upright and compressed, and metadata removed: EXIF with its
// location and camera details, XMP, IPTC, comments, embedded thumbnails and
// the extra pictures some phones append. Colour profiles stay, so colours
// don't shift. GIFs keep their frames (an animation would not survive a
// resize); only their comments go.

const (
	// MaxWidth is the widest image stored: twice the 600 px email column,
	// so images stay sharp on high-density screens.
	MaxWidth = 1200
	// MaxUploadBytes is the largest file accepted for upload (20 MB, a
	// phone photo). What is stored is resized and compressed and must fit
	// MaxBytes.
	MaxUploadBytes = 20 << 20
	// jpegQuality is what photos are compressed to.
	jpegQuality = 82
	// reencodeFrom is the size below which an image that needs no resize
	// isn't re-encoded just to save bytes.
	reencodeFrom = 100 << 10
	// decodeBudget is the most memory one decoded image may take; larger
	// images are only stripped of their metadata.
	decodeBudget = 256 << 20
)

// optimizeSlots bounds how many uploads are decoded at once: a
// 50-megapixel PNG takes about 200 MB while it's resized.
var optimizeSlots = make(chan struct{}, 1)

// optimized is what gets stored for an upload.
type optimized struct {
	data          []byte
	contentType   string
	width, height int
}

// optimize returns the bytes to store for a validated upload. err is set
// only when the file can't be read well enough to store it safely.
func optimize(contentType string, data []byte, width, height int) (optimized, error) {
	switch contentType {
	case "image/jpeg":
		return optimizeJPEG(data, width, height)
	case "image/png":
		return optimizePNG(data, width, height)
	case "image/gif":
		return optimized{data: stripGIF(data), contentType: contentType, width: width, height: height}, nil
	}
	return optimized{data: data, contentType: contentType, width: width, height: height}, nil
}

// fit is the stored size: at most MaxWidth wide, same proportions.
func fit(w, h int) (int, int) {
	if w <= MaxWidth {
		return w, h
	}
	return MaxWidth, max(1, int(math.Round(float64(h)*MaxWidth/float64(w))))
}

var errUnreadable = ImageError("unreadable")

func optimizeJPEG(data []byte, w, h int) (optimized, error) {
	stripped, orientation, icc, ok := stripJPEG(data)
	turn := orientation >= 2 && orientation <= 8
	ow, oh := w, h
	if orientation >= 5 {
		ow, oh = h, w
	}
	resize := ow > MaxWidth
	keep := optimized{data: stripped, contentType: "image/jpeg", width: w, height: h}
	if ok && !turn && !resize && len(stripped) < reencodeFrom {
		return keep, nil
	}
	var img image.Image
	var err error = errUnreadable
	if w*h*4 <= decodeBudget {
		img, err = jpeg.Decode(bytes.NewReader(data))
	}
	if _, cmyk := img.(*image.CMYK); err == nil && cmyk && ok && !turn && !resize {
		// Print-ready CMYK: converting its colours here would shift them.
		return keep, nil
	}
	if err != nil {
		if !ok {
			// Neither re-encoded nor safely stripped: refuse rather than
			// store a location the uploader didn't mean to share.
			return optimized{}, errUnreadable
		}
		if turn {
			// Keep only the orientation, so clients still show it upright.
			keep.data = withOrientation(stripped, orientation)
		}
		return keep, nil
	}
	tw, th := fit(ow, oh)
	m := img
	if turn || resize {
		sw, sh := tw, th
		if orientation >= 5 {
			sw, sh = th, tw
		}
		m = orient(scale(img, sw, sh), orientation)
	}
	enc, err := encodeJPEG(m, icc)
	if err != nil {
		return optimized{}, err
	}
	if ok && !turn && !resize && len(enc) > len(stripped)*9/10 {
		// Already compressed: re-encoding would cost quality for little.
		return keep, nil
	}
	return optimized{data: enc, contentType: "image/jpeg", width: tw, height: th}, nil
}

func optimizePNG(data []byte, w, h int) (optimized, error) {
	stripped, animated, colour, ok := stripPNG(data)
	keep := optimized{data: stripped, contentType: "image/png", width: w, height: h}
	if ok && animated {
		// An animated PNG would lose its frames in a resize.
		return keep, nil
	}
	resize := w > MaxWidth
	if ok && !resize && len(stripped) < reencodeFrom {
		return keep, nil
	}
	var img image.Image
	var err error = errUnreadable
	// 16-bit PNGs decode to 8 bytes a pixel.
	if bpp := 4 + 4*btoi(len(data) > 24 && data[24] == 16); w*h*bpp <= decodeBudget {
		img, err = png.Decode(bytes.NewReader(data))
	}
	if err != nil {
		if !ok {
			return optimized{}, errUnreadable
		}
		return keep, nil
	}
	tw, th := fit(w, h)
	m := img
	if resize {
		m = scalePNG(img, tw, th)
	}
	best := keep
	if enc, err := encodePNG(m, colour); err != nil {
		return optimized{}, err
	} else if resize || !ok || len(enc) < len(stripped)*9/10 {
		best = optimized{data: enc, contentType: "image/png", width: tw, height: th}
	}
	// A photo saved as PNG is many times larger than the same photo as a
	// JPEG. Convert only opaque images that look like photos, and only when
	// the JPEG is well under half the size (screenshots and diagrams stay
	// sharp as PNG).
	if isOpaque(m) && looksLikePhoto(m) {
		enc, err := encodeJPEG(m, iccFromPNG(colour))
		if err == nil && len(enc) < len(best.data)*2/5 {
			return optimized{data: enc, contentType: "image/jpeg", width: tw, height: th}, nil
		}
	}
	return best, nil
}

// ── JPEG ───────────────────────────────────────────────────────────────

// stripJPEG drops metadata segments: EXIF and XMP (APP1), IPTC and
// Photoshop data (APP13), other application segments, comments, the JFIF
// thumbnail and anything after the end of the image (the extra pictures
// some phones append). JFIF density, colour profiles (APP2 ICC_PROFILE)
// and Adobe's colour transform (APP14) stay. It also returns the EXIF
// orientation and the colour profile's segments when that profile is RGB.
// ok is false when the file's structure couldn't be followed.
func stripJPEG(data []byte) (out []byte, orientation int, icc [][]byte, ok bool) {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return data, 0, nil, false
	}
	out = append(make([]byte, 0, len(data)), 0xFF, 0xD8)
	rgbProfile := false
	i := 2
	for i < len(data) {
		if data[i] != 0xFF {
			return data, 0, nil, false
		}
		for i < len(data) && data[i] == 0xFF { // fill bytes
			i++
		}
		if i >= len(data) {
			break
		}
		m := data[i]
		i++
		if m == 0xD9 { // end of image: nothing after it is kept
			return append(out, 0xFF, 0xD9), orientation, keepICC(icc, rgbProfile), true
		}
		if m == 0x01 || m >= 0xD0 && m <= 0xD7 { // markers without a length
			out = append(out, 0xFF, m)
			continue
		}
		if i+2 > len(data) {
			return data, 0, nil, false
		}
		n := int(data[i])<<8 | int(data[i+1])
		if n < 2 || i+n > len(data) {
			return data, 0, nil, false
		}
		seg := data[i : i+n] // length and payload
		payload := seg[2:]
		i += n
		switch {
		case m == 0xE0: // APP0: JFIF density only, without its thumbnail
			if len(payload) >= 14 && string(payload[:5]) == "JFIF\x00" {
				out = append(out, 0xFF, 0xE0, 0, 16)
				out = append(out, payload[:12]...)
				out = append(out, 0, 0)
			}
			continue
		case m == 0xE1: // APP1: EXIF or XMP
			if o := exifOrientation(payload); o != 0 && orientation == 0 {
				orientation = o
			}
			continue
		case m == 0xE2: // APP2: the colour profile stays, MPF and FlashPix go
			if len(payload) < 14 || string(payload[:12]) != "ICC_PROFILE\x00" {
				continue
			}
			if payload[12] == 1 && len(payload) >= 14+20 && string(payload[14+16:14+20]) == "RGB " {
				rgbProfile = true
			}
			icc = append(icc, append([]byte{0xFF, 0xE2}, seg...))
		case m == 0xEE: // APP14: Adobe's colour transform, needed to decode
		case m >= 0xE3 && m <= 0xEF, m == 0xFE: // other application data, comments
			continue
		}
		out = append(out, 0xFF, m)
		out = append(out, seg...)
		if m == 0xDA { // start of scan: copy the compressed data that follows
			j := i
			for j < len(data) {
				if data[j] != 0xFF {
					j++
					continue
				}
				if j+1 >= len(data) {
					j = len(data)
					break
				}
				if nx := data[j+1]; nx == 0x00 || nx >= 0xD0 && nx <= 0xD7 {
					j += 2
					continue
				}
				break
			}
			out = append(out, data[i:j]...)
			i = j
		}
	}
	// Cut short after the image data: close it the way it would be read.
	if bytes.Contains(out, []byte{0xFF, 0xDA}) {
		return append(out, 0xFF, 0xD9), orientation, keepICC(icc, rgbProfile), true
	}
	return data, 0, nil, false
}

func keepICC(icc [][]byte, rgb bool) [][]byte {
	if !rgb {
		return nil
	}
	return icc
}

// exifOrientation reads the Orientation tag (0x0112) of IFD0 from an APP1
// payload ("Exif\0\0" and a TIFF structure). 0 when absent or unreadable.
func exifOrientation(p []byte) int {
	if len(p) < 6+8 || string(p[:6]) != "Exif\x00\x00" {
		return 0
	}
	t := p[6:]
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	if bo.Uint16(t[2:4]) != 42 {
		return 0
	}
	off := bo.Uint32(t[4:8])
	if off < 8 || uint64(off)+2 > uint64(len(t)) {
		return 0
	}
	ifd := t[off:]
	count := int(bo.Uint16(ifd))
	for k := range count {
		e := 2 + k*12
		if e+12 > len(ifd) {
			return 0
		}
		if bo.Uint16(ifd[e:]) != 0x0112 {
			continue
		}
		if bo.Uint16(ifd[e+2:]) != 3 { // SHORT
			return 0
		}
		if v := int(bo.Uint16(ifd[e+8:])); v >= 1 && v <= 8 {
			return v
		}
		return 0
	}
	return 0
}

// withOrientation adds an EXIF segment holding only the orientation, for a
// photo that couldn't be turned upright here.
func withOrientation(jpg []byte, o int) []byte {
	p := []byte("Exif\x00\x00MM\x00\x2a\x00\x00\x00\x08\x00\x01\x01\x12\x00\x03\x00\x00\x00\x01")
	p = append(p, 0, byte(o), 0, 0, 0, 0, 0, 0)
	seg := append([]byte{0xFF, 0xE1, byte((len(p) + 2) >> 8), byte(len(p) + 2)}, p...)
	out := make([]byte, 0, len(jpg)+len(seg))
	out = append(out, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

// encodeJPEG compresses an image as a JPEG, with the colour profile's
// segments right after the start of the image.
func encodeJPEG(m image.Image, icc [][]byte) ([]byte, error) {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, m, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	if len(icc) == 0 {
		return b.Bytes(), nil
	}
	enc := b.Bytes()
	out := make([]byte, 0, len(enc)+len(icc)*600)
	out = append(out, enc[:2]...)
	for _, s := range icc {
		out = append(out, s...)
	}
	return append(out, enc[2:]...), nil
}

// iccSegments splits an ICC profile into APP2 segments.
func iccSegments(profile []byte) [][]byte {
	const per = 65535 - 2 - 14
	n := (len(profile) + per - 1) / per
	if n == 0 || n > 255 {
		return nil
	}
	segs := make([][]byte, 0, n)
	for k := range n {
		part := profile[k*per : min(len(profile), (k+1)*per)]
		l := 2 + 14 + len(part)
		s := append([]byte{0xFF, 0xE2, byte(l >> 8), byte(l)}, "ICC_PROFILE\x00"...)
		s = append(s, byte(k+1), byte(n))
		segs = append(segs, append(s, part...))
	}
	return segs
}

// ── PNG ────────────────────────────────────────────────────────────────

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// stripPNG keeps the chunks that draw the image (critical chunks,
// transparency, colour, pixel density, background, animation) and drops
// text, time, EXIF and anything unknown. colour holds the colour chunks
// (iCCP, sRGB, gAMA, cHRM, cICP) to carry over when the image is
// re-encoded. ok is false when the chunks couldn't be followed.
func stripPNG(data []byte) (out []byte, animated bool, colour [][]byte, ok bool) {
	if !bytes.HasPrefix(data, pngSignature) {
		return data, false, nil, false
	}
	out = append(make([]byte, 0, len(data)), pngSignature...)
	i := len(pngSignature)
	for i+12 <= len(data) {
		n := binary.BigEndian.Uint32(data[i:])
		if uint64(n) > uint64(len(data)-i-12) {
			return data, false, nil, false
		}
		end := i + 12 + int(n)
		chunk := data[i:end]
		typ := string(data[i+4 : i+8])
		i = end
		switch typ {
		case "IHDR", "PLTE", "IDAT", "tRNS", "pHYs", "bKGD", "sBIT", "fcTL", "fdAT":
		case "acTL":
			animated = true
		case "iCCP", "sRGB", "gAMA", "cHRM", "cICP":
			colour = append(colour, chunk)
		case "IEND":
			return append(out, chunk...), animated, colour, true
		default:
			if typ[0]&0x20 != 0 { // ancillary: metadata
				continue
			}
		}
		out = append(out, chunk...)
	}
	return data, false, nil, false
}

// encodePNG writes an image as PNG at the best compression, with the
// original's colour chunks after the header.
func encodePNG(m image.Image, colour [][]byte) ([]byte, error) {
	var b bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&b, m); err != nil {
		return nil, err
	}
	enc := b.Bytes()
	const afterIHDR = 8 + 12 + 13
	if len(colour) == 0 || len(enc) < afterIHDR || string(enc[12:16]) != "IHDR" {
		return enc, nil
	}
	out := make([]byte, 0, len(enc)+256)
	out = append(out, enc[:afterIHDR]...)
	for _, c := range colour {
		out = append(out, c...)
	}
	return append(out, enc[afterIHDR:]...), nil
}

// iccFromPNG turns a PNG's RGB colour profile (iCCP) into JPEG segments.
func iccFromPNG(colour [][]byte) [][]byte {
	for _, c := range colour {
		if string(c[4:8]) != "iCCP" {
			continue
		}
		body := c[8 : len(c)-4]
		nul := bytes.IndexByte(body, 0)
		if nul < 1 || nul+2 > len(body) || body[nul+1] != 0 {
			return nil
		}
		zr, err := zlib.NewReader(bytes.NewReader(body[nul+2:]))
		if err != nil {
			return nil
		}
		profile, err := io.ReadAll(io.LimitReader(zr, 4<<20))
		if err != nil || len(profile) < 20 || string(profile[16:20]) != "RGB " {
			return nil
		}
		return iccSegments(profile)
	}
	return nil
}

// ── GIF ────────────────────────────────────────────────────────────────

// stripGIF drops comments and application data other than looping
// (NETSCAPE2.0, ANIMEXTS1.0), and anything after the trailer. Frames and
// their timing stay. A file whose blocks can't be followed is kept as is.
func stripGIF(data []byte) []byte {
	if len(data) < 13 || (string(data[:6]) != "GIF87a" && string(data[:6]) != "GIF89a") {
		return data
	}
	i := 13
	if data[10]&0x80 != 0 {
		i += 3 << (int(data[10]&7) + 1)
	}
	if i > len(data) {
		return data
	}
	out := append(make([]byte, 0, len(data)), data[:i]...)
	// subBlocks returns the end of a run of data sub-blocks starting at j.
	subBlocks := func(j int) int {
		for j < len(data) {
			n := int(data[j])
			j++
			if n == 0 {
				return j
			}
			j += n
		}
		return -1
	}
	for i < len(data) {
		switch data[i] {
		case 0x3B: // trailer
			return append(out, 0x3B)
		case 0x21: // extension
			if i+2 > len(data) {
				return data
			}
			end := subBlocks(i + 2)
			if end < 0 {
				return data
			}
			label := data[i+1]
			keep := label == 0xF9 || label == 0x01 // graphic control, plain text
			if label == 0xFF && i+3+11 <= end {
				id := string(data[i+3 : i+3+11])
				keep = data[i+2] == 11 && (id == "NETSCAPE2.0" || id == "ANIMEXTS1.0")
			}
			if keep {
				out = append(out, data[i:end]...)
			}
			i = end
		case 0x2C: // image
			j := i + 10
			if j > len(data) {
				return data
			}
			if data[i+9]&0x80 != 0 {
				j += 3 << (int(data[i+9]&7) + 1)
			}
			if j+1 > len(data) {
				return data
			}
			end := subBlocks(j + 1) // after the LZW code size
			if end < 0 {
				return data
			}
			out = append(out, data[i:end]...)
			i = end
		default:
			return data
		}
	}
	return data
}

// ── Pixels ─────────────────────────────────────────────────────────────

// span is the part of the source one output pixel covers along an axis:
// from the source pixel first, with each source pixel's share.
type span struct {
	first int
	w     []float32
}

// spans divides src pixels among dst pixels (dst ≤ src): each output pixel
// covers src/dst source pixels, partly covering those at its edges.
func spans(src, dst int) []span {
	out := make([]span, dst)
	step := float64(src) / float64(dst)
	for i := range dst {
		lo, hi := float64(i)*step, float64(i+1)*step
		j0, j1 := int(lo), min(src, int(math.Ceil(hi)))
		sp := span{first: j0, w: make([]float32, 0, j1-j0)}
		for j := j0; j < j1; j++ {
			sp.w = append(sp.w, float32((math.Min(hi, float64(j+1))-math.Max(lo, float64(j)))/step))
		}
		out[i] = sp
	}
	return out
}

// scale resizes an image down to w×h by area averaging: each output pixel
// is the mean of the source area it covers. Colours are averaged
// premultiplied, so transparent pixels don't darken the edges of logos.
func scale(src image.Image, w, h int) *image.RGBA {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	w, h = min(w, sw), min(h, sh)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xs, ys := spans(sw, w), spans(sh, h)
	row := make([]float32, sw*4)
	hrow := make([]float32, w*4)
	acc := make([]float32, w*4)
	read := rowReader(src)
	for y, sy := range ys {
		clear(acc)
		for k, wy := range sy.w {
			read(sy.first+k, row)
			for x, sx := range xs {
				var r, g, bl, a float32
				o := sx.first * 4
				for _, wx := range sx.w {
					r += row[o] * wx
					g += row[o+1] * wx
					bl += row[o+2] * wx
					a += row[o+3] * wx
					o += 4
				}
				hrow[x*4], hrow[x*4+1], hrow[x*4+2], hrow[x*4+3] = r, g, bl, a
			}
			for x := range acc {
				acc[x] += hrow[x] * wy
			}
		}
		px := dst.Pix[y*dst.Stride : y*dst.Stride+w*4]
		for x := range px {
			px[x] = uint8(min(255, max(0, acc[x]+0.5)))
		}
	}
	return dst
}

// rowReader returns a function that reads source row y (from the bounds'
// top) as premultiplied RGBA floats in 0–255, with fast paths for the
// types the decoders return.
func rowReader(src image.Image) func(y int, row []float32) {
	b := src.Bounds()
	w := b.Dx()
	switch s := src.(type) {
	case *image.YCbCr:
		return func(y int, row []float32) {
			for x := range w {
				yi := s.YOffset(b.Min.X+x, b.Min.Y+y)
				ci := s.COffset(b.Min.X+x, b.Min.Y+y)
				r, g, bl := color.YCbCrToRGB(s.Y[yi], s.Cb[ci], s.Cr[ci])
				row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = float32(r), float32(g), float32(bl), 255
			}
		}
	case *image.Gray:
		return func(y int, row []float32) {
			p := s.Pix[s.PixOffset(b.Min.X, b.Min.Y+y):]
			for x := range w {
				v := float32(p[x])
				row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = v, v, v, 255
			}
		}
	case *image.RGBA:
		return func(y int, row []float32) {
			p := s.Pix[s.PixOffset(b.Min.X, b.Min.Y+y):]
			for x := range w * 4 {
				row[x] = float32(p[x])
			}
		}
	case *image.NRGBA:
		return func(y int, row []float32) {
			p := s.Pix[s.PixOffset(b.Min.X, b.Min.Y+y):]
			for x := range w {
				a := float32(p[x*4+3])
				row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = float32(p[x*4])*a/255, float32(p[x*4+1])*a/255, float32(p[x*4+2])*a/255, a
			}
		}
	case *image.Paletted:
		pal := make([][4]float32, len(s.Palette))
		for i, c := range s.Palette {
			r, g, bl, a := c.RGBA()
			pal[i] = [4]float32{float32(r >> 8), float32(g >> 8), float32(bl >> 8), float32(a >> 8)}
		}
		return func(y int, row []float32) {
			p := s.Pix[s.PixOffset(b.Min.X, b.Min.Y+y):]
			for x := range w {
				var c [4]float32
				if int(p[x]) < len(pal) {
					c = pal[p[x]]
				}
				row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = c[0], c[1], c[2], c[3]
			}
		}
	}
	return func(y int, row []float32) {
		for x := range w {
			r, g, bl, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = float32(r)/257, float32(g)/257, float32(bl)/257, float32(a)/257
		}
	}
}

// scalePNG resizes a PNG, keeping a palette image paletted (logos and
// diagrams stay small) by mapping each pixel to its nearest palette colour.
func scalePNG(src image.Image, w, h int) image.Image {
	m := scale(src, w, h)
	p, ok := src.(*image.Paletted)
	if !ok || len(p.Palette) == 0 {
		return m
	}
	out := image.NewPaletted(m.Bounds(), p.Palette)
	seen := map[uint32]uint8{}
	for i := 0; i+3 < len(m.Pix); i += 4 {
		key := uint32(m.Pix[i])<<24 | uint32(m.Pix[i+1])<<16 | uint32(m.Pix[i+2])<<8 | uint32(m.Pix[i+3])
		idx, ok := seen[key]
		if !ok {
			idx = uint8(p.Palette.Index(color.RGBA{m.Pix[i], m.Pix[i+1], m.Pix[i+2], m.Pix[i+3]}))
			seen[key] = idx
		}
		out.Pix[i/4] = idx
	}
	return out
}

// orient turns an image upright from its EXIF orientation (2–8).
func orient(m *image.RGBA, o int) *image.RGBA {
	if o < 2 || o > 8 {
		return m
	}
	w, h := m.Rect.Dx(), m.Rect.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	out := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for dy := range dh {
		for dx := range dw {
			var sx, sy int
			switch o {
			case 2: // mirrored
				sx, sy = w-1-dx, dy
			case 3: // upside down
				sx, sy = w-1-dx, h-1-dy
			case 4: // mirrored upside down
				sx, sy = dx, h-1-dy
			case 5: // mirrored, turned left
				sx, sy = dy, dx
			case 6: // turned left: turn right
				sx, sy = dy, h-1-dx
			case 7: // mirrored, turned right
				sx, sy = w-1-dy, h-1-dx
			case 8: // turned right: turn left
				sx, sy = w-1-dy, dx
			}
			copy(out.Pix[dy*out.Stride+dx*4:dy*out.Stride+dx*4+4], m.Pix[sy*m.Stride+sx*4:sy*m.Stride+sx*4+4])
		}
	}
	return out
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func isOpaque(m image.Image) bool {
	if o, ok := m.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	return false
}

// looksLikePhoto guesses whether an image is a photograph (many colours)
// rather than a screenshot, diagram or logo (few), from a sample of pixels.
func looksLikePhoto(m image.Image) bool {
	b := m.Bounds()
	step := max(1, b.Dx()*b.Dy()/4000)
	seen := map[uint32]bool{}
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i++
			if i%step != 0 {
				continue
			}
			r, g, bl, _ := m.At(x, y).RGBA()
			seen[(r>>11)<<10|(g>>11)<<5|bl>>11] = true
			if len(seen) > 600 {
				return true
			}
		}
	}
	return false
}
