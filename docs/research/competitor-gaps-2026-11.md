# ReputeCast so với đối thủ: các tính năng còn thiếu (nghiên cứu ngày 10/11/2026)

## Phạm vi và cách làm

- **Đối thủ đã xem: khoảng 45 sản phẩm, chia 5 nhóm.**
  - Creator/newsletter: beehiiv, Substack, Ghost 6, Kit, Buttondown, Flodesk, Kajabi.
  - Email marketing cho SMB và e‑commerce: Mailchimp, Brevo, MailerLite, Klaviyo, Omnisend, ActiveCampaign, GetResponse, HubSpot, Constant Contact.
  - Mã nguồn mở / self-hosted: listmonk, Mautic, Keila, Notifuse, Mailcoach, Sendy, BillionMail, Plunk, useSend, Dittofeed, Laudspeaker, SendPortal, Mailtrain, phpList, MailWizz, Acelle, Postal.
  - Developer, lifecycle, transactional: Customer.io, Loops, Resend, Postmark, Bento, Encharge, SendGrid, Mailgun, Amazon SES, Brevo transactional, MailerSend, Mailtrap, Lettermint, Knock, Novu, Courier, SuprSend, Braze, Iterable.
  - Thay đổi chung của ngành trong 2025–2026: Gmail, Yahoo, Microsoft, Apple, IETF, AI, kênh mới.
- **Nguồn:** changelog, help center, tài liệu và GitHub releases của từng hãng, cập nhật đến 11/2026. Danh sách link ở mục 8.
- **Đối chiếu với ReputeCast:** mỗi mục đều được kiểm tra trong code (`internal/`, `web/src/`, `migrations/`) và trong `docs/spec/`, không chỉ dựa vào README. Mục nào ReputeCast đã có thì bị loại; danh sách các mục bị loại ở mục 5.
- **Đối chiếu với HostRepute:** mỗi mục cũng được kiểm tra trong code của HostRepute (`app.hostrepute.com`: `app/`, `routes/`, `docs/features/`). Mục nào HostRepute đã có thì bị loại: người dùng dùng HostRepute cho tính năng đó, ReputeCast không build lại. Danh sách cũng ở mục 5.
- **Đối chiếu với ReputeCast Cloud:** mỗi mục cũng được kiểm tra trong code của ReputeCast Cloud (`ReputeCastCloud`, nhánh `c17-production`, commit `222be5a`: `app/`, `docs/spec/`, `docs/pr/SUMMARY.md`). Mục nào Cloud đã có thì bị loại: ai cần ReputeCast được host sẵn thì dùng Cloud, ReputeCast không build lại. Danh sách cũng ở mục 5.
- **Ký hiệu:**
  - Trạng thái ở ReputeCast: ❌ chưa có · 🟡 có một phần · 🚫 spec ghi rõ là ngoài phạm vi.
  - Ưu tiên: **P1** nên làm sớm · **P2** đáng làm · **P3** ngách hoặc để sau.
  - Độ khó: **S** vài ngày · **M** 1–3 tuần · **L** một milestone trở lên.
- **Lưu ý:**
  - Mức ưu tiên là đánh giá của tôi. Tôi dựa vào nhóm người dùng của ReputeCast (indie publisher, SaaS founder, team marketing nhỏ, agency) và các nguyên tắc của dự án: mọi tính năng miễn phí, self-hosted, giữ ranh giới với HostRepute.
  - Tính năng của đối thủ có thể khác nhau theo gói cước.

## Tóm tắt

1. **ReputeCast đã rộng hơn hầu hết đối thủ** ở deliverability (Send Readiness, guarded sending, routing nhiều provider), ở analytics trung thực và ở quản trị team. Các khoảng trống còn lại tập trung vào 6 nhóm:
   - **Dữ liệu khách hàng và e‑commerce:**
     - tracking hành vi trên website, custom objects, dữ liệu công ty (B2B);
     - gợi ý sản phẩm, dự đoán CLV và churn;
     - mã giảm giá riêng cho từng người nhận;
     - flow back-in-stock, price-drop, browse abandonment.
   - **Tăng trưởng danh sách:** exit intent, form nhiều bước, A/B test form, targeting, lead ads, Instagram comment-to-DM.
   - **Đo lường:**
     - báo cáo theo vị trí, thiết bị, email client, giờ trong ngày;
     - holdout group;
     - trình tạo báo cáo tùy chỉnh;
     - báo cáo định kỳ gửi qua email.
   - **Tính năng cho creator:**
     - tương tác và phân phối: bình luận, ActivityPub, bản audio/TTS, podcast;
     - kiếm tiền: group subscription, tạm dừng subscription, mạng lưới paid recommendation và quảng cáo, sponsor tự đặt chỗ;
     - chuyển nhà: import bài viết.
   - **Hạ tầng cho developer:** nhận mail (inbound), SDK client, sandbox bắt thư, bước wait-for-event, bước fetch, connected content, environments.
   - **AI agent:** mọi đối thủ lớn đều đã có agent tự lên kế hoạch và dựng campaign hoặc flow. ReputeCast mới có trợ lý cho từng tác vụ, cộng với MCP cho agent bên ngoài.
2. **Có 5 việc nên sửa ngay,** vì chuẩn hoặc nhà cung cấp đã thay đổi trong năm 2026, hoặc vì có rủi ro tuân thủ (mục 1).
3. **Đề xuất theo thứ tự làm** ở mục 7.

---

## 1. Cần sửa hoặc cập nhật ngay

### 1.1 Yêu cầu unsubscribe qua `mailto:` không được xử lý (S–M)

- **Vấn đề:**
  - Header List-Unsubscribe có thêm một địa chỉ `mailto:`, trỏ tới reply-to hoặc From. ReputeCast lại không nhận mail.
  - Vì vậy, yêu cầu hủy đăng ký gửi qua mailto rơi vào hộp thư của publisher và không ai suppress địa chỉ đó.
- **Vì sao quan trọng:**
  - Gmail Postmaster v2 giờ chấm riêng tiêu chí "Honor unsubscribe".
  - CAN-SPAM yêu cầu xử lý yêu cầu hủy trong vòng 10 ngày.
- **Bằng chứng trong code:**
  - Header được tạo ở `internal/sending/engine/loop.go:1248-1257`.
  - Địa chỉ mailto được mô tả ở `internal/readiness/facts.go:488`.
  - `internal/domains/service.go:187` ghi rõ "ReputeCast doesn't receive mail".
- **Đề xuất:**
  - Ngắn hạn (S): thêm tùy chọn bỏ `mailto:`, vì RFC 8058 chỉ cần HTTPS cộng `List-Unsubscribe-Post`. Hoặc để Send Readiness cảnh báo rằng mailto cần có người xử lý.
  - Dài hạn (M): xử lý mailto qua inbound (mục 3.1).

### 1.2 Chuẩn DMARC mới: RFC 9989, 9990, 9991 (tháng 5/2026) (S–M)

- **Thay đổi:** bộ RFC mới thay RFC 7489:
  - bỏ tag `pct=` và `ri=`;
  - thêm `t=` (test mode), `np=` (policy cho subdomain không tồn tại) và `psd=`;
  - tìm policy bằng DNS tree walk thay cho Public Suffix List.
- **Bằng chứng trong code:** `internal/dnscheck/auth.go:330-390` vẫn đọc `Percent` và tìm organizational domain theo RFC 7489 §6.6.3.
- **Đề xuất:**
  - Cập nhật bộ kiểm tra DMARC và DMARC được gợi ý.
  - Cho parser báo cáo DMARC aggregate đọc được định dạng RFC 9990.

### 1.3 Google Postmaster Tools v2 (S, phần việc phía ReputeCast)

- **Thay đổi:**
  - Domain Reputation và IP Reputation đã bị bỏ; API v1 đã ngừng.
  - Bản v2 có Compliance status gồm 11 yêu cầu (trong đó có one-click unsubscribe và honor unsubscribe), cùng một Deliverability verdict.
