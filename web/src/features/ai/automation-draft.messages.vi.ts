import { translate } from '@/i18n'
import { ma } from './automation-draft.messages'

export default translate(ma, {
  title: 'Mô tả automation',
  lead: 'Soạn chuỗi email khi tham gia danh sách. Duyệt từng bước và email trước khi lưu cả bộ.',
  brief: 'Chuỗi này cần làm gì?',
  hint: 'Mô tả email, thời gian chờ và nhánh theo tương tác. Không nhập thông tin liên hệ, thông tin xác thực hay URL.',
  name: 'Tên automation',
  list: 'Danh sách đã chọn',
  chooseList: 'Chọn danh sách',
  audience: 'Sau khi bạn tự xuất bản: chỉ lượt tham gia đã xác nhận trong tương lai, mỗi người một lần, không gồm nhập dữ liệu. Người đọc rời luồng khi rời danh sách. Lưu nháp không đưa thành viên hiện có vào luồng.',
  scope: 'Mẫu này hỗ trợ email, chờ, nhánh tương tác, chia A/B và kết thúc. Giữ nguyên liên kết, ảnh và Liquid đã chọn. Trigger khác, điều kiện liên hệ, webhook và thao tác đăng ký cần dùng trình sửa luồng.',
  privacy: 'Yêu cầu, văn bản biên tập đã chọn, các bước được phép và giọng đã lưu được gửi tới mô hình. ID, tên danh sách và email, địa chỉ đích và chân thư giữ tại đây. Không đọc liên hệ hay truy cập URL.',
  waiting: 'Đang soạn các bước và email. Chưa lưu luồng hay email nào. Dừng chờ vẫn có thể phát sinh phí nhà cung cấp.',
  saved: 'Đã lưu automation và email dưới dạng nháp. Chưa xuất bản hay đưa ai vào luồng.',
  rejected: 'Đã từ chối đề xuất. Không thay đổi luồng hay email nào.',
  leaveBody: 'Yêu cầu chưa lưu sẽ đóng. Mở lại hộp thoại này để duyệt đề xuất đã tạo trước khi hết hạn sau 24 giờ. Luồng và email không bị thay đổi.',
  unsaved: 'Chờ luồng lưu xong trước khi tạo hoặc chấp nhận bản thay thế.',
  replace: 'So sánh graph và nội dung email đã lưu với đề xuất. Chấp nhận tạo các bản sao email mới và thay graph nháp này; email hiện có không thay đổi.',
  unsupported: 'Mẫu này cần lượt tham gia đã xác nhận trong tương lai, mỗi người một lần, không gồm nhập dữ liệu và thoát khi rời danh sách. Giữ quy tắc người nhận khác trong trình sửa luồng.',
  graph: 'Các bước và đường đi',
  emails: 'Bản nháp email',
  exact: 'Xem cấu trúc luồng chi tiết',
  next: 'Tiếp theo: {steps}',
  end: 'Kết thúc đường đi',
  checked: 'Đã kiểm tra graph và nội dung email. Vẫn cần kiểm tra khi xuất bản.',
})
