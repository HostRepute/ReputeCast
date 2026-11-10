/**
 * The allowed IPs field of an API key or MCP agent: entries are addresses
 * or networks (203.0.113.7, 203.0.113.0/24, 2001:db8::/48), one a line or
 * separated by commas. The server reads and checks them (server.ParsePrefix);
 * the browser only splits the text and tells whether an address it already
 * knows (where the key was last used) stays inside the list.
 */

/** One entry of the field, with the line it's on (from 1). */
export type IpEntry = { value: string; line: number }

/** Splits the field on new lines and commas, trimmed; empty pieces are dropped. */
export function ipEntries(text: string): IpEntry[] {
  const out: IpEntry[] = []
  text.split(/\r?\n/).forEach((l, i) => {
    for (const piece of l.split(',')) {
      const value = piece.trim()
      if (value) out.push({ value, line: i + 1 })
    }
  })
  return out
}

type Addr = { bits: 32 | 128; n: bigint }

function v4(s: string): bigint | undefined {
  const parts = s.split('.')
  if (parts.length !== 4) return undefined
  let n = 0n
  for (const p of parts) {
    if (!/^\d{1,3}$/.test(p) || Number(p) > 255) return undefined
    n = (n << 8n) | BigInt(p)
  }
  return n
}

function v6(s: string): bigint | undefined {
  if (s.includes('%')) return undefined
  let text = s
  let tail: bigint | undefined
  if (s.includes('.')) {
    const at = s.lastIndexOf(':')
    tail = v4(s.slice(at + 1))
    if (tail === undefined) return undefined
    text = `${s.slice(0, at + 1)}0:0`
  }
  const halves = text.split('::')
  if (halves.length > 2) return undefined
  const head = halves[0] ? halves[0].split(':') : []
  const rest = halves.length === 2 && halves[1] ? halves[1].split(':') : []
  const missing = 8 - head.length - rest.length
  if (halves.length === 1 ? missing !== 0 : missing < 1) return undefined
  let n = 0n
  for (const g of [...head, ...Array<string>(halves.length === 2 ? missing : 0).fill('0'), ...rest]) {
    if (!/^[0-9a-f]{1,4}$/i.test(g)) return undefined
    n = (n << 16n) | BigInt(parseInt(g, 16))
  }
  return tail === undefined ? n : (n & ~0xffffffffn) | tail
}

/** An address; an IPv4 address written as IPv6 (::ffff:203.0.113.7) is the IPv4 one. */
function addr(s: string): Addr | undefined {
  if (!s.includes(':')) {
    const n = v4(s)
    return n === undefined ? undefined : { bits: 32, n }
  }
  const n = v6(s)
  if (n === undefined) return undefined
  return n >> 32n === 0xffffn ? { bits: 32, n: n & 0xffffffffn } : { bits: 128, n }
}

function network(s: string): (Addr & { prefix: number }) | undefined {
  const [a, len, extra] = s.split('/')
  if (extra !== undefined) return undefined
  const base = addr(a ?? '')
  if (!base) return undefined
  if (len === undefined) return { ...base, prefix: base.bits }
  if (!/^\d{1,3}$/.test(len)) return undefined
  // A 4-in-6 network's length counts the 96 bits of its ::ffff: prefix.
  const prefix = Number(len) - (base.bits === 32 && a?.includes(':') ? 96 : 0)
  return prefix < 0 || prefix > base.bits ? undefined : { ...base, prefix }
}

/** Whether ip is inside one of the networks; false when it can't be read. */
export function ipCovered(networks: string[], ip: string): boolean {
  const a = addr(ip.trim())
  if (!a) return false
  return networks.some((s) => {
    const net = network(s.trim())
    if (!net || net.bits !== a.bits) return false
    const shift = BigInt(net.bits - net.prefix)
    return net.n >> shift === a.n >> shift
  })
}
