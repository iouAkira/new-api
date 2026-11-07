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
import { VChart } from '@visactor/react-vchart'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import dayjs from '@/lib/dayjs'
import { formatQuota } from '@/lib/format'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

import { useIpAuditDetailQuery } from '../hooks'
import type { IpAuditRow } from '../types'
import { StatusTags } from './status-tag'

function formatDateTime(ts: number): string {
  if (!ts) return '-'
  return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm:ss')
}

function formatDay(month: string, day: number): string {
  return dayjs(`${month}-${String(day).padStart(2, '0')}`).format('YYYY-MM-DD')
}

function Kv({
  label,
  value,
  tone,
}: {
  label: React.ReactNode
  value: React.ReactNode
  tone?: 'red' | 'blue' | 'green'
}) {
  return (
    <div className='text-xs text-muted-foreground'>
      {label}
      <span
        className={
          tone === 'red'
            ? 'mt-0.5 block text-[13px] font-semibold text-red-600 dark:text-red-400'
            : tone === 'blue'
              ? 'mt-0.5 block text-[13px] font-semibold text-blue-600 dark:text-blue-400'
              : tone === 'green'
                ? 'mt-0.5 block text-[13px] font-semibold text-green-700 dark:text-green-400'
                : 'mt-0.5 block text-[13px] font-semibold text-foreground'
        }
      >
        {value}
      </span>
    </div>
  )
}

