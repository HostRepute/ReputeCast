import { translate } from '@/i18n'
import { ma } from './attribution.messages'

/**
 * Vietnamese for Earn › Settings › Attribution: the attribution window, the
 * click token, the conversions log and the conversion API card
 * (docs/i18n/glossary-vi.md › Earn). sampleName stays in English: it is the
 * "name" value inside the code samples.
 */
export default translate(ma, {
  title: 'Phân bổ',
  windowTitle: 'Khung thời gian phân bổ',
  windowText:
    'Mỗi lượt nâng cấp hoặc lượt chuyển đổi được ghi nhận cho email chiến dịch hoặc tự động hóa mà người đọc nhấp gần nhất trong khung thời gian này. Nếu không có, nó được ghi nhận theo cách người đọc đến trang nâng cấp.',
  windowLabel: 'Khung thời gian phân bổ',
  windowDays: '{days, plural, other {# ngày}}',
  windowCustom: 'Số ngày khác…',
  windowNumber: 'Số ngày',
  windowPresets: 'Chọn từ danh sách',
  windowError: 'Hãy chọn khung thời gian từ 1 đến 30 ngày.',
  tokenTitle: 'Token lượt nhấp trên tên miền của chính bạn',
  tokenText:
    'Thêm vào các liên kết dẫn đến những tên miền này một <mono>rc_click=…</mono> có chữ ký, không chứa thông tin đọc được, để API chuyển đổi có thể ghi nhận cho email. Token hết hạn cùng khung thời gian phân bổ và không bao giờ được thêm vào liên kết dẫn đến tên miền khác, kể cả liên kết của nhà tài trợ.',
  tokenLabel: 'Token lượt nhấp trên tên miền của chính bạn',
  domains: 'Tên miền',
  domainsNone: 'Chưa có',
  addDomain: 'Thêm tên miền',
  add: 'Thêm',
  removeDomain: 'Gỡ {domain}',
  domainInvalid: '“{domain}” không phải là tên miền. Hãy nhập tên miền dạng app.acme.io, không kèm https:// hay đường dẫn.',
  domainsNeeded: 'Hãy thêm các tên miền sẽ được gắn token lượt nhấp, chẳng hạn app.acme.io.',
  unsaved: 'Thay đổi chưa lưu',
  notSaved: 'Chưa lưu: hãy sửa khung thời gian trước',
  notSavedDomains: 'Chưa lưu: hãy thêm một tên miền trước',
  save: 'Lưu',
  saved: 'Đã lưu phân bổ',
  savedText: 'Đã ghi vào nhật ký kiểm tra.',
  readOnly: 'Thay đổi phân bổ cần quyền <mono>payments:manage</mono>.',
  // The conversions log.
  logTitle: 'Nhật ký chuyển đổi',
  logAll: '{count, plural, =1 {Lượt chuyển đổi duy nhất là} other {Cả # lượt chuyển đổi đều là}} “{name}” trị giá {value}, từ {sender}',
  logAllNoSender: '{count, plural, =1 {Lượt chuyển đổi duy nhất là} other {Cả # lượt chuyển đổi đều là}} “{name}” trị giá {value}',
  logSome: '{count, plural, other {# lượt chuyển đổi}}, mới nhất trước',
  logNone: 'Các lượt chuyển đổi do sản phẩm của bạn gửi sẽ hiện ở đây, mới nhất trước.',
  colTime: 'Thời gian',
  colValue: 'Giá trị',
  colMatched: 'Khớp qua',
  colCredited: 'Ghi nhận cho',
  byToken: 'Token lượt nhấp',
  byEmail: 'Email',
  byNone: 'Không khớp',
  test: 'Thử nghiệm',
  notCredited: 'Không được ghi nhận · không có lượt nhấp email nào trong {days, plural, other {# ngày}}',
  noReader: 'Không được ghi nhận · không khớp người đọc nào',
  logFoot:
    '{count} từ ngày {date} · {value} · {credited} được ghi nhận ({creditedValue}) · {rest} không được ghi nhận ({restValue}). Mỗi lệnh gọi API được ghi lại ở đây, không ghi vào nhật ký kiểm tra.',
  logFootNone: 'Mỗi lệnh gọi API được ghi lại ở đây, không ghi vào nhật ký kiểm tra.',
  showMore: 'Hiện thêm',
  logError: 'Không tải được nhật ký chuyển đổi',
  retry: 'Thử lại',
  // The conversion API.
  apiTitle: 'API chuyển đổi',
  apiText:
    'Dành cho sản phẩm của bạn: ghi lại các lượt nâng cấp và đơn hàng kèm giá trị. Trả về 202 kèm cách khớp lượt chuyển đổi. Lỗi theo chuẩn RFC 9457.',
  keyBadge: 'Chuyển đổi',
  keyLine: '{who} tạo ngày {date} · dùng gần nhất {ago} từ {ip} · chỉ có quyền <mono>conversions:create</mono>',
  keyLineNoIp: '{who} tạo ngày {date} · dùng gần nhất {ago} · chỉ có quyền <mono>conversions:create</mono>',
  keyLineUnused: '{who} tạo ngày {date} · chưa dùng lần nào · chỉ có quyền <mono>conversions:create</mono>',
  keyLineMore: '{who} tạo ngày {date} · có quyền <mono>conversions:create</mono> và các quyền khác',
  manageKeys: 'Quản lý khóa',
  noKey: 'Chưa có khóa nào ghi lại được lượt chuyển đổi. Khóa chuyển đổi chỉ có quyền <mono>conversions:create</mono>.',
  sendTest: 'Gửi một lượt chuyển đổi thử nghiệm',
  testSent: 'Đã ghi lại lượt chuyển đổi thử nghiệm',
  testSentCredited:
    '{matched, select, click_token {Khớp qua token lượt nhấp} email {Khớp qua email} other {Không khớp}} và được ghi nhận cho {name}. Lượt chuyển đổi này không được tính vào bất kỳ số liệu nào.',
  testSentNot:
    'Khớp qua địa chỉ của bạn nhưng không được ghi nhận: bạn chưa nhấp email nào trong khung thời gian. Lượt chuyển đổi này không được tính vào bất kỳ số liệu nào.',
  testSentNone:
    '{email} không phải là người đăng ký ở đây nên không được ghi nhận. Lượt chuyển đổi này không được tính vào bất kỳ số liệu nào.',
  createKey: 'Tạo khóa chuyển đổi',
  otherCurrency:
    'Lượt chuyển đổi bằng đơn vị tiền tệ khác được giữ lại và hiển thị theo đơn vị đó, nhưng không được tính vào các tổng.',
  colError: 'Lỗi',
  colWhen: 'Khi nào',
  colDo: 'Cách xử lý',
  errMissingWhen: 'Không gửi cả <mono>email</mono> lẫn <mono>click_token</mono>',
  errMissingDo: 'Gửi ít nhất một trong hai',
  errValueWhen: '<mono>value</mono> không phải là chuỗi số thập phân',
  errValueDo: 'Gửi “29.00”, không phải 29',
  errCurrencyWhen: 'Không phải mã ISO 4217',
  errCurrencyDo: 'Dùng USD, EUR, GBP…',
  errRangeWhen: 'Cách đây hơn 30 ngày, hoặc ở tương lai',
  errRangeDo: 'Gửi thời điểm chuyển đổi xảy ra',
  errRateWhen: 'Vượt giới hạn tần suất của khóa',
  errRateDo: 'Chờ hết thời gian trong <mono>Retry-After</mono>',
  sampleName: 'Upgraded to Team plan',
  errorLoad: 'Không tải được phân bổ',
})
