# Vietnamese glossary (Tiếng Việt)

The words ReputeCast uses in Vietnamese, so every screen, email and error
says the same thing the same way. It was written from the Vietnamese on the
design's stress-test boards (`StressCampaignsVi`, `StressReviewVi`,
`StressTeamVi`, `LanguageRegion`) and the glossary in
`docs/spec/reputecast-design-addon-connect-reach.md` §4.

> **Status: draft.** These are the first non-English words in ReputeCast. A
> native speaker reviews them before Vietnamese ships in a release (brief
> §8.17). Change a term here first, then in the translations
> (`web/src/**/messages.vi.ts`, `internal/<package>/messages/vi.json`).

How to add or complete a language: [`translating.md`](translating.md).

## Voice

- Plain and specific, like the English: say what happened and what to do.
  No exclamation marks.
- Address the reader as **bạn**. Don't use "quý khách" or "anh/chị".
- Sentence case, as in English. Only the first word and names start with a
  capital: "Chiến dịch mới", not "Chiến Dịch Mới". Days of the week keep
  their capitals: "Thứ Hai", "Chủ Nhật".
- Vietnamese has no plural forms: messages use `{n, plural, other {…}}` only.
  "1 chiến dịch", "6 chiến dịch".
- Numbers and dates come from the region format, never typed by hand:
  187.312 · 99,2% · 18,21 US$ · Th 3, 14 thg 10, 2026 · 09:00.
- Relative times: "vừa xong", "4 phút trước", "2 giờ trước", "hôm qua",
  "sau 5 ngày".
- Overlines (table headers, group labels) stay uppercase and keep their
  marks: "NGƯỜI NHẬN", "HOẠT ĐỘNG GẦN NHẤT".
- Language names start with the common noun "tiếng": "Tiếng Anh" on its own,
  at the start of a sentence and after a colon or "·", "tiếng Anh" inside a
  sentence ("Thêm tiếng Anh", "Người đọc tiếng Đức nhận tiếng Anh"). Write
  `{language}` where the sentence needs it: the app picks the case, for one
  name or a list of them (`internal/i18n/names.go`, `web/src/i18n/names.ts`).

## Never translated

Workspace content (campaign, list, segment, template, form and custom role
names such as "Content Writer"), people's names and email addresses,
permission keys (`campaigns:send`), audit action names, API error codes,
provider names and codes (Amazon SES, Postal, Postmark, SES), product names
(ReputeCast, HostRepute, Gmail, Outlook), DNS records and record types (SPF,
DKIM, DMARC, BIMI, CNAME, MAIL FROM), headers (List-Unsubscribe), code
samples, CLI output, "MCP", "2FA", "TOTP", "IdP", "SSO", "JIT", "A/B",
"HTML", "CSS", "API", "webhook".

## App shell and navigation

| English | Tiếng Việt |
|---|---|
| Overview | Tổng quan |
| Campaigns | Chiến dịch |
| Automations | Tự động hóa |
| Transactional | Giao dịch |
| Audience | Người đăng ký |
| Grow | Tăng trưởng |
| Earn | Kiếm tiền |
| Forms & pages | Biểu mẫu và trang |
| Deliverability | Khả năng vào hộp thư |
| Sending providers | Nhà cung cấp gửi thư |
| Settings | Cài đặt |
| More | Thêm |
| Skip to content | Chuyển đến nội dung |
| Import contacts | Nhập liên hệ |
| HostRepute connected | Đã kết nối HostRepute |
| synced 4 min ago | đồng bộ 4 phút trước |
| Account & security | Tài khoản và bảo mật |
| Notification preferences | Tùy chọn thông báo |
| Workspace settings | Cài đặt không gian làm việc |
| Instance admin | Quản trị hệ thống |
| admins only | chỉ quản trị viên |
| Sign out | Đăng xuất |
| Language: English | Ngôn ngữ: Tiếng Việt (the item names the current language in its own name) |

## Settings tabs

| English | Tiếng Việt |
|---|---|
| General | Chung |
| Senders | Người gửi |
| Providers | Nhà cung cấp |
| Team | Nhóm |
| Roles | Vai trò |
| Approvals | Phê duyệt |
| Keys & MCP | Khóa và MCP |
| Webhooks | Webhook |
| Integrations | Tích hợp |
| Audit log | Nhật ký kiểm tra |

## Roles and people

| English | Tiếng Việt |
|---|---|
| Owner | Chủ sở hữu |
| Admin | Quản trị viên |
| Editor | Biên tập viên |
| Viewer | Người xem |
| custom (role) | tùy chỉnh |
| instance admin (the area, a label: "Instance admin", "· instance admin") | quản trị hệ thống |
| instance admin (a person, in a sentence: "Ask your instance admin") | quản trị viên hệ thống |
| break-glass account | tài khoản khẩn cấp |
| member | thành viên |
| invitee | người được mời |
| workspace | không gian làm việc |
| instance | hệ thống |
| team | nhóm |

## Statuses

| English | Tiếng Việt |
|---|---|
| Draft | Bản nháp |
| Draft · blocked | Bản nháp · bị chặn |
| Scheduled | Đã lên lịch |
| Sending | Đang gửi |
| Sent | Đã gửi |
| Active | Đang hoạt động |
| Invited | Đã mời |
| Deactivated | Đã vô hiệu hóa |
| On · Off | Bật · Tắt |
| Via IdP | Qua IdP |
| Required | Bắt buộc |
| Pass · Warn · Block | Đạt · Cảnh báo · Chặn |
| Ready · Ready with warnings · Blocked | Sẵn sàng · Sẵn sàng, có cảnh báo · Bị chặn |
| Healthy | Ổn định |
| Warming up | Đang khởi động |
| New | Mới |
| Not saved | Chưa lưu |
| Development | Phát triển |
| Auto-paused | Tự tạm dừng |
| Pending approval · Approved | Chờ phê duyệt · Đã phê duyệt |
| Cancelled · Failed | Đã hủy · Thất bại |

## Campaigns

| English | Tiếng Việt |
|---|---|
| Campaign | Chiến dịch |
| New campaign | Chiến dịch mới |
| All · Drafts · Scheduled · Sending · Sent | Tất cả · Bản nháp · Đã lên lịch · Đang gửi · Đã gửi |
| Search campaigns | Tìm chiến dịch |
| Sent · last 30 days | Đã gửi · 30 ngày qua |
| Cost · last 30 days | Chi phí · 30 ngày qua |
| Recipients | Người nhận |
| Delivered | Gửi thành công |
| Human opens | Lượt mở bởi người đọc |
| Clicks | Lượt nhấp |
| click-to-open | nhấp trên lượt mở |
| bounces · complaints | thư trả lại · khiếu nại |
| machine opens shown separately | lượt mở tự động được tính riêng |
| Readiness | Độ sẵn sàng |
| 3 blocking issues | 3 lỗi chặn gửi |
| Edit · Report · Duplicate | Sửa · Báo cáo · Nhân bản |
| not set | chưa chọn |
| created 08:55 | tạo lúc 08:55 |
| edited 09:31 by Linh Nguyen | Linh Nguyen sửa lúc 09:31 |
| All subscribers | Tất cả người đăng ký |
| Broadcasts route | Tuyến Gửi hàng loạt |
| A/B subject, winner B | A/B tiêu đề, B thắng |
| auto-paused once, Yahoo deferrals | tự tạm dừng một lần vì Yahoo hoãn nhận |
| Showing 9 of 48 | Hiển thị 9 trên 48 |
| Previous · Next | Trước · Sau |
| Live monitor | Theo dõi trực tiếp |
| imported from Brevo · Brevo · imported | nhập từ Brevo · Brevo · đã nhập |
| segment (in an audience: 2 segments) | phân khúc (2 phân khúc) |

## Review & send

| English | Tiếng Việt |
|---|---|
| Review & send | Kiểm tra và gửi |
| Back to editor | Quay lại trình soạn thảo |
| Run checks again | Chạy lại kiểm tra |
| Draft · autosaved 2 min ago | Bản nháp · tự lưu 2 phút trước |
| Fix both warnings | Sửa cả hai cảnh báo |
| Schedule send to 31,604 recipients | Lên lịch gửi cho 31.604 người nhận |
| 27 checks passed, 2 warnings, 0 blocks | 27 mục đạt, 2 cảnh báo, 0 mục chặn |
| Sender reputation | Uy tín người gửi |
| blocklist | danh sách chặn |
| suppression list (shared) | danh sách chặn gửi (dùng chung) |
| role addresses | địa chỉ chức danh |
| Duplicates | Trùng lặp |
| Final recipients | Người nhận cuối cùng |
| Route split | Phân bổ tuyến |
| Content · Links · Compliance | Nội dung · Liên kết · Tuân thủ |
| Audience quality | Chất lượng người nhận |
| Routing · stream | Định tuyến · luồng |
| Broadcasts stream | Luồng Gửi hàng loạt |
| Send time · Send now · Schedule | Thời điểm gửi · Gửi ngay · Lên lịch |
| Best time per recipient | Giờ tốt nhất cho từng người nhận |
| Test email · Send test | Thư gửi thử · Gửi thử |
| Auto-pause thresholds | Ngưỡng tự động tạm dừng |
| Bounce rate · Complaint rate | Tỷ lệ trả thư · Tỷ lệ khiếu nại |
| per mailbox provider | theo nhà cung cấp hộp thư |
| warm-up (day 9 of 21) | khởi động (ngày 9/21) |
| 14 /s | 14/giây |
| Cost · Time | Chi phí · Thời gian |

### Send Readiness's checks

The findings the server writes for each check (`internal/readiness`).