export interface DetailSheetProps {
  row: IpAuditRow | undefined
  month: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function DetailSheet(props: DetailSheetProps) {
  const { t } = useTranslation()
  const { row: rowProp, month, open, onOpenChange } = props
  const { resolvedTheme, themeReady } = useChartTheme()

  // Keep the last non-empty row mounted so the exit animation does not
  // collapse the sheet content before it finishes sliding out.
  const [cachedRow, setCachedRow] = useState(rowProp)
  useEffect(() => {
    if (rowProp) setCachedRow(rowProp)
  }, [rowProp])
  const row = rowProp ?? cachedRow

  const { data: detail } = useIpAuditDetailQuery(
    row?.user_id,
    row?.ip,
    month
  )

  const days = detail?.days ?? []
  const chartData = useMemo(
    () => days.map((d) => ({ day: String(d.day), calls: d.calls })),
    [days]
  )
  const chartSpec = useMemo(() => {
    // 0 次的天 title/key/value 全部返回 undefined：内容为空时 VChart 判定
    // isEmpty，直接不弹提示框（x 轴仍保留整月每一天）
    const titleOf = (datum: Record<string, unknown>) =>
      Number(datum?.calls) > 0
        ? formatDay(month, Number(datum?.day) || 0)
        : undefined
    const labelOf = (datum: Record<string, unknown>) =>
      Number(datum?.calls) > 0 ? t('Requests') : undefined
    const valueOf = (datum: Record<string, unknown>) =>
      Number(datum?.calls) > 0
        ? t('{{count}} calls', {
            count: (Number(datum?.calls) || 0).toLocaleString(),
          })
        : undefined
    return {
      type: 'bar',
      data: [{ id: 'calls', values: chartData }],
      xField: 'day',
      yField: 'calls',
      bar: {
        style: {
          fill: '#dc2626',
          fillOpacity: 0.85,
          cornerRadius: [3, 3, 0, 0],
        },
      },
      tooltip: {
        mark: {
          title: { value: titleOf },
          content: [{ key: labelOf, value: valueOf }],
        },
        dimension: {
          title: { value: titleOf },
          content: [{ key: labelOf, value: valueOf }],
        },
      },
    }
  }, [chartData, month, t])

  const activeDays = useMemo(
    () => days.filter((d) => d.calls > 0).sort((a, b) => b.day - a.day),
    [days]
  )

  if (!row) return null

  const handled = row.is_new && row.handled

  const statusText = row.hit
    ? t('Blacklist hit!')
    : row.white
      ? t('Whitelist · safe')
      : handled
        ? t('New · Handled (not seen last month, verified)')
        : row.is_new
          ? t('New Pending (not seen last month)')
          : t('Existing IP')

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        className={sideDrawerContentClassName(
          '!top-[132px] !bottom-auto !h-auto !max-h-[calc(100dvh-156px)] !rounded-l-2xl border-y !shadow-2xl max-w-none sm:!max-w-[620px]'
        )}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle className='flex flex-wrap items-center gap-2'>
            <StatusTags row={row} />
            <span className='font-mono text-[15px] font-semibold'>
              {row.hit && '🚫 '}
              {row.ip}
            </span>
          </SheetTitle>
          <SheetDescription>
            {t('Account')} {row.user_name} · {t('Token')} {row.token_name}
          </SheetDescription>
        </SheetHeader>

        <div className='min-h-0 flex-1 overflow-y-auto px-5 py-4 sm:px-6'>
          <div className='mb-4 grid grid-cols-1 gap-x-4 gap-y-3 sm:grid-cols-2 lg:grid-cols-3'>
            <Kv label={t('IP / Range')} value={<span className='font-mono'>{row.ip}</span>} />
            <Kv
              label={t('Status')}
              value={statusText}
              tone={row.hit ? 'red' : row.white ? 'green' : handled ? 'blue' : undefined}
            />
            <Kv label={t('Account')} value={row.user_name} />
            <Kv label={t('Token')} value={<span className='font-mono'>{row.token_name}</span>} />
            <Kv
              label={t('Calls (This Month)')}
              value={row.cur_calls > 0 ? row.cur_calls.toLocaleString() : '-'}
            />
            <Kv
              label={t('Cost (This Month)')}
              value={row.cur_quota > 0 ? formatQuota(row.cur_quota) : '-'}
            />
            <Kv
              label={t('Calls (Last Month)')}
              value={
                row.prev_calls > 0
                  ? `${row.prev_calls.toLocaleString()}${
                      row.prev_quota > 0 ? ` · ${formatQuota(row.prev_quota)}` : ''
                    }`
                  : '-'
              }
            />
            <Kv
              label={t('First Seen')}
              value={row.first_seen > 0 ? dayjs(row.first_seen * 1000).format('YYYY-MM-DD') : '-'}
            />
            {handled && (
              <Kv
                label={t('Handled At')}
                value={formatDateTime(row.handled_at)}
                tone='blue'
              />
            )}
          </div>

          <div className='text-muted-foreground mt-5 mb-2 text-xs font-semibold tracking-wide uppercase'>
            {t('Daily Calls Trend (This Month)')}
          </div>
          <div className='h-[220px] rounded-lg border p-1.5'>
            {themeReady && chartData.length > 0 && (
              <VChart
                key={`ip-audit-detail-${month}-${row.ip}-${chartData.length}`}
                spec={{
                  ...chartSpec,
                  theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                  background: 'transparent',
                }}
                option={VCHART_OPTION}
              />
            )}
            {chartData.length === 0 && (
              <div className='text-muted-foreground/60 flex h-full items-center justify-center text-xs'>
                {t('No records under current filters')}
              </div>
            )}
          </div>

          <div className='text-muted-foreground mt-5 mb-1 text-xs font-semibold tracking-wide uppercase'>
            {t('Recent Records')}
          </div>
          <div>
            {activeDays.map((d) => (
              <div
                key={d.day}
                className='flex items-center gap-2.5 border-b py-2 text-[12.5px] last:border-b-0'
              >
                <div className='min-w-0 flex-1'>
                  <div className='font-medium'>{formatDay(month, d.day)}</div>
                  <div className='text-muted-foreground/70 text-[11.5px]'>
                    {t('{{count}} calls', { count: d.calls.toLocaleString() })}
                    {d.quota > 0 ? ` · ${formatQuota(d.quota)}` : ''}
                  </div>
                </div>
                <div className='text-muted-foreground text-[11.5px] whitespace-nowrap'>
                  {formatDay(month, d.day).slice(5)}
                </div>
              </div>
            ))}
            {activeDays.length === 0 && (
              <div className='text-muted-foreground/60 py-4 text-center text-xs'>
                {t('No records under current filters')}
              </div>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}
