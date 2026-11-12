import { translate } from '@/i18n'
import { sm } from './segment.messages'

export default translate(sm, {
  visitorsInSegment: 'Trong một phân khúc',
  visitorsNotInSegment: 'Không trong phân khúc',
  segment: 'Phân khúc',
  pickSegment: 'Chọn một phân khúc',
  needsTrackingIn:
    'Cần Theo dõi website trên cùng trang. Biểu mẫu chỉ hiện cho những khách truy cập ReputeCast nhận ra là thuộc phân khúc; không hiện cho ai khác.',
  needsTrackingNotIn: 'Cần Theo dõi website trên cùng trang. Biểu mẫu ẩn với những khách truy cập ReputeCast nhận ra là thuộc phân khúc; mọi người khác đều thấy.',
  inSegmentVisitors: 'khách truy cập trong một phân khúc',
  notInSegmentVisitors: 'khách truy cập không trong phân khúc',
  segmentsError: 'Không tải được các phân khúc.',
})
