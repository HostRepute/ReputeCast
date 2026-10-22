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
| degraded · down | suy giảm · ngừng hoạt động | provider statuses |
| delisting | gỡ khỏi danh sách | "Yêu cầu gỡ khỏi danh sách" |
| event log · raw event · normalized event | nhật ký sự kiện · sự kiện gốc · sự kiện chuẩn hóa | |
| failover · fallback · primary | chuyển dự phòng · dự phòng · chính | "nhà cung cấp dự phòng", "nhà cung cấp chính" |
| feedback loop | vòng phản hồi | |
| flow (an automation's, in sending text) | quy trình tự động | there "luồng" is a stream: "email do các quy trình tự động gửi"; the Automations section names the flow editor's flows |
| guard (Warm-up page) | ngưỡng bảo vệ | "Tỷ lệ trả thư · ngưỡng bảo vệ 2%" |
| held (recipients) | tạm giữ | |
| incident · outage | sự cố · sự cố ngừng hoạt động | |
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

MJML tags (mj-section, mj-body) stay as they are. A block's sample content
is written for the email's readers, in the app language: the first-name
fallback "there" is "bạn".

## Automations

| English | Tiếng Việt |
|---|---|
| A/B split · weight | Chia A/B · tỷ trọng |
| automation (one, or the section) | tự động hóa ("Tự động hóa mới") |
| Branch · Yes · No | Rẽ nhánh · Có · Không |
| Delay (step) · Wait 10 minutes | Trì hoãn · Chờ 10 phút |
| double opt-in | đăng ký kép |
| edge (a connection on the canvas) | đường nối |
| exit · goal · goal rate | thoát · mục tiêu · tỷ lệ đạt mục tiêu |
| fail over · failover | chuyển dự phòng |
| Filter (step) | Lọc |
| flow | luồng (a named stream keeps its capital: "Luồng Tự động hóa") |
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

## Approvals

| English | Tiếng Việt |
|---|---|
| after exclusions | sau khi loại trừ |
| approval inbox | hộp thư phê duyệt |
| approval request · approver · author | yêu cầu phê duyệt · người phê duyệt · tác giả |
| Approve · Reject · Request changes | Phê duyệt · Từ chối · Yêu cầu thay đổi |
| Bypassed · emergency bypass | Đã bỏ qua · bỏ qua khẩn cấp |
| comment · Discussion · evidence | nhận xét · Thảo luận · bằng chứng |
| Contacts in flight | Liên hệ đang trong luồng |
| Invalidated · Approval invalidated | Mất hiệu lực · Phê duyệt không còn hiệu lực |
| median wait | thời gian chờ trung vị (a median, not an average: not "trung bình") |
| Pending · Rejected | Đang chờ · Bị từ chối |
| policy · rule · Rules matched | chính sách · quy tắc · Quy tắc khớp |
| Recheck | Kiểm tra lại |
| snapshot | bản chụp |

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

"Heading" is "Đề mục", never "Tiêu đề", which is an email's subject.

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
| allowed domains | tên miền được phép | the sites a form may be embedded on |
| archive (public page) · issue | trang lưu trữ · số | as subscribers read them: "Lưu trữ bản tin", "Số tiếp theo" |
| audience · merge field · cleaned (Mailchimp) | audience · merge field · cleaned | Mailchimp's own words, as its screens show them |
| branding | thương hiệu | Forms & pages › "Thương hiệu và tên miền" |
| column · map fields | cột · ghép trường | an import's step: "Ghép trường" |
| confirmation link | liên kết xác nhận | |
| consent · consent text · consent evidence | sự đồng ý · nội dung đồng ý · bằng chứng đồng ý | |
| deferral (provider) | hoãn nhận | "deferrals expired": "thư hoãn nhận đã hết hạn" |
| embed code · script embed · embedded (form) | mã nhúng · mã nhúng tập lệnh · nhúng | |
| erase (a contact) | xóa vĩnh viễn | a GDPR erasure; a plain delete stays "xóa" |
| excerpt | đoạn trích | an archived issue's |
| export data (GDPR) | xuất dữ liệu | |
| Forms & public pages | Biểu mẫu và trang công khai | |
| hosted page (a form type) | trang trên ReputeCast | as "hosted" above |
| offer a download | tặng một tệp tải xuống | the file a form gives after confirmation |
| popup (form) | bật lên | |
| preference center | trung tâm tùy chọn | in the app; subscribers read "trang tùy chọn" |
| publisher (name) | đơn vị phát hành | heads every public page |
| re-permission | xin phép lại | "re-permission flow": "quy trình xin phép lại" |
| read-only link | liên kết chỉ xem | a shared report's |
| RSS feed | nguồn cấp RSS | |
| sender health | sức khỏe người gửi | the Overview's score; reputation is "uy tín" |
| signup · submission (form) | lượt đăng ký · lượt gửi | "Views · Submissions": "Lượt xem · Lượt gửi" |
| single opt-in | xác nhận một bước | double opt-in is "xác nhận hai bước" |
| subscribe page · confirmation page · unsubscribe page | trang đăng ký · trang xác nhận · trang hủy đăng ký | |
| suppressed | bị chặn gửi | the list is "danh sách chặn gửi" |
| System stream · System mail | luồng Hệ thống · thư Hệ thống | the stream's name keeps its capital |
| tag | thẻ | |
| title (archived issue) | tựa đề | "tiêu đề" is an email's subject, "đề mục" a heading |

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
