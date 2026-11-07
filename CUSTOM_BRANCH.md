# 自定义分支维护说明（AI Hub SSO）

本 fork 在上游 `QuantumNous/new-api` 之上维护 AI Hub SSO / IP 审计 / 模型节点等改动。
同步方向永远只有 **上游 → 本地**，绝不把自定义改动合回上游。

## 分支约定

- `custom/aihub-sso`
  - 长期维护的规范分支，提交历史始终 **rebase 在上游最新 release tag 之上**，没有 merge commit。
  - 目前包含 6 个自定义提交（顺序固定）：
    1. `chore(deps)`：自定义分支沿用的 go.mod/go.sum 版本
    2. `chore(ci)`：`docker-build-akyakya.yml` 镜像发布 workflow
    3. `feat(aihub-sso)`：AI Hub SSO 登录与开户
    4. `feat(ip-audit)`：系统账号 IP 审计
    5. `feat(model-nodes)`：内部部署节点页
    6. `test(aihub)`：上述功能的数据库兼容测试
- `release/aihub-sso-vX.Y.Z-rc.N`
  - 发布分支 = `custom/aihub-sso` 在某个上游 tag 上的快照，只做 fast-forward，不单独提交。
  - Docker 镜像 tag 由 workflow 按**分支名**生成：`/`、`:`、`@`、空格都会被替换成 `-`。
    例如分支 `release/aihub-sso-v1.0.0-rc.48` → 镜像 tag `release-aihub-sso-v1.0.0-rc.48`。
- 历史备份：`group/aihub-sso-v1.0.0-rc.48` 是整理前的 merge 分支，仅作参考，不再使用。

## 一次性配置（建议）

```bash
# 冲突只解一次：rerere 会记录冲突解法，之后自动套用
git config --global rerere.enabled true
git config --global rerere.autoupdate true

# 本机直连 GitHub 会超时，使用本地代理（如代理端口不同请替换）
export GIT_PROXY=http://127.0.0.1:7897
```

## 上游发布新 tag 后如何升级

```bash
# 1. 拉取上游最新 tag（只拉 tag，不要 push tag）
git fetch upstream --tags

# 2. 把 custom 分支 rebase 到新 tag 上
#    --onto 新tag 旧tag(即当前 custom 的基座) 分支
#    例如从 rc.48 升到 rc.49：
git switch custom/aihub-sso
git rebase --onto v1.0.0-rc.49 v1.0.0-rc.48 custom/aihub-sso

# 3. 若出现冲突：逐个解决后
#      git add <冲突文件>
#      git rebase --continue
#    rerere 开启时，多数冲突会自动复用上次的解法。
#    高发文件：web/src/i18n/locales/*.json、go.sum、web/package.json、web/src/routeTree.gen.ts

# 4. 验证（Go 1.26.10；本机 Go 较旧时用 GOTOOLCHAIN 下载对应工具链）
go build ./...
go test ./service/aihub_sso/... ./middleware/ ./controller/ ./model/ ./router/ \
  -run 'AIHub|SSO|IPAudit|IpAudit|Deployment'
( cd web && bun install --frozen-lockfile && bun run build:check )

# 5. 推送 custom 分支（rebase 会改写历史，必须 force-with-lease）
git -c http.proxy=$GIT_PROXY push --force-with-lease origin custom/aihub-sso

# 6. 生成本版的发布分支（分支名决定镜像 tag）
git switch -c release/aihub-sso-v1.0.0-rc.49 custom/aihub-sso
git -c http.proxy=$GIT_PROXY push -u origin release/aihub-sso-v1.0.0-rc.49

# 7. 触发镜像构建
#    GitHub Actions -> "Publish Docker image to akyakya/new-api" -> Run workflow
#    branch 填 release/aihub-sso-v1.0.0-rc.49
#    或命令行（需要 gh auth login）：
HTTPS_PROXY=$GIT_PROXY gh workflow run docker-build-akyakya.yml \
  --repo iouAkira/new-api -f branch=release/aihub-sso-v1.0.0-rc.49
```

## 注意事项

- **不要 `git push origin --tags`**：fork 里的 `docker-build.yml`（push tag 触发多架构镜像）
  和 `release.yml`（push tag 创建 Release）会为每个 tag 各触发一次。
  上游 tag 只需要 `git fetch` 到本地即可。
- 不要把自定义分支合回上游，也不要向上游开 PR。
- 生成文件（`go.sum`、`routeTree.gen.ts`、locale JSON）在 rebase 冲突时优先重新生成/按 key 三方合并，
  不要整文件覆盖。
- `chore(deps)` 提交如确认不需要偏离上游版本，可在 rebase 时 drop 掉，然后 `go mod tidy` 重新生成 go.sum。
- 回滚某次发布：切到对应的 `release/aihub-sso-*` 分支重新触发构建即可，不要改 `custom/aihub-sso` 历史。
