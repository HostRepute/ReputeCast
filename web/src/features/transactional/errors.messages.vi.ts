import { translate } from '@/i18n'
import { me } from './errors.messages'

/** Vietnamese for the Setup tab's table of the API's errors; the codes stay as they are. */
export default translate(me, {
  template_not_found_when: 'ID không tồn tại trong không gian làm việc này',
  template_not_found_do: 'Kiểm tra ID trong mục Mẫu',
  template_not_published_when: 'Mẫu có bản nháp nhưng chưa có phiên bản đang chạy',
  template_not_published_do: 'Hãy xuất bản trước',
  missing_variables_when: 'Một biến bắt buộc không có trong {data}',
  missing_variables_do: 'Phản hồi liệt kê những biến đó',
  invalid_recipient_when: 'Địa chỉ không phải email hợp lệ',
  invalid_recipient_do: 'Hãy kiểm tra địa chỉ trong ứng dụng của bạn',
  recipient_suppressed_when: 'Địa chỉ đã bị trả thư, đã khiếu nại hoặc là bẫy thư rác',
  recipient_suppressed_do: 'Phản hồi cho biết lý do và ngày',
  sender_not_verified_when: '{from} không phải người gửi Giao dịch đã xác minh',
  sender_not_verified_do: 'Hãy xác minh trong Cài đặt › Người gửi',
  attachments_too_large_when: 'Vượt giới hạn của tuyến',
  attachments_too_large_do: '{size} trên {provider}',
  rate_limited_when: 'Vượt tốc độ của khóa',
  rate_limited_do: 'Chờ theo {header}',
  no_healthy_route_when: 'Mọi nhà cung cấp trong tuyến đều gặp sự cố',
  no_healthy_route_do: 'Thử lại với cùng khóa idempotency',
  idempotency_key_reused_when: 'Cùng một khóa đi kèm một yêu cầu khác trong vòng 24 giờ',
  idempotency_key_reused_do: 'Dùng khóa mới cho thư mới',
  template_not_allowed_when: 'Khóa không được gửi mẫu hoặc danh mục này',
  template_not_allowed_do: 'Dùng một khóa được phép, hoặc mở rộng phạm vi của khóa',
})