| English | Tiếng Việt | Note |
|---|---|---|
| check (a row) · passed · not checked · timed out | mục kiểm tra · đạt · chưa kiểm tra · hết thời gian chờ | |
| the verdict waits for this one | kết luận đang chờ mục này | |
| I understand … (a warning's confirmation) | Tôi hiểu … | as the web's "I understand: {title}." |
| unengaged (contacts) | không tương tác | no open or click in the sunset policy's days |
| Sunset tier · At risk (names) | nhóm Ngừng gửi dần · Có nguy cơ | tier names keep their capitals |
| spam trigger · spam heuristic | dấu hiệu spam | "trips 3 spam heuristics": "vướng 3 dấu hiệu spam" |
| all caps · repeated punctuation · urgency phrasing · money words · click-bait | viết hoa toàn bộ · lặp dấu câu · lời lẽ thúc ép · từ ngữ về tiền · câu nhử nhấp | |
| image-to-text ratio | tỷ lệ hình ảnh trên chữ | |
| Gmail's 102 KB clipping limit · clips | ngưỡng cắt thư 102 KB của Gmail · cắt bớt | |
| inline(d) CSS · minify | CSS nội tuyến · thu gọn | |
| one-click unsubscribe (headers) | (header) hủy đăng ký một lần nhấp | "header" stays in English |
| postal address | địa chỉ bưu chính | |
| bulk-sender rules | quy tắc người gửi hàng loạt | Gmail and Yahoo's |
| URL shortener · redirects through | dịch vụ rút gọn liên kết · chuyển hướng qua | |
| responds · doesn't respond | phản hồi · không phản hồi | a link |
| review thresholds · pause level (SES) | ngưỡng xem xét · mức tạm dừng | |
| no enforcement | không bị thực thi giới hạn | an account's standing |
| terms allow bulk marketing | điều khoản cho phép tiếp thị hàng loạt | |
| production access | quyền gửi chính thức | as in Sending |
| double-tracked · rewrites links | bị theo dõi hai lần · viết lại liên kết | |
| aligned (DMARC, SPF) | (được) căn chỉnh | |
| Easy DKIM CNAME · selector | CNAME Easy DKIM · selector | names and record words stay |
| custom MAIL FROM | MAIL FROM tùy chỉnh | |
| Conditional content · conditional block · version | Nội dung có điều kiện · khối có điều kiện · phiên bản | |
| Visibility rule | quy tắc hiển thị | |
| poll · survey · answer page | bình chọn · khảo sát · trang trả lời | |
| risky · catch-all · disposable (addresses) | rủi ro · catch-all · dùng một lần | |
| inactive (contacts) | không hoạt động | At risk and Sunset together |
| paid members · trial · complimentary · gift (access) | hội viên trả phí · dùng thử · quyền truy cập miễn phí · quà tặng | as Earn has it |
| Sponsor block · creative · booking · Sponsored (label) | khối Nhà tài trợ · mẫu quảng cáo · lượt đặt · Được tài trợ | |
| Upgrade block · offer | khối Nâng cấp · ưu đãi | |
| page copy · preference center | nội dung trang · trung tâm tùy chọn | |
| bare URL · skipped heading level | URL trần · cấp đề mục bị bỏ qua | accessibility |
| linked image | hình có liên kết | |
| self-hosted (pricing) · plan | tự lưu trữ · theo gói | a provider's price |

## Sending and deliverability

Sending providers & routing, the connect wizard, Settings › Senders and
Providers, Deliverability, HostRepute and the live monitor. In alphabetical
order of the English.

| English | Tiếng Việt | Note |
|---|---|---|
| account health (a provider's) | tình trạng tài khoản | |
| adapter · community adapter · adapter registry | adapter · adapter cộng đồng · kho adapter | "adapter" stays in English, like "webhook" |
| aggregate report (DMARC) | báo cáo tổng hợp | |
| alignment (DMARC) | căn chỉnh | "căn chỉnh DMARC" |
| asset (a domain or IP HostRepute monitors) | tài sản | |
| backfill (statuses from an API) | lấy bù | "Lấy bù trạng thái từ API của Mailgun" |
| blocklist · listed · clean | danh sách chặn · bị liệt kê · sạch | |
| bounce mailbox | hộp thư nhận thư trả lại | SMTP providers |
| bounce · hard bounce · soft bounce | thư trả lại · trả thư vĩnh viễn · trả thư tạm thời | the rate is "tỷ lệ trả thư"; as in the Grow section |
| Broadcasts · Automations · System · Transactional (streams) | Gửi hàng loạt · Tự động hóa · Hệ thống · Giao dịch | names, so capitalised in a sentence: "luồng Hệ thống" |
| bulk marketing | tiếp thị hàng loạt | a provider's terms |
| cap (daily, today's) | giới hạn | "Giới hạn hôm nay" |
| catalog (of providers) | danh mục | the wizard's first step |
| condition (routing) | điều kiện | |
| credentials (a provider's) | thông tin xác thực | |
| dedicated IP · shared pool | IP riêng · nhóm IP dùng chung | |
| defer · deferral | hoãn nhận | "Yahoo hoãn nhận" |
| degraded · down | suy giảm · gián đoạn | provider statuses |
| delisting | gỡ khỏi danh sách | "Yêu cầu gỡ khỏi danh sách" |
| event log · raw event · normalized event | nhật ký sự kiện · sự kiện gốc · sự kiện chuẩn hóa | |
| failover · fallback · primary | chuyển dự phòng · dự phòng · chính | "nhà cung cấp dự phòng", "nhà cung cấp chính" |
| feedback loop | vòng phản hồi | |
| flow (an automation's, in sending text) | quy trình tự động | "luồng" is a stream: "email do các quy trình tự động gửi"; on the Automations screens a flow is "quy trình" |
| guard (Warm-up page) | ngưỡng bảo vệ | "Tỷ lệ trả thư · ngưỡng bảo vệ 2%" |
| held (recipients) | tạm giữ | |
| incident · outage | sự cố · sự cố gián đoạn | |
| live sending monitor | theo dõi gửi trực tiếp | "Live monitor" is "Theo dõi trực tiếp" |
| maintainer · unmaintained (adapter) | người duy trì · ngừng duy trì | |
| mapping (webhook, JSONPath) | ánh xạ | |
| message id | mã thư | |
| monitoring · monitored | giám sát · đang giám sát | HostRepute |
| nameserver | máy chủ tên miền | |
| polling (instead of webhooks) | hỏi định kỳ | "hỏi API sự kiện mỗi phút" |
| Postmaster reputation: high · medium · low · bad | cao · trung bình · thấp · kém | |
| production access (SES) | quyền gửi chính thức | the SES sandbox stays "sandbox" |
| provisioning | khởi tạo | SNS topics and configuration sets |
| pulled in · pushed out · two-way (suppression sync) | đã kéo về · đã đẩy đi · hai chiều | |
| quota | hạn mức | |
| read-only link (a live send) | liên kết chỉ xem | |
| recommended (an option, a badge) | khuyên dùng | "IMAP (khuyên dùng)"; a recommendation's heading is "Đề xuất" |
| region (AWS) | khu vực | AWS's own word; not the region format |
| reputation · domain reputation · IP reputation | uy tín · uy tín tên miền · uy tín IP | |
| reroute | định tuyến lại | |
| reverse DNS | DNS ngược | |
| route tester | công cụ thử tuyến | |
| route · routing · weighted split | tuyến · định tuyến · chia theo tỷ trọng | |
| secret (webhook signing) | khóa bí mật | "khóa bí mật ký webhook" |
| seed list | danh sách địa chỉ thử | one address of yours at each mailbox provider |
| self-test (of an endpoint) | tự kiểm tra | |
| Send Readiness | Độ sẵn sàng gửi | |
| share (of a split · of a send) | tỷ trọng · phần | |
| slowdown (warm-up) | giảm tốc | |
| SNDS filter result: Green · Yellow · Red | Xanh · Vàng · Đỏ | |
| spam rate · spam trap | tỷ lệ spam · bẫy spam | |
| spam trap hit | lượt dính bẫy spam | |
| stale (data) · cached results | đã cũ · kết quả đã lưu | |
| stream | luồng | |
| suppression sync | đồng bộ danh sách chặn gửi | |
| throttle · throttling | giới hạn tốc độ | |
| tracking · tracking domain | theo dõi · tên miền theo dõi | |
| verdict (HostRepute): Good · Fair · Poor · Bad | Tốt · Khá · Kém · Xấu | |
| warm-up · warmed up · warm-up guard · ramp | khởi động · đã khởi động xong · bảo vệ khởi động · mức tăng | |
| webhook endpoint · registration | endpoint webhook · đăng ký | |
| write-only (a key) | chỉ ghi | "it can be replaced but never read back" |
| zone file | tệp zone | |

"sandbox", "configuration set", "SNS topic", "IAM", "ARN", "CloudFormation",
"stack", "return-path", "DSN", "ARF", "rua", "JSONPath", "manifest.json",
"SDK", "tunnel", "reverse proxy", "endpoint", "token", "idempotency",
"Postmaster", "SNDS" and "JMRP" stay as they are, and so do the AWS field
names ("Access key ID", "Secret access key", "RoleArn"), the SES region
names, a provider's own stream names (Postmark's "Broadcast" stream:
"luồng Broadcast") and the results in DNS records and reports ("pass",
"fail", "p=quarantine").

### What adapters and the connect flow write

The server's words about a provider: the catalog's cards and chips, an
adapter's fields and terms, the account's standing, the stepper and the
test's timeline.

| English | Tiếng Việt | Note |
|---|---|---|
| Production · Sandbox (an account's standing) | Gửi chính thức · Sandbox | "Gửi chính thức · không bị hạn chế" |
| Two-way · Pull only · Push only · Local only (suppression sync) | Hai chiều · Chỉ kéo về · Chỉ đẩy đi · Chỉ cục bộ | the Adapters tab |
| self-hosted | tự lưu trữ | "Postal (tự lưu trữ)" |
| Any SMTP server · Generic HTTP adapter | Máy chủ SMTP bất kỳ · Adapter HTTP chung | catalog names |
| Batch 50 · Quota discovery · Sandbox detection (chips) | Gửi lô 50 · Dò hạn mức · Phát hiện sandbox | |
| Dedicated IPs, paid · Pro plan | IP riêng, trả phí · gói Pro | the plan's name stays |
| read-only call | lệnh gọi chỉ đọc | "đã kiểm tra bằng một lệnh gọi chỉ đọc" |
| Round trip 2.4 s | Trọn vòng 2,4 giây | the test send |
| Mailpit (development) | Mailpit (môi trường phát triển) | |
| a price in an adapter's words | 0,10 US$ cho mỗi 1.000 email | decimal comma, dot for thousands, US$ after the amount |

A provider's names for its keys, plans, menus and stream types stay as the
provider writes them ("Server API token", "Private API key", "Account SID",
"Auth token", "Full access", "Broadcasts API", "Scale plan"), and so do its
event names ("Delivery", "SpamComplaint") and its error codes.

### What the sending engine, Deliverability and HostRepute write

The live monitor's incident panel (likely causes, recommendations and the
ways to continue), its notifications and feed, Deliverability's account
health and check runs, the HostRepute connection, route notes on a
contact's timeline and why a transactional message failed.

| English | Tiếng Việt | Note |
|---|---|---|
| "October Product Update" auto-paused · Outlook | “October Product Update” tự tạm dừng · Outlook | a campaign's name between “ ” |
| built-in daily check (blocklists) | lần kiểm tra hằng ngày có sẵn | without HostRepute |
| check run · lookups that didn't answer | lần kiểm tra · lượt tra cứu không có phản hồi | |
| connection errors · throttling errors · API errors | lỗi kết nối · lỗi giới hạn tốc độ · lỗi API | "Postal trả về lỗi kết nối trong 3 phút" |
| decision requested · waiting for a decision | đang chờ quyết định · đang chờ bạn quyết định | a held share |
| health check · still failing · answering again | kiểm tra tình trạng · vẫn lỗi · đã phản hồi lại | the outage log's lines |
| never-engaged · lapsed (contacts) | chưa từng tương tác · đã ngừng tương tác | |
| placement ("protects placement") | vị trí thư | as in Placement tests |
| Reputation dashboard (SES) | Bảng điều khiển uy tín | |
| Resume as is · Resume with changes · Cancel the Outlook share · Retry Postal · Move 1,204 to SES | Tiếp tục như cũ · Tiếp tục với thay đổi · Hủy phần của Outlook · Thử lại Postal · Chuyển 1.204 sang SES | the incident panel's ways to continue; "(khuyên dùng)" after the recommended one |
| route note: SES share of the 90/10 split · by rule · instead of · failed over to | Phần của SES trong phân bổ 90/10 · theo quy tắc · thay cho · đã chuyển dự phòng sang | |
| scope (a HostRepute token's) | phạm vi | the scopes' keys stay: monitor:read |
| Send held (a scheduled send) | Lần gửi bị tạm giữ | |
| status page (a provider's) | trang trạng thái | |
| web process (a self-hosted provider's) | tiến trình web | |

The providers' own replies stay as they come: SMTP codes and texts
("421 4.7.0 TSS04"), API errors, the outage log's error lines
("ECONNREFUSED mta.acme.io:443") and a bounce's diagnostic.

## Transactional templates

| English | Tiếng Việt |
|---|---|
| transactional template · transactional email | mẫu giao dịch · email giao dịch |
| draft · publish · v3 live | bản nháp · xuất bản · v3 đang chạy |
| version · Publish again | phiên bản · Xuất bản lại |
| variable · declare · required · optional | biến · khai báo · bắt buộc · không bắt buộc |
| Sensitive | Nhạy cảm |
| sample value · default | giá trị mẫu · mặc định |
| test data (set) | (bộ) dữ liệu thử |
| render · test renders | kết xuất · kết xuất thử |
| Checks · Ready to publish | Kiểm tra · Sẵn sàng xuất bản |
| breaking change | thay đổi gây lỗi |
| Subject · Preheader | Tiêu đề · Dòng xem trước |
| canvas | khung thiết kế |
| Visual · Text (views) | Trực quan · Văn bản |
| plain text | văn bản thuần |
| call (an API call) | lệnh gọi |
| Essential · Optional (categories) | Thiết yếu · Tùy chọn |
| autosaved 09:41 · conflict | tự lưu lúc 09:41 · xung đột |
| Import templates · Needs a manual fix | Nhập mẫu · Cần sửa thủ công |

"payload", "Liquid", "MJML", "Mustachio", "Handlebars", "cURL" and "JSON"
stay as they are.

## Composer

| English | Tiếng Việt |
|---|---|
| A/B test · variant | thử nghiệm A/B · biến thể |
| block · Blocks panel | khối · Khối |
| code view | chế độ mã |
| column · 2 columns | cột · 2 cột |
| compile · compiled HTML | biên dịch · HTML đã biên dịch |
| composer | trình soạn thảo ("Rời trình soạn thảo?") |
| dark mode · light | chế độ tối · sáng |
| footer · header (of an email) | chân thư · đầu thư |
| Inspector | Bảng thuộc tính |
| layer · layer tree | lớp · cây lớp |
| padding · border radius | khoảng đệm · bo góc |
| personalization token · Liquid tag | thẻ cá nhân hóa · thẻ Liquid |
| raw HTML | HTML thô |
| sample · winner (A/B) | nhóm thử · bản thắng |
| section (of an email) · saved section | phần · phần đã lưu |
| Send Readiness | Độ sẵn sàng gửi (the checks before a send, as a name; "runs again" = "được kiểm tra lại") |
| spam heuristic | dấu hiệu spam |
| bare URL · link text · says nothing (a link's text) | URL trần · chữ của liên kết · không nói lên điều gì |
| broken link · shortened link | liên kết hỏng · liên kết rút gọn (a URL shortener is Send Readiness's "dịch vụ rút gọn liên kết") |
| inline CSS | CSS nội tuyến (Gmail's clipping limit is Send Readiness's "ngưỡng cắt thư") |
| contact field · personalization token's fallback | trường liên hệ · giá trị dự phòng |
| plain-text version · postal address | phiên bản văn bản thuần · địa chỉ bưu chính |
| Line 3: (a compile or layout problem) · doesn't compile | Dòng 3: · không biên dịch được |
| title (of an article card, a post) | tựa đề ("tiêu đề" is the email's subject) |
| Email preferences link · View in browser link | Liên kết tùy chọn email · Liên kết xem trên trình duyệt |
| starter (template) | mẫu khởi đầu |
| starters: Blank · Weekly digest · Product update · Plain text · Announcement · Product grid · Blog digest | Trống · Bản tin hằng tuần · Cập nhật sản phẩm · Văn bản thuần · Thông báo · Lưới sản phẩm · Bản tin blog |

MJML tags (mj-section, mj-body) stay as they are. Section names in the
server's sentences use the composer's words: đầu thư, chân thư, hero,
2 cột, văn bản, hình ảnh, nút. A block's sample content
is written for the email's readers, in the app language: the first-name
fallback "there" is "bạn".

## Automations

| English | Tiếng Việt |
|---|---|
| A/B split · weight | Chia A/B · tỷ trọng |
| automation (one, or the section) | tự động hóa ("Tự động hóa mới") |
| Branch · Yes · No | Rẽ nhánh · Có · Không |
| Delay (step) · Wait 10 minutes | Trì hoãn · Chờ 10 phút |
| double opt-in · double opt-in confirmed only | xác nhận hai bước · chỉ người đã hoàn tất xác nhận hai bước |
| edge (a connection on the canvas) | đường nối |
| exit · goal · goal rate | thoát · mục tiêu · tỷ lệ đạt mục tiêu |
| fail over · failover | chuyển dự phòng |
| Filter (step) | Lọc |
| flow · in the flow · leave the flow | quy trình · trong quy trình · rời quy trình ("quy trình tự động" where the kind needs saying; "luồng" is only a stream: "Luồng Tự động hóa") |
| List action | Thao tác danh sách |
| method (HTTP) | phương thức |
| override (a route) | ghi đè; "route override" on a step is "tuyến riêng" |
| re-entry | vào lại |
| retire · retiring (a version) | ngừng hoạt động · đang ngừng |
| send window | khung giờ gửi |
| signing secret · write-only | khóa bí mật để ký · chỉ ghi |
| stats · Live stats | số liệu · Số liệu trực tiếp |
| step · trigger · action · condition | bước · trình kích hoạt · hành động · điều kiện |
| web push | thông báo đẩy (trên web) |
| path (of a branch or split) · connection | nhánh · đường nối |
| starters: Welcome series · Re-engagement · Onboarding · Abandoned cart | Chuỗi chào mừng · Tái tương tác · Hướng dẫn bắt đầu · Giỏ hàng bị bỏ dở |
| Send Readiness content checks | kiểm tra nội dung của Độ sẵn sàng gửi |
| webhook delivery log · failed calls | nhật ký gửi webhook · lệnh gọi thất bại |
| route label: SES, failover Postmark | SES, dự phòng Postmark |
| Removed provider | Nhà cung cấp đã xóa |
| webinar | hội thảo trực tuyến |
| version summary: created from a blank canvas · created from the X template · copied from X · first published version | tạo từ khung trống · tạo từ mẫu X · nhân bản từ X · phiên bản xuất bản đầu tiên |

## Approvals

| English | Tiếng Việt |
|---|---|
| after exclusions | sau khi loại trừ |
| approval inbox | hộp thư phê duyệt |
| approval request · approver · author | yêu cầu phê duyệt · người phê duyệt · tác giả |
| Approve · Reject · Request changes | Phê duyệt · Từ chối · Yêu cầu thay đổi |
| Bypassed · emergency bypass | Đã bỏ qua · bỏ qua khẩn cấp |
| comment · Discussion · evidence | nhận xét · Thảo luận · bằng chứng |
| Contacts in flight | Liên hệ đang trong quy trình |
| Invalidated · Approval invalidated | Mất hiệu lực · Phê duyệt không còn hiệu lực |
| median wait | thời gian chờ trung vị (a median, not an average: not "trung bình") |
| Pending · Rejected | Đang chờ · Bị từ chối |
| policy · rule · Rules matched | chính sách · quy tắc · Quy tắc khớp |
| Recheck | Kiểm tra lại |
| snapshot | bản chụp |
| after approval (a change) | sau khi được phê duyệt ("số người nhận đã thay đổi +14% sau khi được phê duyệt"; said once) |
| rules: audience ≥ 10,000 · warming-up provider · new provider · author lacks send · list requires approval · client sign-off | người nhận ≥ 10.000 · nhà cung cấp đang khởi động · nhà cung cấp mới · tác giả không có quyền gửi · danh sách cần phê duyệt · phê duyệt của khách hàng |
| automation publish · post publish · page publish · template publish (rules) | xuất bản tự động hóa · xuất bản bài viết · xuất bản trang · xuất bản mẫu |
| a send held · suppressed | lần gửi bị tạm giữ · bị chặn gửi |
| online · active 2 h ago (an approver) | đang trực tuyến · hoạt động 2 giờ trước |
| Best time within 24 h of | Giờ tốt nhất trong vòng 24 giờ từ |
| Only people can… (not agents or keys) | Chỉ con người mới… |

## Content languages

| English | Tiếng Việt |
|---|---|
| language version · Source | phiên bản ngôn ngữ · Bản gốc |
| Default language · Fallback language | Ngôn ngữ mặc định · Ngôn ngữ dự phòng |
| Up to date · Outdated · Draft · Missing · Held | Đã cập nhật · Lỗi thời · Bản nháp · Thiếu · Tạm giữ |
| Mark as translated · Mark as up to date | Đánh dấu đã dịch · Đánh dấu đã cập nhật |
| readers (of a language) | người đọc |
| page copy · built-in labels | nội dung trang · nhãn có sẵn |
| Results per language · click map | Kết quả theo ngôn ngữ · bản đồ lượt nhấp |
| no language · other languages (results) | không có ngôn ngữ · ngôn ngữ khác |
| compare view's blocks: Header · Footer · Raw HTML · Social links · Two cards · 3 columns · Hero image · Button · Intro · The email | Đầu thư · Chân thư · HTML thô · Liên kết mạng xã hội · Hai thẻ · 3 cột · Hình ảnh hero · Nút · Phần mở đầu · Toàn bộ email |
| Headline (a block) · no alt text | Dòng tít ("Đề mục" is a heading) · không có văn bản thay thế |
| version labels: copied from · layout of · applied from · block copied from · changed by an agent · edited in the flow editor · translated from | chép từ · bố cục của · áp dụng từ · khối chép từ · do tác tử thay đổi · sửa trong trình chỉnh sửa quy trình · dịch từ |
| a language's changes after approval: added · marked translated · back to draft · sent again · edited | mới thêm · đã đánh dấu là đã dịch · trở lại bản nháp · được gửi lại · đã sửa |
| copy (a form's or page's own text) · Copy in | nội dung · Nội dung bằng |
| Live · Incomplete · Not set up (copy) | Đang dùng · Chưa đủ · Chưa thiết lập |
| string (text to write) | chuỗi |
| visitors | khách truy cập ("Khách truy cập dùng {language}") |

A content language's name comes from the browser's or CLDR's names in
the app language ("Tiếng Tây Ban Nha"), capitalized as they write it; the
app's own picker names each language in itself.

## Conditional content

| English | Tiếng Việt |
|---|---|
| combination (of blocks) · version | tổ hợp · phiên bản |
| conditional block | khối có điều kiện |
| Gmail clips · clipping size | Gmail cắt bớt · ngưỡng cắt bớt |
| includes · excludes | bao gồm · không bao gồm |
| is · isn't · contains · is set · is empty | là · khác · chứa · đã có giá trị · đang trống (as automations' conditions) |
| is at least · is at most · is between … and … | tối thiểu là · tối đa là · nằm trong khoảng … đến … |
| is greater than · is less than · is before · is after | lớn hơn · nhỏ hơn · trước · sau |
| is in · is not in | thuộc · không thuộc |
| Match ALL / ANY of these | Khớp TẤT CẢ / BẤT KỲ điều kiện sau (the uppercase words stay uppercase) |
| operator | toán tử |
| Preview as · Show why | Xem trước dưới dạng · Hiện lý do |
| Visibility (who sees a block) | Hiển thị |

## Accessibility

| English | Tiếng Việt |
|---|---|
| Accessibility | Khả năng tiếp cận |
| issue · serious · moderate | lỗi · nghiêm trọng · trung bình |
| alt text · decorative image | văn bản thay thế · hình trang trí |
| contrast · Passes AA · Fails AA | độ tương phản · Đạt AA · Không đạt AA |
| Heading 1 · Paragraph · Text role | Đề mục 1 · Đoạn văn · Vai trò văn bản |
| layout table | bảng bố cục |
| screen reader | trình đọc màn hình |
| Warn · Block (the policy) | Cảnh báo · Chặn |
| skipped heading level · card titles | cấp đề mục bị bỏ qua · tựa đề thẻ |

"Heading" is "Đề mục", never "Tiêu đề", which is an email's subject.
Content languages in a sentence use their standard Vietnamese names, with
"tiếng" in lower case: "tiếng Đức", "tiếng Tây Ban Nha".

## Earn

Payments, paid subscriptions, offers, gifts, sponsors and revenue. A
"paid member" is a **hội viên trả phí**, never a "thành viên", which is a
team member. A reader's Stripe subscription is a **thuê bao**, never an
"đăng ký": "hủy đăng ký" is unsubscribing.

| English | Tiếng Việt |
|---|---|
| access (paid) · With access | quyền truy cập · Có quyền truy cập |
| access rank · Rank 2 | hạng truy cập · Hạng 2 |
| Access rules | Quy tắc truy cập |
| Active · Trialing · Past due · Cancels soon · Canceled · Ended (paid status) | Đang hoạt động · Đang dùng thử · Quá hạn · Sắp hủy · Đã hủy · Đã kết thúc |
| archive · Archived (a tier) | lưu trữ · Đã lưu trữ |
| attribution · attribution window · credited to (an email) | phân bổ · khung thời gian phân bổ · ghi nhận cho |
| benefit (of a tier) | quyền lợi |
| billing email · billing period · Manage billing | email thanh toán · chu kỳ thanh toán · Quản lý thanh toán |
| book · booking · Booked (a sponsor in an issue) | đặt chỗ · lượt đặt chỗ · Đã đặt |
| buyer · recipient (of a gift) | người mua · người nhận |
| cancel page · gift page · upgrade page · welcome page | trang hủy · trang quà tặng · trang nâng cấp · trang chào mừng |
| charge (a card) | trừ tiền (từ thẻ); "nothing is charged" is "không tính phí" |
| checkout · test checkout | thanh toán · thanh toán thử nghiệm ("Stripe Checkout" is a name) |
| churn · churn rate | rời bỏ · tỷ lệ rời bỏ |
| click token | token lượt nhấp |
| Complimentary · complimentary access | Được miễn phí · quyền truy cập miễn phí |
| conversion · conversion API | chuyển đổi · API chuyển đổi |
| coupon · promotion code · code | phiếu giảm giá · mã khuyến mãi · mã |
| creative (a sponsor's) | nội dung tài trợ |
| customer portal | cổng khách hàng |
| deliver · delivery date (a gift) | giao · ngày giao |
| discount · 20% off | giảm giá · giảm 20% |
| dispute (a card payment's) | tranh chấp |
| Earn's tabs: Overview · Plans & offers · Paid members · Gifts · Sponsors · Products · Settings | Tổng quan · Gói và ưu đãi · Hội viên trả phí · Quà tặng · Nhà tài trợ · Sản phẩm · Cài đặt |
| expansion · contraction · reactivated (MRR movement) | mở rộng · thu hẹp · kích hoạt lại |
| Free (the card) · free subscriber | Miễn phí · người đăng ký miễn phí |
| free-to-paid | miễn phí sang trả phí |
| gift · gift option | quà tặng · tùy chọn quà tặng |
| headline (an upgrade page's, a creative's) | dòng tiêu đề; a subject is "tiêu đề", a heading "đề mục" |
| highlight (a tier) · Most popular | làm nổi bật · Phổ biến nhất |
| hold · on hold · release (an issue) | giữ chỗ · đang giữ chỗ · bỏ giữ chỗ |
| in person (a gift handed over) | trao tận tay |
| issue (of a newsletter: #143) | số (số #143); an accessibility issue is "lỗi" |
| legacy price | giá cũ |
| lifetime value | giá trị trọn đời |
| live · test (a key, a mode, payments) | chính thức · thử nghiệm; "no real payments" is "không có thanh toán thật" |
| live (a page, an offer or a template version that's running) | đang chạy |
| map (a Stripe price to a tier) | gán |
| member email · membership | email hội viên · gói hội viên |
| merchant of record | bên bán hàng thay bạn |
| monthly · yearly · Monthly · Yearly (the choice) · /month | hằng tháng · hằng năm · Theo tháng · Theo năm · /tháng |
| MRR · ARR · MRR movement | MRR · ARR · biến động MRR |
| Needs you (Earn's overview) | Cần bạn xử lý |
| normal price (without an offer) | giá thường |
| offer · signup · retention · referral reward | ưu đãi · ưu đãi đăng ký · ưu đãi giữ chân · phần thưởng giới thiệu |
| Overdue · Unpaid · Paid (a sponsor's payment) | Quá hạn · Chưa thanh toán · Đã thanh toán |
| paid member | hội viên trả phí |
| paid status · paid-only | trạng thái trả phí · chỉ dành cho hội viên trả phí |
| Paid subscriptions (what Earn sells) | Gói trả phí |
| paid-through date | ngày hết kỳ đã trả |
| payment provider | nhà cung cấp thanh toán |
| payment terms (a sponsor's) | thời hạn thanh toán |
| payouts | rút tiền về ngân hàng |
| plan · Plans & offers | gói · Gói và ưu đãi |
| Public · Hidden (a tier's visibility) | Công khai · Ẩn |
| receipt · invoice | biên lai · hóa đơn |
| record (a sponsor's payment, by hand) | ghi nhận |
| redeem · redeem link (a gift) | nhận · liên kết nhận quà |
| redeem · redemption (an offer code) | dùng · lượt dùng |
| refund | hoàn tiền |
| renew · renewal | gia hạn |
| restricted key · secret key | khóa hạn chế · khóa bí mật |
| results so far · final (a sponsor report's) | kết quả tạm thời · chốt |
| revenue · income | doanh thu · thu nhập |
| roll (a Stripe key) | xoay vòng |
| seller ("you're the seller") | bên bán |
| signing secret (a webhook's) · endpoint | khóa bí mật để ký (as Automations has it) · endpoint |
| Since · Member since (a paid member's) | Tham gia · Hội viên từ |
| slot · placement | vị trí |
| sponsor · sponsorship · Sponsored (the block's label) | nhà tài trợ · tài trợ · Được tài trợ |
| sponsor report | báo cáo tài trợ |
| subscription (a reader's, billed by Stripe) | thuê bao |
| Terms of sale · Support address | Điều khoản bán hàng · Địa chỉ hỗ trợ |
| tier · paid tier | gói · gói trả phí |
| timeline (a paid member's) | dòng thời gian |
| trial · free trial | dùng thử · dùng thử miễn phí |
| upgrade · Upgrade block | nâng cấp · khối Nâng cấp |
| webhook silent · gap check | webhook không có tin · kiểm tra sự kiện sót |
| write-only · sealed (a key, a secret) | chỉ ghi · niêm phong |

Some English messages choose a word for English grammar only: an article
(`{article, select, an {an} other {a}} {tier}`) or a possessive
(`{ends, select, s {{sponsor}’} other {{sponsor}’s}}`). Vietnamese keeps the
argument and puts the name in its `other` branch: `{ends, select, other
{{sponsor}}}`. Where the English counts with `#`, keep
`{count, plural, other {# …}}` rather than a bare `{count}`, so the number
is grouped by the region format (48.213).

## Website and pages

The publication website, its posts, the page builder and the public pages'
own words. Words a reader sees follow the built-in labels of the public
pages (`internal/pagelabels`): "bản tin", "số", "Lưu trữ", "Tùy chọn email".

| English | Tiếng Việt | Note |
|---|---|---|
| About (page, template) | Giới thiệu | |
| Access (Website › Access) | Quyền truy cập | |
| archive · Archive (the page, the link) | kho lưu trữ · Lưu trữ | the archive as a whole; "trang lưu trữ" when the page itself is meant |
| band (subscribe band) | dải | a full-width strip of the home page |
| block · layers | khối · lớp | the page builder's |
| built-in page | trang có sẵn | like "nhãn có sẵn" |
| byline · By {name} | tác giả · Tác giả: {name} | |
| canonical URL | URL chính tắc | as Google's tools say it |
| card (paywall card, a post's card) | khung | "thẻ" is a tag |
| certificate | chứng chỉ | |
| countdown | đếm ngược | |
| cover (image) | ảnh bìa | |
| excerpt | đoạn trích | |
| FAQ (block) | Hỏi đáp | |
| featured · Feature on home | nổi bật · Nổi bật trên trang chủ | |
| feed · RSS feed | nguồn cấp · nguồn cấp RSS | |
| free preview · Free preview ends here (the post's block) | phần xem trước miễn phí · Hết phần miễn phí | |
| Hero (block) | Hero | the block's name, as in the composer |
| home page | trang chủ | |
| issue (of a newsletter) | số | "48 số", "Đọc số mới nhất" |
| lead magnet | quà tặng đăng ký | a free file for confirmed subscribers |
| live (the website) · live (a page, a form, a version) | đang hoạt động · đang chạy | as Earn and the signup forms have it |
| member · free member · paid member (of the site) · membership | hội viên · hội viên miễn phí · hội viên trả phí · gói hội viên | a reader who signs in, as Earn has it; "thành viên" is a team member |
| meter | hạn mức đọc miễn phí | "Hạn mức đọc" as a heading |
| navigation · header · footer | điều hướng · đầu trang · chân trang | |
| page · Pages | trang · Trang | |
| page builder | trình dựng trang | |
| Pay what you want | Trả tùy ý | |
| paywall | tường phí | |
| post · web-only post | bài viết · bài viết chỉ trên web | |
| publish · unpublish · unschedule | xuất bản · gỡ xuất bản · hủy lên lịch | Unpublished: Đã gỡ xuất bản |
| redirect | chuyển hướng | |
| referrer (where views come from) | nguồn truy cập | a person who refers is "người giới thiệu" |
| registration (event) | ghi danh | "đăng ký" is a signup |
| sales page | trang bán hàng | |
| signups · upgrades · views · visitors | lượt đăng ký · lượt nâng cấp · lượt xem · khách truy cập | |
| site · website | trang web | |
| sitemap | sơ đồ trang web | |
| slug | đường dẫn | the page's or tag's part of the address |
| social image | ảnh chia sẻ | the image a shared link shows |
| store product (block) | sản phẩm cửa hàng | |
| tab (a panel's: the Web tab) | tab | "thẻ" is a tag |
| tag | thẻ | |
| tagline | khẩu hiệu | |
| testimonial | lời nhận xét | |
| theme | giao diện | |
| tier · paid tier | gói · gói trả phí | as Earn has it; a tier's name, such as "Insider", stays as it is |
| title (a post's, a page's, the site's) · SEO title | tựa đề · tựa đề SEO | as an archived issue's; "tiêu đề" is an email's subject ("tiêu đề thư" beside a title) |
| visibility · Public · Members · Paid | chế độ hiển thị · Công khai · Hội viên · Trả phí | |
| visitor | khách truy cập | someone who isn't signed in |
| webinar | hội thảo trực tuyến | |

## Team

| English | Tiếng Việt |
|---|---|
| Team | Nhóm |
| Invite member | Mời thành viên |
| Members | Thành viên |
| Everyone with access to Acme Newsletter | Mọi người có quyền truy cập Acme Newsletter |
| Search name or email | Tìm tên hoặc email |
| Any role · Any status · Any 2FA | Mọi vai trò · Mọi trạng thái · Mọi trạng thái 2FA |
| Role · scope | Vai trò · phạm vi |
| Sign-in | Đăng nhập |
| Last active | Hoạt động gần nhất |
| Manage · Reactivate | Quản lý · Kích hoạt lại |
| Password · passkey | Mật khẩu · khóa truy cập |
| just now · yesterday | vừa xong · hôm qua |
| 2 selected · Bulk: | Đã chọn 2 · Hàng loạt: |
| Change role · Require 2FA now · Deactivate | Đổi vai trò · Bắt buộc 2FA ngay · Vô hiệu hóa |
| Required · off, 3 days of grace left | Bắt buộc · đang tắt còn 3 ngày gia hạn |
| deactivated Sep 30, 2026 | vô hiệu hóa ngày 30 thg 9, 2026 |
| Pending invitations · 1 | Lời mời đang chờ · 1 |
| Links last 7 days by default | Liên kết có hiệu lực 7 ngày theo mặc định |
| Invitee · Sent · Expires | Người được mời · Đã gửi · Hết hạn |
| invited by Linh Nguyen | Linh Nguyen mời |
| Beta Program list only | chỉ danh sách Beta Program |
| in 5 days | sau 5 ngày |
| Resend · Copy link · Revoke | Gửi lại · Sao chép liên kết · Thu hồi |

## Language and region

| English | Tiếng Việt |
|---|---|
| Language & region | Ngôn ngữ và khu vực |
| App language | Ngôn ngữ ứng dụng |
| Instance default (English) | Mặc định của hệ thống (English) |
| Region format | Định dạng khu vực |
| Follows the app language | Theo ngôn ngữ ứng dụng |
| Time · 24-hour · 12-hour | Giờ · 24 giờ · 12 giờ |
| Week starts on · Monday · Sunday | Tuần bắt đầu vào · Thứ Hai · Chủ Nhật |
| From the region format | Theo định dạng khu vực |
| Preview | Xem trước |
| Help translate ReputeCast | Giúp dịch ReputeCast |
| Save · Undo | Lưu · Hoàn tác |
| Profile | Hồ sơ |
| timezone | múi giờ |
| quiet hours | giờ yên lặng |
| Settings › General | Cài đặt › Chung |
| Invitation language | Ngôn ngữ lời mời |
| Default language · Default region format | Ngôn ngữ mặc định · Định dạng khu vực mặc định |

Languages are always named in their own language, in every interface
language: English, Tiếng Việt, Deutsch (Deutschland). "ReputeCast standard"
is a name and stays in English.

## Account and security

| English | Tiếng Việt |
|---|---|
| Full name | Họ và tên |
| Upload photo · Remove | Tải ảnh lên · Xóa |
| Change email · verification link | Đổi email · liên kết xác minh |
| Current password · New password | Mật khẩu hiện tại · Mật khẩu mới |
| security key · platform passkey | khóa bảo mật · khóa truy cập trên thiết bị |
| Rename · Keep it | Đổi tên · Giữ lại |
| Time-based codes (TOTP) | Mã theo thời gian (TOTP) |
| Set up · Replace app | Thiết lập · Thay ứng dụng |
| Regenerate codes | Tạo lại mã |
| 7 of 10 remain | Còn 7 trên 10 mã |
| Connected sign-ins | Đăng nhập đã liên kết |
| Link · Unlink · Managed | Liên kết · Hủy liên kết · Được quản lý |
| session · Active sessions · This session | phiên · Phiên đang hoạt động · Phiên này |
| Sign out all other sessions | Đăng xuất mọi phiên khác |
| Device · Browser · IP · location (approx.) | Thiết bị · Trình duyệt · IP · vị trí (ước tính) |
| My workspaces · Leave workspace | Không gian làm việc của tôi · Rời không gian làm việc |
| In-app · All events | Trong ứng dụng · Tất cả sự kiện |
| MCP agent | tác tử MCP |

## Instance admin, keys, webhooks and the audit log

| English | Tiếng Việt | Note |
|---|---|---|
| actor (audit) · Any actor type | người thực hiện · Mọi loại người thực hiện | who did it: a member, a key, an agent or the system |
| agent capability: Allow · Deny · Approve · Pause only (Settings › MCP) | Cho phép · Từ chối · Phê duyệt · Chỉ tạm dừng | the matrix's cells; “Phê duyệt” as the caption quotes it |
| All retained (date range) | Toàn bộ thời gian lưu giữ | |
| app registration (Entra) · app integration (Okta) | đăng ký ứng dụng · tích hợp ứng dụng | “the Entra app registration”: “đăng ký ứng dụng Entra” |
| batch (of streamed entries) | lô | as in "endpoint theo lô" |
| break-glass Owner | Chủ sở hữu có tài khoản khẩn cấp | never "Chủ sở hữu khẩn cấp" |
| break-glass password · break-glass passkey | mật khẩu khẩn cấp · khóa truy cập khẩn cấp | what a break-glass Owner keeps while SSO is enforced |
| claim · Client ID · client secret · Scopes (OIDC) | claim · Client ID · client secret · Scope | named as the identity provider's console names them |
| danger zone | Vùng nguy hiểm | |
| delivery (webhook) · Delivery log | lần gửi · Nhật ký gửi | |
| Development build | Bản dựng phát triển | |
| digest (email) · Hourly digest · Daily digest | email tổng hợp · Tổng hợp mỗi giờ · Tổng hợp hằng ngày | notification preferences |
| discovery · issuer · Issuer URL (OIDC) | discovery · nhà phát hành · URL nhà phát hành (issuer) | |
| Effective permissions | Quyền thực tế | a member's page; the column is "Thực tế" |
| endpoint · payload | endpoint · payload | not translated |
| enforce (SSO, two-factor) · enforcement | bắt buộc · chế độ bắt buộc | "Enforce SSO for acme.io": "Bắt buộc SSO cho acme.io" |
| entry (audit) · Entry details | mục · Chi tiết mục nhật ký | |
| Failing · Paused (an endpoint) | Đang lỗi · Đã tạm dừng | "Healthy" is "Ổn định" |
| Force sign-out | Buộc đăng xuất | |
| Generic OIDC (a preset) | OIDC chung | the IdP's short name in sentences is “IdP” |
| Given (approvals a member gave) | Đã xét duyệt | |
| group mapping · group object ID | ánh xạ nhóm · object ID của nhóm | |
| hourly cleanup | lần dọn dẹp hằng giờ | |
| identity provider | nhà cung cấp danh tính | "IdP" stays |
| immutable (audit entries) | không thể sửa đổi | |
| instance-level (entries) · Instance level | cấp hệ thống · Cấp hệ thống | |
| JIT account creation | tạo tài khoản tự động (JIT) | |
| just-in-time account · JIT default role | tài khoản JIT · vai trò mặc định của JIT | |
| Locked (sign-in) · Unlock sign-in | Bị khóa · Mở khóa đăng nhập | a lockout is "khóa đăng nhập" |
| Make instance admin · Remove instance admin | Đặt làm quản trị viên hệ thống · Gỡ quyền quản trị hệ thống | |
| overage link (Entra's groups) | liên kết overage | Entra's own word |
| preset (an SSO provider's, a key's) | cấu hình sẵn | |
| proposal (an agent asked a person) · proposal recorded | đề xuất · đã ghi đề xuất | “Ops asked to pause …”: “Ops đề xuất tạm dừng …” |
| provider status: degraded · down | đang suy giảm · đang gián đoạn | as Channels and SMS' Suy giảm · Gián đoạn |
| re-authentication | xác thực lại | |
| Redirect URI | URI chuyển hướng | |
| reference ID · request ID | mã tham chiếu · mã yêu cầu | |
| result: Allowed · Denied · Error | Được phép · Bị từ chối · Lỗi | |
| retention | thời gian lưu giữ | |
| revoke · rotate · replace (a key or secret) | thu hồi · xoay vòng · thay | |
| secret · signing secret | khóa bí mật · khóa bí mật để ký | "secrets" in general: thông tin bí mật |
| Send approvals (Settings › General) | Phê duyệt trước khi gửi | |
| Sending (notification category) | Gửi thư | so it isn't read as the status "Đang gửi" |
| SIEM or warehouse | SIEM hoặc kho dữ liệu | |
| single sign-on | đăng nhập một lần | "SSO" stays |
| stream (audit entries, noun) · stream (verb) · streaming endpoint | luồng · truyền · endpoint truyền luồng | the System stream is "luồng Hệ thống" |
| test sign-in | đăng nhập thử | |
| Then allowed | Sau đó được phép | a sign-in that was rejected, then allowed |
| token (an agent's) | token | "agent token": "token của tác tử" |
| tool (MCP) | công cụ | tool names (get_campaign_report) stay as they are |
| Trusted from Okta (two-factor) | Tin cậy từ Okta | the IdP enforces the second factor |
| typed confirmation | xác nhận bằng cách nhập tên | |
| Up to date · Update available (a release) | Đã là bản mới nhất · Có bản cập nhật | language versions keep "Đã cập nhật" |
| user (an account on the instance) | người dùng | a workspace's people are "thành viên" |
| Workspace ID | Mã không gian làm việc | |
| workspace switcher | bộ chuyển không gian làm việc | |

## Channels and SMS

| English | Tiếng Việt |
|---|---|
| 10DLC · local · mobile · toll-free (number kinds) | 10DLC · địa phương · di động · miễn cước ("số di động", "số miễn cước") |
| adapter · manifest · contract test suite | adapter · manifest · bộ kiểm thử hợp đồng (developer words, kept) |
| auto-pause · Auto-paused | tự động tạm dừng · Tự tạm dừng |
| brand (in a 10DLC registration) · brand prefix | thương hiệu · tiền tố thương hiệu |
| budget · spend · price per segment | ngân sách · chi tiêu · giá mỗi đoạn tin |
| carrier | nhà mạng |
| channel · Email · SMS · Push · WhatsApp | kênh · Email · SMS · Thông báo đẩy · WhatsApp |
| consent · SMS consent · consent text · evidence | sự đồng ý · đồng ý nhận SMS · nội dung đồng ý · bằng chứng |
| credentials · Live · Test credentials | thông tin xác thực · Chính thức · Thông tin xác thực thử |
| delivery receipt | báo nhận (what a carrier reports back; "Last receipt" is "Báo nhận gần nhất") |
| fallback (of a route) | dự phòng |
| filtered by carriers | bị nhà mạng lọc |
| held · skipped | bị tạm giữ · bị bỏ qua |
| inbound (text) · reply | tin đến · tin trả lời |
| keyword | từ khóa (STOP, START, HELP and YES stay in English: people text them) |
| landline · not in service · invalid number · unreachable | số cố định · ngừng hoạt động · số không hợp lệ · không liên lạc được |
| link preview | bản xem trước liên kết (a messaging app's machine click, never a person) |
| local time · "Berlin local" (a reply's time) | giờ địa phương · "giờ Berlin" |
| opt in · opt-in confirmation | đăng ký nhận · tin xác nhận đăng ký |
| opt out · opt-outs · opt-out text · opt-out reply | hủy nhận · lượt hủy nhận · câu hủy nhận · tin trả lời hủy nhận |
| opt-out sync · Two-way · Read only | đồng bộ hủy nhận · Hai chiều · Chỉ đọc |
| outside quiet hours (a text waits or is skipped) | ngoài khung giờ gửi (the window texts may go out in, not "ngoài giờ yên lặng") |
| provider status: Healthy · Degraded · Down | Ổn định · Suy giảm · Gián đoạn |
| quiet hours · sending window | giờ yên lặng · khung giờ gửi |
| raw event · normalized | sự kiện gốc · đã chuẩn hóa |
| registration · registered | đăng ký · đã đăng ký |
| route · Routing & rules · route tester | tuyến · Định tuyến và quy tắc · kiểm tra tuyến |
| segment (of a text) | đoạn tin (the billed part of 160 GSM-7 or 70 UCS-2 characters; never "phân khúc", an audience segment) |
| sender ID · alphanumeric sender ID | ID người gửi · ID người gửi dạng chữ và số |
| short-link domain · public shortener | tên miền liên kết rút gọn · dịch vụ rút gọn công khai |
| signature · signing secret | chữ ký · khóa bí mật ký |
| status callback (Twilio) | callback trạng thái |
| suppression list (SMS) | danh sách chặn gửi SMS |
| test text · test number | tin nhắn thử · số thử |
| text (a message) · to text | tin nhắn · nhắn tin |
| Text message terms | Điều khoản tin nhắn |
| throughput · 14 texts a second | thông lượng · 14 tin nhắn mỗi giây |

"GSM-7", "UCS-2", "Unicode", "VoIP" and "emoji" stay as they are. A
phone number, a sender ID and a country code are data: never translated.

## Web push and WhatsApp

| English | Tiếng Việt |
|---|---|
| access token · permanent access token · system user | mã truy cập · mã truy cập vĩnh viễn · người dùng hệ thống |
| Allow · Not now · Granted (a prompt's numbers) | Cho phép · Để sau · Đã cấp quyền |
| anonymous · linked (a subscription) | ẩn danh · đã liên kết (với liên hệ) |
| app secret · verify token · callback URL · phone number ID | khóa bí mật ứng dụng · mã xác minh · URL callback · ID số điện thoại |
| banner (a notification's style on a desktop) | biểu ngữ |
| Bell button · Dialog · Slide-in bar (prompt styles) | Nút chuông · Hộp thoại · Thanh trượt vào |
| business account · display name | tài khoản doanh nghiệp · tên hiển thị ("WhatsApp Business Account" is a name) |
| category (of a template): Marketing · Utility · Authentication | danh mục: Tiếp thị · Tiện ích · Xác thực |
| conversation · customer service window | cuộc hội thoại · khung chăm sóc khách hàng (short: "khung 24 giờ", never the quiet-hours "khung giờ gửi") |
| expiry · Expires after | thời hạn · Hết hạn sau |
| free-form reply · quick reply | tin trả lời tự do · trả lời nhanh |
| frequency cap | giới hạn tần suất |
| header · body · footer · buttons (of a template) | phần đầu · phần thân · phần chân · nút |
| Home Screen (iPhone, iPad) | Màn hình chính |
| icon · badge · image (of a notification) | biểu tượng · huy hiệu · hình ảnh |
| In review · In appeal · submit (a template) | Đang xét duyệt · Đang kháng nghị · gửi duyệt |
| messaging limit · rolling 24 hours | giới hạn nhắn tin · 24 giờ gần nhất |
| notification (one push message) | thông báo |
| permission prompt (the browser's own) | hộp xin quyền (của trình duyệt) |
| prompt · prompt designer | lời nhắc · trình thiết kế lời nhắc |
| publication site | trang web của ấn phẩm |
| push (in a sentence) · Push (the channel) | thông báo đẩy · Thông báo đẩy |
| push service (the browser maker's) | dịch vụ đẩy |
| quality rating · High · Medium · Low | xếp hạng chất lượng · Cao · Trung bình · Thấp |
| rate card · Edit rates | bảng giá · Sửa bảng giá |
| read receipts | thông báo đã xem (WhatsApp's own Vietnamese) |
| Rotate keys · public key · private key | Xoay vòng khóa · khóa công khai · khóa riêng tư |
| Sealed | Đã niêm phong |
| Shown (a count) · Shown (an event) · Gone · removed | Lượt hiển thị · Đã hiển thị · Hết hạn · đã xóa |
| site (where readers subscribe) · scope (of a service worker) | trang · phạm vi |
| snippet | đoạn mã |
| subscriber · subscription | người đăng ký · lượt đăng ký |
| Sync now | Đồng bộ ngay |
| Tapped (a WhatsApp button) | Đã nhấn |
| template (WhatsApp) | mẫu (mẫu tin nhắn where it must be explicit) |
| token (that a link carries to know the reader) | mã nhận diện (never "mã thông báo") |

"service worker", "VAPID", "Meta", "Graph API" and Meta's own screens and
paths (WhatsApp Manager, Meta for Developers › … › Configuration) stay as
they are.

## Surveys and polls

| English | Tiếng Việt |
|---|---|
| 0–10 scale · Emoji scale · faces | thang 0–10 · thang emoji · khuôn mặt |
| answer · answer page | câu trả lời · trang trả lời |
| Breakdown · Break down by | Phân tích · Phân tích theo |
| completion rate · drop-off · started · completed | tỷ lệ hoàn thành · bỏ dở · đã bắt đầu · đã hoàn thành |
| Embedded · Popup · Hosted page (form kinds) | Nhúng · Bật lên · Trang trên ReputeCast |
| link scanner · security gateway · prefetching proxy | trình quét liên kết · cổng bảo mật · proxy tải trước (machine visits, never a person) |
| Not likely · Very likely (an NPS scale's ends) | Không có khả năng · Rất có khả năng |
| Open · Closed (a survey or a poll) | Đang mở · Đã đóng |
| Owner (the forms list's column) | Người phụ trách (not the role "Chủ sở hữu") |
| poll (the block) · a poll | Bình chọn · cuộc bình chọn |
| promoters · passives · detractors | người ủng hộ · người trung lập · người phản đối ("NPS" and "Net Promoter Score" stay) |
| question · respondent · response | câu hỏi · người trả lời · lượt trả lời |
| submission | lượt gửi |
| survey | khảo sát |
| Top answer | Được chọn nhiều nhất |
| vote · voter | lượt bình chọn · người bình chọn |

## Notifications

| English | Tiếng Việt |
|---|---|
| digest (recurring) · issue (of a digest) | bản tin định kỳ · kỳ bản tin |
| event type (a row of Notification preferences) | loại sự kiện |
| Everyone with campaigns:pause | Mọi người có quyền campaigns:pause |
| paid tier (Earn) | hạng thành viên |
| placement test · seed mailbox | kiểm tra vị trí hộp thư · hộp thư mẫu |
| Sending (the notification center's tab) | Gửi thư |
| verification credits | credit xác minh |

## Agency accounts

| English | Tiếng Việt |
|---|---|
| Account manager · Agency manager (staff levels) | Phụ trách khách hàng · Quản lý agency |
| agency · Agency (the area) · agency mode | agency · Agency · chế độ agency (Vietnamese marketing says "agency") |
| assign · assignment | phân công · việc phân công |
| client · client workspace | khách hàng · không gian làm việc của khách hàng |
| client people (a client's own people) | người của khách hàng (never agency staff) |
| Client approver · Client viewer (roles) | Người duyệt phía khách hàng · Người xem phía khách hàng |
| client sign-off | phê duyệt của khách hàng |
| copy (of a library item) · behind | bản sao · đã cũ (the library has a newer version) |
| flat (rate) · monthly fee | cố định · phí hằng tháng |
| incident · Needs attention | sự cố · Cần chú ý |
| level (of staff) | cấp |
| library · shared library · library item | thư viện · thư viện dùng chung · mục thư viện |
| provider cost · plan | chi phí nhà cung cấp · gói |
| rebill · rebill rate · to rebill | tính phí lại · mức tính phí lại · cần tính phí lại |
| saved section | phần đã lưu |
| sender health · good · fair · poor | sức khỏe người gửi · tốt · khá · kém |
| setup steps · Setting up | bước thiết lập · Đang thiết lập |
| staff | nhân sự |
| usage | mức sử dụng |
| workspace template · apply (a template) | mẫu không gian làm việc · áp dụng |

## Grow, Growth and scoring

| English | Tiếng Việt | Note |
|---|---|---|
| awaiting confirmation | chờ xác nhận | a referral whose subscriber hasn't confirmed |
| band (score) · Cold · Warm · Hot | mức (mức điểm) · Lạnh · Ấm · Nóng | the names are fixed, like the English |
| burst · shared IP · numbered addresses | tăng đột biến · IP dùng chung · địa chỉ đánh số | the signals that hold a referral |
| cap (a rule's) | giới hạn | "6 a week": "6 mỗi tuần" |
| claim · claim link | nhận · liên kết nhận thưởng | a coupon reward |
| cohort retention | giữ chân theo nhóm | Audience › Growth |
| counted · not counted | đã tính · không tính | "Count" and "Don't count" in Review: "Tính", "Không tính" |
| coupon (Stripe) | phiếu giảm giá | |
| decay | giảm dần | |
| double opt-in | xác nhận hai bước | two-factor is "xác thực hai lớp" |
| engagement tier · Active · Cooling · At risk · Sunset | nhóm tương tác · Tích cực · Nguội dần · Có nguy cơ · Ngừng gửi dần | List health's tiers, as Scoring names them; the sunset policy is "chính sách ngừng gửi dần" |
| field condition | điều kiện trường | |
| fulfill · fulfilled · To fulfill | trao · đã trao · Cần trao | a manual reward; "sent" stays "đã gửi" |
| Growth (Audience › Growth) | Nguồn tăng trưởng | so it doesn't read as Grow ("Tăng trưởng") |
| held for review | tạm giữ để duyệt | short: "tạm giữ" |
| hold period · in hold | thời gian chờ · đang chờ | "the 7-day hold": "thời gian chờ 7 ngày" |
| hosted (form, page) | trên ReputeCast · do ReputeCast lưu trữ | |
| invite page | trang mời | |
| join · departure | lượt tham gia · lượt rời đi | Growth's joined and left: "Tham gia", "Rời đi" |
| leaderboard | bảng xếp hạng | |
| List health | Sức khỏe danh sách | as Growth and Scoring refer to it |
| milestone · milestone email | mốc thưởng · email mốc thưởng | "mốc" alone when the number follows: "Mốc 3" |
| net (growth) | tăng ròng · tăng trưởng ròng | |
| newsletter | bản tin | |
| partner · partner code · partner link · partner profile · partnership | đối tác · mã đối tác · liên kết đối tác · hồ sơ đối tác · quan hệ đối tác | |
| recommendation · recommend · Recommending you | đề xuất · đề xuất · Đề xuất bạn | |
| referral · referrer · referral link · referral program | lượt giới thiệu · người giới thiệu · liên kết giới thiệu · chương trình giới thiệu | |
| referral hub | trang phần thưởng | the referrer's own page; "trang giới thiệu" reads as an About page |
| Referral · Recommendations · Reward block | khối Giới thiệu · khối Đề xuất · khối Phần thưởng | the composer's Grow blocks |
| referring page · referring domain | trang dẫn đến · tên miền dẫn đến | never "giới thiệu", which is a referral |
| review queue · Review (tab) | hàng chờ duyệt · Duyệt | |
| reward | phần thưởng | |
| rule · points | quy tắc · điểm | |
| score · scoring | điểm (điểm số) · chấm điểm | |
| self-referral | tự giới thiệu | |
| soft bounce · hard bounce | trả thư tạm thời · trả thư vĩnh viễn | |
| source (growth) | nguồn | |
| spread (of scores) | phân bố | |
| subscribe request | yêu cầu đăng ký | what one instance sends a partner |
| threshold | ngưỡng | |
| token (Liquid) | biến | as transactional templates' variables |
| trigger (automation) | trình kích hoạt | |

## Audience, imports, forms and reports

| English | Tiếng Việt | Note |
|---|---|---|
| accent color · page background | màu nhấn · nền trang | public page branding |
| Added by the team (a contact's source) | Do nhóm thêm vào | a suppression added by hand is "Thủ công" |
| allowed domains | tên miền được phép | the sites a form may be embedded on |
| archive (public page) · issue | trang lưu trữ · số | as subscribers read them: "Lưu trữ bản tin", "Số tiếp theo" |
| audience · merge field · cleaned (Mailchimp) | audience · merge field · cleaned | Mailchimp's own words, as its screens show them |
| batch (an import's) | lô | the live log: "Lô 3 · 500 hàng" |
| bounce types (a report's) | Vĩnh viễn · Tạm thời · Bị chặn | "Vĩnh viễn · người nhận không tồn tại", "Tạm thời · hộp thư đầy" |
| branding | thương hiệu | Forms & pages › "Thương hiệu và tên miền" |
| catch-all domain · known complainers | tên miền catch-all · người hay khiếu nại | a verification service's risky groups |
| column · map fields | cột · ghép trường | an import's step: "Ghép trường" |
| confirmation link | liên kết xác nhận | |
| consent · consent text · consent evidence | sự đồng ý · nội dung đồng ý · bằng chứng đồng ý | |
| custom domain | tên miền tùy chỉnh | Branding & domain |
| data center (Mailchimp) | trung tâm dữ liệu | the API key's suffix, like us21 |
| deferral (provider) | hoãn nhận | "deferrals expired": "thư hoãn nhận đã hết hạn" |
| disposable domain · role address · likely typo | tên miền dùng một lần · địa chỉ chức danh · có thể đánh máy sai | an import's risky groups |
| embed code · script embed · embedded (form) | mã nhúng · mã nhúng tập lệnh · nhúng | |
| erase (a contact) | xóa vĩnh viễn | a GDPR erasure; a plain delete stays "xóa" |
| excerpt | đoạn trích | an archived issue's |
| export data (GDPR) | xuất dữ liệu | |
| Forms & public pages | Biểu mẫu và trang công khai | |
| hosted page (a form type) | trang trên ReputeCast | as "hosted" above |
| Import from {tool} (a source) | Nhập từ {tool} | a contact's or suppression's source: "Nhập từ Mailchimp" |
| live log (an import's) | nhật ký trực tiếp | |
| merge (a duplicate) | gộp | "Merge · fills first name": "Gộp · điền tên" |
| Migration (where a suppression came from) | Chuyển từ công cụ khác | an import from another email tool |
| offer a download | tặng một tệp tải xuống | the file a form gives after confirmation |
| popup (form) | bật lên | |
| preference center | trung tâm tùy chọn | in the app; subscribers read "trang tùy chọn" |
| publisher (name) | đơn vị phát hành | heads every public page |
| re-permission | xin phép lại | "re-permission flow": "quy trình xin phép lại" |
| read-only link | liên kết chỉ xem | a shared report's |
| RSS feed | nguồn cấp RSS | |
| self-hosted (a provider) | tự lưu trữ | "Postal (tự lưu trữ)" |
| sender health | sức khỏe người gửi | the Overview's score; reputation is "uy tín" |
| signup · submission (form) | lượt đăng ký · lượt gửi | "Views · Submissions": "Lượt xem · Lượt gửi" |
| single opt-in | xác nhận một bước | double opt-in is "xác nhận hai bước" |
| site key · secret key (Turnstile) | khóa site · khóa bí mật | "Turnstile" stays |
| source status (an import's: subscribed, cleaned…) | as the source writes it | the consent table and the consent history keep the tool's own word |
| spam trap (audience, imports) | bẫy thư rác | the suppression list's and the importer's word, as their screens say it |
| subscribe page · confirmation page · unsubscribe page | trang đăng ký · trang xác nhận · trang hủy đăng ký | |
| suppressed | bị chặn gửi | the list is "danh sách chặn gửi" |
| System stream · System mail | luồng Hệ thống · thư Hệ thống | the stream's name keeps its capital |
| tag | thẻ | |
| title (archived issue) | tựa đề | "tiêu đề" is an email's subject, "đề mục" a heading |
| workbook · sheet (XLSX) | sổ làm việc · trang tính | as Excel's Vietnamese names them |

## Products, stores and transactional email

| English | Tiếng Việt | Note |
|---|---|---|
| abandoned checkout · abandoned cart | thanh toán bị bỏ dở · giỏ hàng bị bỏ dở | as Shopify's Vietnamese admin says it |
| accepted marketing · didn't accept | đã đồng ý nhận tiếp thị · không đồng ý | a store customer's consent |
| attribution · credited to | ghi nhận doanh thu · ghi nhận cho | "Attribution window": "Khoảng thời gian ghi nhận" |
| Bought a product (trigger · condition) | Mua sản phẩm · Đã mua sản phẩm | |
| buyer · customer | người mua · khách hàng | a product's buyer; a store's customer |
| checkout · at checkout | thanh toán · khi thanh toán | "Stripe Checkout" keeps its name |
| click token | mã lượt nhấp | |
| collection (store) | bộ sưu tập | |
| compare-at price | giá so sánh | |
| cover (product) | ảnh bìa | |
| delivery · delivery email · delivery template | giao hàng · email giao hàng · mẫu email giao hàng | a digital product's download link; a delivered email is still "Gửi thành công" |
| dispute | tranh chấp | never "khiếu nại", which is a spam complaint |
| download link · downloads | liên kết tải xuống · lượt tải | "4 of 5 downloads left": "còn 4/5 lượt tải" |
| failover · primary · fallback | chuyển dự phòng · chính · dự phòng | a route's providers |
| Gross · Net | Tổng thu · Thực thu | product revenue before and after refunds |
| idempotency key | khóa idempotency | |
| list offer | lời mời vào danh sách | the checkbox next to a product's Buy button |
| live · test (payments, keys, modes) | chính thức · thử nghiệm (as Earn has it) | the message log's live updates are "Trực tiếp" |
| median delivery | thời gian gửi trung vị | |
| message (transactional) · Messages (tab) | thư · Thư | |
| order | đơn hàng | |
| Order placed · First order · Placed an order | Đơn hàng được đặt · Đơn hàng đầu tiên · Đã đặt hàng | store triggers and the goal |
| Pay what you want · One-time | Trả tùy ý · Một lần | a product's price |
| product · digital product | sản phẩm · sản phẩm số | |
| Queued · Deferred · Bounced · Complained · Rejected | Đang chờ gửi · Bị hoãn · Bị trả lại · Bị khiếu nại · Bị từ chối | a message's status; Delivered is "Gửi thành công" |
| Raw (SMTP) | Thư thô (SMTP) | mail an app sends already rendered |
| reconciliation | đối soát | a store's nightly check |
| refund · refunded | hoàn tiền · đã hoàn tiền | |
| revoke access · restore access | thu hồi quyền truy cập · khôi phục quyền truy cập | a product order's download |
| route · Transactional route | tuyến · tuyến Giao dịch | as "Tuyến Gửi hàng loạt" |
| sales page · default product page · shop | trang bán hàng · trang sản phẩm mặc định · cửa hàng | |
| send-only key · SMTP credential | khóa chỉ gửi · thông tin xác thực SMTP | "SMTP relay" keeps its name |
| single opt-in | đăng ký một bước | double opt-in is "xác nhận hai bước" |
| store · connect · disconnect · sync | cửa hàng · kết nối · ngắt kết nối · đồng bộ | Shopify and WooCommerce; their menu paths and field names stay in English |
| suppressed · opted out | bị chặn gửi · đã tắt nhận | why a message wasn't sent |
| webhook topic | chủ đề | orders/create stays as it is |
| write-only · sealed | chỉ ghi · được niêm phong | a secret that's never shown again |

## Recurring digests

| English | Tiếng Việt | Note |
|---|---|---|
| approval rule · approver | quy tắc phê duyệt · người phê duyệt | as in Approvals |
| archive (public) | kho lưu trữ công khai | where sent issues are listed |
| back catalogue | bài cũ | a feed's older posts |
| category · tag (of a post) | chuyên mục · thẻ | WordPress's own Vietnamese words |
| Collecting · Draft ready · Not sent (an issue) | Đang thu thập · Bản nháp sẵn sàng · Không được gửi | an issue's own states; the others are the campaign words |
| Create a draft · Send automatically · draft mode | Tạo bản nháp · Gửi tự động · chế độ bản nháp | the two sending modes |
| excerpt · featured image | đoạn trích · ảnh đại diện | featured image as WordPress says it |
| fallback image | ảnh dự phòng | |
| feed · feed address | nguồn cấp · địa chỉ nguồn cấp | as Integrations; RSS 2.0, Atom 1.0 and JSON Feed 1.1 keep their names |
| feed item · Feed items (the block) | mục · Mục nguồn cấp | |
| feed token | biến nguồn cấp | a Liquid token: "biến", as in Grow |
| held · release (an issue) | tạm giữ · bỏ tạm giữ | "Release issue": "Bỏ tạm giữ số này" |
| hours before (preparation) | giờ trước giờ gửi | "{count} hours before": "{count} giờ trước giờ gửi"; "trước 3 giờ" would read as 3 o'clock |
| issue · Next issue · issue #143 | số · Số tiếp theo · số #143 | as a magazine's issue, and as the public pages say it |
| late limit | giới hạn trễ | how late an issue may still go out |
| layout: List · Cards · Grid | Danh sách · Thẻ · Lưới | the Feed items block's layouts |
| left out (by a filter) | bị loại | "Left out by a filter": "Bị bộ lọc loại" |
| owner (of a digest) | người phụ trách | the member issues send under; not the Owner role |
| percentage points (pts) | điểm % | "{delta} pts": "{delta} điểm %" |
| post | bài viết | |
| prepare · Prepared at 07:00 | chuẩn bị · Chuẩn bị lúc 07:00 | |
| recurring digest · digest | bản tin định kỳ | as Integrations; the Recurring tab is "Định kỳ" |
| Regular (campaign) | Thông thường | the other choice in New campaign |
| sending window | khung giờ gửi | as Automations' send window |
| skip · skipped | bỏ qua · đã bỏ qua | |
| SSRF guard | lớp bảo vệ SSRF | |
| starter (a template to begin from) | mẫu khởi đầu | a starter's name ("Blog digest") stays as it is; what a new digest's starter writes into the email (subject, preheader) follows the workspace's content language |
| starting point | điểm bắt đầu | |
| turn on · pause (a digest) | bật · tạm dừng | |
| weekdays (Monday to Friday) | Thứ Hai đến Thứ Sáu | short: "Th 2–Th 6", the region format's own abbreviations |

## AI assistant

| English | Tiếng Việt | Note |
|---|---|---|
| accept · edit · reject (a suggestion) | chấp nhận · sửa · từ chối | |
| Accepted with edits · No decision | Đã chấp nhận, có chỉnh sửa · Chưa quyết định | outcomes in the activity log |
| activity log (AI) | nhật ký hoạt động AI | the audit log stays "nhật ký kiểm tra" |
| AI assistant · Assist (the menu) | trợ lý AI · Trợ lý | |
| AI budget | ngân sách AI | |
| cap · monthly cap | hạn mức · hạn mức hằng tháng | in dollars or tokens; "No cap": "Không có hạn mức" |
| condition schema · segment builder | lược đồ điều kiện · trình tạo phân khúc | |
| deployment (Azure) | bản triển khai | |
| describe the segment · Draft (its button) | mô tả phân khúc · Soạn nháp | what it drafts is a "bản nháp" |
| feature (of the assistant) | tính năng | |
| Image input · Structured output | Đọc hình ảnh · Đầu ra có cấu trúc | what a model can do |
| input · output (tokens, prices) | đầu vào · đầu ra | |
| instruction (to rewrite) | chỉ dẫn | "Rewrite with an instruction": "Viết lại theo chỉ dẫn" |
| keep · discard (a summary) | giữ · bỏ | "kept summaries": "bản tóm tắt đã giữ" |
| model · connect a model | mô hình · kết nối mô hình | model names stay as they are |
| private address | địa chỉ nội bộ | a server on a private network |
| prompt · test prompt | lời nhắc · lời nhắc thử | |
| refused by the model | mô hình từ chối | |
| resets (the usage) | đặt lại | "resets {date}": "đặt lại vào {date}" |
| rewrite · shorten · friendlier · more formal · more direct | viết lại · rút gọn · thân thiện hơn · trang trọng hơn · trực tiếp hơn | |
| subject ideas | gợi ý tiêu đề | |
| suggestion | gợi ý | |
| summary (of results) | bản tóm tắt | "Summarize results": "Tóm tắt kết quả" |
| token (of a model) | token | what the provider bills; a Liquid token is "biến" |
| tone | giọng văn | "Rewrite, shorten and tone": "Viết lại, rút gọn và đổi giọng" |
| usage | mức sử dụng | |

## Placement tests

| English | Tiếng Việt | Note |
|---|---|---|
| app password · app-specific password | mật khẩu ứng dụng · mật khẩu dành riêng cho ứng dụng | Google's and Apple's own words |
| arrived late | đến muộn | |
| copy (of the test email) | bản sao | |
| deferral | lần hoãn nhận | as Campaigns' "hoãn nhận" |
| folder · tab | thư mục · tab | a mailbox's own names (Promotions, Junk Email) stay as it gives them |
| folders: Sent · Draft · Archive · Trash | Thư đã gửi · Thư nháp · Lưu trữ · Thùng rác | the folders a test never reads |
| hand-off | bàn giao | when the sending provider takes a copy |
| headers | header | "All headers": "Mọi header"; header names stay |
| Health (Deliverability › Health) | Tình trạng | as Integrations' Health |
| Healthy · Warming up · Degraded · Down · Paused · Sandbox (a provider) | Ổn định · Đang khởi động · Suy giảm · Gián đoạn · Đã tạm dừng · Sandbox | |
| Inbox · Tab · Spam · Not arrived | Hộp thư đến · Tab · Thư rác · Không đến | the four results |
| median time to arrive | thời gian đến trung vị | a median, as Approvals' median wait |
| pass · fail · aligned (authentication) | đạt · không đạt · đã căn chỉnh | SPF, DKIM and DMARC keep their names |
| placement · placement test | vị trí thư · kiểm tra vị trí thư | where a copy lands |
| Queued (a copy) | Chờ gửi | |
| remove (a seed) | gỡ | deleting copies stays "xóa" |
| round (of copies) | đợt | "Round 2": "Đợt 2" |
| sample (the seeds) | nhóm thử | as an A/B test's sample |
| seed · seed mailbox | hộp thư thử | mailboxes that exist only for tests |
| stop waiting | ngừng chờ | |
| test (one run) | lần kiểm tra | "Tests · 12": "Lần kiểm tra · 12" |
| test token | mã kiểm tra | what marks a copy as a test's |
| vs (providers compared) | so với | "SES vs Postal": "SES so với Postal" |
| warm-up cap | giới hạn khởi động | a warming-up provider's daily cap |
| warm-up planner · warm-up plan | trình lập kế hoạch khởi động · kế hoạch khởi động | |

## Address verification

| English | Tiếng Việt | Note |
|---|---|---|
| abuse · catch-all domain · disposable · role address | hay khiếu nại · tên miền nhận mọi thư · dùng một lần · địa chỉ chức danh | why an address is risky; tags such as verify:catch-all stay as written |
| Add credits · Low credits | Nạp tín dụng · Sắp hết tín dụng | |
| balance · credits | số dư · tín dụng | the prepaid credits at the service |
| bulk check · real-time check | kiểm tra hàng loạt · kiểm tra theo thời gian thực | |
| check (one address) | lượt kiểm tra | "{price} a check": "{price} mỗi lượt kiểm tra" |
| data region | vùng dữ liệu | |
| Found by | Phát hiện bởi | which check found a risky address |
| imports (of contacts) | nhập liên hệ | "Bulk checks and imports": "Kiểm tra hàng loạt và nhập liên hệ" |
| job · Resume | tác vụ · Tiếp tục | |
| spam trap | bẫy spam | |
| Uses (of a service) | Lượt dùng | |
| Valid · Invalid · Risky · Unknown | Hợp lệ · Không hợp lệ · Rủi ro · Không xác định | the four results |
| verification service · verify | dịch vụ xác minh · xác minh | |

## Words used everywhere

| English | Tiếng Việt |
|---|---|
| contact | liên hệ |
| subscriber | người đăng ký |
| list | danh sách |
| segment | phân khúc |
| template | mẫu |
| sending provider | nhà cung cấp gửi thư |
| mailbox provider | nhà cung cấp hộp thư |
| sender | người gửi |
| domain | tên miền |
| record (DNS) | bản ghi |
| verified | đã xác minh |
| sign in · sign out | đăng nhập · đăng xuất |
| password · passkey | mật khẩu · khóa truy cập |
| authenticator app | ứng dụng xác thực |
| recovery code | mã khôi phục |
| two-factor | xác thực hai lớp (2FA) |
| invitation · invite | lời mời · mời |
| permission | quyền |
| Ask Marcus Chen or Linh Nguyen. | Hãy hỏi Marcus Chen hoặc Linh Nguyen. |
| notification | thông báo |
| approval | phê duyệt |
| retry · try again | thử lại |
| cancel · close · back | hủy · đóng · quay lại |
| copy · copied | sao chép · đã sao chép |
| optional · required | không bắt buộc · bắt buộc |
