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
