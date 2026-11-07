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
import { t } from 'i18next'

import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

import type {
  IpAuditAlertConfig,
  IpAuditApiResponse,
  IpAuditConfig,
  IpAuditDetailData,
  IpAuditListData,
  IpAuditListEntry,
  IpAuditListPayload,
  IpAuditListType,
  IpAuditQueryParams,
  IpAuditSaveConfigPayload,
  IpAuditStatusPayload,
} from './types'

// ============================================================================
// Audit view (aggregated per user/token/ip, current vs previous month)
// ============================================================================

export async function getIpAudit(
  params: IpAuditQueryParams,
  signal?: AbortSignal
): Promise<IpAuditApiResponse<IpAuditListData>> {
  const queryParams = new URLSearchParams()
  queryParams.set('month', params.month)
  queryParams.set('page', String(params.page))
  queryParams.set('page_size', String(params.pageSize))
  queryParams.set('account', params.account)
  queryParams.set('status', params.status)
  if (params.keyword) queryParams.set('keyword', params.keyword)
  const res = await api.get(`/api/ip_audit/audit?${queryParams.toString()}`, {
    signal,
  })
  return res.data
}

export async function getIpAuditMonthRows(
  month: string,
  signal?: AbortSignal
): Promise<IpAuditListData> {
  const params: IpAuditQueryParams = {
    month,
    page: 1,
    pageSize: 100,
    account: 'all',
    status: 'all',
    keyword: '',
  }
  const first = requireServerSuccess(await getIpAudit(params, signal))
  if (!first.data) throw new Error(t('Failed to load audit data'))
  const data = { ...first.data, rows: [...first.data.rows] }
  while (data.rows.length < data.total) {
    params.page++
    const next = requireServerSuccess(await getIpAudit(params, signal))
    if (!next.data || next.data.rows.length === 0) {
      throw new Error(t('Failed to load audit data'))
    }
    data.rows.push(...next.data.rows)
  }
  return data
}

// CSV export follows the current filters; the browser downloads the blob.
export async function exportIpAuditCsv(
  params: IpAuditQueryParams
): Promise<void> {
  const queryParams = new URLSearchParams()
  queryParams.set('month', params.month)
  queryParams.set('account', params.account)
  queryParams.set('status', params.status)
  if (params.keyword) queryParams.set('keyword', params.keyword)
  const res = await api.get<Blob>(
    `/api/ip_audit/audit/export?${queryParams.toString()}`,
    { responseType: 'blob' }
  )
  const url = URL.createObjectURL(res.data)
  const link = document.createElement('a')
  link.href = url
  link.download = `ip_audit_${params.month}.csv`
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

export async function getIpAuditDetail(
  userId: number,
  ip: string,
  month: string
): Promise<IpAuditApiResponse<IpAuditDetailData>> {
  const queryParams = new URLSearchParams()
  queryParams.set('user_id', String(userId))
  queryParams.set('ip', ip)
  queryParams.set('month', month)
  const res = await api.get(
    `/api/ip_audit/audit/detail?${queryParams.toString()}`
  )
  return res.data
}

// ============================================================================
// Handling status (mark handled / reopen)
// ============================================================================

export async function updateIpAuditStatus(
  data: IpAuditStatusPayload
): Promise<IpAuditApiResponse> {
  const res = await api.post('/api/ip_audit/status', data)
  return res.data
}

// ============================================================================
// Blacklist / whitelist entries
// ============================================================================

export async function getIpAuditList(
  type: IpAuditListType
): Promise<IpAuditApiResponse<IpAuditListEntry[]>> {
  const res = await api.get(`/api/ip_audit/list?type=${type}`)
  return res.data
}

export async function addIpAuditListEntry(
  data: IpAuditListPayload
): Promise<IpAuditApiResponse<IpAuditListEntry>> {
  const res = await api.post('/api/ip_audit/list', data)
  return res.data
}

export async function deleteIpAuditListEntry(
  id: number
): Promise<IpAuditApiResponse> {
  const res = await api.delete(`/api/ip_audit/list/${id}`)
  return res.data
}

// ============================================================================
// Config (monitored accounts + Feishu alert)
// ============================================================================

export async function getIpAuditConfig(): Promise<
  IpAuditApiResponse<IpAuditConfig>
> {
  const res = await api.get('/api/ip_audit/config')
  return res.data
}

export async function saveIpAuditConfig(
  data: IpAuditSaveConfigPayload
): Promise<IpAuditApiResponse<IpAuditConfig>> {
  const res = await api.put('/api/ip_audit/config', data)
  return res.data
}

// ============================================================================
// Feishu test message
// ============================================================================

// The backend falls back to the saved config for fields omitted from the
// payload, so the form's current values win without requiring a save first.
export async function sendIpAuditAlertTest(
  alert: Pick<IpAuditAlertConfig, 'webhook' | 'appkey' | 'template_id'>
): Promise<IpAuditApiResponse<string>> {
  const res = await api.post('/api/ip_audit/alert/test', {
    webhook: alert.webhook,
    appkey: alert.appkey,
    template_id: alert.template_id,
  })
  return res.data
}
