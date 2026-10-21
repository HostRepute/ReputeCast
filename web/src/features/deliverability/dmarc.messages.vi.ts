import { translate } from '@/i18n'
import { md } from './dmarc.messages'

/** Vietnamese for Deliverability › DMARC reports (docs/i18n/glossary-vi.md). Results and p=… policies stay as the reports write them. */
export default translate(md, {
  title: 'Báo cáo DMARC · {days} ngày qua',
  captionRua: 'báo cáo tổng hợp gửi đến {rua}',
  captionUploaded: '{count} báo cáo tổng hợp đã tải lên',
  passBadge: '{rate} đạt',
  tableCaption: 'Kết quả DMARC theo nguồn gửi',
  colSource: 'Nguồn',
  colMessages: 'Thư',
  colResult: 'Kết quả',
  notYourProvider: 'không phải nhà cung cấp của bạn',
  unknownEnforced: 'Có người đang gửi thư dưới tên {domain} từ một máy chủ lạ; p={policy} đã giữ thư đó ngoài hộp thư đến.',
  unknownNotEnforced: 'Có người đang gửi thư dưới tên {domain} từ một máy chủ lạ; với p={policy}, bên nhận vẫn có thể chuyển thư đó.',
  investigate: 'Điều tra nguồn',

  // ── Investigate source ───────────────────────────────────────────────
  sourceTitle: 'Nguồn gửi {ip}',
  sourceKnown: '{provider} gửi thư từ IP này.',
  sourceUnknown: 'IP này không thuộc nhà cung cấp nào của bạn. Nó đã gửi thư dưới tên {domain} trong 7 ngày qua.',
  nothingChanged: 'Không có gì bị thay đổi.',
  messages7d: 'Thư, 7 ngày',
  messagesInReports: '{messages} trong {reports} báo cáo',
  seen: 'Ghi nhận',
  seenRange: '{from} đến {to}',
  reportedBy: 'Báo cáo bởi',
  reverseDns: 'DNS ngược',
  none: 'không có',
  fromDomains: 'Tên miền người gửi',
  receiversDid: 'Bên nhận đã xử lý',
  dispositionNone: 'đã chuyển (không có hành động DMARC)',
  dispositionQuarantine: 'đã cách ly (thư mục spam)',
  dispositionReject: 'đã từ chối',
  unknownAdvice:
    'Nếu đây là dịch vụ bạn dùng (CRM, hệ thống hỗ trợ khách hàng, công cụ lập hóa đơn), hãy thêm nó vào bản ghi SPF của {domain} và bật ký DKIM ở đó. Nếu không phải, bạn không cần làm gì: {hasPolicy, select, yes {p={policy}} other {chính sách DMARC của bạn}} {delivered, select, yes {nên được nâng lên quarantine để bên nhận ngừng chuyển thư đó} other {giữ thư đó ngoài hộp thư đến}}.',

  // ── Without reports ──────────────────────────────────────────────────
  emptyTitle: 'Tóm tắt báo cáo DMARC',
  noReports: 'Chưa có báo cáo',
  emptyRua:
    'Báo cáo tổng hợp cần một hộp thư để nhận. Hãy trỏ <mono>rua=</mono> đến <mono>{rua}</mono> và HostRepute sẽ phân tích chúng cho bạn, hoặc dán báo cáo XML vào đây.',
  emptyNoRua:
    'Báo cáo tổng hợp cần một hộp thư để nhận. Hãy trỏ <mono>rua=</mono> đến HostRepute (phân tích sẵn cho bạn) hoặc đến bất kỳ hộp thư nào của bạn rồi dán báo cáo XML vào đây.',
  useHostRepute: 'Dùng HostRepute làm rua',

  // ── Upload ───────────────────────────────────────────────────────────
  uploadButton: 'Tải lên báo cáo XML',
  chooseFile: 'Hãy chọn tệp báo cáo trước.',
  uploaded: 'Đã thêm báo cáo từ {org}',
  uploadedDetail: '{messages} thư cho {domain}',
  uploadTitle: 'Tải lên báo cáo tổng hợp DMARC',
  uploadDescription: 'Tệp .xml mà nhà cung cấp hộp thư gửi đến địa chỉ rua= của bạn, hoặc tệp .xml.gz hay .zip chứa nó. Chỉ số liệu trong báo cáo được lưu.',
  upload: 'Tải lên',
  reportFile: 'Tệp báo cáo',
})