- **HostRepute đã có:** Postmaster v2 với Compliance status, Deliverability verdict và delivery error theo lý do (`app/Services/Postmaster/PostmasterSyncService.php` trong `app.hostrepute.com`). ReputeCast không dựng lại các màn hình này.
- **Bằng chứng trong code:** `internal/store/models.go:2819-2820` và tool MCP `get_deliverability` (`internal/mcpserver/tools.go:591`) vẫn trả về `domain_reputation` và `ip_reputation`.
- **Đề xuất:**
  - Bỏ, hoặc đánh dấu lỗi thời, hai trường cũ.
  - Chỗ đang hiện reputation thì trỏ sang trang domain Postmaster của HostRepute, theo kiểu touchpoint ở `reputecast-design-expansion-plan.md:75-78`.

### 1.4 Quy tắc bulk sender của Microsoft (Outlook.com, có hiệu lực từ 5/5/2025) (S)

- **Thay đổi:** domain gửi từ 5.000 thư/ngày phải đạt SPF, DKIM và DMARC alignment. Nếu không, thư bị từ chối với mã `550 5.7.515`.
- **Bằng chứng trong code:** Send Readiness mới chỉ có quy tắc của Gmail và Yahoo (`internal/readiness/assemble.go:1714-1717`).
- **Đề xuất:**
  - Thêm Microsoft vào dòng kiểm tra `compliance.bulk_sender`.
  - Nhận diện mã 5.7.515 khi phân loại bounce.

### 1.5 Gợi ý sửa lỗi gõ email trên form chưa được dùng (S)

- **Vấn đề:** ReputeCast đã có sẵn thư viện sửa lỗi gõ domain nhưng chưa đưa ra form. Kit thì hiện "Did you mean gmail.com?" ngay trên form.
- **Bằng chứng trong code:**
  - Hàm `TypoFix` ở `internal/emailaddr/emailaddr.go:318`.
  - Hàm này mới được dùng khi import và trong cảnh báo transactional; `internal/forms` chưa gọi.
- **Đề xuất:** hiện gợi ý trên form nhúng, popup và trang subscribe.

---

## 2. Tính năng ReputeCast đã có nhưng đối thủ làm sâu hơn

### 2.1 Composer và nội dung email

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Countdown timer trong email (GIF tạo lúc mở thư) | Omnisend, MailerLite; Sendtric, NiftyImages; Kit (qua app MILA, loopOffer) | ❌ Chỉ có countdown trên page (`internal/pages/blocks.go:20`) | P1 | M |
| Mã giảm giá riêng cho từng người nhận (tạo trên Shopify/Woo hoặc lấy từ pool) | Klaviyo, Mailchimp, Omnisend, GetResponse | 🟡 Có mã dùng một lần cho referral reward (upload CSV); offer là coupon Stripe dùng chung | P1 | M |
| Gmail Promotions annotations: deal badge, mã giảm giá, carousel (JSON-LD trong `<head>`) | Stripo, Salesforce MC; hướng dẫn của Litmus và Validity | ❌ Email không có JSON-LD | P2 | S |
| Thư viện ảnh stock và GIF (Unsplash, Pexels, Giphy) | Kit, MailerLite, beehiiv, Mailchimp (Shutterstock), Constant Contact (Getty) | ❌ | P2 | S |
| Trình sửa ảnh (crop, xoay, filter, chú thích) | beehiiv, Ghost, Kit | ❌ Chỉ tối ưu dung lượng ảnh | P2 | M |
| Sửa link sau khi đã gửi | Ghost, MailerLite | ❌ | P2 | S (link đã đi qua redirect có token) |
| Thư viện template lớn; import từ Stripo, Beefree, Figma, Canva | Stripo (hơn 1.650 mẫu), Mailchimp, Klaviyo (plugin Figma), MailerLite (Canva) | 🟡 7 template mẫu (`internal/templates/starters/`), import MJML/HTML | P2 | M |
| Kiểm tra tương thích email client theo dữ liệu caniemail | Resend, Mailtrap | ❌ Có render thật qua Litmus/Mailgun Inspect bằng key riêng | P2 | S |
| Template khóa bố cục, chỉ cho sửa các vùng được chỉ định | Keila, Mailcoach, Beefree SDK, Klaviyo | 🟡 Có synced block và brand kit | P2 | M |
| Connected content: gọi API lúc render để lấy dữ liệu | Braze, Iterable, Notifuse (data feeds) | ❌ Feed chỉ có RSS cho digest | P2 | M |
| Ảnh động hoặc ảnh cá nhân hóa tạo lúc mở thư (tồn kho, kết quả poll, tên người nhận) | Movable Ink, Litmus Personalize, NiftyImages | ❌ | P3 | M |
| Tạo ảnh bằng AI | Brevo, Klaviyo, ActiveCampaign, beehiiv | 🚫 `docs/spec/reputecast-design-addon-create.md:315` | P3 | S (dùng key riêng) |
| Nhiều người gõ cùng lúc trên một bản nháp | beehiiv, Kit, Resend Broadcasts | 🟡 Có presence, comment và xử lý xung đột autosave; chưa sửa đồng thời | P3 | L |
| Đính kèm file trong campaign | listmonk, phpList, Mautic | ❌ (transactional thì có) | P3 | S |
| Markdown thuần, đăng bài bằng cách gửi email, CLI soạn bài trong Git | Buttondown, Keila, Mailcoach | 🟡 Write mode có phím tắt Markdown | P3 | S |
| Block cho bài viết: toggle/accordion, footnote, Mermaid, LaTeX, code | Substack, Ghost, Buttondown | ❌ | P3 | S |
| Interactive email bằng CSS (carousel, accordion) | Stripo, Mailmodo | ❌ | P3 | M |
| AMP for Email | Gmail và Yahoo; SendGrid, Mailgun, SparkPost hỗ trợ gửi | ❌ | P3 | L |
| Video trong email (thumbnail kèm nút play, GIF từ video) | Stripo, HubSpot | ❌ | P3 | S |
| Spintax | BillionMail, Acelle | ❌ | P3 | S |

### 2.2 A/B testing

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| A/B test cho form và landing page | Klaviyo, MailerLite, Omnisend | ❌ với form; 🚫 với page (`docs/spec/reputecast-design-addon-publish-sell.md:448`) | P1 | M |
| Multivariate: kết hợp subject × người gửi × nội dung × giờ gửi, tối đa 8 tổ hợp | Mailchimp, MailerLite | 🟡 2–3 biến thể, mỗi lần một chiều (`internal/campaigns/composer.go:117-147`) | P2 | M |
| Bandit: AI sinh N subject rồi dồn dần lưu lượng sang biến thể thắng | Braze, Klaviyo, Jacquard | ❌ | P2 | M |
| Experiment nhiều bước trong automation, tự chọn nhánh thắng | Customer.io, Knock, Courier, Iterable | 🟡 Split tối đa 5 nhánh có trọng số, không tự chọn nhánh thắng | P2 | M |
| Thử 4–5 subject một lúc | Kit (5), beehiiv (4) | 🟡 Tối đa 3 | P3 | S |
| A/B cho SMS, push, WhatsApp | Mailchimp, Klaviyo | 🚫 `docs/spec/reputecast-design-addon-channels-agencies.md:478` | P3 | M |

