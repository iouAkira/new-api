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
import { Download, Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  NativeSelect,
  NativeSelectOption,
} from '@/components/ui/native-select'

import type { IpAuditAccount, IpAuditStatusFilter } from '../types'

const CAPSULE_INPUT_CLASS =
  'h-8 w-[220px] rounded-full border px-3.5 text-[13px] focus-visible:ring-1 focus-visible:ring-inset'

/**
 * Capsule filter bar (frozen UX): keyword search, account select, status
 * select and a month picker, all as pill controls without external labels.
 * The keyword is debounced before it reaches the query.
 */
export function FilterBar({
  accounts,
  month,
  account,
  status,
  keyword,
  onMonthChange,
  onAccountChange,
  onStatusChange,
  onKeywordChange,
  onManageAccounts,
  onExport,
  exporting,
}: {
  accounts?: IpAuditAccount[]
  month: string
  account: string
  status: IpAuditStatusFilter
  keyword: string
  onMonthChange: (month: string) => void
  onAccountChange: (account: string) => void
  onStatusChange: (status: IpAuditStatusFilter) => void
  onKeywordChange: (keyword: string) => void
  onManageAccounts: () => void
  onExport: () => void
  exporting?: boolean
}) {
  const { t } = useTranslation()
  const [keywordInput, setKeywordInput] = useState(keyword)

  useEffect(() => {
    setKeywordInput(keyword)
  }, [keyword])

  useEffect(() => {
    const timer = setTimeout(() => {
      if (keywordInput !== keyword) onKeywordChange(keywordInput.trim())
    }, 400)
    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keywordInput])

  return (
    <div className='flex flex-wrap items-center gap-2.5 px-5 py-3'>
      <Input
        value={keywordInput}
        placeholder={t('Filter by IP / token...')}
        className={CAPSULE_INPUT_CLASS}
        onChange={(event) => setKeywordInput(event.target.value)}
      />
      <NativeSelect
        value={account}
        className='max-w-[180px]'
        onChange={(event) => onAccountChange(event.target.value)}
      >
        <NativeSelectOption value='all'>
          {t('All Accounts')}
        </NativeSelectOption>
        {(accounts ?? []).map((item) => (
          <NativeSelectOption key={item.user_id} value={String(item.user_id)}>
            {item.username || item.name || String(item.user_id)}
          </NativeSelectOption>
        ))}
      </NativeSelect>
      <NativeSelect
        value={status}
        className='max-w-[180px]'
        onChange={(event) =>
          onStatusChange(event.target.value as IpAuditStatusFilter)
        }
      >
        <NativeSelectOption value='all'>{t('All Statuses')}</NativeSelectOption>
        <NativeSelectOption value='hit'>{t('Blacklist Hit')}</NativeSelectOption>
        <NativeSelectOption value='new'>{t('New Pending')}</NativeSelectOption>
        <NativeSelectOption value='handled'>
          {t('New Handled')}
        </NativeSelectOption>
        <NativeSelectOption value='white'>{t('Whitelist')}</NativeSelectOption>
        <NativeSelectOption value='exist'>{t('Existing')}</NativeSelectOption>
      </NativeSelect>
      <Input
        type='month'
        value={month}
        className={CAPSULE_INPUT_CLASS + ' w-[160px]'}
        onChange={(event) => onMonthChange(event.target.value || month)}
      />
      <Button
        size='sm'
        className='ml-auto h-8 rounded-full px-3.5 text-[13px]'
        onClick={onManageAccounts}
      >
        <Plus className='size-3.5' />
        {t('Add Account')}
      </Button>
      <Button
        variant='outline'
        size='sm'
        className='h-8 rounded-full px-3.5 text-[13px]'
        disabled={exporting}
        onClick={onExport}
      >
        <Download className='size-3.5' />
        {t('Export CSV')}
      </Button>
    </div>
  )
}
