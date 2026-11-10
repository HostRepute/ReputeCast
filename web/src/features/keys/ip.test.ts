import { describe, expect, it } from 'vitest'
import { ipCovered, ipEntries } from './ip'

describe('ipEntries', () => {
  it('splits on lines and commas and keeps the line of each entry', () => {
    expect(ipEntries(' 203.0.113.7\n\n198.51.100.0/24, 2001:db8::/48 ,\r\nx')).toEqual([
      { value: '203.0.113.7', line: 1 },
      { value: '198.51.100.0/24', line: 3 },
      { value: '2001:db8::/48', line: 3 },
      { value: 'x', line: 4 },
    ])
    expect(ipEntries('  \n ')).toEqual([])
  })
})

describe('ipCovered', () => {
  it('finds an address in a network or as itself', () => {
    expect(ipCovered(['203.0.113.0/24'], '203.0.113.200')).toBe(true)
    expect(ipCovered(['203.0.113.7'], '203.0.113.7')).toBe(true)
    expect(ipCovered(['203.0.113.0/24'], '203.0.114.1')).toBe(false)
    expect(ipCovered(['2001:db8::/48'], '2001:db8:0:5::1')).toBe(true)
    expect(ipCovered(['2001:db8::/48'], '2001:db9::1')).toBe(false)
    expect(ipCovered(['0.0.0.0/0'], '198.51.100.4')).toBe(true)
  })
  it('reads IPv4 written as IPv6', () => {
    expect(ipCovered(['::ffff:203.0.113.0/120'], '203.0.113.9')).toBe(true)
    expect(ipCovered(['203.0.113.0/24'], '::ffff:203.0.113.9')).toBe(true)
  })
  it('never matches what it cannot read or another family', () => {
    expect(ipCovered(['not-an-ip', '203.0.113.0/33', 'fe80::1%eth0'], '203.0.113.1')).toBe(false)
    expect(ipCovered(['2001:db8::/32'], '203.0.113.1')).toBe(false)
    expect(ipCovered(['203.0.113.0/24'], '')).toBe(false)
  })
})