### 2.3 Analytics và báo cáo

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Báo cáo theo quốc gia và thành phố (GeoIP) | Mailchimp, MailerLite, Sendy, Mautic, Acelle, beehiiv | ❌ Không dùng GeoIP ở đâu | P1 | S (MaxMind GeoLite hoặc DB-IP, tùy chọn) |
| Báo cáo theo thiết bị, hệ điều hành, email client | Mailchimp, Customer.io, Mautic, Acelle, Litmus | 🟡 Đã nhận diện client ở từng sự kiện (`internal/tracking/classify.go`) nhưng chưa có báo cáo tổng hợp | P1 | S |
| Holdout / global control group, đo mức tăng thực sự (incrementality) | Klaviyo, Braze, Iterable, Customer.io | ❌ Chỉ có holdout cho A/B và canary | P1 | M |
| Báo cáo số liệu gửi định kỳ qua email (tuần/tháng) cho tác giả và các bên liên quan | Buttondown, Klaviyo, HubSpot, Mautic, Lettermint | ❌ Đã có digest cho thông báo, nhưng không có digest số liệu | P1 | S |
| Lượt mở và click theo giờ trong ngày, theo thứ trong tuần | Mautic, Sendy | ❌ Biểu đồ chỉ tính theo giờ kể từ lúc gửi | P2 | S |
| Trình tạo báo cáo và dashboard tùy chỉnh | Mailchimp, MailerLite, Brevo, Klaviyo, Mautic, HubSpot | ❌ | P2 | M–L |
| So sánh kết quả từng segment trong cùng một lần gửi | Mailchimp | ❌ | P2 | S |
| Cohort mua hàng, CLV theo segment, phân tích theo sản phẩm và danh mục | Klaviyo, Brevo | 🟡 Có cohort theo growth source | P2 | M |
| Hỏi đáp số liệu trên toàn workspace | Mailchimp Analytics AI, ActiveCampaign, Klaviyo, Omnisend | 🟡 "Ask about this report" chỉ trong phạm vi một báo cáo | P2 | M |
| Web analytics đầy đủ: real-time, nguồn, UTM, vị trí, thiết bị, lọc free/paid | Ghost (Tinybird), beehiiv, Notifuse | 🟡 Chỉ đếm lượt xem và referrer của post/page (`internal/website/report.go`) | P2 | M |
| Xuất dữ liệu sang warehouse (S3, BigQuery, Snowflake), kết nối BI | Customer.io, Braze, SuprSend, Resend (Webhooks Ingester) | ❌ Có CSV và webhook | P2 | M |
| So sánh với kỳ trước theo khoảng ngày tùy chọn | beehiiv, MailerLite | 🟡 So với các số trước trong báo cáo campaign | P3 | S |
| Đo thời gian đọc (read/skim/glance), lượt in | Litmus, Yahoo Performance Feeds | ❌ (đã có số lượt forward) | P3 | M |
| Unsubscribe-to-open ratio, CTOR theo từng link | Mautic, Sendy | 🟡 Có CTOR tổng | P3 | S |
| Nhiều mô hình attribution (linear, time decay), cửa sổ theo kênh | Klaviyo, HubSpot, Brevo, Omnisend | 🚫 Chỉ last-click (`docs/spec/reputecast-design-addon-audience-insights.md:286`) | P3 | M |
| Benchmark theo ngành | Klaviyo, Mailchimp | ❌ Cần dữ liệu tập trung, khó làm với self-hosted | P3 | L |

### 2.4 Form và popup

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Hiện popup khi người đọc sắp rời trang (exit intent) | beehiiv, Brevo, Klaviyo, GetResponse, Mautic | ❌ Chỉ hiện sau một khoảng delay hoặc khi cuộn đến % trang (`internal/forms/embed/form.js:421-427`) | P1 | S |
| Targeting: theo URL, segment, vị trí, UTM; ẩn với người đã đăng ký | Klaviyo, Omnisend | ❌ Chỉ có danh sách site được phép | P1 | M |
| Form nhiều bước (hỏi sở thích trước, email sau) | Klaviyo, Omnisend (4 bước), Mautic | ❌ | P1 | M |
| Inbox deep link sau khi đăng ký (nút "Mở Gmail") | Ghost, Buttondown (báo tỷ lệ xác nhận tăng khoảng 6%) | ❌ | P1 | S |
| Thêm kiểu hiển thị: slide-in, thanh dính, teaser, toàn màn hình, hiệu ứng | beehiiv, Klaviyo, Omnisend, Mautic (Focus items) | 🟡 Có embedded, popup, hosted | P2 | S–M |
| Form dạng trò chơi (spin-to-win, cào thẻ) | Klaviyo, Omnisend | ❌ | P3 | M |
| Hẹn giờ bật/tắt form | Omnisend, Klaviyo | ❌ | P3 | S |
| Progressive profiling, kiosk mode, giới hạn số lượt gửi | Mautic, MailerLite (app iPad) | ❌ | P3 | M |
| Đăng ký bằng Google One Tap | các nền tảng cho người đọc | ❌ | P3 | S |
| SMS tap-to-text (mở app SMS với keyword điền sẵn) | Mailchimp, Klaviyo | ❌ | P3 | S |
| Quiz | Klaviyo, Octane AI | 🚫 `docs/spec/reputecast-design-addon-create.md:314` | P3 | M |

### 2.5 Automation

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Trigger theo vòng đời gói trả phí: nâng/hạ tier, bắt đầu subscription, kết thúc (win-back), trial sắp hết | Ghost (11/2026), beehiiv ("Upgraded"), Encharge, Loops | ❌ Đã có webhook `subscription.*` (`internal/earn/hooks.go:18`) nhưng chưa có trigger tương ứng; hiện chỉ làm vòng được qua segment theo `paid_status` | P1 | S |
| Trigger e‑commerce: browse abandonment, added to cart, back in stock, price drop, reorder, search abandonment | Klaviyo, Omnisend, Brevo, Mailchimp | ❌ Chỉ có order placed, first order, abandoned checkout; browse abandonment là 🚫 (`docs/spec/reputecast-design-addon-publish-sell.md:447`) | P1 | M |
| Bước wait-until-event có timeout | Customer.io, Knock, Resend, SuprSend, Dittofeed | 🟡 Branch theo event trong một khung thời gian, và bước goal | P2 | S |
| Bước fetch/HTTP: lấy dữ liệu rồi dùng ở các bước sau | Knock, Novu, Customer.io, SuprSend | 🟡 Bước webhook chỉ gửi đi, không lấy kết quả về | P2 | M |
| Bước gọi LLM và lưu kết quả vào field | Customer.io, Knock, Braze | ❌ | P2 | M |
| Bước throttle; bước batch/digest gom nhiều sự kiện thành một thư | Knock, Novu, SuprSend, Courier | ❌ (digest chỉ dành cho RSS) | P2 | M |
| Thoát flow khi người nhận trả lời email (stop-on-reply) | Notifuse, Mautic | ❌ Cần inbound (mục 3.1) | P2 | M |
| Export/import automation ra file để chuyển giữa các instance | Mautic 7, Dittofeed (qua Git) | ❌ | P2 | S |
| Gọi flow khác; hủy một lượt chạy theo key | SuprSend, Knock | ❌ | P3 | S |
| Ngày không gửi; hạn "không gửi sau ngày X" cho từng bước | Kajabi, Flodesk | 🟡 Có send window | P3 | S |
| Debug: chạy lại lượt lỗi, chạy thử một bước, xem sự kiện test trực tiếp | Knock, Resend, Courier | 🟡 Lượt lỗi được để riêng sau 5 lần thử; có test send | P3 | S |
| Bước chạy code tùy chỉnh (JS/Python) | Klaviyo | ❌ | P3 | M |
| Flow mẫu do AI đề xuất theo ngữ cảnh | Omnisend, Mailchimp | 🟡 Có 4 flow mẫu | P3 | S |
| Trigger từ Typeform, Calendly, Eventbrite, Patreon | Kit App Store | ❌ Làm được qua API event và Zapier | P3 | S mỗi app |

### 2.6 Audience và dữ liệu

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Đồng ý tracking riêng cho từng contact, theo CNIL/Garante: tách open và click, mặc định không track khi chưa rõ | Notifuse, Brevo, Klaviyo, Customer.io | ❌ Đã có retention và tắt tracking theo template transactional | P1 (cho EU) | M |
| Gộp contact thủ công, gợi ý contact trùng | Klaviyo, Mautic | 🟡 Mỗi email là một contact; import tự gộp trùng | P2 | M |
| Phân nhóm RFM (recency, frequency, monetary) | Klaviyo | ❌ | P2 | S (SQL trên dữ liệu store) |
| Nhiều địa chỉ email trên một profile | Klaviyo | ❌ | P3 | L |
| Làm giàu dữ liệu: ảnh, bio, công ty, mạng xã hội | Kit (Subscriber Signals), Mautic (Clearbit) | ❌ | P3 | M (cần dịch vụ ngoài) |
| Người phụ trách contact, ghi chú, lifecycle stage | Mautic, ActiveCampaign | ❌ | P3 | S |
| Trình xem event log: tìm, lọc, xuất mọi sự kiện | Brevo | 🟡 Có timeline cho từng contact | P3 | S |
| Segment tĩnh, thủ công, random bucket | beehiiv, Dittofeed | 🟡 Segment động và tag | P3 | S |

