package webpush

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

// Message encryption (RFC 8291) in the aes128gcm content coding (RFC 8188).
//
// The application server makes a fresh ECDH key pair and a random 16-byte
// salt for every message. The ECDH secret with the browser's key, mixed
// with the browser's authentication secret, gives the input keying
// material; the salt turns it into a 16-byte AES-GCM key and a 12-byte
// nonce. The body is a header (salt, record size, the server's public key)
// followed by one record: the payload, the 0x02 delimiter, and the tag.

const (
	// RecordSize is the record size written in the header. Push services
	// accept 4096-byte bodies (RFC 8030 §7.2), so one record carries the
	// whole message.
	RecordSize = 4096
	// headerLen is salt (16) + record size (4) + key id length (1) + the
	// server's public key (65).
	headerLen = 16 + 4 + 1 + 65
	tagLen    = 16
	// MaxPayload is the most plaintext one message carries.
	MaxPayload = RecordSize - headerLen - tagLen - 1
)

// Errors from encrypting and decrypting.
var (
	ErrTooLarge = fmt.Errorf("webpush: the payload is over %d bytes", MaxPayload)
	ErrBody     = errors.New("webpush: the body isn't aes128gcm content")
)

// Encrypt encrypts plaintext for a subscription with a fresh key pair and
// salt. The result is the request body, sent with Content-Encoding:
// aes128gcm.
func Encrypt(sub Subscription, plaintext []byte) ([]byte, error) {
	as, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return encrypt(sub.P256dh, sub.Auth, as, salt, plaintext)
}

// encrypt is Encrypt with the server's key and the salt given, so the
// tests can reproduce RFC 8291's example.
func encrypt(uaPublic, authSecret []byte, as *ecdh.PrivateKey, salt, plaintext []byte) ([]byte, error) {
	if len(plaintext) > MaxPayload {
		return nil, ErrTooLarge
	}
	if len(salt) != 16 || len(authSecret) != 16 {
		return nil, ErrKeys
	}
	ua, err := ecdh.P256().NewPublicKey(uaPublic)
	if err != nil {
		return nil, ErrKeys
	}
	secret, err := as.ECDH(ua)
	if err != nil {
		return nil, err
	}
	asPublic := as.PublicKey().Bytes()
	ikm, err := inputKey(secret, authSecret, uaPublic, asPublic)
	if err != nil {
		return nil, err
	}
	aead, nonce, err := contentKeys(ikm, salt)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, headerLen+len(plaintext)+1+tagLen)
	out = append(out, salt...)
	out = binary.BigEndian.AppendUint32(out, RecordSize)
	out = append(out, byte(len(asPublic)))
	out = append(out, asPublic...)
	record := make([]byte, 0, len(plaintext)+1)
	record = append(record, plaintext...)
	record = append(record, 0x02) // the last (and only) record's delimiter
	return aead.Seal(out, nonce, record, nil), nil
}

// inputKey combines the ECDH secret and the authentication secret into the
// 32-byte input keying material (RFC 8291 §3.3–3.4).
func inputKey(ecdhSecret, authSecret, uaPublic, asPublic []byte) ([]byte, error) {
	prk, err := hkdf.Extract(sha256.New, ecdhSecret, authSecret)
	if err != nil {
		return nil, err
	}
	info := make([]byte, 0, 14+len(uaPublic)+len(asPublic))
	info = append(info, "WebPush: info\x00"...)
	info = append(info, uaPublic...)
	info = append(info, asPublic...)
	return hkdf.Expand(sha256.New, prk, string(info), 32)
}

// contentKeys derives the content encryption key and the nonce from the
// input keying material and the salt (RFC 8188 §2.2–2.3).
func contentKeys(ikm, salt []byte) (cipher.AEAD, []byte, error) {
	prk, err := hkdf.Extract(sha256.New, ikm, salt)
	if err != nil {
		return nil, nil, err
	}
	cek, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, nil, err
	}
	nonce, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, nil, err
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	return aead, nonce, nil
}

// header is an aes128gcm header.
type header struct {
	salt  []byte
	rs    int
	keyID []byte
	body  []byte
}

func parseHeader(body []byte) (header, error) {
	if len(body) < 21 {
		return header{}, ErrBody
	}
	h := header{salt: body[:16], rs: int(binary.BigEndian.Uint32(body[16:20]))}
	idlen := int(body[20])
	if len(body) < 21+idlen || h.rs < 18 {
		return header{}, ErrBody
	}
	h.keyID = body[21 : 21+idlen]
	h.body = body[21+idlen:]
	return h, nil
}

// Decrypt reads a Web Push message as the browser does, with the
// subscription's private key and authentication secret. The in-process push
// service uses it to check what it receives.
func Decrypt(ua *ecdh.PrivateKey, authSecret, body []byte) ([]byte, error) {
	h, err := parseHeader(body)
	if err != nil {
		return nil, err
	}
	as, err := ecdh.P256().NewPublicKey(h.keyID)
	if err != nil {
		return nil, fmt.Errorf("%w: the key id isn't a P-256 key", ErrBody)
	}
	secret, err := ua.ECDH(as)
	if err != nil {
		return nil, err
	}
	ikm, err := inputKey(secret, authSecret, ua.PublicKey().Bytes(), h.keyID)
	if err != nil {
		return nil, err
	}
	return decryptRecords(ikm, h)
}

// decryptContent decrypts aes128gcm content with known input keying
// material (RFC 8188's own examples).
func decryptContent(ikm, body []byte) ([]byte, error) {
	h, err := parseHeader(body)
	if err != nil {
		return nil, err
	}
	return decryptRecords(ikm, h)
}

// decryptRecords opens each record with the nonce XORed with its sequence
// number and strips the delimiter and padding.
func decryptRecords(ikm []byte, h header) ([]byte, error) {
	aead, nonce, err := contentKeys(ikm, h.salt)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	rest := h.body
	for seq := uint64(0); len(rest) > 0; seq++ {
		n := min(h.rs, len(rest))
		if n <= tagLen {
			return nil, ErrBody
		}
		iv := make([]byte, len(nonce))
		copy(iv, nonce)
		var s [8]byte
		binary.BigEndian.PutUint64(s[:], seq)
		for i := range 8 {
			iv[len(iv)-8+i] ^= s[i]
		}
		plain, err := aead.Open(nil, iv, rest[:n], nil)
		if err != nil {
			return nil, fmt.Errorf("%w: record %d doesn't authenticate", ErrBody, seq)
		}
		last := n == len(rest)
		end := len(plain) - 1
		for end >= 0 && plain[end] == 0 {
			end--
		}
		if end < 0 {
			return nil, fmt.Errorf("%w: record %d has no delimiter", ErrBody, seq)
		}
		switch {
		case last && plain[end] != 0x02, !last && plain[end] != 0x01:
			return nil, fmt.Errorf("%w: record %d has the wrong delimiter", ErrBody, seq)
		}
		out.Write(plain[:end])
		rest = rest[n:]
	}
	return out.Bytes(), nil
}
