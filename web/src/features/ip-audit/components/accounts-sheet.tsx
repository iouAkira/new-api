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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  sideDrawerContentClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { useIpAuditConfigQuery, useSaveIpAuditConfig } from '../hooks'
import type { IpAuditAccount } from '../types'

/**
 * Right-side sheet for managing the monitored project accounts
 * (persisted as the accounts part of the ip.audit config).
 */
export function AccountsSheet({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const { data: config } = useIpAuditConfigQuery()
  const saveConfig = useSaveIpAuditConfig()

  const [username, setUsername] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [pendingRemove, setPendingRemove] = useState<IpAuditAccount | null>(
    null
  )

  useEffect(() => {
    if (open) {
      setUsername('')
      setError(null)
    }
  }, [open])

  const accounts = config?.accounts ?? []

  // Accounts are added by username (login name); the backend resolves it to
  // a user_id and reads display_name from the users table for display.
  const persist = (nextUsernames: string[]) => {
    saveConfig.mutate({ usernames: nextUsernames, intent: 'accounts' })
  }

  const handleAdd = () => {
    const trimmed = username.trim()
    if (trimmed === '' || /\s/.test(trimmed)) {
      setError(t('Enter a valid username'))
      return
    }
    if (accounts.some((account) => account.username === trimmed)) {
      setError(t('{{id}} is already monitored', { id: trimmed }))
      return
    }
    setError(null)
    persist([...accounts.map((account) => account.username), trimmed])
    setUsername('')
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        className={sideDrawerContentClassName(
          '!top-[132px] !bottom-auto !h-auto !max-h-[calc(100dvh-156px)] !rounded-l-2xl border-y !shadow-2xl max-w-none sm:!max-w-[480px]'
        )}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Manage Accounts')}</SheetTitle>
          <SheetDescription>
            {t(
              'Manage the project accounts whose API keys are distributed; their calling IPs are audited month over month'
            )}
          </SheetDescription>
        </SheetHeader>

        <div className='min-h-0 flex-1 overflow-y-auto px-5 py-5 sm:px-6'>
          <div className='mb-5 flex flex-col gap-3.5 sm:flex-row sm:items-end'>
            <div className='w-full sm:w-[220px]'>
              <Label className='mb-2 text-[13px] font-medium'>
                {t('User Account')}
              </Label>
              <Input
                value={username}
                aria-invalid={Boolean(error)}
                className='h-10 rounded-[10px] px-3.5'
                placeholder={t('Username (login name)')}
                onChange={(event) => setUsername(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') handleAdd()
                }}
              />
            </div>
            <Button
              className='w-full sm:w-auto'
              disabled={saveConfig.isPending}
              onClick={handleAdd}
            >
              <Plus className='size-3.5' />
              {t('Add')}
            </Button>
          </div>
          {error && (
            <p className='mb-4 text-[12.5px] text-red-600 dark:text-red-400'>
              {error}
            </p>
          )}

          <Table>
            <TableHeader>
              <TableRow className='bg-muted/40 hover:bg-muted/40'>
                <TableHead className='w-[130px] px-3'>
                  {t('User Account')}
                </TableHead>
                <TableHead className='px-3'>{t('Account Name')}</TableHead>
                <TableHead className='w-[90px] px-3 text-right'>
                  {t('Actions')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {accounts.map((account) => (
                <TableRow key={account.user_id}>
                  <TableCell className='px-3 font-mono font-semibold'>
                    {account.username || account.user_id}
                  </TableCell>
                  <TableCell className='max-w-[240px] truncate px-3'>
                    {account.name || account.username || account.user_id}
                  </TableCell>
                  <TableCell className='px-3 text-right'>
                    <Button
                      variant='outline'
                      size='sm'
                      disabled={saveConfig.isPending}
                      onClick={() => setPendingRemove(account)}
                    >
                      {t('Remove')}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {accounts.length === 0 && (
            <div className='text-muted-foreground/70 py-6 text-center text-[13px]'>
              {t(
                'No accounts are monitored yet — add one to audit its calling IPs month over month'
              )}
            </div>
          )}

          <div className='bg-muted/50 text-muted-foreground mt-4 rounded-[10px] border p-3.5 text-[12.5px] leading-relaxed'>
            {t(
              'After adding, this account\'s monthly calling IPs are compared with last month; new IPs not whitelisted are flagged as abnormal, and blacklist hits alert in red.'
            )}
          </div>
        </div>

        <ConfirmDialog
          open={pendingRemove != null}
          onOpenChange={(open) => !open && setPendingRemove(null)}
          title={t('Remove this account?')}
          desc={
            <>
              {t(
                'This account will no longer be audited; its rows disappear from the audit view.'
              )}{' '}
              <span className='font-mono font-semibold'>
                {pendingRemove?.username}
              </span>
            </>
          }
          confirmText={
            saveConfig.isPending ? t('Removing...') : t('Remove')
          }
          destructive
          isLoading={saveConfig.isPending}
          handleConfirm={() => {
            if (!pendingRemove) return
            persist(
              accounts
                .filter((item) => item.user_id !== pendingRemove.user_id)
                .map((item) => item.username)
            )
            setPendingRemove(null)
          }}
        />
      </SheetContent>
    </Sheet>
  )
}
