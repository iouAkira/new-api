/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

/**
 * IPv4 helpers shared by the IP Audit feature: dotted-quad parsing,
 * three-format rule matching (single IP / CIDR / closed range) and the
 * client-side validation used by the add-entry sheet. The semantics mirror
 * the backend matcher so the UI can compute matched rules locally.
 */

/** Parse a dotted-quad IPv4 string to a uint32; returns null when invalid. */
export function ipToInt(ip: string): number | null {
  const parts = ip.trim().split('.')
  if (parts.length !== 4) return null
  let value = 0
  for (const part of parts) {
    if (part === '' || !/^\d+$/.test(part)) return null
    const n = Number(part)
    if (n < 0 || n > 255) return null
    value = value * 256 + n
  }
  return value
}

/** Whether a rule entry (single IP / CIDR / closed range) matches an IP. */
export function entryMatchesIp(entry: string, ip: string): boolean {
  const value = ipToInt(ip)
  if (value === null) return false
  const rule = String(entry).trim()

  if (rule.includes('/')) {
    const [base, bitsRaw] = rule.split('/')
    const baseValue = ipToInt(base)
    const bits = Number(bitsRaw)
    if (baseValue === null || bitsRaw.trim() === '' || !Number.isInteger(bits) || bits < 0 || bits > 32) {
      return false
    }
    const size = Math.pow(2, 32 - bits)
    const network = Math.floor(baseValue / size) * size
    return value >= network && value < network + size
  }

  if (rule.includes('-')) {
    const [loRaw, hiRaw] = rule.split('-')
    const lo = ipToInt(loRaw)
    const hi = ipToInt(hiRaw)
    if (lo === null || hi === null || lo > hi) return false
    return value >= lo && value <= hi
  }

  return ipToInt(rule) === value
}

/** Ids of list entries whose rule matches the given IP (may be several). */
export function matchedEntryIds(
  entries: { id: number; ip: string }[],
  ip: string
): number[] {
  return entries
    .filter((entry) => entryMatchesIp(entry.ip, ip))
    .map((entry) => entry.id)
}

export interface EntryFormatError {
  key: string
  params?: Record<string, string>
}

/**
 * Validate one of the three accepted rule formats.
 * Returns a translation-ready error, or null when the value is valid.
 */
export function validateEntryFormat(raw: string): EntryFormatError | null {
  const entry = String(raw ?? '').trim()
  if (!entry) return { key: 'Please enter an IP / range' }

  if (entry.includes('/')) {
    const parts = entry.split('/')
    if (parts.length !== 2) {
      return { key: 'CIDR format should be network/prefix, e.g. 10.0.0.0/24' }
    }
    const [base, bitsRaw] = parts
    if (ipToInt(base) === null) {
      return {
        key: 'Invalid network base: {{base}}',
        params: { base: base.trim() },
      }
    }
    const bits = Number(bitsRaw)
    if (bitsRaw.trim() === '' || !Number.isInteger(bits) || bits < 0 || bits > 32) {
      return { key: 'Prefix must be an integer between 0 and 32' }
    }
    return null
  }

  if (entry.includes('-')) {
    const parts = entry.split('-')
    if (parts.length !== 2) {
      return {
        key: 'Range format should be startIP-endIP, e.g. 172.26.64.0-172.26.147.0',
      }
    }
    const lo = ipToInt(parts[0])
    const hi = ipToInt(parts[1])
    if (lo === null || hi === null) {
      return { key: 'Invalid range endpoints' }
    }
    if (lo > hi) {
      return { key: 'Start IP must not be greater than end IP' }
    }
    return null
  }

  if (ipToInt(entry) === null) {
    return {
      key: 'Invalid IP format, expected dotted decimal like 172.26.1.100',
    }
  }
  return null
}
