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
import { useState } from 'react'

import {
  useAddIpAuditListEntry,
  useDeleteIpAuditListEntry,
  useExportIpAuditCsv,
  useIpAuditConfigQuery,
  useIpAuditListQuery,
  useIpAuditQuery,
  useUpdateIpAuditStatus,
} from '../hooks'
import { matchedEntryIds } from '../lib/ip-utils'
import type {
  IpAuditListType,
  IpAuditRow,
  IpAuditStatusFilter,
} from '../types'
import { AccountsSheet } from './accounts-sheet'
import { AuditTable } from './audit-table'
import { DetailSheet } from './detail-sheet'
import { FilterBar } from './filter-bar'
import { StatCards } from './stat-cards'

function currentMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

/** 审计视图 tab: stat cards + capsule filter bar + audit table + sheets. */
export function AuditView() {
  const [month, setMonth] = useState(currentMonth)
  const [account, setAccount] = useState('all')
  const [status, setStatus] = useState<IpAuditStatusFilter>('all')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const [detailRow, setDetailRow] = useState<IpAuditRow | undefined>()
  const [accountsOpen, setAccountsOpen] = useState(false)

  const { data, isLoading } = useIpAuditQuery({
    month,
    page,
    pageSize,
    account,
    status,
    keyword,
  })
  const { data: config } = useIpAuditConfigQuery()
  const { data: blacklist } = useIpAuditListQuery(1)
  const { data: whitelist } = useIpAuditListQuery(2)

  const updateStatus = useUpdateIpAuditStatus()
  const addEntry = useAddIpAuditListEntry()
  const deleteEntry = useDeleteIpAuditListEntry()
  const exportCsv = useExportIpAuditCsv()

  const resetPage = () => setPage(1)

  const handleMonthChange = (value: string) => {
    setMonth(value)
    resetPage()
  }
  const handleAccountChange = (value: string) => {
    setAccount(value)
    resetPage()
  }
  const handleStatusChange = (value: IpAuditStatusFilter) => {
    setStatus(value)
    resetPage()
  }
  const handleKeywordChange = (value: string) => {
    setKeyword(value)
    resetPage()
  }
  const handlePageSizeChange = (value: number) => {
    setPageSize(value)
    resetPage()
  }

  const handleHandle = (row: IpAuditRow) => {
    updateStatus.mutate({
      user_id: row.user_id,
      ip: row.ip,
      action: 'handle',
      note: '',
    })
  }

  const handleReopen = (row: IpAuditRow) => {
    updateStatus.mutate({
      user_id: row.user_id,
      ip: row.ip,
      action: 'reopen',
      note: '',
    })
  }

  const handleAddToList = (row: IpAuditRow, type: IpAuditListType) => {
    addEntry.mutate({
      type,
      ip: row.ip,
      remark: `Audit · ${row.is_new ? 'new this month' : 'existing'} · user ${row.user_id}`,
    })
  }

  const handleRemoveFromList = (row: IpAuditRow, type: IpAuditListType) => {
    const entries = type === 1 ? (blacklist ?? []) : (whitelist ?? [])
    // Remove every rule that matches this IP (a single IP may be covered by
    // both a single-IP rule and a range rule at the same time).
    const ids = matchedEntryIds(entries, row.ip)
    ids.forEach((id) => deleteEntry.mutate(id))
  }

  return (
    <div>
      <StatCards stats={data?.stats} isLoading={isLoading} />

      <div className='bg-card overflow-hidden rounded-xl border'>
        <div className='border-b'>
          <FilterBar
            accounts={config?.accounts}
            month={month}
            account={account}
            status={status}
            keyword={keyword}
            onMonthChange={handleMonthChange}
            onAccountChange={handleAccountChange}
            onStatusChange={handleStatusChange}
            onKeywordChange={handleKeywordChange}
            onManageAccounts={() => setAccountsOpen(true)}
            onExport={() =>
              exportCsv.mutate({ month, page, pageSize, account, status, keyword })
            }
            exporting={exportCsv.isPending}
          />
        </div>
        <AuditTable
          rows={data?.rows ?? []}
          total={data?.total ?? 0}
          page={page}
          pageSize={pageSize}
          isLoading={isLoading}
          onPageChange={setPage}
          onPageSizeChange={handlePageSizeChange}
          onDetail={setDetailRow}
          onHandle={handleHandle}
          onReopen={handleReopen}
          onAddToList={handleAddToList}
          onRemoveFromList={handleRemoveFromList}
        />
      </div>

      <DetailSheet
        row={detailRow}
        month={month}
        open={detailRow != null}
        onOpenChange={(open) => !open && setDetailRow(undefined)}
      />

      <AccountsSheet
        open={accountsOpen}
        onOpenChange={setAccountsOpen}
      />
    </div>
  )
}
