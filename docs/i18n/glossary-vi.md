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
| signing secret (a webhook's) | mã bí mật ký |
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
