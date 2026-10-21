import { translate } from '@/i18n'
import { md } from './domain.messages'

/** Vietnamese for the domain step and a domain's records (docs/i18n/glossary-vi.md). DMARC's policy words stay as DNS writes them. */
export default translate(md, {
  // ── The step ─────────────────────────────────────────────────────────
  setupFailed: 'Không thiết lập được {domain}',
  skipFailed: 'Không bỏ qua được bước tên miền',
  sendingDomain: 'Tên miền gửi thư',
  fetching: 'Đang lấy…',
  useDomain: 'Dùng {domain}',
  enterDomain: 'Hãy nhập tên miền bạn dùng để gửi thư.',
  showRecords: 'Xem các bản ghi DNS',
  headingChosen: 'Thêm các bản ghi DNS này vào {domain}',
  headingChoose: 'Bạn sẽ gửi thư từ tên miền nào?',
  leadChosen: 'Đây đúng là các bản ghi {provider} cần cho tên miền này. Trạng thái cập nhật trực tiếp; hầu hết nhà đăng ký tên miền cập nhật trong vài phút, một số mất đến 48 giờ.',
  leadChoose: '{provider} ký thư cho tên miền trong địa chỉ người gửi của bạn. ReputeCast hỏi {provider} các bản ghi mà tên miền đó cần và kiểm tra khi chúng xuất hiện.',
  loadFailed: 'Không tải được bản ghi',
  loading: 'Đang tải bản ghi…',
  skip: 'Tạm bỏ qua · vẫn gửi thử được từ địa chỉ đã xác minh',
  chooseFirst: 'Hãy chọn tên miền gửi thư trước, hoặc tạm bỏ qua.',
  continueTest: 'Tiếp tục gửi thử',

  // ── Tools above the records ──────────────────────────────────────────
  nameservers: 'Máy chủ tên miền: {host}',
  addViaCloudflare: 'Thêm mọi bản ghi qua API Cloudflare',
  copyZone: 'Sao chép dạng tệp zone',
  allVerified: 'Mọi bản ghi đã xác minh · kiểm tra lần cuối lúc {time}',
  checking: 'Kiểm tra mỗi 30 giây · lần cuối lúc {time}',

  // ── The records ──────────────────────────────────────────────────────
  recordsTitle: 'Bản ghi · đã xác minh {verified} trên {total}',
  verifiedCount: '{count} đã xác minh',
  pendingCount: '{count} đang chờ',
  missingCount: '{count} còn thiếu',
  caption: 'Bản ghi DNS của {domain}',
  colPurpose: 'Mục đích',
  colType: 'Loại',
  colName: 'Tên',
  colValue: 'Giá trị',
  colStatus: 'Trạng thái',
  colCopy: 'Sao chép',
  found: 'Đang có: <mono>{value}</mono>',
  copyValueLabel: 'Sao chép giá trị {record}',
  dmarcTip: '<b>Mẹo DMARC:</b> {tip}',
  statusVerified: 'Đã xác minh',
  statusPending: 'Đang chờ',
  statusMissing: 'Còn thiếu',
  statusUnknown: 'Không kiểm tra được',

  // ── BIMI and the tracking domain ─────────────────────────────────────
  addTrackingFailed: 'Không thêm được tên miền theo dõi',
  optionalTitle: 'Không bắt buộc',
  bimiPublished: 'Đã công bố: logo của bạn hiển thị trong Gmail, Yahoo và Apple Mail ở những nơi bên nhận hỗ trợ.',
  bimiInfo:
    'Hiển thị logo của bạn trong Gmail, Yahoo và Apple Mail. Cần DMARC ở mức quarantine hoặc reject và một logo SVG Tiny PS; chứng chỉ Verified Mark Certificate là không bắt buộc với Gmail.',
  trackingTitle: 'Tên miền theo dõi tùy chỉnh',
  trackingLine:
    '{state, select, verified {<mono>{domain}</mono> CNAME đến <mono>{target}</mono>: đã xác minh, liên kết và pixel đếm lượt mở dùng tên miền của bạn.} pending {<mono>{domain}</mono> CNAME đến <mono>{target}</mono>: đang chờ DNS.} wrong_value {<mono>{domain}</mono> CNAME đến <mono>{target}</mono>: đang trỏ đến nơi khác.} missing {<mono>{domain}</mono> CNAME đến <mono>{target}</mono>: hãy thêm bản ghi CNAME ở nhà cung cấp DNS của bạn.} suggested {<mono>{domain}</mono> CNAME đến <mono>{target}</mono>, để liên kết và pixel đếm lượt mở dùng tên miền của bạn.} other {<mono>{domain}</mono> CNAME đến <mono>{target}</mono>.}}',
  addTracking: 'Thêm tên miền theo dõi',

  // ── Cloudflare ───────────────────────────────────────────────────────
  cfAdded: '{count, plural, other {Đã thêm # bản ghi trong Cloudflare}}',
  cfNothing: 'Cloudflare đã có đủ mọi bản ghi',
  cfFailedSome: 'Không thêm được: {records}. Hãy thêm chúng thủ công.',
  cfChecks: 'ReputeCast kiểm tra lại chúng mỗi 30 giây.',
  cfTitle: 'Thêm các bản ghi qua API Cloudflare',
  cfDescription: 'ReputeCast thêm các bản ghi còn thiếu vào zone {domain} và sửa các bản ghi sai. Các bản ghi hiện có mà nó không cần vẫn được giữ nguyên.',
  adding: 'Đang thêm…',
  pasteToken: 'Hãy dán token API Cloudflare trước.',
  addRecords: 'Thêm bản ghi',
  tokenLabel: 'Token API Cloudflare',
  tokenHint: 'Cần quyền {permission} trên zone này. Chỉ dùng một lần và không bao giờ được lưu.',
})
