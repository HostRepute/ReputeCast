package media_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/media"
)

// Why an upload was refused reads in the uploader's language; API clients
// and members who read English get English.
func TestUploadProblemsInVietnamese(t *testing.T) {
	env := apptest.New(t)
	linh := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	ws := env.CreateWorkspace("Acme Newsletter", linh.ID)
	marcus := env.CreateUser(apptest.UserOpts{Email: "marcus@acme.io", Name: "Marcus Chen"})
	env.AddMember(ws, marcus.ID, "admin")
	// Linh reads Vietnamese (uploads are multipart, so the app's language
	// comes from her own choice here rather than Accept-Language).
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE users SET language = 'vi' WHERE id = $1`, linh.ID); err != nil {
		t.Fatal(err)
	}
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	en := env.Client()
	en.SignInDemo("marcus@acme.io")

	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	p := upload(t, vi, "logo.svg", svg).MustStatus(t, 422).Problem(t)
	if p["detail"] != "Hình ảnh SVG không hiển thị trong Gmail hoặc Outlook. Hãy tải lên tệp PNG, JPEG hoặc GIF." {
		t.Fatalf("svg, in Vietnamese: %v", p)
	}
	small := pngBytes(t, 4, 4, false)
	big := append(small, bytes.Repeat([]byte{0}, media.MaxUploadBytes+1-len(small))...)
	p = upload(t, vi, "huge.png", big).MustStatus(t, 422).Problem(t)
	if p["detail"] != "Tệp nặng 20,1 MB; giới hạn là 20 MB. Hãy xuất hình ảnh nhỏ hơn rồi thử lại." {
		t.Fatalf("too large, in Vietnamese: %v", p)
	}
	// GIFs are kept as uploaded, so their limit is the stored one.
	gif := append([]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00"), bytes.Repeat([]byte{0}, media.MaxBytes)...)
	p = upload(t, vi, "huge.gif", gif).MustStatus(t, 422).Problem(t)
	if p["detail"] != "Ảnh GIF nặng 5,1 MB; ảnh GIF tối đa 5 MB vì được lưu nguyên như khi tải lên để giữ đủ các khung hình động." {
		t.Fatalf("GIF too large, in Vietnamese: %v", p)
	}
	missing := "/workspaces/acme-newsletter/media/0192e8f0-0000-7000-8000-000000000000"
	p = vi.Do("DELETE", missing, nil).MustStatus(t, 404).Problem(t)
	if p["detail"] != "Không tìm thấy hình ảnh." || p["code"] != "not_found" {
		t.Fatalf("not found, in Vietnamese: %v", p)
	}

	p = upload(t, en, "logo.svg", svg).MustStatus(t, 422).Problem(t)
	if p["detail"] != "SVG images don't display in Gmail or Outlook. Upload a PNG, JPEG or GIF." {
		t.Fatalf("svg, in English: %v", p)
	}
	p = upload(t, en, "huge.png", big).MustStatus(t, 422).Problem(t)
	if p["detail"] != "The file is 20.1 MB; the limit is 20 MB. Export the image smaller and retry." {
		t.Fatalf("too large, in English: %v", p)
	}
	p = upload(t, en, "huge.gif", gif).MustStatus(t, 422).Problem(t)
	if p["detail"] != "The GIF is 5.1 MB; GIFs can be up to 5 MB because they're stored as uploaded, so animations keep their frames." {
		t.Fatalf("GIF too large, in English: %v", p)
	}
	p = en.Do("DELETE", missing, nil).MustStatus(t, 404).Problem(t)
	if p["detail"] != "Image not found." {
		t.Fatalf("not found, in English: %v", p)
	}
}
