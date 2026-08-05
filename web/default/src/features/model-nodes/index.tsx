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
import { useQuery } from '@tanstack/react-query'
import {
  Activity,
  ChevronLeft,
  ChevronRight,
  Loader2,
  MapPin,
  RefreshCw,
  Search,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { MaskedValueDisplay } from '@/components/masked-value-display'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'

import { getDeploymentNodes } from './api'
import type { DeploymentNode } from './types'

const CHANNEL_STATUS_ENABLED = 1

// 脱敏：所有 key 都只露首尾，中间打码（短的也打码）
function maskKey(k: string): string {
  if (!k) return '—'
  const n = k.length
  if (n > 10) return `${k.slice(0, 6)}••••${k.slice(-4)}`
  if (n > 6) return `${k.slice(0, 3)}••••${k.slice(-2)}`
  return `${k.slice(0, 2)}••••`
}

function StatusBadge({ status }: { status: number }) {
  const { t } = useTranslation()
  if (status === CHANNEL_STATUS_ENABLED) {
    return (
      <Badge className='bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'>
        {t('Enabled')}
      </Badge>
    )
  }
  return (
    <Badge variant='outline' className='text-amber-700 dark:text-amber-300'>
      {t('Disabled')}
    </Badge>
  )
}

function KpiCard({
  label,
  value,
  tone,
}: {
  label: string
  value: number | string
  tone?: string
}) {
  return (
    <Card>
      <CardContent className='flex flex-col items-center justify-center gap-1 py-4 text-center'>
        <span className='text-muted-foreground text-base font-medium'>
          {label}
        </span>
        <span
          className={cn('text-3xl font-semibold tabular-nums leading-none', tone)}
        >
          {value}
        </span>
      </CardContent>
    </Card>
  )
}

function LoadingSkeleton() {
  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 md:grid-cols-4'>
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className='h-20 w-full' />
        ))}
      </div>
      <Skeleton className='h-64 w-full' />
    </div>
  )
}

