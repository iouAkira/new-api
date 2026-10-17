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
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import { STAT_ALARM_CLASS } from '../constants'
import type { IpAuditStats } from '../types'

function StatCard({
  label,
  value,
  danger,
  alarm,
}: {
  label: React.ReactNode
  value: React.ReactNode
  danger?: boolean
  alarm?: boolean
}) {
  return (
    <div
      className={cn(
        'bg-card flex flex-col items-center rounded-xl border px-3.5 py-4 text-center',
        alarm && STAT_ALARM_CLASS
      )}
    >
      <div className='text-muted-foreground flex items-center justify-center gap-1.5 text-xs'>
        {label}
      </div>
      <div
        className={cn(
          'mt-1.5 text-2xl font-bold tabular-nums',
          danger && 'text-red-600 dark:text-red-400'
        )}
      >
        {value}
      </div>
    </div>
  )
}

/**
 * Five centered stat cards (no bottom sub-text, per the frozen prototype).
 * Pure display — account management lives in the filter bar's add button.
 */
export function StatCards({
  stats,
  isLoading,
}: {
  stats?: IpAuditStats
  isLoading?: boolean
}) {
  const { t } = useTranslation()

  if (isLoading && !stats) {
    return (
      <div className='mb-4 grid grid-cols-2 gap-3.5 md:grid-cols-3 xl:grid-cols-5'>
        {Array.from({ length: 5 }).map((_, index) => (
          <Skeleton key={index} className='h-[86px] rounded-xl' />
        ))}
      </div>
    )
  }

  const value = (n: number | undefined) =>
    (n ?? 0).toLocaleString()

  return (
    <div className='mb-4 grid grid-cols-2 gap-3.5 md:grid-cols-3 xl:grid-cols-5'>
      <StatCard
        label={t('Monitored Accounts')}
        value={value(stats?.accounts)}
      />
      <StatCard
        label={
          <>
            🚫 {t('Blacklist Hits')}
          </>
        }
        value={value(stats?.hits)}
        danger
        alarm={(stats?.hits ?? 0) > 0}
      />
      <StatCard
        label={<>★ {t('New Pending (This Month)')}</>}
        value={value(stats?.new_pending)}
        danger
      />
      <StatCard
        label={t('Abnormal Requests')}
        value={value(stats?.abnormal_req)}
        danger
      />
      <StatCard
        label={t('Total IPs (This Month)')}
        value={value(stats?.total_ip)}
      />
    </div>
  )
}
