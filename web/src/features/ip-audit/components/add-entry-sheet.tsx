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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

import { useAddIpAuditListEntry, useIpAuditListQuery } from '../hooks'
import { validateEntryFormat } from '../lib/ip-utils'
import type { IpAuditListType } from '../types'

/**
 * Right-side sheet (620px) for adding one blacklist/whitelist rule.
 * The three accepted formats are validated client-side with inline error
 * text under the input; duplicate detection is also local (unique (type,ip)
 * is enforced by the backend too).
 */
export function AddEntrySheet({
  type,
  open,
  onOpenChange,
}: {
  type: IpAuditListType
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const isBlacklist = type === 1
  const { data: entries } = useIpAuditListQuery(type)
  const addEntry = useAddIpAuditListEntry()

  const [ip, setIp] = useState('')
  const [remark, setRemark] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setIp('')
      setRemark('')
      setError(null)
    }
  }, [open])

  const validate = (): string | null => {
    const formatError = validateEntryFormat(ip)
    if (formatError) {
      return t(formatError.key, formatError.params)
    }
    const trimmed = ip.trim()
    if ((entries ?? []).some((entry) => entry.ip === trimmed)) {
      return t('{{ip}} already exists in this list, no need to add again', {
        ip: trimmed,
      })
    }
    return null
  }

  const handleConfirm = () => {
    const validationError = validate()
    if (validationError) {
      setError(validationError)
      return
    }
    setError(null)
    addEntry.mutate(
      { type, ip: ip.trim(), remark: remark.trim() },
      {
        onSuccess: (res) => {
          if (res.success) onOpenChange(false)
        },
      }
    )
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        className={sideDrawerContentClassName(
          '!top-[132px] !bottom-auto !h-auto !max-h-[calc(100dvh-156px)] !rounded-l-2xl border-y !shadow-2xl max-w-none sm:!max-w-[480px]'
        )}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {isBlacklist ? '🚫 ' : '✅ '}
            {isBlacklist
              ? t('Add Blacklist Rule')
              : t('Add Whitelist Rule')}
          </SheetTitle>
          <SheetDescription>
            {t(
              'Supports three formats: single IP · CIDR · closed range, e.g. 172.26.1.100 / 10.0.0.0/24 / 172.26.64.0-172.26.147.0'
            )}
          </SheetDescription>
        </SheetHeader>

        <div className='min-h-0 flex-1 overflow-y-auto px-5 py-5 sm:px-6'>
          <div className='mb-5'>
            <Label className='mb-2 text-[13px] font-medium'>
              {t('IP / Range')}
            </Label>
            <Input
              value={ip}
              autoFocus
              aria-invalid={Boolean(error)}
              className='h-10 rounded-[10px] px-3.5 text-sm'
              placeholder='172.26.1.100 / 10.0.0.0/24 / 172.26.64.0-172.26.147.0'
              onChange={(event) => setIp(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') handleConfirm()
              }}
            />
            <p className='text-muted-foreground/70 mt-1.5 text-xs leading-relaxed'>
              {t(
                'Supports three formats: single IP · CIDR · closed range, e.g. 172.26.1.100 / 10.0.0.0/24 / 172.26.64.0-172.26.147.0'
              )}
            </p>
            {error && (
              <p className='mt-1.5 text-[12.5px] text-red-600 dark:text-red-400'>
                {error}
              </p>
            )}
          </div>

          <div className='mb-5'>
            <Label className='mb-2 text-[13px] font-medium'>
              {t('Remark')}
            </Label>
            <Input
              value={remark}
              className='h-10 rounded-[10px] px-3.5 text-sm'
              placeholder={
                isBlacklist
                  ? t(
                      'Why blacklist (e.g. departed staff, external partner, past anomaly)'
                    )
                  : t(
                      'Why whitelist (e.g. DC segment, confirmed new server)'
                    )
              }
              onChange={(event) => setRemark(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') handleConfirm()
              }}
            />
            <p className='text-muted-foreground/70 mt-1.5 text-xs leading-relaxed'>
              {t(
                'Shown in the list and hit details so others can understand this rule'
              )}
            </p>
          </div>

          <div className='bg-muted/50 text-muted-foreground rounded-[10px] border p-3.5 text-[12.5px] leading-relaxed'>
            {isBlacklist
              ? t(
                  'Once blacklisted, any call from the IP / range to a monitored account triggers an immediate red alert in the audit view (alert only, calls are not blocked).'
                )
              : t(
                  'Once whitelisted, the IP / range is not flagged or alerted even if it is new this month.'
                )}
          </div>
        </div>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' className='w-full sm:w-auto' />}>
            {t('Cancel')}
          </SheetClose>
          <Button
            className={
              isBlacklist
                ? 'w-full bg-red-600 text-white hover:bg-red-700 sm:w-auto'
                : 'w-full sm:w-auto'
            }
            disabled={addEntry.isPending}
            onClick={handleConfirm}
          >
            {addEntry.isPending ? t('Saving...') : t('Confirm Add')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