export function ModelNodes() {
  const { t } = useTranslation()
  const [keyword, setKeyword] = useState('')
  const [modelPage, setModelPage] = useState(1)
  const MODEL_PAGE_SIZE = 10

  const nodesQuery = useQuery({
    queryKey: ['model-nodes'],
    queryFn: async () => {
      const res = await getDeploymentNodes()
      if (!res.success || !Array.isArray(res.data)) {
        throw new Error(res.message || t('No deployment nodes found.'))
      }
      return res.data
    },
    staleTime: 30 * 1000,
    retry: false,
  })

  const loading = nodesQuery.isLoading
  const refreshing = nodesQuery.isFetching && !loading
  const allNodes = nodesQuery.data ?? []

  const nodes = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return allNodes
    return allNodes.filter(
      (n) =>
        n.model.toLowerCase().includes(kw) || n.ip.toLowerCase().includes(kw)
    )
  }, [allNodes, keyword])

  const stats = useMemo(() => {
    const ips = new Set<string>()
    const models = new Set<string>()
    let enabled = 0
    for (const n of nodes) {
      ips.add(n.ip)
      models.add(n.model)
      if (n.channel_status === CHANNEL_STATUS_ENABLED) enabled++
    }
    return {
      instances: nodes.length,
      distinctNodes: ips.size,
      distinctModels: models.size,
      enabled,
      disabled: nodes.length - enabled,
    }
  }, [nodes])

  // 按 IP 聚合
  const byServer = useMemo(() => {
    const map = new Map<string, DeploymentNode[]>()
    for (const n of nodes) {
      const arr = map.get(n.ip) ?? []
      arr.push(n)
      map.set(n.ip, arr)
    }
    return [...map.entries()].sort((a, b) => a[0].localeCompare(b[0]))
  }, [nodes])

  // 按模型聚合（表格按模型、IP 排序）
  const byModelRows = useMemo(
    () =>
      [...nodes].sort(
        (a, b) =>
          a.model.localeCompare(b.model) || a.ip.localeCompare(b.ip)
      ),
    [nodes]
  )

  // 按模型表格分页
  const modelTotalPages = Math.max(
    1,
    Math.ceil(byModelRows.length / MODEL_PAGE_SIZE)
  )
  const curModelPage = Math.min(Math.max(1, modelPage), modelTotalPages)
  const pagedModelRows = byModelRows.slice(
    (curModelPage - 1) * MODEL_PAGE_SIZE,
    curModelPage * MODEL_PAGE_SIZE
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        <span className='inline-flex min-w-0 items-center gap-2'>
          <MapPin className='size-5 shrink-0 text-primary' />
          <span className='truncate'>{t('Model Nodes')}</span>
        </span>
      </SectionPageLayout.Title>

      <SectionPageLayout.Content>
        <div className='space-y-5'>
          {/* 工具条 */}
          <div className='flex flex-wrap items-center gap-3'>
            <div className='relative max-w-xs flex-1'>
              <Search className='pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground' />
              <Input
                value={keyword}
                onChange={(e) => {
                  setKeyword(e.target.value)
                  setModelPage(1)
                }}
                placeholder={t('Search model / IP')}
                className='pl-8'
              />
            </div>
            <div className='text-muted-foreground ml-auto text-sm'>
              {t('Instances')}: {stats.instances}
            </div>
            <Button
              variant='outline'
              size='sm'
              onClick={() => nodesQuery.refetch()}
              disabled={refreshing}
            >
              {refreshing ? (
                <Loader2 className='size-4 animate-spin' />
              ) : (
                <RefreshCw className='size-4' />
              )}
              {t('Refresh')}
            </Button>
          </div>

          {loading ? (
            <LoadingSkeleton />
          ) : nodesQuery.isError ? (
            <ErrorState
              icon={Activity}
              title={t('No deployment nodes found.')}
              description={
                nodesQuery.error instanceof Error
                  ? nodesQuery.error.message
                  : undefined
              }
              onRetry={() => nodesQuery.refetch()}
            />
          ) : nodes.length === 0 ? (
            <EmptyState
              icon={MapPin}
              title={t('No deployment nodes found.')}
              description={t(
                'No internal model deployment nodes matched the current filter.'
              )}
            />
          ) : (
            <>
              {/* 概览 */}
              <section className='space-y-2'>
                <h2 className='text-base font-semibold'>{t('Overview')}</h2>
                <div className='grid grid-cols-2 gap-3 md:grid-cols-4'>
                <KpiCard label={t('Instances')} value={stats.instances} />
                <KpiCard
                  label={t('Distinct Nodes')}
                  value={stats.distinctNodes}
                  tone='text-primary'
                />
                <KpiCard
                  label={t('Distinct Models')}
                  value={stats.distinctModels}
                />
                <KpiCard
                  label={t('Enabled') + ' / ' + t('Disabled')}
                  value={`${stats.enabled} / ${stats.disabled}`}
                />
                </div>
              </section>

              {/* 按服务器 */}
              <section className='space-y-2'>
                <h2 className='text-base font-semibold'>{t('By Server')}</h2>
                <div className='grid grid-cols-1 items-start gap-3 lg:grid-cols-2 xl:grid-cols-3'>
                  {byServer.map(([ip, group]) => {
                    const anyEnabled = group.some(
                      (n) => n.channel_status === CHANNEL_STATUS_ENABLED
                    )
                    return (
                      <Card key={ip} className='flex h-36 flex-col'>
                        <CardHeader className='shrink-0 pb-3'>
                          <CardTitle className='flex items-center justify-between gap-2'>
                            <span className='font-mono text-base break-all'>
                              {ip}
                            </span>
                            <StatusBadge
                              status={
                                anyEnabled
                                  ? CHANNEL_STATUS_ENABLED
                                  : group[0]?.channel_status ?? 0
                              }
                            />
                          </CardTitle>
                        </CardHeader>
                        <CardContent className='flex-1 overflow-hidden pt-0'>
                          <div className='h-full space-y-1 overflow-y-auto pr-1 [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-foreground/40 [&::-webkit-scrollbar-track]:bg-transparent'>
                            {group.map((n, idx) => (
                              <div
                                key={`${n.model}-${n.ip}-${n.port}-${idx}`}
                                className='flex items-center gap-2 text-sm'
                              >
                                <span className='size-1.5 shrink-0 rounded-full bg-primary/50' />
                                <span className='truncate font-medium'>
                                  {n.model}
                                </span>
                              </div>
                            ))}
                          </div>
                        </CardContent>
                      </Card>
                    )
                  })}
                </div>
              </section>

              {/* 按模型 */}
              <section className='space-y-2'>
                <h2 className='text-base font-semibold'>{t('By Model')}</h2>
                <Card>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Model')}</TableHead>
                        <TableHead>IP</TableHead>
                        <TableHead>Port</TableHead>
                        <TableHead>Key</TableHead>
                        <TableHead>{t('Status')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {pagedModelRows.map((n, idx) => (
                        <TableRow
                          key={`${n.model}-${n.ip}-${n.port}-${idx}`}
                        >
                          <TableCell className='font-medium'>
                            {n.model}
                          </TableCell>
                          <TableCell className='whitespace-nowrap font-mono text-xs'>
                            {n.ip}
                          </TableCell>
                          <TableCell className='whitespace-nowrap font-mono text-xs tabular-nums'>
                            {n.port || '—'}
                          </TableCell>
                          <TableCell>
                            {n.key ? (
                              <MaskedValueDisplay
                                label='API Key'
                                fullValue={n.key}
                                maskedValue={maskKey(n.key)}
                                copyTooltip={t('Copy')}
                                copyAriaLabel='Copy API key'
                              />
                            ) : (
                              <span className='text-muted-foreground'>—</span>
                            )}
                          </TableCell>
                          <TableCell>
                            <StatusBadge status={n.channel_status} />
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                  {byModelRows.length > MODEL_PAGE_SIZE && (
                    <div className='flex items-center justify-end gap-3 border-t px-3 py-2'>
                      <span className='text-muted-foreground text-xs tabular-nums'>
                        {curModelPage} / {modelTotalPages}
                      </span>
                      <Button
                        variant='outline'
                        size='icon'
                        className='size-7'
                        disabled={curModelPage <= 1}
                        onClick={() => setModelPage((p) => Math.max(1, p - 1))}
                      >
                        <ChevronLeft className='size-4' />
                      </Button>
                      <Button
                        variant='outline'
                        size='icon'
                        className='size-7'
                        disabled={curModelPage >= modelTotalPages}
                        onClick={() =>
                          setModelPage((p) => Math.min(modelTotalPages, p + 1))
                        }
                      >
                        <ChevronRight className='size-4' />
                      </Button>
                    </div>
                  )}
                </Card>
              </section>
            </>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
