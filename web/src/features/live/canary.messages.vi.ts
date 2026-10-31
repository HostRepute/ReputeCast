import { translate } from '@/i18n'
import { mc } from './canary.messages'

/**
 * Vietnamese for the live monitor's canary ("gửi thử": the most engaged
 * few percent first; "điều kiện đạt" are its gates).
 */
export default translate(mc, {
  titleWatching:
    '{seeds, plural, =0 {Gửi thử · {size} người nhận tương tác nhiều nhất ({percent}) · theo dõi đến {until} (còn {left} phút)} other {Gửi thử · {size} người nhận tương tác nhiều nhất ({percent}) và {seeds} hộp thư thử · theo dõi đến {until} (còn {left} phút)}}',
  titleSending:
    '{seeds, plural, =0 {Gửi thử · đang gửi tới {size} người nhận tương tác nhiều nhất ({percent})} other {Gửi thử · đang gửi tới {size} người nhận tương tác nhiều nhất ({percent}) và {seeds} hộp thư thử}}',
  titleHeld: 'Gửi thử bị giữ lại: {gate} không đạt điều kiện',
  titleStopped:
    '{seeds, plural, =0 {Gửi thử · đã dừng lúc {time} · {size} người nhận tương tác nhiều nhất ({percent}) đã nhận} other {Gửi thử · đã dừng lúc {time} · {size} người nhận tương tác nhiều nhất ({percent}) và {seeds} hộp thư thử đã nhận}}',
  sentLine: 'Đã gửi từ {from} đến {to}. {queued} người nhận đang chờ cho đến khi gửi thử đạt.',
  sendingLine: 'Đang gửi từ {from}. {queued} người nhận đang chờ cho đến khi gửi thử đạt.',
  stoppedLine: '{queued} người nhận đang chờ đã bị loại khỏi lần gửi này. Kết quả gửi thử vẫn ở trong báo cáo.',

  sumGates:
    '{collecting, plural, =0 {{passing} trên {total} điều kiện đang đạt và chưa điều kiện nào không đạt.} other {{passing} trên {total} điều kiện đang đạt và chưa điều kiện nào không đạt; {names} vẫn đang thu thập bằng chứng.}}',
  sumWatchingUntil: 'Theo dõi đến {until}.',
  sumFailed: 'Điều kiện {gate} không đạt: lần gửi bị giữ lại cho đến khi có người tiếp tục hoặc dừng.',
  sumStopped: 'Đã dừng lúc {time}. {queued} người nhận đang chờ sẽ không nhận được email này.',
  sumRates: 'Đã đổi tốc độ của lần gửi chính: {changes}. Nếu tiếp tục lúc {at}, mọi người nhận được vào khoảng {time}.',

  why: 'Vì sao gửi thử:',
  whyNeverMailed: '{count, plural, other {# người nhận chưa từng nhận thư từ {domain}.}}',
  whyNewDomain: '{domain} còn mới: đây là trong 30 ngày đầu gửi từ tên miền này.',
  whyNewIP: '{count, plural, other {Địa chỉ IP gửi {ips} chưa từng gửi thư cho không gian làm việc này.}}',
  ipOn: '{ip} trên {provider}',
  whyOverUsual: 'Số người nhận {audience} lớn hơn 2 lần mức thông thường {usual}.',
  whyChosen: 'Gửi thử đã được chọn trong Kiểm tra và gửi.',
  rulesLink: 'Quy tắc gửi thử',
  honest:
    'Gửi thử đi tới những người nhận tương tác nhiều nhất, nên nó kiểm tra việc bị chặn, hoãn nhận và vị trí vào hộp thư, không cho biết các liên hệ mới nhập sẽ phản ứng ra sao. Khiếu nại của họ được theo dõi trong lần gửi chính và {hours} giờ sau đó.',

  gates: 'Điều kiện đạt',
  gate: '{key, select, bounce {Thư trả lại} block_bounce {Trả lại do chặn/chính sách} deferral {Hoãn nhận} complaint {Khiếu nại} other {Vị trí ở hộp thư thử}}',
  gateState: '{state, select, passing {Đang đạt} failing {Không đạt} other {Đang thu thập bằng chứng}}',
  valueOf: '{rate} · {events} trên {base}',
  limit: 'giới hạn {limit}',
  limitPerMailbox: 'giới hạn {limit} cho mỗi nhà cung cấp hộp thư',
  limitSeeds: 'giới hạn {limit} vào thư rác',
  complaintValue: '{unreported, select, none {{count} từ {reported}} other {{unreported} không có số liệu · {count} từ {reported}}}',
  complaintNeeds: 'đã giao {base} trên {min} thư tới {mailbox} trước khi điều kiện này có thể không đạt',
  seedValue: '{promotions, plural, =0 {Hộp thư đến {inbox} · Thư rác {spam}} other {Hộp thư đến {inbox} · Quảng cáo {promotions} · Thư rác {spam}}}',
  seedPending: '{pending} chưa tìm thấy',
  seedMissing: '{missing} không đến',
  seedsTitle: 'Vị trí ở hộp thư thử theo nhà cung cấp hộp thư',
  seedsBar: '{seeds} hộp thư thử: {inbox} hộp thư đến, {promotions} thẻ Quảng cáo, {spam} thư rác',
  seedLegendInbox: 'Hộp thư đến {count}',
  seedLegendPromotions: 'Thẻ Quảng cáo {count}',
  seedLegendSpam: 'Thư rác {count}',
  seedsOf: '{count, plural, other {# hộp thư thử}}',
  seedsIncl: '{mailbox, select, gmail {gồm Google Workspace} outlook {gồm Microsoft 365} yahoo {gồm AOL} other {GMX, Web.de}}',
  seedsNote: 'Hộp thư thử là hộp thư kiểm tra của chính bạn trong <a>Hộp thư thử</a>; chúng cho biết thư vào đâu, không cho biết người đọc phản ứng ra sao.',

  choose: 'Chọn cách tiếp tục',
  continueNow: 'Tiếp tục gửi tới mọi người ngay',
  continueNowBody:
    'Bắt đầu lần gửi chính ngay thay vì lúc {until}. Các điều kiện còn đang thu thập bằng chứng trở thành ngưỡng của lần gửi chính. Mọi người nhận được vào khoảng {eta}.',
  continueHeldBody: 'Gửi tới {queued} người nhận mà gửi thử đã giữ lại. Mọi ngưỡng vẫn được bật cho lần gửi chính. Mọi người nhận được vào khoảng {eta}.',
  watchMore: 'Theo dõi thêm 10 phút',
  watchMoreBody: 'Giữ {queued} người nhận đang chờ đến {until}. Mọi người nhận được vào khoảng {eta}.',
  stop: 'Dừng lần gửi',
  stopBody: '{queued} người nhận đang chờ sẽ không nhận được email này. {size} người nhận của lần gửi thử đã nhận được.',
  stopConfirm: 'Dừng {name} với {queued} người nhận đang chờ?',
  stopConfirmBody: 'Không thể hoàn tác; bạn có thể gửi một bản mới sau.',
  keepWatching: 'Tiếp tục theo dõi',
  autoNote:
    'Nếu không ai chọn: lúc {until}, khi mọi điều kiện có đủ bằng chứng đều đạt, ReputeCast tự tiếp tục gửi tới mọi người. Nếu một điều kiện không đạt, lần gửi được giữ lại và bạn sẽ được hỏi.',
  stoppedBy: 'Đã dừng lúc {time} bởi {name}',
  stoppedAt: 'Đã dừng lúc {time}',
  continued: 'Đã tiếp tục gửi tới mọi người',
  watched: 'Đang theo dõi thêm 10 phút',
  stopped: 'Đã dừng lần gửi',
  decideFailed: 'Chưa thực hiện được',
  needsApprove: 'Tiếp tục sớm cần người có thể phê duyệt lần gửi{helpers, select, none {} other {: hãy hỏi {helpers}}}.',
  needsSend: 'Chỉ người có thể gửi chiến dịch mới quyết định về gửi thử{helpers, select, none {} other {: hãy hỏi {helpers}}}.',

  queuedBig: 'người nhận đang chờ cho đến khi gửi thử đạt',
  progressLine: 'Đã gửi {sent} trên {total} ({percent}) · nếu tiếp tục lúc {until}, mọi người nhận được vào khoảng {eta}',
  progressLineNow: 'Đã gửi {sent} trên {total} ({percent})',
  barLabel: 'Đã gửi thử: {parts}. {queued} đang chờ.',
  viaProvider: 'Gửi thử qua {provider} {count}',
  queuedLegend: 'Đang chờ {count}{routes, select, none {} other { · {routes}}}',
  abNote: 'Thử nghiệm A/B tiêu đề chạy trên lần gửi chính.',
  progressLabel: 'Tiến độ gửi thử',

  deliveredSub: '{rate} của gửi thử{retrying, plural, =0 {} other { · # đang thử lại}}',
  deferredSub: '{rate} của gửi thử',
  sinceDelivered: '{count} đã giao sau đó',
  retryingCount: '{count} đang thử lại',
  complainedSub: '{unreported, select, none {từ {reported}} other {từ {reported} · {unreported} không có số liệu}}',
  tileDeferred: 'Hoãn nhận',

  badge: '{phase, select, sending {Gửi thử · đang gửi} watching {Gửi thử · đang theo dõi} held {Gửi thử · bị giữ} other {Đã dừng}}',
  passed: 'Gửi thử {percent} · đạt lúc {time}',
  passedEarly: 'Gửi thử {percent} · tiếp tục lúc {time}',
  mailboxCaption: 'kết quả gửi thử · tốc độ áp dụng cho lần gửi chính',
  colCanarySent: 'Đã gửi thử',
  colDeferredOnce: 'Hoãn nhận ít nhất một lần',
  colMainRate: 'Tốc độ gửi chính',
  colThresholds: 'Ngưỡng',
  colQueued: 'Đang chờ',
  deliveredRetrying: '{rate} · {retrying} đang thử lại',
  noneSoFar: 'chưa có',
  minNote:
    '{count, plural, =0 {Ngưỡng của một nhà cung cấp hộp thư cần đã giao {min} thư tới đó mới có thể bị vượt, trừ khiếu nại và thư trả lại do chặn/chính sách rõ ràng vượt giới hạn; trong lần gửi thử chưa nhà cung cấp nào đạt số đó.} other {Ngưỡng của một nhà cung cấp hộp thư cần đã giao {min} thư tới đó mới có thể bị vượt, trừ khiếu nại và thư trả lại do chặn/chính sách rõ ràng vượt giới hạn; trong lần gửi thử chỉ {names} đạt số đó.}}',
  ratesTimes: 'Thời gian giả định lần gửi chính bắt đầu lúc {until}.',
})
