import { translate } from '@/i18n'
import { mb } from './bulk.messages'

/**
 * Vietnamese for the bulk-terms warning (docs/i18n/glossary-vi.md, "Sending
 * and deliverability"). A provider's own "Broadcast" message stream keeps
 * its name; ReputeCast's Broadcasts stream is "Gửi hàng loạt".
 */
export default translate(mb, {
  termsAllowed: 'Được phép',
  termsYourPolicy: 'Theo chính sách của bạn',
  termsNotAllowed: 'Không được phép',
  termsBroadcastOnly: 'Chỉ luồng Broadcast',
  selfHosted: '{name} (tự lưu trữ)',

  streamCreated: 'Đã tạo luồng Broadcast của {name}',
  streamCreatedDetail: 'Webhook cũng đã được đăng ký cho luồng này.',
  streamFailed: 'Chưa tạo được luồng Broadcast',

  actionCreate: 'Tạo luồng Broadcast và lưu',
  actionOther: 'Dùng {provider} và lưu',
  actionNone: 'Lưu mà không có dự phòng',
  actionKeep: 'Giữ nguyên và lưu',

  titleFallback: 'Gửi hàng loạt sẽ chuyển dự phòng sang một nhà cung cấp có điều khoản không cho phép tiếp thị hàng loạt',
  titleRoute: 'Gửi hàng loạt sẽ đi qua một nhà cung cấp có điều khoản không cho phép tiếp thị hàng loạt',
  bodySet:
    'Bạn đã đặt {where, select, fallback {dự phòng} split {phần chia} other {điều kiện}} của Gửi hàng loạt là {hasVariant, select, yes {luồng thư "{variant}" của {name}} other {"{variant}" của {name}}}.',
  bodyTerms:
    '{hasVariant, select, yes {Điều khoản của {name} chỉ cho phép luồng {variantLower} gửi thư một-một; gửi bản tin qua đó có nguy cơ bị khóa tài khoản.} other {Điều khoản của {name} chỉ cho phép tài khoản này gửi thư một-một; gửi bản tin qua đó có nguy cơ bị khóa tài khoản.}}',
  bodyCreatable: '{name} vẫn cho phép thư hàng loạt trên luồng thư Broadcast, và ReputeCast có thể tạo một luồng như vậy.',

  routeTitle: 'Tuyến Gửi hàng loạt',
  routeSub: 'chiến dịch',
  routeSubVolume: 'chiến dịch · {smallest} đến {largest} người nhận mỗi lần gửi',
  primary: 'Chính',
  weightedWith: 'Chia tỷ trọng với {list}',
  weightItem: '{provider} · {share}',
  weightItemWarming: '{provider} · {share}, đang khởi động',
  chooseProvider: 'Chọn nhà cung cấp',
  fallback: 'Dự phòng',
  nonePauseAsk: 'Không có · tạm dừng và hỏi',
  termsBadge: 'Điều khoản: không cho phép tiếp thị hàng loạt',
  failoverPolicy: 'Chính sách chuyển dự phòng',
  failoverHint: 'Áp dụng khi nhà cung cấp chính gặp lỗi trong lúc gửi.',
  policyPause: 'Tạm dừng và hỏi (mặc định cho Gửi hàng loạt)',
  policyWarm: 'Tự động, chỉ sang nhà cung cấp đã khởi động xong',
  policyAuto: 'Tự động',

  fixTitle: 'Cách khắc phục',
  createTitle: 'Tạo luồng Broadcast cho {name} (khuyến nghị)',
  createDescription:
    'ReputeCast tạo "ReputeCast Broadcasts" trong máy chủ {name} của bạn, đăng ký webhook và dùng nó làm {role, select, fallback {dự phòng} other {tuyến}}. Khoảng 1 phút.',
  otherTitle: '{role, select, fallback {Dùng dự phòng khác} other {Dùng nhà cung cấp khác}}',
  otherNone: 'Chưa có nhà cung cấp nào khác đã kết nối có điều khoản cho phép thư hàng loạt.',
  otherWarming: '{provider} là nhà cung cấp duy nhất còn lại, và đang khởi động: chuyển dự phòng sang nó sẽ bị bảo vệ khởi động chặn cho đến ngày {day}.',
  otherUse: 'Dùng {name} thay thế.',
  noneTitle: 'Không có dự phòng, tạm dừng và hỏi',
  noneDescription: 'Nếu {primary} gặp lỗi giữa lúc gửi, chiến dịch sẽ tạm dừng và bạn quyết định.',
  thePrimary: 'nhà cung cấp chính',
  keepTitle: 'Vẫn giữ {name} {variant}',
  keepDescription: 'Được ghi vào nhật ký kiểm tra. Độ sẵn sàng gửi sẽ báo Chặn cho mọi lần gửi hàng loạt cho đến khi bạn thay đổi điều này.',

  termsTitle: 'Điều khoản của từng nhà cung cấp đã kết nối cho phép gì',
  termsSub: 'từ manifest của từng adapter · xem lại mỗi quý',
  colProvider: 'Nhà cung cấp',
  colBulk: 'Tiếp thị hàng loạt',
  colNotes: 'Ghi chú từ adapter',
  colStreams: 'Luồng có thể phục vụ',
  notCreated: 'Chưa được tạo trong không gian làm việc này.',
})
