import { translate } from '@/i18n'
import { m } from './messages'

/** Vietnamese for the connect wizard's frame and shared parts (docs/i18n/glossary-vi.md). */
export default translate(m, {
  // ── Top bar ──────────────────────────────────────────────────────────
  exitSetup: 'Thoát thiết lập',
  stepBadge: 'Bước {n}/6',
  title: 'Kết nối nhà cung cấp gửi thư',

  // ── Stepper ──────────────────────────────────────────────────────────
  stepsLabel: 'Các bước thiết lập',
  stepDone: '<vh>Xong:</vh> {step}',
  stepProvider: 'Nhà cung cấp',
  stepProviderLine: 'Chọn từ danh mục',
  stepConnection: 'Kết nối',
  stepConnectionLine: 'Khóa, khu vực, webhook',
  stepDomain: 'Xác minh tên miền',
  stepDomainLine: 'DKIM, return-path, SPF, DMARC',
  stepTest: 'Gửi thử',
  stepTestLine: 'Theo dõi sự kiện trả về',
  stepStreams: 'Luồng',
  stepStreamsLine: 'Gửi hàng loạt, Tự động hóa, Hệ thống',
  stepContacts: 'Liên hệ',
  stepContactsLine: 'Nhập hoặc bắt đầu trống',
  stepStreamsCurrent: '{provider} sẽ gửi những gì',
  stepStreamsCurrentNoName: 'Nhà cung cấp sẽ gửi những gì',
  stepsFoot: 'Hầu hết mọi người hoàn tất trong khoảng 10 phút. Bạn có thể thêm nhà cung cấp sau và gửi qua nhiều nhà cung cấp cùng lúc.',

  // ── Status badges of a row ───────────────────────────────────────────
  stateDone: 'Xong',
  stateFailed: 'Thất bại',
  stateSkipped: 'Bỏ qua',
  statePending: 'Đang chờ',
  stateYes: 'Có',
  statePartial: 'Một phần',
  stateNo: 'Không',

  // ── Copying ──────────────────────────────────────────────────────────
  copied: '{what, select, zone {Đã sao chép tệp zone} secret {Đã sao chép khóa bí mật webhook} url {Đã sao chép URL webhook} other {Đã sao chép giá trị {label}}}',
  copyFailed:
    '{what, select, zone {Không sao chép được tệp zone} secret {Không sao chép được khóa bí mật webhook} url {Không sao chép được URL webhook} other {Không sao chép được giá trị {label}}}',
  copyBlocked: 'Trình duyệt đã chặn bộ nhớ tạm. Hãy chọn văn bản và tự sao chép.',

  // ── Read-only notes ──────────────────────────────────────────────────
  readOnlyCatalog: 'Bạn có thể xem danh mục và các thiết lập người khác đã bắt đầu. Để kết nối nhà cung cấp, bạn cần quyền {perm}.',
  readOnlySetup: 'Bạn có thể theo dõi thiết lập này. Để kết nối hoặc thay đổi nhà cung cấp, bạn cần quyền {perm}.',
})
