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

import { cn } from '@/lib/utils'

import { TAG_STYLES } from '../constants'
import type { IpAuditRow } from '../types'

function Tag({
  className,
  children,
}: {
  className?: string
  children: React.ReactNode
}) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium whitespace-nowrap',
        className
      )}
    >
      {children}
    </span>
  )
}

/**
 * Handling-status tags for one audit row. Purely about the handling state
 * (new pending / new handled / existing); blacklist and whitelist membership
 * is orthogonal and is shown next to the IP and in the list action column.
 */
export function StatusTags({ row }: { row: IpAuditRow }) {
  const { t } = useTranslation()

  const pending = row.is_new && !row.handled
  const handled = row.is_new && row.handled
  const existing = !row.is_new

  return (
    <span className='flex flex-wrap items-center gap-1'>
      {handled && <Tag className={TAG_STYLES.handled}>{t('New · Handled')}</Tag>}
      {pending && <Tag className={TAG_STYLES.new}>★ {t('New Pending')}</Tag>}
      {existing && <Tag className={TAG_STYLES.exist}>{t('Existing')}</Tag>}
    </span>
  )
}
