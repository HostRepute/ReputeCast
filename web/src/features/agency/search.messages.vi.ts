import { translate } from '@/i18n'
import { m } from './search.messages'

export default translate(m, {
  title: 'Tìm trong mọi khách hàng',
  typeMore: 'Hãy nhập ít nhất 2 ký tự: tên chiến dịch, tự động hóa, mẫu hoặc biểu mẫu, hoặc đúng địa chỉ email của một liên hệ.',
  failed: 'Không tìm được',
  nameCaption:
    'Với tư cách {name}: {clients, plural, other {# khách hàng của bạn}} và thư viện · {n, plural, other {# kết quả}}',
  nameCaptionAll: 'Với tư cách {name}: mọi khách hàng và thư viện · {n, plural, other {# kết quả}}',
  contactCaption: 'Với tư cách {name} · liên hệ chỉ khớp khi đúng email hoặc số điện thoại, ở những khách hàng bạn được xem liên hệ',
  noMatch: 'Không có gì khớp với “{q}” trong các khách hàng của bạn hay thư viện.',
  noContact:
    '{n, plural, =0 {Không có khách hàng nào bạn được xem liên hệ.} =1 {Không có liên hệ nào với địa chỉ đó ở khách hàng bạn được xem liên hệ.} other {Không có liên hệ nào với địa chỉ đó ở # khách hàng bạn được xem liên hệ.}}',
  skipped:
    '{n, plural, other {# khách hàng không được tìm liên hệ: bạn không được xem liên hệ ở đó.}}',
  audited: 'Lượt tìm này được ghi vào nhật ký kiểm tra: ai đã tìm, và tìm địa chỉ nào.',
  libraryGroup: 'Thư viện {agency}',
  libraryHit: '{kind} · v{version}',
  kindCampaign: 'Chiến dịch',
  kindAutomation: 'Tự động hóa',
  kindSection: 'Phần đã lưu',
  kindTemplate: 'Mẫu chiến dịch',
  kindForm: 'Biểu mẫu đăng ký',
  kindContact: 'Liên hệ',
  contactSubscribed: 'Liên hệ · {lists} · đăng ký {date}',
  contactLeft: 'Liên hệ · {lists} · hủy đăng ký {date}',
  contactLists: 'Liên hệ · {lists}',
  stApproval: 'Đang chờ phê duyệt',
  stDraft: 'Bản nháp',
  stScheduled: 'Đã lên lịch',
  stApproved: 'Đã phê duyệt',
  stSending: 'Đang gửi',
  stSent: 'Đã gửi',
  stPaused: 'Đã tạm dừng',
  stLive: 'Đang hoạt động',
})
