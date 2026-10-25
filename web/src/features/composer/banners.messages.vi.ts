import { translate } from '@/i18n'
import { mn } from './banners.messages'

/** Vietnamese for the composer's banners (xung đột khi lưu, nội dung khôi phục từ trình duyệt). */
export default translate(mn, {
  conflictTitle: '{who} đã lưu v{version} trong lúc bạn đang sửa.',
  conflictBoth: 'Cả hai bạn đều sửa {parts}. Nội dung bạn sửa vẫn nằm trong trình duyệt này cho đến khi bạn chọn; dù chọn cách nào cũng không mất gì.',
  conflictTheirs: '{name} đã sửa {parts}. Nội dung bạn sửa vẫn nằm trong trình duyệt này cho đến khi bạn chọn; dù chọn cách nào cũng không mất gì.',
  conflictPlain: 'Nội dung bạn sửa vẫn nằm trong trình duyệt này cho đến khi bạn chọn; dù chọn cách nào cũng không mất gì.',
  resolveGroup: 'Giải quyết xung đột',
  keepMine: 'Giữ bản của tôi',
  keepMineText: 'Lưu bản của bạn thành v{next} trên v{version}. Bản v{version} của {name} vẫn nằm trong lịch sử phiên bản.',
  compare: 'So sánh',
  compareText: 'Hiện hai phiên bản cạnh nhau để bạn lấy từng phần từ mỗi bản.',
  loadTheirs: 'Mở bản của họ',
  loadTheirsText: 'Giữ bản của bạn trong lịch sử phiên bản thành v{next}, rồi mở bản v{version} của {name}.',
  someone: 'Một người khác',
  loadTheirsDone: 'Đang mở bản v{version} của {name}',
  loadTheirsDoneDetail: 'Nội dung bạn sửa nằm trong lịch sử phiên bản ở v{kept}.',
  loadTheirsFailed: 'Không mở được bản của họ',
  partSubject: 'tiêu đề',
  partBody: 'nội dung email',
})
