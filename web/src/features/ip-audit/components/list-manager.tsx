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
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import { IP_AUDIT_LIST_TYPE, TAG_STYLES } from '../constants'
import {
  useDeleteIpAuditListEntry,
  useIpAuditListQuery,
  useIpAuditMonthRowsQuery,
} from '../hooks'
import { entryMatchesIp } from '../lib/ip-utils'
import type { IpAuditListEntry, IpAuditListType } from '../types'
import { AddEntrySheet } from './add-entry-sheet'

function currentMonth(): string {
  return dayjs().format('YYYY-MM')
}

function formatDateTime(ts: number): string {
  if (!ts) return '-'
  return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm:ss')
}

function ListPanel({
  type,
  onAdd,
}: {
  type: IpAuditListType
  onAdd: (type: IpAuditListType) => void
}) {
  const { t } = useTranslation()
  const isBlacklist = type === IP_AUDIT_LIST_TYPE.BLACKLIST
  const { data: entries, isLoading } = useIpAuditListQuery(type)
  const { data: monthData } = useIpAuditMonthRowsQuery(currentMonth())
  const deleteEntry = useDeleteIpAuditListEntry()
  const [pendingDelete, setPendingDelete] = useState<IpAuditListEntry | null>(
    null
  )

  const rows = entries ?? []
  const monthRows = monthData?.rows ?? []

  return (
    <div className='bg-card mb-4 overflow-hidden rounded-xl border'>
      <div className='flex flex-wrap items-center gap-2.5 border-b px-5 py-3'>
        <span className='text-[13.5px] font-semibold'>
          {isBlacklist ? '🚫 ' : '✅ '}
          {isBlacklist ? t('Blacklist') : t('Whitelist')}
        </span>
        <span className='text-muted-foreground/70 text-xs'>
          {isBlacklist
            ? t('Suspicious, red alert on any usage')
            : t('Confirmed safe, excluded from audit alerts')}{' '}
          · {t('{{count}} entries', { count: rows.length })}
        </span>
        <div className='flex-1' />
        <Button
          variant={isBlacklist ? 'destructive' : 'default'}
          onClick={() => onAdd(type)}
        >
          <Plus className='size-3.5' />
          {t('Add')}
        </Button>
      </div>

      <Table className='min-w-[760px]'>
        <TableHeader>
          <TableRow className='bg-muted/40 hover:bg-muted/40'>
            <TableHead className='w-[220px] px-5'>{t('IP / Range')}</TableHead>
            <TableHead className='px-5'>{t('Remark')}</TableHead>
            <TableHead className='w-[120px] px-5'>{t('Added By')}</TableHead>
            <TableHead className='w-[170px] px-5'>{t('Added At')}</TableHead>
            <TableHead className='w-[260px] px-5'>{t('Calls (This Month)')}</TableHead>
            <TableHead className='w-[100px] px-5'>{t('Actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((entry) => {
            const matched = monthRows.filter((row) =>
              entryMatchesIp(entry.ip, row.ip)
            )
            return (
              <TableRow key={entry.id}>
                <TableCell className='px-5 font-mono text-[12.5px] font-semibold'>
                  {entry.ip}
                </TableCell>
                <TableCell className='max-w-[240px] truncate px-5'>
                  {entry.remark}
                </TableCell>
                <TableCell className='px-5'>{entry.operate_by}</TableCell>
                <TableCell className='px-5'>
                  {formatDateTime(entry.created_at)}
                </TableCell>
                <TableCell className='px-5'>
                  {matched.length > 0 ? (
                    <span className='flex flex-wrap gap-1'>
                      {matched.map((row) => (
                        <span
                          key={`${row.user_id}-${row.ip}`}
                          className={cn(
                            'inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium',
                            isBlacklist ? TAG_STYLES.hit : TAG_STYLES.white
                          )}
                        >
                          {(isBlacklist ? '🚫 ' : '✅ ') + row.ip} ·{' '}
                          {row.cur_calls.toLocaleString()}
                        </span>
                      ))}
                    </span>
                  ) : (
                    <span
                      className={cn(
                        'inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium',
                        TAG_STYLES.exist
                      )}
                    >
                      {t('Not seen this month')}
                    </span>
                  )}
                </TableCell>
                <TableCell className='px-5'>
                  <Button
                    variant='outline'
                    size='sm'
                    disabled={deleteEntry.isPending}
                    onClick={() => setPendingDelete(entry)}
                  >
                    {t('Remove')}
                  </Button>
                </TableCell>
              </TableRow>
            )
          })}
          {rows.length === 0 && !isLoading && (
            <TableRow>
              <TableCell colSpan={6} className='px-5 py-3'>
                <div className='text-muted-foreground/70 py-6 text-center text-[13px]'>
                  {isBlacklist
                    ? t(
                        'Blacklist is empty — once a suspicious IP is added, any call to monitored accounts triggers a red alert in the audit view'
                      )
                    : t(
                        'Whitelist is empty — IPs added here are not flagged as abnormal even if new this month'
                      )}
                </div>
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>

      <ConfirmDialog
        open={pendingDelete != null}
        onOpenChange={(open) => !open && setPendingDelete(null)}
        title={t('Delete this entry?')}
        desc={
          <>
            {t('This will remove the rule; matched IPs will no longer match it.')}{' '}
            <span className='font-mono font-semibold'>
              {pendingDelete?.ip}
            </span>
          </>
        }
        confirmText={
          deleteEntry.isPending ? t('Deleting...') : t('Delete')
        }
        destructive
        isLoading={deleteEntry.isPending}
        handleConfirm={() => {
          if (!pendingDelete) return
          deleteEntry.mutate(pendingDelete.id, {
            onSettled: () => setPendingDelete(null),
          })
        }}
      />
    </div>
  )
}

/** 名单管理 tab: whitelist + blacklist panels with counts and add sheets. */
export function ListManager() {
  const { t } = useTranslation()
  const [addType, setAddType] = useState<IpAuditListType | null>(null)

  return (
    <div>
      <ListPanel type={IP_AUDIT_LIST_TYPE.WHITELIST} onAdd={setAddType} />
      <ListPanel type={IP_AUDIT_LIST_TYPE.BLACKLIST} onAdd={setAddType} />

      <p className='text-muted-foreground/70 mt-2.5 px-1 text-xs leading-relaxed'>
        {t(
          "Lists only detect and alert; they do not block calls. When an IP hits both lists, the blacklist wins. To reject calls directly, use the token's allowed-IP whitelist."
        )}
      </p>

      <AddEntrySheet
        type={IP_AUDIT_LIST_TYPE.BLACKLIST}
        open={addType === IP_AUDIT_LIST_TYPE.BLACKLIST}
        onOpenChange={(open) => !open && setAddType(null)}
      />
      <AddEntrySheet
        type={IP_AUDIT_LIST_TYPE.WHITELIST}
        open={addType === IP_AUDIT_LIST_TYPE.WHITELIST}
        onOpenChange={(open) => !open && setAddType(null)}
      />
    </div>
  )
}