### 2.7 Gửi thư và deliverability

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Tự thu báo cáo DMARC aggregate (địa chỉ rua riêng, đọc qua IMAP) | Postmark DMARC Digests, MailerSend, Valimail | 🟡 Upload tay; HostRepute chưa thu (mục 5) | P2 | M |
| BIMI: kiểm tra chứng chỉ VMC/CMC và logo SVG Tiny-PS; checklist cho Apple Branded Mail | Red Sift, Mailgun, Resend | 🟡 Chỉ kiểm tra bản ghi DNS `default._bimi` (`internal/dnscheck/auth.go:410-440`) | P2 | S |
| Lưu media trên S3/R2 kèm CDN | listmonk, Notifuse, Plunk, Mautic | ❌ Media nằm trên data volume; khi tách `--role` ra nhiều node thì cần ổ dùng chung | P2 | S |
| Gửi rải đều theo phút trong một khung giờ | Klaviyo | 🟡 Có warm-up và throttle | P3 | S |
| Gửi từ hộp thư Gmail/M365 qua SMTP OAuth2 | Notifuse, Dittofeed, Keila | ❌ | P3 | M |
| Bắt buộc TLS theo route hoặc theo thư; tự xoay khóa DKIM | Lettermint, MailerSend | ❌ | P3 | S–M |
| Ẩn nội dung và PII theo dự án; tự chọn thời gian lưu | Lettermint, Mailgun, Postmark | 🟡 Transactional lưu được 0–90 ngày | P3 | S |
| Tự cấu hình DNS (Domain Connect, Entri) | Customer.io, Ghost, beehiiv | 🟡 Có Cloudflare API và xuất zone file | P3 | M |
| Header campaign ID cho Yahoo Performance Feeds | Yahoo | 🟡 Đã có Feedback-ID | P3 | S |

### 2.8 Transactional và developer

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| SDK client chính thức (Node, Python, Go, PHP, Ruby…) | Resend (8 ngôn ngữ), Postmark, Lettermint, Bento, useSend | 🟡 Chỉ có code mẫu; thư mục `sdk/` là SDK để viết adapter | P1 | M (sinh từ OpenAPI) |
| Sandbox bắt thư: hộp thư giả để xem HTML và header | Mailtrap, Postmark (sandbox server), Lettermint, Mailcoach | 🟡 Test key chỉ gửi cho thành viên và địa chỉ test | P2 | M |
| CLI cho API: gửi thư, quản lý tài nguyên từ terminal và CI | Resend, Loops, Mailgun, Lettermint, Courier | 🟡 CLI hiện chỉ dùng cho vận hành | P2 | S |
| Environments (dev → staging → prod), workflow-as-code | Knock, Novu, Courier, SuprSend, Dittofeed | ❌ | P2 | L |
| Giới hạn API key theo IP; ký request | Postmark, Lettermint, MailWizz, BillionMail | ❌ | P2 | S |
| Email hook cho Supabase Auth, Clerk, Auth0; mail driver cho Laravel, Symfony | Notifuse, Resend, Lettermint, Mailcoach | ❌ | P2 | S mỗi tích hợp |
| Template bằng React Email / jsx-email | Resend, useSend | ❌ Dùng MJML và import HTML | P3 | S (nhận HTML đã render sẵn) |
| Layout và partial cho template transactional | Postmark, Knock, Novu | 🟡 Có synced block | P3 | S |
| Schema cho event và sinh type | SuprSend, Resend | ❌ | P3 | S |
| Tự thu hồi key bị lộ qua GitHub secret scanning | Resend | ❌ | P3 | S |
| Terraform/OpenTofu provider | Lettermint, Mailtrap | ❌ | P3 | M |
| Bulk API có endpoint xem tiến độ | Postmark | 🟡 Batch tối đa 100 | P3 | S |
| Remote MCP qua OAuth, Agent Skills, llms.txt cho developer | Resend, Postmark, Loops, Mailgun | 🟡 MCP qua API key | P3 | S |

### 2.9 Website và xuất bản

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Import bài viết từ Substack, Ghost, WordPress, beehiiv (giữ slug, ảnh, redirect) | beehiiv, Ghost, Kit, Buttondown (chuyển nhà hộ) | ❌ Mới import được contact, paid member và template, chưa import bài viết | P1 | M |
| Nút share trên post; link share trong email | Ghost | ❌ Chỉ referral mới có share link | P2 | S |
| Tối ưu cho AI search: llms.txt, bản `.md` của mỗi bài, schema tùy chỉnh, đánh dấu paywall, kiểm soát crawler AI | Ghost, beehiiv, Buttondown | 🟡 Đã có JSON-LD (`internal/publicpages/site_post.go:230`), robots.txt, sitemap; chưa có llms.txt | P2 | S |
| Gift link: cho người không trả phí đọc một bài trả phí | Ghost, beehiiv | ❌ | P2 | S |
| Trang link-in-bio | beehiiv, Flodesk, Mailchimp | ❌ Block `bio` hiện là tiểu sử tác giả | P2 | S (dựng bằng page builder) |
| Thanh thông báo trên toàn site | Ghost, Buttondown | ❌ | P3 | S |
| Quản lý redirect (wildcard, công cụ kiểm tra) | beehiiv, Buttondown | 🟡 Tự redirect khi đổi slug | P3 | S |
| AI website builder | beehiiv | ❌ | P3 | M |
| Backlink nội bộ bằng `[[`; nhắc các chỗ "TK" còn sót | beehiiv, Ghost, Buttondown | ❌ | P3 | S |
| Dịch bài tự động cho người đọc | Substack | ❌ Đã có language version do publisher tự viết | P3 | S |
| Cài theme; sửa code theme ngay trên trình duyệt | Ghost | 🚫 `docs/spec/reputecast-design-addon-publish-sell.md:442` | P3 | L |

### 2.10 Earn (kiếm tiền)

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Group/team subscription: mua nhiều suất, có trang quản lý cho người mua, xuất hóa đơn | beehiiv, Substack | 🚫 `docs/spec/reputecast-design-addon-earn.md:395-399` | P1 | M |
| Sponsor tự đặt chỗ: tự mua slot qua Stripe, tự nộp creative, publisher duyệt | Buttondown, Substack (Creator Kits) | 🚫 `docs/spec/reputecast-design-addon-earn.md:395`; hiện payment của sponsor ghi tay | P1 | M |
| Tạm dừng thanh toán 1, 3 hoặc 6 tháng | beehiiv | 🚫 `docs/spec/reputecast-design-addon-earn.md:397` | P2 | S (Stripe `pause_collection`) |
| Thêm chu kỳ thanh toán: quý, trọn đời; tier donation | beehiiv, Ghost | ❌ Chỉ có tháng và năm | P2 | S |
| Tip/donation một lần | Ghost, Kit, beehiiv | ❌ (product pay-what-you-want dùng thay được một phần) | P2 | S |
| Media kit công khai | Substack | ❌ | P2 | S |
| Upsell, order bump, mua thử end-to-end | Kit, Kajabi, Flodesk | ❌ | P2 | M |
| Mã giảm giá và bundle cho digital product | Kit, beehiiv | 🚫 `docs/spec/reputecast-design-addon-publish-sell.md:444-445` | P2 | S |
| Bán thời gian/coaching kèm đặt lịch | beehiiv, MailerLite Bookings, Kajabi | ❌ | P2 | M |
| Nhiều loại tiền tệ; Apple Pay, Google Pay; tùy biến giao diện Stripe Checkout | Ghost (135 loại tiền), beehiiv, Kit | 🚫 Đa tiền tệ (`docs/spec/reputecast-design-addon-earn.md:397`); phương thức thanh toán do Stripe quyết định | P2 | S–M |
| Paddle, Lemon Squeezy (merchant of record) | Acelle, Encharge | 🟡 Có trong catalog nhưng hiện "Not available yet" (`internal/payments/catalog.go:3-10`) | P2 | M |
| Đồng bộ role Discord theo tier | beehiiv | ❌ | P2 | S |
| Founding member tier; pledge (cam kết trả trước khi bật thanh toán) | Substack | ❌ | P3 | S |
| Paid trial (trial có thu phí) | beehiiv | 🚫 `docs/spec/reputecast-review-2026-11.md:871` | P3 | S |
| Trang "Perks" cho paid member | Substack | ❌ | P3 | S |
| Feed podcast riêng tư theo tier (Transistor, Hello Audio) | Ghost, Kit | ❌ | P3 | S |

