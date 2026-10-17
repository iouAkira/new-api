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
import type { IpAuditAlertConfig, IpAuditStatusFilter } from './types'

/** List entry type constants (mirrors the backend model). */
export const IP_AUDIT_LIST_TYPE = {
  BLACKLIST: 1,
  WHITELIST: 2,
} as const

export const IP_AUDIT_STATUS_FILTERS: IpAuditStatusFilter[] = [
  'all',
  'hit',
  'new',
  'handled',
  'white',
  'exist',
]

export const DEFAULT_ALERT_CONFIG: IpAuditAlertConfig = {
  enabled: false,
  webhook: '',
  appkey: '',
  template_id: '',
  event_hit: true,
  event_new: true,
  silence_minutes: 1440,
}

export const SILENCE_MINUTES_OPTIONS = [
  { value: 30, labelKey: '30 minutes' },
  { value: 60, labelKey: '1 hour' },
  { value: 360, labelKey: '6 hours' },
  { value: 1440, labelKey: '24 hours' },
  { value: 10080, labelKey: '7 days' },
]

export const PAGE_SIZE_OPTIONS = [10, 20, 50]

/**
 * Semantic tag palette (frozen by the UX prototype):
 * red = blacklist hit, amber = new pending, blue = handled,
 * green = whitelist, gray = existing / neutral network tag.
 */
export const TAG_STYLES = {
  hit: 'border-red-600 bg-red-600 text-white dark:border-red-500 dark:bg-red-500',
  new: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-400',
  handled:
    'border-blue-200 bg-blue-50 text-blue-600 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-400',
  white: 'border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400',
  exist: 'border-zinc-200 bg-zinc-100 text-zinc-500 dark:border-zinc-700/60 dark:bg-zinc-800/60 dark:text-zinc-400',
} as const

/** Card style for the "blacklist hits" stat card when hits > 0. */
export const STAT_ALARM_CLASS =
  'border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950/40'

/** Row background for blacklist-hit rows. */
export const ROW_HIT_CLASS =
  'bg-red-50/80 hover:bg-red-100/80 dark:bg-red-950/20 dark:hover:bg-red-950/30'
