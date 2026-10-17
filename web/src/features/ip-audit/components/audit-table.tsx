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
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  NativeSelect,
  NativeSelectOption,
} from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import dayjs from '@/lib/dayjs'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import { PAGE_SIZE_OPTIONS, ROW_HIT_CLASS } from '../constants'
import type { IpAuditListType, IpAuditRow } from '../types'
import { StatusTags } from './status-tag'

function formatFullDate(ts: number): string {
  if (!ts) return ''
  return dayjs(ts * 1000).format('YYYY-MM-DD')
}

function formatDateTime(ts: number): string {
  if (!ts) return ''
  return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm:ss')
}

function CellSub({
  children,
  tone,
}: {
  children: React.ReactNode
  tone?: 'red' | 'green' | 'blue'
}) {
  return (
    <span
      className={cn(
        'mt-0.5 block text-[11px] leading-snug',
        tone === 'red' && 'text-red-600 dark:text-red-400',
        tone === 'green' && 'text-green-600 dark:text-green-400',
        tone === 'blue' && 'text-blue-600 dark:text-blue-400',
        !tone && 'text-muted-foreground/70'
      )}
    >
      {children}
    </span>
  )
}

function Pager({
  total,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
}: {
  total: number
  page: number
  pageSize: number
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
}) {
  const { t } = useTranslation()
  const pages = Math.max(1, Math.ceil(total / pageSize))

  const from = Math.max(1, page - 2)
  const to = Math.min(pages, page + 2)
  const buttons: React.ReactNode[] = []
  if (from > 1) {
    buttons.push(
      <Button key={1} variant='outline' size='icon-sm' className='h-7 w-7 text-xs' onClick={() => onPageChange(1)}>
        1
      </Button>
    )
    if (from > 2) buttons.push(<span key='el-left' className='text-muted-foreground/60 px-0.5 text-xs'>…</span>)
  }
  for (let p = from; p <= to; p++) {
    buttons.push(
      <Button
        key={p}
        variant={p === page ? 'default' : 'outline'}
        size='icon-sm'
        className='h-7 w-7 text-xs'
        onClick={() => onPageChange(p)}
      >
        {p}
      </Button>
    )
  }
  if (to < pages) {
    if (to < pages - 1) buttons.push(<span key='el-right' className='text-muted-foreground/60 px-0.5 text-xs'>…</span>)
    buttons.push(
      <Button key={pages} variant='outline' size='icon-sm' className='h-7 w-7 text-xs' onClick={() => onPageChange(pages)}>
        {pages}
      </Button>
    )
  }

  return (
    <div className='flex flex-wrap items-center justify-end gap-4 border-t px-4 py-3 text-xs text-muted-foreground'>
      <span>{t('{{total}} items in total', { total })}</span>
      <div className='flex items-center gap-1.5'>
        <Button
          variant='outline'
          size='icon-sm'
          className='h-7 w-7'
          disabled={page <= 1}
          aria-label={t('Previous page')}
          onClick={() => onPageChange(page - 1)}
        >
          <ChevronLeft className='size-3.5' />
        </Button>
        {buttons}
        <Button
          variant='outline'
          size='icon-sm'
          className='h-7 w-7'
          disabled={page >= pages}
          aria-label={t('Next page')}
          onClick={() => onPageChange(page + 1)}
        >
          <ChevronRight className='size-3.5' />
        </Button>
      </div>
      <span>{t('Page {{page}} of {{pages}}', { page, pages })}</span>
      <span className='inline-flex items-center gap-1.5'>
        <NativeSelect
          size='sm'
          value={pageSize}
          onChange={(event) => onPageSizeChange(Number(event.target.value))}
        >
          {PAGE_SIZE_OPTIONS.map((size) => (
            <NativeSelectOption key={size} value={size}>
              {t('{{count}} per page', { count: size })}
            </NativeSelectOption>
          ))}
        </NativeSelect>
      </span>
    </div>
  )
}

export interface AuditTableProps {
  rows: IpAuditRow[]
  total: number
  page: number
  pageSize: number
  isLoading?: boolean
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
  onDetail: (row: IpAuditRow) => void
  onHandle: (row: IpAuditRow) => void
  onReopen: (row: IpAuditRow) => void
  onAddToList: (row: IpAuditRow, type: IpAuditListType) => void
  onRemoveFromList: (row: IpAuditRow, type: IpAuditListType) => void
}