### 2.11 Grow (tăng trưởng)

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Đồng bộ lead ads (Meta, LinkedIn, TikTok, Google) | Mailchimp, Brevo, beehiiv, HubSpot | ❌ | P1 | M (cần Meta duyệt app) |
| Magic link: subscribe một chạm từ email của ESP khác (dùng merge tag của Mailchimp, Kit…) | beehiiv, SparkLoop | 🟡 Partner link chỉ chạy giữa các instance ReputeCast; trang subscribe điền sẵn được email | P2 | S |
| Instagram comment-to-DM để thu email | Klaviyo, Omnisend, ManyChat | ❌ | P2 | M |
| Meta Conversions API (gửi sự kiện đăng ký từ server) | beehiiv | ❌ | P2 | S |
| Widget Upscribe ngay sau khi đăng ký | SparkLoop, Kit, beehiiv | 🟡 Có "While you're here" trên trang xác nhận | P3 | S |
| Chế độ slot cho recommendation (xáo trộn, ghim, smart); nút "recommend back" | beehiiv, Kit | 🟡 Tối đa 3 slot, cố định | P3 | S |
| Giveaway/sweepstakes | Gleam, KingSumo | ❌ | P3 | M |
| QR code cho form và trang | Buttondown, Mailchimp | ❌ | P3 | S |

### 2.12 Commerce

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Tracking website (pixel hoặc GTM) cho site bất kỳ | Mailchimp, Klaviyo, Omnisend, Brevo | ❌ | P1 | M |
| Gợi ý sản phẩm: bán chạy, "vừa xem", collaborative filtering | Klaviyo, Omnisend, Brevo, Mailchimp, GetResponse | ❌ Chỉ có chọn sản phẩm cố định | P1 | M |
| Dự đoán CLV, nguy cơ churn, ngày mua tiếp theo | Klaviyo, Mailchimp, Braze | ❌ (`docs/spec/reputecast-design-addon-audience-insights.md:283`) | P2 | M |
| Thêm nền tảng: BigCommerce, Magento, PrestaShop, Wix, Squarespace | Klaviyo, Omnisend, Brevo | 🚫 `docs/spec/reputecast-design-addon-publish-sell.md:447` | P2 | M mỗi nền tảng |
| Review sản phẩm, loyalty, wallet pass | Klaviyo Reviews, Brevo (Loyalty, Wallet) | ❌ | P3 | L |

### 2.13 SMS, push, WhatsApp

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Hộp thư SMS hai chiều | Klaviyo, ActiveCampaign, Brevo | 🚫 `docs/spec/reputecast-design-addon-channels-agencies.md:474` | P2 | M |
| MMS, RCS | Klaviyo (RCS đã GA), Mautic (MMS) | 🚫 `docs/spec/reputecast-design-addon-channels-agencies.md:472` | P2 | M |
| Mobile push (APNs/FCM) và SDK | Klaviyo, Customer.io, Braze, Dittofeed | 🚫 `docs/spec/reputecast-design-addon-channels-agencies.md:479` | P2 | L |
| Tạo template WhatsApp ngay trong app; carousel, click-to-chat | ActiveCampaign, Klaviyo | 🚫 | P3 | M |
| Quy định TCPA mới, giờ yên lặng theo luật từng bang của Mỹ, kiểm tra tuổi | Klaviyo, Attentive | 🟡 Có quiet hours 08:00–21:00 | P3 | S |

### 2.14 AI

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| AI agent tự lên kế hoạch, dựng campaign/flow/segment, rồi chờ người duyệt | Klaviyo Composer, HubSpot, Customer.io, Iterable Nova, ActiveCampaign, Brevo Aura, beehiiv Agent | 🚫 "writing a whole campaign", "a chat window" (`docs/spec/reputecast-design-addon-create.md:315`); agent bên ngoài vào được qua MCP | P2 | L |
| Prompt-to-workflow: mô tả bằng lời thành automation | Brevo, MailerLite (qua MCP) | ❌ Đã có prompt-to-segment | P2 | M |
| Brand voice học từ các số đã gửi | HubSpot, Kit (HeyNews), Klaviyo | ❌ | P2 | S |
| Connector ChatGPT/Claude một chạm qua OAuth (có trong danh bạ connector) | Brevo, Klaviyo, Mailchimp, Omnisend, MailerLite, Kit, beehiiv | 🟡 MCP qua API key | P2 | M |
| Tự kiểm tra theo lịch: flow chồng nhau, flow kém hiệu quả | Klaviyo, ActiveCampaign 3.0 | ❌ | P3 | M |
| Xem trước bản tóm tắt AI mà Gmail/Apple sẽ hiện cho người nhận | mới xuất hiện | ❌ | P3 | S |

### 2.15 Team, bảo mật, quản trị instance

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| SAML SSO | Mautic, Mailchimp, Customer.io | 🚫 Chỉ có OIDC (`docs/spec/reputecast-design-addon-team-access.md:77`) | P2 | M |
| SCIM | Customer.io, Lettermint | 🚫 | P2 | M |
| Giới hạn IP cho đăng nhập và cho API | MailWizz, BillionMail, Postmark | ❌ Chỉ có allowlist domain email cho JIT SSO | P2 | S |
| White-label giao diện admin (logo, màu, domain riêng cho app) | Acelle, Sendy, MailWizz, Dittofeed EE | 🟡 Có tên và logo instance; tắt được "Sent with ReputeCast" trên public page | P2 | S |
| LDAP | Mailtrain | ❌ | P3 | M |
| Xác thực lại trước thao tác nhạy cảm (step-up) | Brevo | 🟡 Một số thao tác đã hỏi lại mật khẩu | P3 | S |
| Đăng nhập thay khách hàng để hỗ trợ (impersonate) | Acelle | ❌ | P3 | M |

### 2.16 Tích hợp và hệ sinh thái

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Tự gắn UTM cho link trong email; tích hợp GA4 | hầu hết ESP | 🟡 Đã tự gắn UTM cho SMS, push, sponsor, store; link email mới chỉ được kiểm tra (`internal/readiness/rules.go:84`) | P1 | S |
| Slack/Teams/Discord cho thông báo của team về việc gửi trong ReputeCast | Klaviyo, HubSpot, Dittofeed, Mailgun | 🚫 Chỉ qua webhook; cảnh báo reputation đã có trong HostRepute (mục 5) | P2 | S |
| Đồng bộ CRM (Salesforce, HubSpot, Pipedrive) | Mautic, Encharge, ActiveCampaign, beehiiv | 🚫 | P2 | M mỗi CRM |
| Google Sheets, Typeform, Calendly, Eventbrite, Patreon, Linktree, ManyChat | Kit App Store, beehiiv | ❌ Làm được qua Zapier/Make/n8n | P2 | S mỗi app |
| Đồng bộ audience lên Meta, Google, TikTok | Mailchimp, Klaviyo, MailerLite, HubSpot | ❌ | P2 | M |
| Marketplace plugin có hook (UI, driver, cổng thanh toán) | Mautic, Acelle, MailWizz, phpList | 🟡 Có SDK adapter cho provider gửi thư | P3 | L |

### 2.17 Vận hành self-hosted và ngôn ngữ

| Tính năng | Đối thủ có | ReputeCast hiện tại | Ưu tiên | Độ khó |
|---|---|---|---|---|
| Helm chart, CloudFormation, cài một chạm (Railway, Render, Cloudron, Coolify, aaPanel) | Dittofeed, useSend, listmonk, BillionMail | 🟡 Chỉ có Docker Compose và binary (`deploy/`) | P1 | S–M |
| Số ngôn ngữ giao diện admin | listmonk 38, Mautic hàng chục, Keila 9, Notifuse 8 | 🟡 5 ngôn ngữ (EN, VI, DE, ES, FR) | P2 | M (cộng đồng dịch) |
| Ngôn ngữ RTL (Ả Rập, Hebrew) cho email, trang và app | listmonk | 🚫 Hiện bị từ chối (`internal/languages/catalog.go:33`) | P2 | L |
| Tự cập nhật trong app; VACUUM theo lịch | MailWizz, listmonk | 🟡 Có thông báo phiên bản mới và lệnh `doctor` | P3 | S |

---

## 3. Những mảng sản phẩm ReputeCast chưa có

### 3.1 Nhận mail (inbound) và xử lý reply — P1, M–L

