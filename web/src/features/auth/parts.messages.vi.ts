import { translate } from '@/i18n'
import { m } from './parts.messages'

export default translate(m, {
  notYou: 'không phải bạn?',
  digitLabel: 'Chữ số {n}',

  passkeyUnsupported: 'Trình duyệt này không hỗ trợ khóa truy cập. Hãy dùng trình duyệt khác, hoặc đăng nhập bằng mật khẩu.',
  passkeyAddCancelled: 'Việc thêm khóa truy cập đã bị hủy hoặc hết thời gian. Hãy thử lại khi thiết bị của bạn sẵn sàng.',
  passkeySignInCancelled: 'Hộp thoại khóa truy cập đã đóng hoặc hết thời gian. Hãy thử lại, hoặc dùng mật khẩu.',
  passkeyAlreadyRegistered: 'Thiết bị này đã có khóa truy cập cho tài khoản của bạn.',
  passkeyWrongDomain: 'Khóa truy cập chỉ dùng được trên URL công khai của hệ thống. Hãy mở ReputeCast ở địa chỉ đó rồi thử lại.',

  totpEnterDigits: 'Hãy nhập 6 chữ số mà ứng dụng hiển thị.',
  tryAgain: 'Thử lại',
  qrLabel: 'Mã QR cho ứng dụng xác thực',
  scanHelp: 'Quét bằng 1Password, Google Authenticator, Authy hoặc bất kỳ ứng dụng TOTP nào. Không quét được? Hãy nhập khóa bằng tay:',
  copy: 'Sao chép',
  keyCopied: 'Đã sao chép khóa',
  copyFailed: 'Không sao chép được',
  copyKeyByHand: 'Hãy chọn khóa và sao chép bằng tay.',
  keyFacts: 'Tài khoản {account} · nhà phát hành {issuer} · mã 30 giây',
  preparingKey: 'Đang chuẩn bị khóa…',
  codeLabel: 'Nhập mã 6 chữ số mà ứng dụng hiển thị',
  confirmContinue: 'Xác nhận và tiếp tục',
  cancel: 'Hủy',

  codesFileTitle: 'Mã khôi phục {instance} cho {email}',
  codesFileGenerated: 'Tạo ngày {date}. Mỗi mã cho bạn đăng nhập một lần nếu bạn mất ứng dụng xác thực hoặc khóa truy cập.',
  codesFileKeep: 'Hãy cất chúng ở nơi an toàn, như trình quản lý mật khẩu. Tạo mã mới sẽ thay thế các mã này.',
  download: 'Tải .txt',
  print: 'In',
  copyAll: 'Sao chép tất cả',
  codesCopied: 'Đã sao chép mã khôi phục',
  copyCodesByHand: 'Hãy chọn các mã và sao chép bằng tay.',
  codesList: 'Mã khôi phục',
  shownOnce: 'Chỉ hiện một lần',
  shownOnceBody: 'Mỗi mã cho bạn đăng nhập một lần nếu bạn mất ứng dụng xác thực. Hãy cất chúng ở nơi an toàn; bạn sẽ không thấy lại chúng.',
  savedCodes: 'Tôi đã lưu các mã này',
  finish: 'Hoàn tất',
  confirmSavedFirst: 'Hãy xác nhận bạn đã lưu các mã trước',
})
