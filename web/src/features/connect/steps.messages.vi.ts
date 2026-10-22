import { translate } from '@/i18n'
import { mst } from './steps.messages'

/** Vietnamese for the test send, streams and contacts steps (docs/i18n/glossary-vi.md). */
export default translate(mst, {
  // ── Test send ────────────────────────────────────────────────────────
  testHeading: 'Gửi thử và xem thư đến nơi',
  testLeadSmtp: 'Bước này kiểm tra trọn vòng: ReputeCast chuyển thư cho máy chủ SMTP, máy chủ nhận thư, và thư trả lại quay về qua hộp thư nhận thư trả lại.',
  testLead: 'Bước này kiểm tra trọn vòng: ReputeCast chuyển thư cho {provider}, {provider} gửi thư đi, và sự kiện gửi thành công quay về.',
  testLeadSns: 'Bước này kiểm tra trọn vòng: ReputeCast chuyển thư cho {provider}, {provider} gửi thư đi, và sự kiện gửi thành công quay về qua SNS.',
  testLoadFailed: 'Không tải được bước gửi thử',
  testLoading: 'Đang tải bước gửi thử…',
  continueStreams: 'Tiếp tục chọn luồng',
  defaultSubject: 'Thư thử của ReputeCast từ {provider}',
  testFailed: 'Không gửi thử được',
  testMessage: 'Thư thử',
  from: 'Người gửi',
  to: 'Người nhận',
  subject: 'Tiêu đề',
  sending: 'Đang gửi…',
  addSenderFirst: 'Hãy thêm người gửi trước.',
  sendAnother: 'Gửi thử lần nữa',
  sendTest: 'Gửi thử',
  sendSeeds: 'Gửi đến danh sách địa chỉ thử (Gmail, Outlook, Yahoo, iCloud)',
  authSeenBy: 'Kết quả xác thực {receiver} thấy',
  clipped: '{clipped, select, yes {Bị cắt bớt · {size} KB} other {Không bị cắt bớt · {size} KB}}',
  alsoSent: 'Cũng đã gửi đến {addresses}.',
  timelineTitle: 'Dòng thời gian gửi thư',
  listening: 'Đang chờ sự kiện',
  timelineEmpty: 'Hãy gửi thử: từng bước sẽ hiện ở đây khi các sự kiện quay về.',
  seedsSent: 'Đã gửi thử đến danh sách địa chỉ thử',
  seedsSentDetail: 'Mỗi nhà cung cấp hộp thư cho thấy nó đã xếp thư vào đâu.',
  seedsTitle: 'Gửi đến danh sách địa chỉ thử',
  seedsLine: 'Mỗi nhà cung cấp hộp thư một địa chỉ của bạn. ReputeCast giữ danh sách này cho lần gửi thử sau.',
  enterOne: 'Hãy nhập ít nhất một địa chỉ.',
  send: 'Gửi',

  // ── Streams ──────────────────────────────────────────────────────────
  rolePrimary: 'Chính',
  roleSplit: 'Chia theo tỷ trọng',
  roleFallback: 'Dự phòng',
  roleSplitWeight: 'Chia · {weight}',
  streamsSaveFailed: 'Không lưu được các luồng',
  streamsHeading: '{provider} nên gửi những luồng nào?',
  streamsLead:
    'Mỗi luồng là một loại thư. Mỗi luồng có tuyến, chính sách chuyển dự phòng và configuration set riêng, nên một chiến dịch có vấn đề không bao giờ chặn được thư đặt lại mật khẩu.',
  streamsFinished: 'Thiết lập này đã xong: đây là các luồng nó đã thiết lập. Định tuyến giờ có trang riêng. <link>Mở Định tuyến</link>',
  nextTitle: 'Các bước tiếp theo nên làm',
  tip: 'Mẹo',
  set: 'Đã đặt',
  pickOne: 'Hãy chọn ít nhất một luồng cho {provider}.',
  continueContacts: 'Tiếp tục với liên hệ',
  nextTo: 'Bên cạnh {providers}',
  primaryReplacing: 'Chính, thay cho {providers}',
  share: 'Tỷ trọng của {provider}',
  role: 'Vai trò',
  configurationSet: 'Configuration set',
  failover: 'Chuyển dự phòng',
  tracking: 'Theo dõi',

  // ── Contacts ─────────────────────────────────────────────────────────
  contactsHeading: 'Đưa liên hệ của bạn vào',
  contactsLead: 'Việc nhập giữ nguyên sự đồng ý: liên hệ đã hủy đăng ký, bị trả thư và đã khiếu nại sẽ vào danh sách chặn gửi và không bao giờ được đăng ký lại.',
  contactsLeadPull:
    'Việc nhập giữ nguyên sự đồng ý: liên hệ đã hủy đăng ký, bị trả thư và đã khiếu nại sẽ vào danh sách chặn gửi và không bao giờ được đăng ký lại. Bạn cũng có thể kéo danh sách chặn gửi hiện có từ {provider}.',
  optionsFailed: 'Không tải được các tùy chọn nhập',
  optionsLoading: 'Đang tải các tùy chọn nhập…',
  finishFailed: 'Không hoàn tất được thiết lập',
  pullTitle: 'Kéo danh sách chặn gửi hiện có từ {provider}',
  recommended: 'Khuyên dùng',
  pullLine:
    'Danh sách chặn gửi cấp tài khoản {short} của bạn có {count, plural, other {# địa chỉ}} từ những lần gửi trước. Nhập chúng ngay giúp ReputeCast không bao giờ thử gửi đến các địa chỉ đó, trên bất kỳ nhà cung cấp nào.',
  pullCheckbox: 'Nhập {count, plural, other {# địa chỉ bị chặn gửi}} từ {short} (thư trả lại {bounces}, khiếu nại {complaints})',
  suppressionsFrom: 'Danh sách chặn gửi từ {provider}',
  emptyTitle: 'Hoặc bắt đầu trống',
  emptySuggested:
    'Hãy tạo danh sách ngay và phát triển chúng bằng biểu mẫu đăng ký, API hoặc một lần nhập sau. {count, plural, =1 {Một danh sách được} =2 {Hai danh sách được} =3 {Ba danh sách được} =4 {Bốn danh sách được} =5 {Năm danh sách được} =6 {Sáu danh sách được} other {# danh sách được}} gợi ý từ tên không gian làm việc của bạn; bạn có thể đổi tên bất cứ lúc nào.',
  emptyKeep:
    '{count, plural, =1 {Hãy giữ danh sách của không gian làm việc và phát triển nó bằng biểu mẫu đăng ký, API hoặc một lần nhập sau.} other {Hãy giữ # danh sách của không gian làm việc và phát triển chúng bằng biểu mẫu đăng ký, API hoặc một lần nhập sau.}}',
  startEmpty: 'Bắt đầu trống và hoàn tất',
  setupFinished: 'Thiết lập này đã xong.',
  finish: 'Hoàn tất thiết lập',
})