- **Đối thủ có:** Postmark, SendGrid, Mailgun, Resend, Lettermint, MailerSend, SES Mail Manager, Plunk, Postal. Buttondown có hộp thư gom reply, comment và mention.
- **ReputeCast:** không nhận mail. Bằng chứng: DESIGN_INVENTORY #86, `internal/domains/service.go:187`; spec để ngoài phạm vi (`docs/spec/reputecast-design-addon-transactional.md:243-246`).
- **Inbound mở khóa được nhiều việc:**
  - xử lý unsubscribe qua mailto (1.1);
  - stop-on-reply cho automation;
  - hộp thư reply cho newsletter;
  - tự thu báo cáo DMARC rua và ARF;
  - inbound parse webhook cho developer.
- **Cách làm hợp kiến trúc hiện tại:** nhận qua provider (SES receiving, Mailgun routes, Postmark inbound, Resend inbound) như một optional interface của adapter. Không cần tự chạy MTA.

### 3.2 Bình luận, cộng đồng, social web

- **Đối thủ có:**
  - Ghost: bình luận theo luồng, like, ghim, trang kiểm duyệt; ActivityPub, để người dùng Mastodon, Threads và Flipboard follow được.
  - Substack: Notes, Chat, live video.
  - beehiiv: Community (ra mắt 7/2026).
  - Buttondown: bình luận, trả lời bình luận bằng email, webmention, cross-post sang Bluesky và Mastodon.
- **ReputeCast:** 🚫 `docs/spec/reputecast-design-addon-publish-sell.md:441`.
- **Đề xuất:**
  - Bình luận trên post kèm kiểm duyệt: P1, M.
  - ActivityPub: P2, L. Hợp với một dự án mã nguồn mở.
  - Cross-post lên Bluesky/Mastodon: P3, S.

### 3.3 Audio, podcast, video

- **Đối thủ có:**
  - beehiiv: host podcast, chèn quảng cáo động, audio newsletter bằng TTS.
  - Substack: tác giả tự đọc bài (voiceover), podcast, video, livestream, app TV.
  - Ghost: tích hợp Transistor.
  - Kit: app Waveletter.
- **ReputeCast:** 🚫 `docs/spec/reputecast-design-addon-publish-sell.md:443`.
- **Đề xuất:**
  - Bước nhỏ nhất là bản audio TTS cho mỗi bài: tự host bằng Piper/Kokoro, hoặc dùng key riêng (P2, S–M).
  - Feed RSS cho podcast: P3, M.

### 3.4 Webinar, sự kiện, đặt lịch, khóa học

- **Đối thủ có:**
  - beehiiv Webinars: tối đa 1.000 người, bán vé, bán bản ghi.
  - Substack Live.
  - GetResponse: phòng webinar, trình tạo khóa học.
  - MailerLite Bookings, Brevo Meetings, Kajabi.
  - Constant Contact: sự kiện và bán vé.
- **ReputeCast:** mới có trang đăng ký webinar, file `.ics` và automation nhắc lịch.
- **Đề xuất:**
  - Tích hợp Zoom, YouTube Live hoặc Jitsi, và bán vé qua product: P2, M.
  - Đặt lịch hẹn: P3, M.
  - Khóa học: P3, L.

### 3.5 Mạng lưới tăng trưởng trả phí và mạng quảng cáo

- **Đối thủ có:**
  - beehiiv Paid Recommendations: trả theo từng subscriber đã xác minh, ví dụ $1,68–2,80; năm 2026 Boosts được đổi tên thành tính năng này.
  - beehiiv Ad Network: CPM, CPC, programmatic.
  - Kit: Creator Network và Sponsor Network.
  - Substack: Lists và Creator match.
  - Kajabi Amplify, SparkLoop.
- **ReputeCast:** 🚫 `docs/spec/reputecast-design-addon-grow.md:336-340`.
- **Vướng mắc:** các tính năng này cần một bên trung gian tập trung, trái với mô hình self-hosted.
- **Hướng khả thi:** tích hợp SparkLoop, Paved hoặc Passionfroot thay vì tự xây (P3).

### 3.6 CRM, live chat, helpdesk

- **Đối thủ có:**
  - Brevo: CRM, Conversations, gọi điện.
  - ActiveCampaign, HubSpot.
  - Klaviyo Helpdesk, Bento Chat.
- **ReputeCast:** 🚫.
- **Đề xuất:** làm đồng bộ CRM (mục 2.16), không tự xây CRM.

### 3.7 Dữ liệu khách hàng (CDP): custom objects, B2B, event pipeline

- **Đối thủ có:**
  - Customer.io: objects và relationships, collections, Data Pipelines, reverse ETL, xuất sang warehouse.
  - Klaviyo: custom objects, import từ warehouse.
  - Dittofeed: API tương thích Segment, computed properties.
  - Mautic: companies, tracking khách vô danh rồi gộp khi họ để lại danh tính.
  - Brevo: companies, connector tới các database.
- **ReputeCast:** có custom field, API event (đến 16 KB), segment bằng SQL.
- **Đề xuất:**
  - JS tracker và gộp khách vô danh: P1, M. Đây cũng là nền cho browse abandonment.
  - API tương thích Segment (identify/track) và làm đích cho RudderStack/Segment: P2, M.
  - Companies: P2, M.
  - Reverse ETL từ Postgres/BigQuery: P3, M.
  - Custom objects: P3, L.

### 3.8 In-app message, notification inbox, mobile SDK

- **Đối thủ có:** Knock; Novu (mã nguồn mở, có component `<Inbox/>`); Courier; SuprSend; Customer.io; Braze; Laudspeaker.
- **ReputeCast:** có web push, nhưng chuông thông báo chỉ dành cho team.
- **Phù hợp với:** SaaS founder.
- **Ưu tiên:** P2–P3, L.

### 3.9 Dự đoán và AI decisioning

- **Đối thủ có:** Klaviyo, Braze Decisioning Studio, Iterable Nova, Mailchimp.
- **ReputeCast:**
  - `best_time` lấy giờ mở thư thường gặp nhất của từng contact (`internal/store/queries/sending.sql:650-652`).
  - Scoring chạy theo luật.
- **Đề xuất:**
  - Tối ưu giờ gửi bằng mô hình, chỉ dựa trên tín hiệu của người thật: P2, M.
  - CLV/churn cho store bằng BG/NBD: P2, M.
  - Bandit: xem mục 2.2.

### 3.10 Kênh mới

- **Đối thủ có:**
  - RCS (Klaviyo).
  - Telegram (HubSpot, SendPulse).
  - Zalo OA/ZNS cho thị trường Việt Nam.
  - LINE (Customer.io), Viber.
  - Slack, Discord, Teams làm kênh gửi (Knock, Novu, SuprSend).
  - Apple Messages for Business.
  - Thư giấy/postcard (Lob, PostGrid).
  - Wallet pass (Brevo).
  - Đăng bài mạng xã hội (Mailchimp, Constant Contact, HubSpot).
- **Đề xuất:**
  - Telegram: P2, S.
  - Zalo ZNS: P2 cho thị trường Việt Nam (app đã có tiếng Việt), M.

### 3.11 MTA tích hợp sẵn

- **Đối thủ có:** BillionMail (Postfix, Dovecot, Rspamd, RoundCube), Mailtrain (ZoneMTA), Postal.
- **ReputeCast:** hỗ trợ Postal như một provider.
- **Đề xuất:** đóng gói Postal vào docker-compose như một tùy chọn (P3, M), không tự viết MTA.

---

## 4. Thay đổi của ngành 2025–2026 cần theo dõi (ngoài mục 1)

- **Gmail:**
  - AI Inbox và AI Overviews (1/2026).
  - Tab Promotions có kiểu sắp xếp "Most relevant" và mục Purchases (9/2025). Hai thay đổi này làm annotations (2.1) và schema Order trong email transactional đáng làm hơn.
- **Apple:**
  - Apple Mail tự xếp thư vào category ngay trên máy; Apple Intelligence tóm tắt thư thay cho preview text.
  - Branded Mail qua Apple Business (3/2026): miễn phí nhưng không có API, nên chỉ làm được checklist.
- **Yahoo:**
  - Subscription Hub có tab "Discover" cho người gửi hỗ trợ subscribe một chạm.
  - Sender Hub có thêm Insights.
  - Performance Feeds (phải đăng ký) cung cấp dữ liệu đọc, lướt, forward ở phía mailbox provider, không bị Apple MPP làm sai lệch.
