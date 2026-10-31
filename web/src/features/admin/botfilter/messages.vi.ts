import { translate } from '@/i18n'
import { m } from './messages'

export default translate(m, {
  documentTitle: 'Bộ lọc bot',
  introTitle: 'Chỉ lượt mở và lượt nhấp của người thật được tính',
  intro:
    'Lượt mở và lượt nhấp từ các mạng và user agent này là của máy, hoặc được bảo vệ quyền riêng tư nếu đến từ proxy bảo vệ quyền riêng tư. Báo cáo, phân khúc, tự động hóa và điểm số ở mọi không gian làm việc đều bỏ qua chúng. Các mục có sẵn đi kèm các bản phát hành của ReputeCast; hãy thêm những mục mà thư của người nhận đi qua.',
  nothingChanged: 'Không có gì bị thay đổi.',
  builtinSummary: 'đi kèm các bản phát hành của ReputeCast',
  remove: 'Gỡ',
  removeLabel: 'Gỡ {entry}',
  noName: '–',

  // ── Networks ─────────────────────────────────────────────────────────
  networksTitle: 'Mạng',
  networksTitleCount: 'Mạng · {count}',
  networksCaption: 'Proxy bảo vệ quyền riêng tư và trình quét thư, theo địa chỉ hoặc dải IP',
  networksLoading: 'Đang tải các mạng',
  networksLoadFailed: 'Không tải được các mạng',
  addNetwork: 'Thêm mạng',
  networksTable: 'Các mạng do quản trị viên thêm',
  networksBuiltinTable: 'Các mạng có sẵn',
  networksNoneAdded: 'Chưa có mạng nào được thêm: các mạng có sẵn bên dưới vẫn được áp dụng.',
  colNetwork: 'Mạng',
  colRunBy: 'Bên vận hành',
  colKind: 'Loại',
  colAdded: 'Đã thêm',
  colActions: 'Thao tác',
  netPrivacy: 'Proxy bảo vệ quyền riêng tư',
  netScanner: 'Trình quét thư',
  networksBuiltin: 'Có sẵn · {count} mạng',

  // ── User agents ──────────────────────────────────────────────────────
  agentsTitle: 'User agent',
  agentsTitleCount: 'User agent · {count}',
  agentsCaption: 'Đoạn chữ trong user agent của trình quét, script và bản xem trước liên kết, và của thiết bị của người thật mà các quy tắc tưởng là máy',
  agentsLoading: 'Đang tải các user agent',
  agentsLoadFailed: 'Không tải được các user agent',
  addAgent: 'Thêm user agent',
  agentsTable: 'Các user agent do quản trị viên thêm',
  agentsBuiltinTable: 'Các user agent có sẵn',
  agentsNoneAdded: 'Chưa có user agent nào được thêm: các user agent có sẵn bên dưới vẫn được áp dụng.',
  colPattern: 'Đoạn chữ trong user agent',
  colWho: 'Là ai hoặc là gì',
  kindScanner: 'Trình quét',
  kindScript: 'Script',
  kindHeadless: 'Trình duyệt headless',
  kindPreview: 'Xem trước liên kết',
  kindPerson: 'Thiết bị của người thật',
  agentsBuiltin: 'Có sẵn · {count} user agent',
  agentsBuiltinNote: 'Trình quét bảo mật, script, trình duyệt headless và bản xem trước liên kết, và {people} là của người thật',
  agentsBuiltinNoPeople: 'Trình quét bảo mật, script, trình duyệt headless và bản xem trước liên kết',

  // ── How the filter decides ───────────────────────────────────────────
  rulesTitle: 'Cách bộ lọc quyết định',
  rulesCaption: 'Cùng các quy tắc ở mọi không gian làm việc',
  rulesOpens: 'Lượt mở',
  rulesOpensBody:
    'Của máy: yêu cầu HEAD, không có user agent, các user agent ở trên, các mạng trình quét khi thư đến, và mọi lượt mở trong vòng 2 giây sau khi thư đến. Được bảo vệ quyền riêng tư: Apple Mail Privacy Protection và các proxy bảo vệ quyền riêng tư ở trên. Proxy hình ảnh của Gmail và Yahoo được tính là người thật sau 30 giây đầu tiên.',
  rulesClicks: 'Lượt nhấp',
  rulesClicksBody:
    'Của máy: các user agent ở trên, các mạng trình quét trong vòng 2 phút sau khi thư đến, lượt nhấp trong vòng 10 giây sau khi thư đến, nhấp từ 3 liên kết trở lên trong vòng 5 giây, và liên kết ẩn mà không người nào nhìn thấy. Mỗi lượt nhấp chờ một phút rồi mới được tính, để các lượt nhấp xung quanh giúp quyết định.',
  rulesWords: 'Quy tắc',
  rulesWordsBody:
    'User agent có một từ kết thúc bằng “bot”, hoặc có “crawler”, “spider”, “slurp” hay “preview”, là của máy, trừ khi từ đó là thiết bị của người thật trong danh sách.',
  rulesChanges: 'Thay đổi',
  rulesChangesBody:
    'Áp dụng cho lượt mở và lượt nhấp mới trong vòng 5 phút; các lượt đã được tính vẫn giữ nguyên. Mọi thay đổi đều có trong <a>nhật ký kiểm tra</a> của hệ thống.',

  // ── Add a network ────────────────────────────────────────────────────
  dialogEyebrow: 'Hộp thoại · quản trị viên hệ thống',
  addNetworkTitle: 'Thêm một mạng',
  addNetworkDescription:
    'Lượt mở và lượt nhấp từ mạng này được tính là của máy, hoặc là được bảo vệ quyền riêng tư, ở mọi không gian làm việc trong vòng 5 phút.',
  cidrLabel: 'Địa chỉ hoặc dải',
  cidrHint: 'Một địa chỉ hoặc một mạng, ví dụ 203.0.113.0/24 hoặc 2001:db8::/48: không rộng hơn /8 với IPv4 hoặc /16 với IPv6.',
  runByLabel: 'Bên vận hành',
  runByPlaceholder: 'Cổng thư của chúng tôi',
  kindLegend: 'Loại',
  netScannerHint:
    'Lượt mở và lượt nhấp từ mạng này khi thư đến là của máy. Các lượt sau đó có thể là người thật đứng sau cổng thư, và được tính.',
  netPrivacyHint: 'Mạng này tải hình ảnh khi thư đến, dù có ai đọc hay không, nên lượt mở của nó được bảo vệ quyền riêng tư.',
  noteLabel: 'Ghi chú',
  notePlaceholder: 'Đây là gì, bạn thấy nó ở đâu',
  cidrFirst: 'Hãy nhập địa chỉ hoặc dải trước',
  nameFirst: 'Hãy cho biết bên vận hành trước',
  networkAdded: 'Đã thêm {cidr} vào bộ lọc bot',
  networkRemoved: 'Đã gỡ {cidr} khỏi bộ lọc bot',

  // ── Add a user agent ─────────────────────────────────────────────────
  addAgentTitle: 'Thêm một user agent',
  addAgentDescription:
    'Đoạn chữ trong user agent của một máy mà thư của người nhận gặp phải, hoặc của thiết bị của người thật mà các quy tắc tưởng là máy. Áp dụng ở mọi không gian làm việc trong vòng 5 phút.',
  itIsLegend: 'Đây là',
  aMachine: 'Một máy',
  aMachineHint: 'Trình quét, trình kiểm tra liên kết hoặc script: lượt mở và lượt nhấp của nó bị bỏ qua ở bất cứ đâu đoạn chữ này có trong user agent.',
  aPerson: 'Thiết bị của người thật',
  aPersonHint:
    'Một thiết bị hoặc ứng dụng có một từ trong user agent mà các quy tắc tưởng là của máy, ví dụ một từ kết thúc bằng “bot”: lượt mở và lượt nhấp của nó được tính là của người thật.',
  patternLabel: 'Đoạn chữ trong user agent',
  patternHint: 'Không phân biệt chữ hoa chữ thường. Ít nhất 3 ký tự; thiết bị của người thật là một từ, chỉ gồm chữ và số.',
  whoLabel: 'Là ai hoặc là gì',
  whoPlaceholder: 'Cổng thư của chúng tôi',
  patternFirst: 'Hãy nhập ít nhất 3 ký tự của user agent trước',
  whoFirst: 'Hãy cho biết đó là ai hoặc là gì trước',
  agentAdded: 'Đã thêm “{pattern}” vào bộ lọc bot',
  agentRemoved: 'Đã gỡ “{pattern}” khỏi bộ lọc bot',

  // ── Remove ───────────────────────────────────────────────────────────
  removeBadge: 'Gỡ khỏi bộ lọc bot',
  removeNetworkTitle: 'Gỡ {cidr}?',
  removeNetworkDescription:
    'Lượt mở và lượt nhấp từ mạng này lại được tính là của người thật ở mọi không gian làm việc trong vòng 5 phút. Các lượt đã được tính vẫn giữ nguyên, và thay đổi này có trong nhật ký kiểm tra của hệ thống.',
  removeAgentTitle: 'Gỡ “{pattern}”?',
  removeAgentDescription:
    'Lượt mở và lượt nhấp có đoạn chữ này trong user agent lại được tính là của người thật ở mọi không gian làm việc trong vòng 5 phút. Các lượt đã được tính vẫn giữ nguyên, và thay đổi này có trong nhật ký kiểm tra của hệ thống.',
  removePersonDescription:
    'Lượt mở và lượt nhấp từ các thiết bị có từ này trong user agent lại được tính là của máy ở mọi không gian làm việc trong vòng 5 phút. Các lượt đã được tính vẫn giữ nguyên, và thay đổi này có trong nhật ký kiểm tra của hệ thống.',
  addedBy: 'Người thêm',
  addedByLine: '{hasName, select, yes {{name} vào {date}} other {{date}}} · {what}',
})
