import { translate } from '@/i18n'
import { mi } from './incidents.messages'

/**
 * Vietnamese for Sending providers in an incident (docs/i18n/glossary-vi.md,
 * "Sending and deliverability"). "backfill" is "lấy bù"; "reroute" is
 * "định tuyến lại".
 */
export default translate(mi, {
  // ── Re-check ─────────────────────────────────────────────────────────
  stillDown: '{name} vẫn ngừng hoạt động',
  answered: '{name} đã phản hồi',
  stillDownDetail: 'Lần kiểm tra tự động tiếp theo chạy trong vòng 5 phút.',
  refreshed: 'Đã làm mới tình trạng.',
  checkFailed: 'Kiểm tra thất bại',

  // ── Outage banner ────────────────────────────────────────────────────
  outageTitle: '{provider} ngừng hoạt động: lỗi API từ {time} ({since})',
  outageReturns: '{hasOp, select, yes {{op} trả về {codes}} other {API trả về {codes}}}{hasRate, select, yes { ở {rate} số lần gọi} other {}}.',
  outageHandled: '{handled} đã tự xử lý.',
  outageWaiting: '{waiting} {verb, select, are {đang chờ bạn quyết định} other {đang chờ bạn quyết định}}.',
  outageHandledWaiting: '{handled} đã tự xử lý; {waiting} {verb, select, are {đang chờ bạn quyết định} other {đang chờ bạn quyết định}}.',
  statusPage: 'Trạng thái {provider}',
  recheckNow: 'Kiểm tra lại {provider} ngay',

  failoverActive: 'Đang chuyển dự phòng',
  fallbackUnavailable: 'Dự phòng không khả dụng',
  waitingDecision: 'Đang chờ quyết định',
  held: 'Tạm giữ',
  sentDirectly: '{list} gửi từ {time} đã đi thẳng qua {primary}.',
  nothingSent: 'Chưa gửi gì trên {stream} từ {time}.',
  fallbackBody: 'Nhà cung cấp chính {primary} {unavailable, select, yes {cũng không khả dụng} other {vẫn ổn định}}; dự phòng {provider} của nó không khả dụng.',
  suggestion: 'Gợi ý: thêm {provider} làm dự phòng thứ hai cho {stream}.',
  failoverBody:
    'Chính {provider} → dự phòng {fallback}, tự động từ {time}. {hasList, select, yes {{count, plural, other {Đã định tuyến lại # email cho đến nay: {list}.}}} other {{count, plural, other {Đã định tuyến lại # email cho đến nay.}}}}',
  switchBack: 'Sẽ chuyển lại về {provider} sau {minutes, plural, other {# phút ổn định}}.',
  heldBody: 'Thư {stream} mới cho {provider} được tạm giữ đến khi nó hồi phục.',
  heldBodyPolicy: 'Thư {stream} mới cho {provider} được tạm giữ đến khi nó hồi phục: chuyển dự phòng là {policy}.',

  // ── A broadcast held for a decision ──────────────────────────────────
  waitingFor: 'Đang chờ {provider}',
  waitingForDetail: '"{name}" sẽ tự tiếp tục khi {provider} ổn định trở lại.',
  moved: '{count, plural, other {Đã chuyển # người nhận}}',
  movedDetail: '"{name}" tiếp tục trên {provider}.',
  movedDetailFallback: '"{name}" tiếp tục trên nhà cung cấp dự phòng.',
  cancelled: 'Đã hủy "{name}"',
  cancelledDetail: '{count, plural, other {# người nhận đang tạm giữ sẽ không được gửi.}}',
  decisionFailed: 'Chưa lưu được quyết định',
  policyWarmOnly: 'Chuyển dự phòng Gửi hàng loạt chỉ sang nhà cung cấp đã khởi động xong, và hiện không có nhà cung cấp nào như vậy.',
  policyPause: 'Chuyển dự phòng Gửi hàng loạt là tạm dừng và hỏi, vì chuyển một lần gửi hàng loạt sang nhà cung cấp chưa khởi động giữa chừng có thể làm hại uy tín.',
  decidedWait: '"{name}": đang chờ {provider}; sẽ tự tiếp tục sau {minutes, plural, other {# phút ổn định}}.',
  decidedMove: '"{name}": phần tạm giữ đã được chuyển.',
  decidedCancel: '"{name}": lần gửi đã bị hủy.',
  openLive: 'Mở màn hình theo dõi trực tiếp',
  heldLine: '"{name}" đã gửi được {through} phần của {provider} khi {provider} gặp sự cố: {sent} đã gửi, {held} đang tạm giữ.',
  othersKeep: '{count, plural, other {{others} vẫn tiếp tục gửi phần của mình.}}',
  waitAutoResume: 'Chờ {provider}, tự tiếp tục',
  moveTo: 'Chuyển sang {provider}',
  moveToOverCap: 'Chuyển sang {provider} · vượt giới hạn, không khuyến nghị',
  moveToWarming: 'Chuyển sang {provider} · đang khởi động',
  cancelSend: 'Hủy lần gửi',
  moveTitle: '{count, plural, other {Chuyển # người nhận sang {provider}?}}',
  moveOverCap: '{provider} đang khởi động: số này vượt giới hạn hôm nay của nó, và bảo vệ khởi động có thể tạm dừng nó trong vài phút. Chờ {current} là lựa chọn an toàn hơn.',
  moveBody: 'Phần tạm giữ của "{name}" sẽ tiếp tục trên {provider} ngay.',
  keepWaiting: 'Tiếp tục chờ',
  cancelTitle: 'Hủy "{name}"?',
  cancelBody: '{held} người nhận đang tạm giữ sẽ không nhận chiến dịch này. {sent} thư đã gửi vẫn giữ nguyên, và phần còn lại của lần gửi dừng lại. Không thể hoàn tác.',

  // ── Incident timeline ────────────────────────────────────────────────
  timelineTitle: 'Dòng thời gian sự cố',
  allIncidents: 'Mọi sự cố',
  incidentKind:
    '{kind, select, broadcast_held {đang chờ} failover {chuyển dự phòng} failover_ended {đã chuyển lại} provider_down {ngừng hoạt động} provider_recovered {đã hồi phục} campaign_started {đã bắt đầu} webhooks_silent {im lặng} webhooks_resumed {đã hoạt động lại} other {{raw}}}',

  // ── Silent webhooks ──────────────────────────────────────────────────
  testReceived: 'Đã nhận sự kiện thử',
  testFailed: 'Sự kiện thử thất bại',
  registered: 'Đã đăng ký lại webhook',
  registeredDetail: '{provider} giờ gửi sự kiện đến {url}.',
  registerFailed: 'Không đăng ký lại được webhook',
  backfilling: '{count, plural, other {Đang lấy bù trạng thái cho # thư}}',
  backfillingDetail: 'Trạng thái từ {provider} được chuẩn hóa khi về đến.',
  backfillFailed: 'Không bắt đầu lấy bù được',
  manualTitle: 'Đăng ký lại webhook trong {provider}',
  manualDescription: '{provider} không có API cho webhook, nên hãy thêm lại endpoint của ReputeCast ngay trong {provider}.',
  manualStep1: 'Trong {provider}, mở máy chủ thư, rồi Webhooks, và xóa webhook ReputeCast cũ nếu vẫn còn.',
  manualStep2: 'Thêm một webhook với URL này, cho mọi sự kiện:',
  manualStep3: 'Quay lại đây, chọn Gửi sự kiện thử: thanh thông báo sẽ biến mất ngay khi sự kiện về đến.',
  silentForHours: 'Webhook của {provider} đã im lặng {hours, plural, other {# giờ}}',
  silentFor: 'Webhook của {provider} đã im lặng {duration}',
  silentBody:
    'Không có sự kiện gửi thành công, trả thư hay khiếu nại nào từ {time}. {provider} đã nhận {count} thư trong thời gian đó, nên việc gửi vẫn hoạt động, nhưng trạng thái của chúng chưa rõ và cơ chế bảo vệ trả thư cho tuyến này không còn tác dụng. Tuyến {provider} được đánh dấu Suy giảm{held, select, yes { và phần Gửi hàng loạt của nó bị tạm giữ đến khi sự kiện trở lại.} other {.}}',
  sendTestEvent: 'Gửi sự kiện thử',
  reregister: 'Đăng ký lại webhook',
  backfillFrom: 'Lấy bù từ API {provider}',
  checkedTitle: 'Những gì ReputeCast đã kiểm tra',
  causesTitle: 'Nguyên nhân có thể',
  impactTitle: 'Ảnh hưởng đến nay',
  impactUnknown: 'Thư chưa rõ trạng thái',
  impactCampaigns: 'Chiến dịch bị ảnh hưởng',
  impactBounces: 'Thư trả lại chưa được chặn gửi',
  unknown: 'chưa rõ',
  impactShare: 'Phần Gửi hàng loạt của {provider}',
  shareHeld: 'tạm giữ',
  shareNotUsed: 'không dùng',
  backfillNote: 'Lấy bù sẽ kéo trạng thái thư từ API của {provider} và chuẩn hóa, để báo cáo và danh sách chặn gửi theo kịp.',
  noBackfillNote: 'Không thể hỏi {provider} về các sự kiện đã qua; chúng sẽ về khi webhook của nó hoạt động trở lại.',

  checkApi: 'API {provider} kết nối được: {ok, select, no {không} other {có}}{hasAt, select, yes {, {time}} other {}}{hasDetail, select, yes { ({detail})} other {}}.',
  checkRegisteredNo: 'Webhook đã đăng ký trong {provider}: không{hasDetail, select, yes {, {detail}} other {}}.',
  checkRegisteredYes: 'Webhook đã đăng ký trong {provider}: có{hasId, select, yes {, id {id}} other {}}, trỏ đến <mono>{url}</mono>.',
  checkEndpoint: 'Endpoint của chúng tôi truy cập được từ internet: {ok, select, no {không} other {có}}{hasDetail, select, yes {, {detail}} other {}}.',
  checkProviderLog:
    'Sự kiện {provider} đã gửi trong 6 giờ qua: {count}{hasQueued, select, yes { (nhật ký webhook của chính {provider} cho thấy {queued} đang chờ, {count} đã gửi)} other {}}.',

  causeWorkerStuckFirst:
    '<b>Tiến trình gửi webhook của {provider} bị kẹt.</b> Sự kiện đang chờ mà không cái nào được gửi cho thấy lỗi ở {postal, select, yes {tiến trình postal-worker} other {tiến trình gửi webhook}}, thường khởi động lại là ổn.',
  causeWorkerStuck: 'Tiến trình gửi webhook của {provider} bị kẹt: sự kiện đang chờ nhưng không cái nào được gửi.',
  causeFirewallFirst: '<b>Một thay đổi tường lửa trên máy chủ {provider}</b> đang chặn HTTPS đi ra đến {host}.',
  causeFirewall: 'Một thay đổi tường lửa trên máy chủ {provider} đang chặn HTTPS đi ra đến {host}.',
  causeSecretFirst: 'Khóa bí mật của endpoint đã được xoay vòng trong ReputeCast nhưng không trong {provider} (chữ ký sẽ lỗi với mã 401, và chúng tôi chưa nhận được yêu cầu nào).',
  causeSecret: 'Ít khả năng hơn: khóa bí mật của endpoint đã được xoay vòng trong ReputeCast nhưng không trong {provider} (chữ ký sẽ lỗi với mã 401, và chúng tôi chưa nhận được yêu cầu nào).',
  causeRemovedFirst: '<b>Webhook đã bị gỡ hoặc tắt trong {provider}.</b> Hãy đăng ký lại để {provider} trỏ về ReputeCast.',
  causeRemoved: 'Webhook đã bị gỡ hoặc tắt trong {provider}.',
  causeUnreachableFirst: '<b>Endpoint của ReputeCast không truy cập được từ internet.</b> Hãy kiểm tra DNS, TLS và reverse proxy cho {host}.',
  causeUnreachable: 'Endpoint của ReputeCast không truy cập được từ internet (DNS, TLS hoặc reverse proxy cho {host}).',

  awaitingTitle: 'Thư đang chờ trạng thái từ {provider}',
  awaitingCount: '{count} · mới nhất trước',
  colAccepted: 'Đã nhận lúc',
  colRecipient: 'Người nhận',
  colCampaign: 'Chiến dịch',
  colMessageId: 'Mã thư {provider}',
  colStatus: 'Trạng thái',
  acceptedNoEvent: 'đã nhận · chưa có sự kiện',
  awaitingStatus:
    '{status, select, delivered {gửi thành công} bounced {bị trả lại} deferred {bị hoãn nhận} complained {bị khiếu nại} rejected {bị từ chối} other {đã nhận}}{hasTime, select, yes { {time}} other {}}',

  // ── Rejected credentials ─────────────────────────────────────────────
  authTitle: '{provider} trả về {status} {statusText}: {credential} không còn hợp lệ',
  authEveryCall: 'Mọi lệnh gọi từ {time} đều thất bại.',
  authFailingOver: 'Luồng {stream} ({mail}) đang tự động chuyển dự phòng sang {fallback};',
  authFailingOverRerouted: 'Luồng {stream} ({mail}) đang tự động chuyển dự phòng sang {fallback}; đã định tuyến lại {count} thư cho đến nay.',
  streamMail:
    '{stream, select, system {xác nhận đăng ký hai bước, liên kết trung tâm tùy chọn} automation {email do tự động hóa gửi} broadcast {chiến dịch} other {đặt lại mật khẩu, biên nhận và cảnh báo}}',
  authRotated: 'Có thể ai đó đã xoay vòng {credential} trong {provider}.',
  update: 'Cập nhật {credential}',
  viewFailed: 'Xem các lệnh gọi thất bại',
  auditLog: 'Nhật ký kiểm tra',
  lastSuccessCall: 'Lệnh gọi thành công gần nhất',
  noneRecorded: 'chưa ghi nhận',
  errorFact: 'Lỗi',
  credentialsRejected: 'Thông tin xác thực bị từ chối',
  authWebhooks: '{hasEvent, select, yes {sự kiện gần nhất lúc {time}} other {chưa có sự kiện}} · không bị ảnh hưởng cho đến khi sửa xong {credential}',
  streamsAffected: 'Luồng bị ảnh hưởng',
  streamRole: '{stream} · {role, select, split {chia} primary {chính} fallback {dự phòng} other {điều kiện}}',
  failoverTo: 'Đang chuyển dự phòng sang {provider}',
})