export function AuditTable(props: AuditTableProps) {
  const { t } = useTranslation()
  const {
    rows,
    total,
    page,
    pageSize,
    isLoading,
    onPageChange,
    onPageSizeChange,
    onDetail,
    onHandle,
    onReopen,
    onAddToList,
    onRemoveFromList,
  } = props

  // 紧凑按钮保证一行放下并整体居中于列名正下方（mr-1 会打破 justify-center 的对称）
  const listBtn = 'h-7 px-2 text-xs whitespace-nowrap'
  const renderListActions = (row: IpAuditRow) => {
    if (row.hit) {
      return (
        <Button variant='outline' size='sm' className={listBtn} onClick={() => onRemoveFromList(row, 1)}>
          {t('Remove from Blacklist')}
        </Button>
      )
    }
    if (row.white) {
      return (
        <Button variant='outline' size='sm' className={listBtn} onClick={() => onRemoveFromList(row, 2)}>
          {t('Remove from Whitelist')}
        </Button>
      )
    }
    return (
      <>
        <Button
          variant='destructive'
          size='sm'
          className={listBtn}
          onClick={() => onAddToList(row, 1)}
        >
          {t('+ Black')}
        </Button>
        <Button
          variant='outline'
          size='sm'
          className={listBtn}
          onClick={() => onAddToList(row, 2)}
        >
          {t('+ White')}
        </Button>
      </>
    )
  }

  // 处理动作只看处理状态（新增待处理 → 标记已处理；已处理 → 撤销），与黑白名单互不影响
  const renderHandleAction = (row: IpAuditRow) => {
    const pending = row.is_new && !row.handled
    const handled = row.is_new && row.handled
    if (!pending && !handled) {
      return <span className='text-muted-foreground/40 text-xs'>—</span>
    }
    return pending ? (
      <Button
        variant='outline'
        size='sm'
        className='border-blue-200 text-blue-600 hover:bg-blue-50 dark:border-blue-800 dark:hover:bg-blue-950'
        onClick={() => onHandle(row)}
      >
        {t('Handle')}
      </Button>
    ) : (
      <Button
        variant='outline'
        size='sm'
        className='border-blue-200 text-blue-600 hover:bg-blue-50 dark:border-blue-800 dark:hover:bg-blue-950'
        onClick={() => onReopen(row)}
      >
        {t('Cancel Handling')}
      </Button>
    )
  }

  return (
    <>
      <div className='overflow-x-auto'>
        <Table className='table-fixed'>
          <TableHeader>
            <TableRow className='bg-muted/40 hover:bg-muted/40'>
              <TableHead className='w-[10%] px-4 text-center'>{t('Account')}</TableHead>
              <TableHead className='w-[10%] px-4 text-center'>{t('Token')}</TableHead>
              <TableHead className='w-[10%] px-4 text-center'>{t('IP')}</TableHead>
              <TableHead className='w-[10%] px-4 text-center'>{t('Status')}</TableHead>
              <TableHead className='w-[10%] px-4 text-center'>{t('Requests')}</TableHead>
              <TableHead className='w-[10%] px-4 text-center'>{t('Consumption')}</TableHead>
              <TableHead className='w-[10%] px-4 text-center'>{t('First Seen')}</TableHead>
              <TableHead className='w-[10%] px-4 text-center'>{t('Last Call')}</TableHead>
              <TableHead className='w-[10%] px-4 text-center'>{t('Blacklist / Whitelist')}</TableHead>
              <TableHead className='w-[10%] px-4 text-center'>{t('Handling')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading &&
              Array.from({ length: 6 }).map((_, index) => (
                <TableRow key={`skeleton-${index}`}>
                  {Array.from({ length: 10 }).map((__, cell) => (
                    <TableCell key={cell} className='px-4 py-3.5'>
                      <Skeleton className='h-4 w-full' />
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            {!isLoading &&
              rows.map((row) => (
                <TableRow
                  key={`${row.user_id}-${row.token_name}-${row.ip}`}
                  className={cn('cursor-pointer', row.hit && ROW_HIT_CLASS)}
                  onClick={() => onDetail(row)}
                >
                  <TableCell className='px-4 text-center'>
                    <span className='block truncate font-medium'>
                      {row.user_name}
                    </span>
                  </TableCell>
                  <TableCell className='px-4 text-center font-mono text-[12.5px]'>
                    <span className='block truncate'>{row.token_name}</span>
                  </TableCell>
                  <TableCell className='px-4 text-center'>
                    <span className='font-mono text-[12.5px] font-semibold'>
                      {row.hit && '🚫 '}
                      {row.ip}
                    </span>
                    {row.hit && <CellSub tone='red'>{t('In blacklist')}</CellSub>}
                    {!row.hit && row.white && (
                      <CellSub tone='green'>{t('Confirmed safe')}</CellSub>
                    )}
                    {!row.hit && !row.white && row.handled && (
                      <CellSub tone='blue'>
                        {t('Verified · {{date}}', {
                          date: formatDateTime(row.handled_at),
                        })}
                      </CellSub>
                    )}
                  </TableCell>
                  <TableCell className='px-4'>
                    <div className='flex justify-center'>
                      <StatusTags row={row} />
                    </div>
                  </TableCell>
                  <TableCell className='px-4 text-center'>
                    {row.cur_calls > 0 ? row.cur_calls.toLocaleString() : '-'}
                  </TableCell>
                  <TableCell className='px-4 text-center'>
                    {row.cur_quota > 0 ? formatQuota(row.cur_quota) : '-'}
                  </TableCell>
                  <TableCell className='px-4 text-center text-[12.5px]'>
                    {formatFullDate(row.first_seen)}
                  </TableCell>
                  <TableCell className='px-4 text-center text-[12.5px]'>
                    {formatDateTime(row.last_seen)}
                  </TableCell>
                  <TableCell className='px-4'>
                    <div
                      className='flex flex-wrap items-center justify-center gap-1'
                      onClick={(event) => event.stopPropagation()}
                    >
                      {renderListActions(row)}
                    </div>
                  </TableCell>
                  <TableCell className='px-4'>
                    <div
                      className='flex items-center justify-center gap-1'
                      onClick={(event) => event.stopPropagation()}
                    >
                      {renderHandleAction(row)}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </div>
      {!isLoading && rows.length === 0 && (
        <div className='text-muted-foreground/70 py-10 text-center text-[13px]'>
          {t('No records under current filters')}
        </div>
      )}
      {total > 0 && (
        <Pager
          total={total}
          page={page}
          pageSize={pageSize}
          onPageChange={onPageChange}
          onPageSizeChange={onPageSizeChange}
        />
      )}
    </>
  )
}
