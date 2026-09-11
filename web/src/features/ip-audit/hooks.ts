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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import i18next from 'i18next'
import { toast } from 'sonner'

import {
  addIpAuditListEntry,
  deleteIpAuditListEntry,
  exportIpAuditCsv,
  getIpAudit,
  getIpAuditConfig,
  getIpAuditDetail,
  getIpAuditList,
  saveIpAuditConfig,
  sendIpAuditAlertTest,
  updateIpAuditStatus,
} from './api'
import { DEFAULT_ALERT_CONFIG } from './constants'
import type {
  IpAuditAlertConfig,
  IpAuditConfig,
  IpAuditListData,
  IpAuditListPayload,
  IpAuditListType,
  IpAuditQueryParams,
  IpAuditStatusPayload,
} from './types'

const EMPTY_AUDIT_DATA: IpAuditListData = {
  rows: [],
  stats: { accounts: 0, hits: 0, new_pending: 0, abnormal_req: 0, total_ip: 0 },
  total: 0,
}

// ============================================================================
// Queries
// ============================================================================

export function useIpAuditQuery(params: IpAuditQueryParams) {
  return useQuery({
    queryKey: [
      'ip-audit',
      params.month,
      params.account,
      params.status,
      params.keyword,
      params.page,
      params.pageSize,
    ],
    queryFn: async () => {
      const res = await getIpAudit(params)
      if (!res.success) {
        toast.error(res.message || i18next.t('Failed to load audit data'))
        return EMPTY_AUDIT_DATA
      }
      return res.data ?? EMPTY_AUDIT_DATA
    },
    placeholderData: (previous) => previous,
  })
}

/**
 * Full-month rows (all accounts, all statuses) used by the list manager to
 * compute which monitored IPs currently match each blacklist/whitelist rule.
 */
export function useIpAuditMonthRowsQuery(month: string) {
  return useQuery({
    queryKey: ['ip-audit-month-rows', month],
    queryFn: async () => {
      const res = await getIpAudit({
        month,
        page: 1,
        pageSize: 500,
        account: 'all',
        status: 'all',
        keyword: '',
      })
      if (!res.success) return EMPTY_AUDIT_DATA
      return res.data ?? EMPTY_AUDIT_DATA
    },
  })
}

export function useIpAuditDetailQuery(
  userId: number | undefined,
  ip: string | undefined,
  month: string
) {
  return useQuery({
    queryKey: ['ip-audit-detail', userId, ip, month],
    queryFn: async () => {
      const res = await getIpAuditDetail(userId!, ip!, month)
      if (!res.success) {
        return { days: [] }
      }
      return res.data ?? { days: [] }
    },
    enabled: userId != null && ip != null,
  })
}

export function useIpAuditListQuery(type: IpAuditListType) {
  return useQuery({
    queryKey: ['ip-audit-list', type],
    queryFn: async () => {
      const res = await getIpAuditList(type)
      if (!res.success) {
        toast.error(res.message || i18next.t('Failed to load audit data'))
        return []
      }
      return res.data ?? []
    },
  })
}

export function useIpAuditConfigQuery() {
  return useQuery({
    queryKey: ['ip-audit-config'],
    queryFn: async () => {
      const res = await getIpAuditConfig()
      if (!res.success) {
        toast.error(res.message || i18next.t('Failed to load audit data'))
        return {
          accounts: [],
          alert: { ...DEFAULT_ALERT_CONFIG },
        } satisfies IpAuditConfig
      }
      return (
        res.data ?? {
          accounts: [],
          alert: { ...DEFAULT_ALERT_CONFIG },
        }
      )
    },
  })
}

// ============================================================================
// Mutations
// ============================================================================

function useInvalidateIpAudit() {
  const queryClient = useQueryClient()
  return {
    onSuccess: () => {
      // Query keys are element-matched, not string-prefixed. ['ip-audit']
      // covers the main table/stats key ['ip-audit', month, ...] but none of
      // the dashed keys, so every distinct first element must be listed here.
      queryClient.invalidateQueries({ queryKey: ['ip-audit'] })
      queryClient.invalidateQueries({ queryKey: ['ip-audit-config'] })
      queryClient.invalidateQueries({ queryKey: ['ip-audit-list'] })
      queryClient.invalidateQueries({ queryKey: ['ip-audit-month-rows'] })
      queryClient.invalidateQueries({ queryKey: ['ip-audit-detail'] })
    },
  }
}

export function useUpdateIpAuditStatus() {
  const invalidate = useInvalidateIpAudit()

  return useMutation({
    mutationFn: (data: IpAuditStatusPayload) => updateIpAuditStatus(data),
    onSuccess: (res, variables) => {
      if (res.success) {
        toast.success(
          variables.action === 'handle'
            ? i18next.t('Marked {{ip}} as handled', { ip: variables.ip })
            : i18next.t('Reopened {{ip}} as pending', { ip: variables.ip })
        )
        invalidate.onSuccess()
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || i18next.t('Operation failed'))
    },
  })
}

export function useAddIpAuditListEntry() {
  const invalidate = useInvalidateIpAudit()

  return useMutation({
    mutationFn: (data: IpAuditListPayload) => addIpAuditListEntry(data),
    onSuccess: (res, variables) => {
      if (res.success) {
        toast.success(
          variables.type === 1
            ? i18next.t('Added {{ip}} to the blacklist', { ip: variables.ip })
            : i18next.t('Added {{ip}} to the whitelist', { ip: variables.ip })
        )
        invalidate.onSuccess()
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || i18next.t('Operation failed'))
    },
  })
}

export function useDeleteIpAuditListEntry() {
  const invalidate = useInvalidateIpAudit()

  return useMutation({
    mutationFn: (id: number) => deleteIpAuditListEntry(id),
    onSuccess: (res) => {
      if (res.success) {
        toast.success(i18next.t('Entry deleted'))
        invalidate.onSuccess()
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || i18next.t('Operation failed'))
    },
  })
}

export function useSaveIpAuditConfig() {
  const invalidate = useInvalidateIpAudit()

  return useMutation({
    mutationFn: ({
      usernames,
      alert,
    }: {
      usernames?: string[]
      alert?: IpAuditAlertConfig
      intent: 'accounts' | 'alert'
    }) => saveIpAuditConfig({ usernames, alert }),
    onSuccess: (res, variables) => {
      if (res.success) {
        // Both surfaces share PUT /config; the intent only picks the toast.
        toast.success(
          variables.intent === 'accounts'
            ? i18next.t('Monitored accounts updated')
            : i18next.t('Alert config saved')
        )
        invalidate.onSuccess()
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || i18next.t('Operation failed'))
    },
  })
}

export function useExportIpAuditCsv() {
  return useMutation({
    mutationFn: exportIpAuditCsv,
    onError: (error: Error) => {
      toast.error(error.message || i18next.t('Operation failed'))
    },
  })
}

export function useSendIpAuditAlertTest() {
  return useMutation({
    mutationFn: sendIpAuditAlertTest,
    onSuccess: (res) => {
      if (res.success) {
        toast.success(i18next.t('Test message sent to the Feishu group'))
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || i18next.t('Operation failed'))
    },
  })
}
