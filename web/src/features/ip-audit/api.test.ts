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
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { getIpAuditMonthRows } from './api'

afterEach(() => vi.restoreAllMocks())

it('loads every audit page, including rows beyond the server page-size limit', async () => {
  const rows = Array.from({ length: 205 }, (_, i) => ({ ip: `10.0.0.${i}` }))
  const signal = new AbortController().signal
  const get = vi.spyOn(api, 'get').mockImplementation(async (url, config) => {
    const params = new URL(url ?? '', 'https://example.test').searchParams
    const page = Number(params.get('page'))
    expect(params.get('page_size')).toBe('100')
    expect(config?.signal).toBe(signal)
    return {
      data: {
        success: true,
        data: {
          rows: rows.slice((page - 1) * 100, page * 100),
          total: 205,
          stats: {},
        },
      },
    }
  })
  const data = await getIpAuditMonthRows('2026-09', signal)
  expect(data.rows).toEqual(rows)
  expect(get).toHaveBeenCalledTimes(3)
})

it.each(['failure', 'empty'])(
  'rejects incomplete data when a later page is %s',
  async (kind) => {
    vi.spyOn(api, 'get')
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: { rows: [{ ip: '10.0.0.1' }], total: 2, stats: {} },
        },
      })
      .mockResolvedValueOnce({
        data:
          kind === 'failure'
            ? { success: false, message: 'Query failed' }
            : { success: true, data: { rows: [], total: 2, stats: {} } },
      })
    await expect(getIpAuditMonthRows('2026-09')).rejects.toThrow()
  }
)
