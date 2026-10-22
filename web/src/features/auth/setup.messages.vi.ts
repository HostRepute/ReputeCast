import { translate } from '@/i18n'
import { m } from './setup.messages'

export default translate(m, {
  documentTitle: 'Thiết lập {name} · ReputeCast',

  signInToContinue: 'Hãy đăng nhập để tiếp tục thiết lập',
  loadFailed: 'Không tải được phần thiết lập',
  signedOut: 'Đã đăng xuất',
  error: 'Lỗi',
  sessionEnded: 'Phiên của bạn đã kết thúc. Hãy đăng nhập lại; các bước bạn đã hoàn tất vẫn được lưu.',
  nothingChanged: 'Không có gì bị thay đổi.',
  signIn: 'Đăng nhập',
  loading: 'Đang tải phần thiết lập…',

  stepOf: 'Bước {n}/{total}',
  twoFactorOn: 'Đã bật xác thực hai lớp',
  accountProtected: 'Tài khoản của bạn đã được bảo vệ.',

  progressLabel: 'Tiến độ thiết lập',
  setUpName: 'Thiết lập {name}',
  finishAccount: 'Hoàn tất thiết lập tài khoản của bạn',
  signedInOwner: 'Đã đăng nhập với {email}, tài khoản Chủ sở hữu do <mono>{command}</mono> tạo',
  signedInReason: 'Đã đăng nhập với {email}. {reason}',
  stepsLabel: 'Các bước thiết lập',
  stepLabel:
    '{step, select, password {Thay mật khẩu tạm thời} second_factor {Thêm khóa truy cập hoặc ứng dụng xác thực} recovery_codes {Lưu mã khôi phục} instance {Thông tin cơ bản của hệ thống} workspace {Tạo không gian làm việc đầu tiên} provider {Kết nối nhà cung cấp gửi thư} other {{label}}}',
  stepCompleted: '(đã hoàn tất)',
  stepCurrent: '(bước hiện tại)',
  doneSoFar: 'Đã hoàn tất',
  doneTempPassword: 'Đã thay mật khẩu tạm thời',
  donePasskey: 'Đã thêm khóa truy cập “{name}”',
  doneAuthenticator: 'Đã thiết lập ứng dụng xác thực',
  doneRecovery: '{count, plural, other {Đã tải xuống # mã khôi phục}}',
  doneInstance: 'Đã lưu thông tin cơ bản của hệ thống',
  doneWorkspace: 'Đã tạo không gian làm việc “{name}”',
  firstOwnerRequired: 'Chủ sở hữu đầu tiên bắt buộc dùng xác thực hai lớp; không thể bỏ qua bước này.',

  // ── Password ─────────────────────────────────────────────────────────
  passwordTitle: 'Thay mật khẩu tạm thời',
  passwordSub: 'Mật khẩu do <mono>{command}</mono> in ra chỉ dùng được cho đến khi bạn chọn mật khẩu của riêng mình. Hãy chọn một mật khẩu chỉ bạn biết.',
  tempPasswordRequired: 'Hãy nhập mật khẩu tạm thời được in ra trong terminal.',
  passwordTooShort: 'Hãy dùng ít nhất {min} ký tự. Cụm mật khẩu cũng được.',
  passwordsDiffer: 'Hai mật khẩu không khớp.',
  notSaved: 'Chưa lưu',
  tempPassword: 'Mật khẩu tạm thời',
  newPassword: 'Mật khẩu mới',
  passwordHint: 'Ít nhất {min} ký tự. Cụm mật khẩu cũng được.',
  passwordHintBreached: 'Ít nhất {min} ký tự. Được kiểm tra với các vụ lộ dữ liệu đã biết.',
  confirmPassword: 'Xác nhận mật khẩu mới',
  saveContinue: 'Lưu và tiếp tục',

  // ── Second factor ────────────────────────────────────────────────────
  totpTitle: 'Thiết lập ứng dụng xác thực',
  totpSub: 'Quét mã QR, rồi nhập mã mà ứng dụng hiển thị để xác nhận.',
  secondFactorTitle: 'Thêm khóa truy cập hoặc ứng dụng xác thực',
  passkeyQuickest: 'Khóa truy cập là cách nhanh nhất: Touch ID, Face ID, Windows Hello hoặc khóa bảo mật.',
  passkeyAdded: 'Đã thêm khóa truy cập',
  addPasskey: 'Thêm khóa truy cập (khuyến nghị)',
  setUpAuthenticator: 'Thiết lập ứng dụng xác thực',
  changeLater: 'Bạn có thể thêm khóa truy cập khác và đổi ứng dụng xác thực sau trong Tài khoản và bảo mật.',

  // ── Recovery codes ───────────────────────────────────────────────────
  recoveryTitle: 'Lưu mã khôi phục',
  recoverySub: 'Mỗi mã giúp bạn đăng nhập một lần nếu bạn mất khóa truy cập hoặc ứng dụng xác thực. Bạn chỉ thấy các mã này lúc này.',
  shownOnce: 'Chỉ hiện một lần',
  earlierCodes: 'Không thể hiện lại các mã trước đây của bạn. Hãy tạo bộ mã mới để lưu; các mã cũ sẽ ngừng hoạt động.',
  generateCodes: 'Tạo mã khôi phục',

  // ── Instance basics ──────────────────────────────────────────────────
  instanceTitle: 'Thông tin cơ bản của hệ thống',
  instanceSub: 'Các cài đặt này áp dụng cho mọi không gian làm việc trên bản cài đặt này. Bạn có thể đổi sau trong Quản trị hệ thống.',
  instanceNameRequired: 'Hãy đặt tên cho hệ thống, ví dụ Acme ReputeCast.',
  urlInvalid: 'Hãy dùng địa chỉ đầy đủ mà mọi người mở, ví dụ https://reputecast.acme.io.',
  instanceName: 'Tên hệ thống',
  instanceNameHint: 'Hiển thị trên trang đăng nhập và trong email gửi cho nhóm của bạn.',
  defaultTimezone: 'Múi giờ mặc định',
  defaultTimezoneHint: 'Không gian làm việc mới bắt đầu với múi giờ này.',
  publicUrl: 'URL công khai',
  checkAgain: 'Kiểm tra lại',
  publicUrlHint:
    'Dùng cho webhook của nhà cung cấp, URI chuyển hướng SSO, trang công khai và liên kết trong mọi email. Nếu đổi sau này, các webhook hiện có sẽ lỗi cho đến khi được đăng ký lại.',
  urlCheckLabel: 'Kết quả kiểm tra URL',
  checkReachable: 'Kết nối',
  checking: 'Đang kiểm tra…',
  checked: 'Đã kiểm tra {ago}',
  secondsAgo: '{n} giây trước',
  minutesAgo: '{n} phút trước',
  checkingHost: 'Đang kiểm tra {host}…',
  updateCheck: 'Kiểm tra bản phát hành mới trên GitHub mỗi ngày một lần',
  updateCheckHint: 'Chỉ lấy số phiên bản. ReputeCast không bao giờ gửi dữ liệu sử dụng đi bất cứ đâu. Bạn có thể tắt tùy chọn này với bản cài đặt ngoại tuyến.',
  saveCreateWorkspace: 'Lưu và tạo không gian làm việc đầu tiên',

  // ── First workspace ──────────────────────────────────────────────────
  workspaceTitle: 'Tạo không gian làm việc đầu tiên',
  workspaceSub:
    'Mỗi không gian làm việc có danh sách, chiến dịch, tự động hóa, nhà cung cấp gửi thư và nhóm riêng. Bạn có thể tạo thêm sau từ bộ chuyển không gian làm việc.',
  workspaceNameRequired: 'Hãy đặt tên cho không gian làm việc, ví dụ Bản tin Acme.',
  notCreated: 'Chưa tạo',
  workspaceName: 'Tên không gian làm việc',
  workspaceNamePlaceholder: 'Bản tin Acme',
  workspaceNameHint: 'Hiển thị trong bộ chuyển không gian làm việc và trên các trang công khai cho đến khi bạn đặt thương hiệu.',
  timezone: 'Múi giờ',
  timezoneHint: 'Lịch gửi và báo cáo dùng múi giờ này.',
  createWorkspace: 'Tạo không gian làm việc',

  // ── Sending provider ─────────────────────────────────────────────────
  providerTitle: 'Kết nối nhà cung cấp gửi thư',
  providerNeedsWorkspace: 'Hãy tạo không gian làm việc trước.',
  continue: 'Tiếp tục',
  providerSub:
    'ReputeCast gửi thư qua nhà cung cấp của chính bạn: Amazon SES, Postmark, Mailgun, SendGrid, Resend, một máy chủ Postal hoặc bất kỳ máy chủ SMTP nào. Kết nối một nhà cung cấp mất khoảng 10 phút.',
  ready: 'Sẵn sàng',
  readyBody:
    '{instance} đã được thiết lập và {workspace} đã được tạo; bạn là Chủ sở hữu của không gian này. Lời mời, đặt lại mật khẩu và thông báo qua email sẽ bắt đầu hoạt động khi có nhà cung cấp xử lý luồng Hệ thống.',
  skipForNow: 'Bỏ qua lúc này',
  connectProvider: 'Kết nối nhà cung cấp gửi thư',
})
