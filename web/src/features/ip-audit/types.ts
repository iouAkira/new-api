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

// ============================================================================
// Shared response envelope
// ============================================================================

export interface IpAuditApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

// ============================================================================
// Audit rows (GET /api/ip_audit/audit)
// ============================================================================

export interface IpAuditRow {
  user_id: number
  user_name: string
  token_name: string
  ip: string
  cur_calls: number
  cur_quota: number
  prev_calls: number
  prev_quota: number
  is_new: boolean
  white: boolean
  hit: boolean
  handled: boolean
  handled_by: string
  handled_at: number
  first_seen: number
  last_seen: number
}

export interface IpAuditStats {
  accounts: number
  hits: number
  new_pending: number
  abnormal_req: number
  total_ip: number
}

export interface IpAuditListData {
  rows: IpAuditRow[]
  stats: IpAuditStats
  total: number
}

export type IpAuditStatusFilter =
  | 'all'
  | 'hit'
  | 'new'
  | 'handled'
  | 'white'
  | 'exist'

export interface IpAuditQueryParams {
  month: string
  page: number
  pageSize: number
  account: string
  status: IpAuditStatusFilter
  keyword: string
}

// ============================================================================
// Detail (GET /api/ip_audit/audit/detail)
// ============================================================================

export interface IpAuditDetailDay {
  day: number
  calls: number
  quota: number
}

export interface IpAuditDetailData {
  days: IpAuditDetailDay[]
}

// ============================================================================
// Handling status (POST /api/ip_audit/status)
// ============================================================================

export interface IpAuditStatusPayload {
  user_id: number
  ip: string
  action: 'handle' | 'reopen'
  note: string
}

// ============================================================================
// Blacklist / whitelist (GET/POST /api/ip_audit/list, DELETE /api/ip_audit/list/:id)
// ============================================================================

/** 1 = blacklist, 2 = whitelist */
export type IpAuditListType = 1 | 2

export interface IpAuditListEntry {
  id: number
  type: IpAuditListType
  ip: string
  remark: string
  operate_by: string
  created_at: number
}

export interface IpAuditListPayload {
  type: IpAuditListType
  ip: string
  remark: string
}

// ============================================================================
// Config (GET/PUT /api/ip_audit/config)
// ============================================================================

export interface IpAuditAccount {
  user_id: number
  username: string
  name: string
}

export interface IpAuditAlertConfig {
  enabled: boolean
  webhook: string
  appkey: string
  template_id: string
  event_hit: boolean
  event_new: boolean
  silence_minutes: number
}

export interface IpAuditConfig {
  accounts: IpAuditAccount[]
  alert: IpAuditAlertConfig
}

/** PUT /config payload — usernames omitted when only saving the alert config. */
export interface IpAuditSaveConfigPayload {
  usernames?: string[]
  alert?: IpAuditAlertConfig
}
