import { translate } from '@/i18n'
import { im } from './issueMessages'

/**
 * Vietnamese for a digest's issues where a campaign shows (approvals, Review
 * & send, the report), whose English is issueMessages.ts. An issue is a
 * "số": "Phê duyệt số #3".
 */
export default translate(im, {
  recurringDigest: 'Bản tin định kỳ',
  recurringIssue: 'Số của bản tin định kỳ',
  openDigest: 'Mở bản tin định kỳ',

  // Approvals: inbox and review.
  approveIssue: 'Phê duyệt số #{number}',
  postsTitle: 'Bài viết trong số này',
  postsCaption: '{count, plural, other {# bài viết}} từ {digest}',
  noPosts: 'Số này không có bài viết nào.',
  sinceTitle: 'Kể từ số #{number}',
  sinceNothing: 'Không có gì thay đổi: mẫu, người nhận, tuyến và lịch gửi vẫn như lúc được phê duyệt ngày {date}.',
  sinceChanged: 'Đã thay đổi kể từ lần phê duyệt ngày {date}: {changes}. Phần còn lại vẫn như đã phê duyệt.',
  sinceFirst: 'Số đầu tiên: chưa có số nào trước đó được phê duyệt, nên mọi thứ đều mới với người phê duyệt.',
  change: '{what, select, template {mẫu} audience {người nhận} route {tuyến} schedule {lịch gửi} other {cài đặt}}',

  // Review & send.
  issueOf: 'Số #{number} của {name} · chuẩn bị lúc {time} · {recipients} người nhận',
  issueOfCollecting: 'Số #{number} của {name} · {recipients} người nhận',
  issueOfPrepared: 'Số #{number} của {name} · chuẩn bị lúc {time}',
  issueOfPlain: 'Số #{number} của {name}',
  rulesEach: 'Quy tắc áp dụng cho từng số của bản tin định kỳ.',
  releaseTitle: 'Bỏ tạm giữ',
  releaseBody: 'Thay cho Lên lịch. Hãy bỏ tạm giữ khi lỗi chặn gửi đã được sửa; sau đó số này chờ phê duyệt và được gửi lúc {time}.',
  releaseBodyNoApproval: 'Thay cho Lên lịch. Hãy bỏ tạm giữ khi lỗi chặn gửi đã được sửa; sau đó số này được gửi lúc {time}.',
  release: 'Bỏ tạm giữ số này',
  released: 'Đã bỏ tạm giữ số #{number}.',
  skip: 'Bỏ qua số này',
  lateLimit: 'Giới hạn trễ: bị bỏ qua nếu chưa gửi trước {date}.',
  autoTitle: 'Bản tin định kỳ tự gửi',
  autoBody:
    'Số #{number} tự được gửi lúc {time}, {approval, select, yes {sau khi được phê duyệt} other {không cần ai bấm Gửi}}. Hãy sửa trong trình soạn thảo, hoặc tạo lại từ mẫu trên trang bản tin định kỳ.',
  draftBody: 'Chế độ bản nháp: số #{number} được gửi khi có người gửi nó tại đây. Số này sẽ bị bỏ qua nếu chưa được gửi trước {date}.',
  scheduleNote: 'Bản tin định kỳ sẽ gửi số này vào {when}.',

  // The overview's "Next scheduled campaign".
  nextScheduled: 'Chiến dịch kế tiếp đã lên lịch',
  issueName: '{name} #{number}',
  nextIssueLine: '{when} · {posts, plural, =0 {chưa có bài viết nào} other {đã có # bài viết}}{approval, select, yes { · cần phê duyệt} other {}}',

  // The report.
  comparedTitle: 'So với số #{number}',
  comparedFirst: 'Số đầu tiên của {name}: chưa có số nào trước đó để so sánh.',
  comparedCaption: 'Số #{number} của {name}',
  metricDelivered: 'Gửi thành công',
  metricOpens: 'Lượt mở bởi người đọc',
  metricClicks: 'Lượt nhấp',
  metricUnsubscribes: 'Lượt hủy đăng ký',
  vsIssue: '{delta} so với #{number}',
  pts: '{delta} điểm %',
})