- **MJML 5:** include bị tắt mặc định. ReputeCast cố ý ở lại MJML 4 vì grapesjs-mjml và gomjml đều là MJML 4 (`CLAUDE.md`). Cần theo dõi khi hai thư viện này lên bản 5.
- **SMS tại Mỹ:**
  - Quy tắc TCPA "revoke-all" (hủy một loại tin là hủy tất cả) lùi đến 31/1/2027.
  - 10DLC bắt buộc Authentication+ 2.0 từ 8/2025.
- **European Accessibility Act:** có hiệu lực từ 28/6/2025.
  - Bộ kiểm tra accessibility là điểm mạnh của ReputeCast.
  - Có thể thêm xuất báo cáo tuân thủ WCAG 2.2 (P3, S).
- **Thị trường:**
  - MCP đã thành tiêu chuẩn chung.
  - Notifuse chuyển sang BSL 1.1 (9/2026) và thu phí cho SSO, phân quyền, bản dịch template và audit log. ReputeCast cho miễn phí tất cả, nên đây là một điểm để nhấn mạnh khi so sánh.
  - Salesforce mua Braze (10/2026); Beefree đổi tên thành RGE Studio (5/2026).

---

## 5. Những mục các báo cáo gọi là "thiếu" nhưng ReputeCast, HostRepute hoặc ReputeCast Cloud đã có

Đã kiểm tra trong code, nên không tính là khoảng trống.

**Đã có trong ReputeCast:**

- **Thu hút và đồng ý (consent):**
  - Double opt-in chống link scanner: trang xác nhận có nút Confirm, token hết hạn, giới hạn tần suất gửi lại.
  - Bằng chứng đồng ý: IP, trình duyệt, trang, đúng nội dung consent và phiên bản.
  - Chống spam form: Turnstile, honeypot, chặn domain dùng một lần và địa chỉ role.
  - Lý do unsubscribe; tạm dừng nhận thư 30/90 ngày (tương đương "snooze").
  - Preference center có chọn tần suất.
- **Soạn và duyệt:**
  - Quy trình duyệt (approvals) cho campaign, automation, template, post và page.
  - Link xem trước cho reviewer không có tài khoản; presence trong composer.
  - Cảnh báo Gmail cắt thư (clipping).
  - Brand kit lấy từ URL website.
- **Gửi và deliverability:**
  - Gửi theo giờ địa phương của người nhận và theo giờ hay mở của từng contact (dựa trên luật).
  - Gửi lại cho người chưa mở/đã click; giới hạn tần suất marketing của cả workspace.
  - Xác minh địa chỉ qua 5 dịch vụ.
  - Placement test bằng seed mailbox, nhận diện cả folder lẫn tab; canary.
  - SES tenants; dedicated IP kèm warm-up; tracking domain riêng.
  - Kiểm tra BIMI ở mức DNS; Feedback-ID.
  - Xử lý DSN/ARF qua bounce mailbox cho route SMTP.
- **Developer:**
  - MCP server với 61 tool, resource và prompt.
  - Webhook: xoay secret, ký song song cũ và mới trong 24 giờ; gửi gộp tối đa 100 sự kiện.
  - Transactional: idempotency key, hẹn giờ gửi, hủy hoặc dời lịch.
- **Website và kiếm tiền:**
  - Đăng nhập bằng mã 6 số bên cạnh magic link.
  - Post chỉ cho member (tương đương freewall).
  - JSON-LD, sitemap, RSS.
  - Stripe Tax; product pay-what-you-want; gift có ngày giao.
- **Khác:**
  - Đếm lượt forward.
  - Bản dịch theo ngôn ngữ cho campaign, automation và template (Notifuse thu phí tính năng này).
  - SSO qua OIDC có map nhóm; audit log; dark mode.

**Lưu ý:** một báo cáo nói ReputeCast đã có SAML/SCIM. Thực tế chưa có, mới chỉ có OIDC.

**Đã có trong HostRepute (`app.hostrepute.com`).** Người dùng dùng HostRepute cho các tính năng này; ReputeCast không build lại.

- **Chấm điểm spam trước khi gửi:** Campaign Preflights nhận mã nguồn hoặc file `.eml`, chấm điểm bằng Rspamd, kiểm tra nội dung (thiếu bản text, HTML quá lớn, thư toàn ảnh, URL rút gọn, link lệch domain, List-Unsubscribe hỏng), quét link và kiểm tra alignment SPF, DKIM, DMARC.
  - Bằng chứng: `app/Services/CampaignPreflight/`.
  - Đã bỏ: dòng chấm điểm spam ở 2.1, phần điểm spam của sandbox ở 2.8, dòng chấm điểm spam ở mục 6.
  - ReputeCast không tự gửi nội dung thư sang HostRepute (`reputecast-design-expansion-plan.md:74`), nên người dùng tự tải `.eml` lên HostRepute.
- **Compliance status và Deliverability verdict của Google Postmaster v2,** gồm one-click unsubscribe, honor unsubscribe và delivery error theo lý do.
  - Bằng chứng: `app/Services/Postmaster/PostmasterSyncService.php`, bảng `gmail_postmaster_compliance_snapshots`.
  - Đã thu hẹp 1.3: ReputeCast chỉ bỏ hai trường cũ và trỏ sang HostRepute.
- **Cảnh báo reputation qua Slack, Discord, Telegram, PagerDuty, Pushover và webhook,** có lịch nhận theo múi giờ.
  - Bằng chứng: `app/Services/AlertNotificationService.php`.
  - Đã thu hẹp dòng Slack/Teams/Discord ở 2.16 về thông báo của chính ReputeCast.
- **Giám sát liên tục:** blocklist kèm hỗ trợ delisting, DNS check theo lịch, chứng chỉ TLS và STARTTLS, uptime, hạn domain, SNDS, status page, báo cáo reputation định kỳ.
  - Đã bỏ dòng tương ứng ở mục 6.

**Chưa có trong HostRepute, nên vẫn giữ:**

- Thu báo cáo DMARC aggregate (2.7): `app.hostrepute.com` không có bảng, parser hay hộp thư rua nào. `reputecast-design-expansion-plan.md:64` ghi HostRepute làm việc này, nhưng code chưa có.
- Kiểm tra chứng chỉ VMC/CMC và logo BIMI (2.7): HostRepute chỉ tìm bản ghi `v=BIMI1` (`app/Services/Deliverability/DeliverabilityCheckService.php:366`) và cảnh báo khi thiếu tag `a=`.
- Kiểm tra theo caniemail (2.1): Campaign Preflights không có.

**Đã có trong ReputeCast Cloud** (`ReputeCastCloud`, nhánh `c17-production`, commit `222be5a`). Ai cần ReputeCast được host sẵn, có gói giá và hóa đơn, thì dùng ReputeCast Cloud; ReputeCast không build lại.

- **Chế độ SaaS cho instance được host:** mỗi khách một instance riêng; gói giá theo số contact, đếm hằng đêm trên mọi workspace; trial 14 ngày; thanh toán và hóa đơn qua Paddle (merchant of record); dừng instance khi nợ quá hạn; staff console; duyệt signup.
  - Bằng chứng: `docs/spec/domain.md` §5–6, `docs/spec/billing.md`, `app/Billing/TeamBilling.php`.
  - Đã bỏ: mục 3.11 cũ ("Chế độ reseller/SaaS cho đơn vị host") và dòng "Chế độ reseller" ở Đợt 3.
  - Phần ReputeCast còn phải làm cho Cloud, chưa có trên roadmap của ReputeCast (`docs/spec/provisioning.md:599-602` của Cloud):
    - sending hold khi vượt gói hoặc nợ quá hạn. Cloud gọi việc này là M54, nhưng M54 của ReputeCast đã là dark mode. Trong lúc chờ, Cloud chỉ báo staff (D10);
    - `usage --json` và `domains --json` (Cloud gọi là M55), để thay hai truy vấn Cloud đang đọc thẳng schema của ReputeCast.

**Chưa có trong ReputeCast Cloud, nên vẫn giữ:**

- White-label giao diện admin và đăng nhập thay khách hàng (2.15): Cloud không có white-label; nó chỉ gửi link đăng nhập một lần tới Owner của instance, không cho staff đăng nhập thay (`docs/spec/domain.md` §5.5).
- Helm chart, CloudFormation, bộ cài một chạm, tự cập nhật trong app, VACUUM theo lịch (2.17): Cloud tự cài và nâng cấp instance (theo ring, backup trước, có rollback), nhưng chỉ trên server của Cloud. Người tự host trên hạ tầng của mình vẫn cần các tính năng này.

