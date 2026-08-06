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
import { api } from '@/lib/api'

import type { DeploymentNodesResponse } from './types'

// 获取内部（私有化部署）模型的部署节点列表。仅管理员可调用（后端 AdminAuth 拦截）。
export async function getDeploymentNodes(tag?: string) {
  const params = tag ? { tag } : undefined
  const res = await api.get<DeploymentNodesResponse>('/api/deployment-nodes', {
    params,
  })
  return res.data
}
