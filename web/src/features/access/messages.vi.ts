import { translate } from '@/i18n'
import { m } from './messages'

export default translate(m, {
  signInPassword: 'Mật khẩu',
  signInPasskeyFirst: 'Khóa truy cập',
  signInPasskey: 'khóa truy cập',

  twoFAOn: '{method, select, passkey {Bật · khóa truy cập} other {Bật · TOTP}}',
  twoFAIdp: 'Qua IdP',
  twoFARequired: 'Bắt buộc · đang tắt',
  graceToday: 'hết hạn gia hạn hôm nay',
  graceDays: 'còn {days} ngày gia hạn',
  graceOver: 'đã hết thời gian gia hạn',
  twoFAOff: 'Tắt',

  scopeList: 'chỉ danh sách {list}',
  scopeOnly: 'chỉ {name}',
  scopeLists: '{count} danh sách',
  scopeParts: 'chỉ {parts}',

  granted: 'Được cấp',
  notGranted: 'Không được cấp',
})