---

## 6. Những mục spec để ngoài phạm vi: có nên quyết định lại?

| Mục (spec) | Diễn biến ở đối thủ năm 2026 | Đề xuất |
|---|---|---|
| Bình luận và cộng đồng (`publish-sell.md:441`) | Ghost, Substack, beehiiv (7/2026), Buttondown đều có | **Xem lại**: làm bình luận trước |
| Group subscription, tạm dừng subscription (`earn.md:395-399`) | beehiiv (6/2026), Substack | **Xem lại** |
| Sponsor tự đặt chỗ và thanh toán (`earn.md:395`) | Buttondown, Substack | **Xem lại** |
| A/B test page (`publish-sell.md:448`) | Klaviyo, MailerLite, Omnisend (A/B form) | **Xem lại** cùng với A/B form |
| Browse abandonment (`publish-sell.md:447`) | Klaviyo, Omnisend, Mailchimp | **Xem lại** sau khi có JS tracker |
| RTL (`connect-reach.md:418`) | listmonk có tiếng Ả Rập và Hebrew | **Xem lại** nếu nhắm thị trường Trung Đông |
| SAML/SCIM (`team-access.md:77`) | Mautic, Customer.io, Lettermint | **Xem lại** cho khách doanh nghiệp |
| Hộp thư SMS hai chiều (`channels-agencies.md:474`) | Klaviyo, ActiveCampaign | Xem lại sau khi có inbound email |
| Inbound email (`transactional.md:243-246`) | Hầu hết đối thủ transactional | **Xem lại**: mở khóa nhiều thứ (3.1) |
| Podcast, video hosting (`publish-sell.md:443`) | beehiiv, Substack | Giữ ngoài phạm vi; làm TTS trước |
| Marketplace/directory tập trung, paid recommendation (`grow.md:336-340`) | beehiiv, Kit, Substack | Giữ: trái mô hình self-hosted |
| Mạng quảng cáo | beehiiv, Kit, Kajabi | Giữ; có thể tích hợp bên thứ ba |
| Tự xây CRM | Brevo, ActiveCampaign, HubSpot | Giữ; chỉ làm đồng bộ |
| Sản phẩm vật lý (`publish-sell.md:444`) | Shopify/Woo đảm nhận | Giữ |
| Mobile push native (`channels-agencies.md:479`) | Klaviyo, Customer.io, Braze | Giữ cho đến khi có kế hoạch SDK |

---

## 7. Đề xuất thứ tự làm

### Đợt 1: việc nhỏ, giá trị cao (mỗi việc S)

1. Sửa unsubscribe qua mailto (1.1).
2. Cập nhật theo DMARCbis, thêm quy tắc Microsoft, bỏ hai trường reputation cũ của Postmaster (1.2–1.4).
3. Gợi ý sửa lỗi gõ email trên form (1.5) và inbox deep link sau khi đăng ký (2.4).
4. Exit intent; ẩn form với người đã đăng ký (2.4).
5. Trigger theo vòng đời gói trả phí (2.5).
6. Báo cáo theo email client và thiết bị; GeoIP tùy chọn; giờ trong ngày (2.3).
7. Tự gắn UTM cho link trong email (2.16).
8. Báo cáo số liệu định kỳ qua email (2.3).
9. Nút share trên post và llms.txt (2.9).
10. Giới hạn IP cho API key (2.8).

### Đợt 2: chiến lược (mỗi việc M)

11. Inbound email qua provider (3.1): mở đường cho hộp thư reply, stop-on-reply, tự thu DMARC rua và ARF.
12. Countdown timer trong email; mã giảm giá riêng cho từng người nhận (2.1).
13. Form nhiều bước, targeting, A/B test form (2.4).
14. Import bài viết từ Substack, Ghost, WordPress (2.9).
15. Bình luận trên post (3.2).
16. JS tracker, kèm browse abandonment, back-in-stock và gợi ý sản phẩm (2.12, 3.7).
17. Holdout group (2.3).
18. SDK client sinh từ OpenAPI; Helm chart và các bộ cài một chạm (2.8, 2.17).
19. Group subscription, tạm dừng subscription, sponsor tự đặt chỗ (2.10).
20. Lead ads Meta/LinkedIn (2.11).

### Đợt 3: dài hạn hoặc cần quyết định sản phẩm

- AI agent.
- ActivityPub.
- TTS và podcast.
- CDP và custom objects.
- Mobile push và in-app.
- RCS, Zalo, Telegram.
- SAML và SCIM.
- RTL.

---

## 8. Nguồn chính

**Creator/newsletter**
- beehiiv: https://product.beehiiv.com/archive · https://product.beehiiv.com/p/automations-v4 · https://product.beehiiv.com/p/community · https://www.beehiiv.com/support/article/41790356251159-boosts-to-paid-recommendations-what-changed
- Substack: https://on.substack.com · https://techcrunch.com/2026/01/22/substack-launches-a-tv-app/
- Ghost: https://ghost.org/changelog/ · https://ghost.org/changelog/6/
- Kit: https://updates.kit.com/changelog · https://developers.kit.com/changelog
- Buttondown: https://buttondown.com/features · https://buttondown.com/blog?category=changelog
- Flodesk: https://updates.flodesk.com/announcements · Kajabi: https://feedback.kajabi.com/changelog

**SMB / e‑commerce**
- Mailchimp: https://mailchimp.com/whats-new · Brevo: https://www.brevo.com/releases-main-page
- MailerLite: https://www.mailerlite.com/whats-new · Klaviyo: https://www.klaviyo.com/whats-new
- Omnisend: https://support.omnisend.com/en/articles/17225594-what-s-new-october-2026
- ActiveCampaign: https://www.activecampaign.com/release-updates · GetResponse: https://www.getresponse.com/features

**Mã nguồn mở**
- listmonk: https://github.com/knadh/listmonk/releases · Mautic: https://github.com/mautic/mautic/releases
- Keila: https://github.com/pentacent/keila/releases · Notifuse: https://github.com/Notifuse/notifuse
- Mailcoach: https://www.mailcoach.app/changelog · Sendy: https://sendy.co/get-updated
- BillionMail: https://github.com/aaPanel/BillionMail · Plunk: https://github.com/useplunk/plunk/releases
- useSend: https://github.com/usesend/useSend/releases · Dittofeed: https://github.com/dittofeed/dittofeed
- MailWizz: https://www.mailwizz.com/features/ · Acelle: https://acellemail.com/features · Postal: https://docs.postalserver.io

**Developer/lifecycle**
- Customer.io: https://docs.customer.io/release-notes/ · Loops: https://loops.so/changelog
- Resend: https://resend.com/changelog · Postmark: https://postmarkapp.com/updates
- Mailgun: https://www.mailgun.com/releases/ · Amazon SES: https://docs.aws.amazon.com/ses/latest/dg/tenants.html
- Mailtrap: https://feedback.mailtrap.io/changelog · Lettermint: https://lettermint.co/changelog
- Knock: https://knock.app/changelog · Novu: https://www.novu.co/changelog/
- Courier: https://www.courier.com/changelog · SuprSend: https://docs.suprsend.com/changelog/overview
- Braze: https://www.braze.com/docs/releases/home · Iterable: https://iterable.com/blog/fall-2026-product-launch/

**Chuẩn và mailbox provider**
- DMARC RFC 9989: https://www.rfc-editor.org/info/rfc9989/
- Postmaster Tools v2: https://support.google.com/mail/answer/16594218 · https://developers.google.com/workspace/gmail/postmaster/guides/migration-v2
- Microsoft (bulk sender): https://techcommunity.microsoft.com/blog/microsoftdefenderforoffice365blog/strengthening-email-ecosystem-outlook%E2%80%99s-new-requirements-for-high%E2%80%90volume-senders/4399730
- Gmail annotations: https://developers.google.com/workspace/gmail/promotab/overview
- Yahoo: https://senders.yahooinc.com/subhub/ · https://senders.yahooinc.com/email-deliverability-performance-feeds/
- BIMI CMC: https://bimigroup.org/announcing-common-mark-certificates
- Apple Business: https://www.apple.com/newsroom/2026/03/introducing-apple-business-a-new-all-in-one-platform-for-businesses-of-all-sizes/
