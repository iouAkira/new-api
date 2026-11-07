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
import { Send } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { PasswordInput } from '@/components/password-input'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  NativeSelect,
  NativeSelectOption,
} from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'

import { DEFAULT_ALERT_CONFIG, SILENCE_MINUTES_OPTIONS } from '../constants'
import {
  useIpAuditConfigQuery,
  useSaveIpAuditConfig,
  useSendIpAuditAlertTest,
} from '../hooks'
import type { IpAuditAlertConfig } from '../types'

/** 告警设置 tab: Feishu alert form and test button. */
export function AlertSettings() {
  const { t } = useTranslation()
  const { data: config } = useIpAuditConfigQuery()
  const saveConfig = useSaveIpAuditConfig()
  const sendTest = useSendIpAuditAlertTest()

  const [form, setForm] = useState<IpAuditAlertConfig>(DEFAULT_ALERT_CONFIG)

  useEffect(() => {
    if (config?.alert) setForm(config.alert)
  }, [config])

  const update = <K extends keyof IpAuditAlertConfig>(
    key: K,
    value: IpAuditAlertConfig[K]
  ) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const handleSave = () => {
    if (!config) return
    saveConfig.mutate({
      usernames: config.accounts.map((account) => account.username),
      alert: form,
      intent: 'alert',
    })
  }

  return (
    <div>
      <div className='bg-card mb-4 overflow-hidden rounded-xl border'>
        <div className='flex flex-wrap items-center gap-2.5 border-b px-5 py-3'>
          <span className='text-[13.5px] font-semibold'>🔔 {t('Feishu Alert')}</span>
          <span className='text-muted-foreground/70 text-xs'>
            {t('Push a card message to the Feishu group when the audit finds anomalies')}
          </span>
          <div className='flex-1' />
          <Switch
            id='ip-audit-alert-enabled'
            checked={form.enabled}
            onCheckedChange={(checked) => update('enabled', checked)}
          />
          <Label
            htmlFor='ip-audit-alert-enabled'
            className='cursor-pointer text-[13px] font-semibold'
          >
            {t('Enable Alert')}
          </Label>
        </div>

        <div className='grid grid-cols-1 gap-x-3.5 gap-y-4 px-5 py-4 text-[13px] sm:grid-cols-[130px_1fr] sm:items-center'>
          <Label className='text-muted-foreground sm:justify-end sm:text-right'>
            {t('Webhook URL')}
          </Label>
          <Input
            value={form.webhook}
            className='h-9'
            placeholder={t(
              'https://open.feishu.cn/open-apis/bot/v2/hook/xxxx — fill the iPaaS gateway URL when proxied'
            )}
            onChange={(event) => update('webhook', event.target.value)}
          />

          <Label className='text-muted-foreground sm:justify-end sm:text-right'>
            AppKey
          </Label>
          <PasswordInput
            value={form.appkey}
            className='h-9'
            placeholder={t(
              'Required when proxied through the iPaaS gateway; leave empty for direct Feishu access'
            )}
            onChange={(event) => update('appkey', event.target.value)}
          />

          <Label className='text-muted-foreground sm:justify-end sm:text-right'>
            {t('Card Template ID')}
          </Label>
          <PasswordInput
            value={form.template_id}
            className='h-9'
            placeholder={t(
              'Optional Feishu template card template_id; leave empty to use the built-in red alert card'
            )}
            onChange={(event) => update('template_id', event.target.value)}
          />

          <Label className='text-muted-foreground sm:justify-end sm:text-right'>
            {t('Alert Events')}
          </Label>
          <div className='flex flex-col gap-1.5'>
            <div className='flex items-center gap-2'>
              <Checkbox
                id='ip-audit-event-hit'
                checked={form.event_hit}
                onCheckedChange={(checked) => update('event_hit', checked)}
              />
              <Label
                htmlFor='ip-audit-event-hit'
                className='cursor-pointer font-normal'
              >
                {t(
                  'Blacklist hit — every audit discovery after a hit is included in the alert'
                )}
              </Label>
            </div>
            <div className='flex items-center gap-2'>
              <Checkbox
                id='ip-audit-event-new'
                checked={form.event_new}
                onCheckedChange={(checked) => update('event_new', checked)}
              />
              <Label
                htmlFor='ip-audit-event-new'
                className='cursor-pointer font-normal'
              >
                {t('New pending IP — appeared this month and not marked handled')}
              </Label>
            </div>
          </div>

          <Label className='text-muted-foreground sm:justify-end sm:text-right'>
            {t('Silence Period')}
          </Label>
          <div className='flex flex-col gap-1'>
            <NativeSelect
              value={form.silence_minutes}
              className='w-fit'
              onChange={(event) =>
                update('silence_minutes', Number(event.target.value))
              }
            >
              {SILENCE_MINUTES_OPTIONS.map((option) => (
                <NativeSelectOption key={option.value} value={option.value}>
                  {t(option.labelKey)}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            <span className='text-muted-foreground/70 text-xs'>
              {t(
                'The same IP will not be alerted repeatedly within the silence period to avoid spam'
              )}
            </span>
          </div>
        </div>

        <div className='flex justify-end gap-2.5 border-t px-5 py-3'>
          <Button
            variant='outline'
            disabled={sendTest.isPending}
            onClick={() =>
              sendTest.mutate({
                webhook: form.webhook,
                appkey: form.appkey,
                template_id: form.template_id,
              })
            }
          >
            <Send className='size-3.5' />
            {sendTest.isPending ? t('Testing...') : t('Send Test Message')}
          </Button>
          <Button disabled={saveConfig.isPending} onClick={handleSave}>
            {saveConfig.isPending ? t('Saving...') : t('Save Config')}
          </Button>
        </div>
      </div>

      <p className='text-muted-foreground/70 mt-2.5 px-1 text-xs leading-relaxed'>
        {t(
          'Alerts are triggered after each audit computation (same delivery as the GPU monitor service: template card first, rich-text card when no template; appkey header when going through the iPaaS gateway). New IP alerts are sent once and not repeated after being marked handled; blacklist hits are controlled by the silence period.'
        )}
      </p>
    </div>
  )
}
