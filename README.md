# Dujiao-Next

Dujiao-Next is a digital goods e-commerce platform. This repository contains the complete
application: the Go backend, the customer storefront, and the admin panel.

## 网页安装向导（V免签改造版）

完整 Linux 安装包包含商城前台、管理后台、V免签 Fox 驱动和首次安装页面。服务器无需安装 Go、Node、PHP，也不需要手动创建 MariaDB 数据库。自动发货需要可连接的 Redis。

1. 上传完整安装包并解压，在解压目录用 root 执行一次 `bash start.sh`。脚本安装到 `/opt/dujiao-next-vmq` 并配置开机启动；已有商城数据不会被覆盖。
2. 打开脚本显示的专属安装链接，将 `SERVER_IP` 替换为公网 IP。默认端口 `8081`，与 V免签 `8080` 分开；也可在 1Panel 创建反向代理，目标填写 `http://127.0.0.1:8081`，通过自己的域名打开同一个安装链接。
3. 在网页填写 Redis 的宿主机地址、映射端口及密码，设置管理员和网站地址。密码至少 8 个字符，包含大写、小写和数字。专属链接会自动填写安装授权码，无需手工复制；直接访问 `/install` 时，授权码可从 `/opt/dujiao-next-vmq/.install-token` 读取。
4. 程序验证 Redis 数据库 `0` 和 `1` 的连接，在临时目录完成建表、管理员哈希和站点配置初始化，成功后一次性提交。连接失败或初始化失败可以修改信息重试。页面等到商城健康检查通过才跳转 `/admin/`。

网页安装的配置位于 `/opt/dujiao-next-vmq/db/config.yml`，数据库位于同目录 `dujiao.db`；管理员明文密码不写入配置。备份整个 `db/`（包含配置和密钥）及 `uploads/`。旧版根目录 `config.yml` 仍按原方式读取。已有配置或数据库会锁定安装，配置丢失时应恢复备份，不要删除数据库重新安装。

安装向导支持简体中文、繁体中文和英文。启动脚本创建 `dujiao-next-vmq.service`，查看日志用 `journalctl -u dujiao-next-vmq -n 60 --no-pager`。正式收款前请为商城配置 HTTPS，并做一笔小额实付确认手机监控、回调和自动发货。改造版保持 `BuildType=source`，避免官方在线更新覆盖 V免签驱动；本次没有增加新的在线更新功能。

## V免签 Fox 对接（本地改造版）

此源码增加 `V免签 Fox` 支付驱动，适用于 Fox V3 的 HMAC-SHA256 接口，支持人民币支付宝和微信收款。请使用本改造源码编译的完整程序；直接运行官方在线安装脚本下载的程序不包含此驱动。自编译时保留 `BuildType=source`，防止后台升级用官方程序覆盖改造。

安装后在后台「支付渠道」新增渠道：

| 项目 | 填写内容 |
| --- | --- |
| 支付服务商 | V免签 Fox |
| 支付方式 | 支付宝或微信；同时收款时分别创建两个渠道 |
| 交互方式 | 跳转支付 |
| 网关地址 | Fox 服务根地址，例如 `http://服务器IP:8080`；不加 `/api/order/create` |
| 通讯密钥 | Fox 后台的通讯密钥；不是商城 JWT、管理员密码或易支付商户密钥 |
| 异步通知地址 | `https://商城域名/api/v1/payments/callback` |
| 同步返回地址 | `https://商城域名/pay` |

通知和返回地址会按当前后台访问域名预填，保存前确认它们是商城正式域名。商城货币设置为 `CNY`，Fox 上传对应收款码，配套监控 APK 保持在线。Fox 默认要求回调使用公网 HTTPS，因此正式收款前须完成商城域名和证书配置。Fox 的全局默认通知/返回地址可留空，本驱动每次下单都会传入。

顾客付款后由 Fox 验签回调确认订单，商城沿用现有的支付成功和发货流程。金额区分产生的实际付款金额由 Fox 收银台显示，回调按签名中的原始订单金额核对；重复通知不会重复入账。浏览器返回页本身不会确认付款。部署后仍需做一笔小额实付，确认手机监控、回调和卡密发货整条链路。

## ❤️ Brand Partners (Sponsors)

<table>

<tr>
<td width="180"><a href="https://www.vmrack.net/?ref_code=5iXmGUMf5f5"><img src="assets/partners/vmrack.jpeg" alt="CCTK.AI" width="150"></a></td>
<td><a href="https://www.vmrack.net/?ref_code=5iXmGUMf5f5">Vmrack.com</a> 全球自动化云基础设施服务商 提供先进的云服务器、裸金属、CDN、媒体处理、对象存储和网络解决方案，助力企业轻松上云。
⚡️官方合作闪购款，仅需76刀/年，三网优化线路，助力您业务起飞，<a href="https://www.vmrack.net/vps/flash-deals/2082383856451452928?ref_code=5iXmGUMf5f5">👉点我直达</a>
</td>
</tr>

<tr>
<td width="180"><a href="https://www.99cdn.com/"><img src="assets/partners/99cdn.jpg" alt="openmodel" width="150"></a></td>
<td>99CDN 自建 CDN 平台，自主管理节点 · 智能流量调度 · 多级缓存加速。 <a href="https://www.99cdn.com/">99CDN</a> 是 EasyLink 旗下的商业化自建 CDN 与 DNS 智能调度平台，支持边缘缓存、分片缓存、多级回源、GTM 调度与边缘计算能力。</td>
</tr>

<tr>
<td width="180"><a href="https://niub.me"><img src="assets/partners/niub.png" alt="openmodel" width="150"></a></td>
<td> <a href="https://niub.me">NIUB — 数字服务，一站直达(DujiaoNext自营旗舰店)</a> 正在寻找更便捷的 AI 服务、社交账号或数字礼品卡？NIUB（niub.me）专注提供多种虚拟商品与数字服务，让不同类型的数字需求都能在一个站点完成选购。
我们重视清晰的商品信息、明确的交付方式和负责任的售后支持。每件商品的账号类型、适用地区、有效期限、使用条件与售后范围，均以对应商品页面说明为准。
访问 NIUB，探索更多数字服务与虚拟商品。</td>
</tr>



</table>

## Tech Stack

| Layer | Stack |
| --- | --- |
| Backend | Go 1.26 · Gin · GORM · SQLite / PostgreSQL |
| Auth | JWT (separate admin / user realms) · Casbin RBAC · TOTP 2FA |
| Async | asynq on Redis (optional — the server runs without it) |
| Config | Viper (`config.yml`) |
| Frontend | Vue 3 · Vite · TypeScript · Tailwind CSS v4 · pnpm 10 |
| Admin UI | shadcn-vue / reka-ui |

## Repository Layout

```
.
├── cmd/server/               # entry point; also hosts the `admin` operator subcommands
├── internal/
│   ├── app/                  # composition root
│   │   ├── container/        # dependency-injection container
│   │   ├── httpserver/       # Gin router, route groups, middleware
│   │   └── jobs/             # asynq worker service and consumers
│   ├── bootstrap/            # per-module wiring (adapters.go + wiring.go)
│   ├── modules/              # 35 business modules — one vertical slice per domain
│   ├── workflows/            # use cases that span several modules
│   ├── platform/             # framework-facing infrastructure
│   │   ├── database/gormdb/  # connection, auto-migration
│   │   └── http/             # response envelope, Gin helpers
│   ├── shared/               # dependency-free primitives (money, jsonmap, serial …)
│   ├── authz/                # Casbin RBAC: policy model, built-in role seeds
│   ├── web/                  # SPA embedding and mounting (build-tag gated)
│   ├── architecture/         # architecture guard tests — no production code
│   ├── cache/ config/ constants/ crypto/ i18n/ logger/ queue/ version/
│   └── admincmd/ htmltext/ persistence/ telegramidentity/ testkit/ upstream/
├── frontend/
│   ├── admin/                # admin panel SPA        (dev :5174)
│   └── user/                 # customer storefront SPA (dev :5173)
├── config.yml.example
├── Dockerfile                # single full-stack image
└── .goreleaser.yaml
```

Runtime directories created on first start: `db/` (SQLite), `uploads/`, `logs/`.

## Architecture

A modular monolith. Each domain under `internal/modules/<name>/` is a vertical slice with its
own layers:

| Layer | Holds | May import |
| --- | --- | --- |
| `domain/` | entities, value objects, business invariants | nothing from the other layers |
| `application/` | use cases, port interfaces | `domain`, `contract` |
| `infrastructure/` | GORM stores, gateways, queue adapters | `domain`, `application` ports |
| `transport/` | HTTP handlers, presenters | `application` contracts |
| `contract/` | port interfaces the application layer depends on, and the module's public surface for other modules | — |

**These rules are enforced by tests, not convention.** `internal/architecture/` parses every
import in the tree and fails the build on violations. The main ones:

- `domain` must not reach into `application`, `infrastructure`, or `transport`
- `application` must not import Gin or asynq — no transport libraries in use cases
- only a module's `infrastructure/gormstore` adapter may import GORM
- `transport` depends on application contracts, never on concrete stores
- `internal/shared` stays free of modules, GORM, Gin, and asynq
- `internal/platform` must not depend on business modules

Run them with the rest of the suite: `go test ./internal/architecture/...`

Modules never import each other's internals — they talk through `contract/`, and the wiring
lives in `internal/bootstrap/<module>/`.

### RBAC

Every `/api/v1/admin/...` route passes through Casbin. The permission catalog is generated
from the live route table, but the **built-in roles are hand-maintained** in
`internal/authz/bootstrap.go`. Adding an admin route without adding it to a role seed leaves
that route reachable only by the super admin. `internal/app/httpserver/rbac_coverage_test.go`
checks that every registered route is covered.

## Build Tags

| Tag | Effect |
| --- | --- |
| *(none)* | API only. No SPAs mounted — the default for local development. |
| `fullstack` | Embeds `internal/web/dist/{admin,user}` into the binary via `go:embed`. |
| `release` | Production behavior for outbound URL building. |

`go:embed all:dist/admin all:dist/user` requires **both** directories to exist, so a
`fullstack` build fails outright if the frontends were not built first. A plain `go build`
does not compile `embed_fullstack.go` — after touching `internal/web/`, verify with
`go build -tags release,fullstack ./cmd/server`.

## Run Modes

```bash
./dujiao-next                 # all    — HTTP server + background worker (default)
./dujiao-next -mode api       # HTTP server only
./dujiao-next -mode worker    # background worker only
```

Operator subcommands ship in the same binary, so a container needs no extra tooling:

```bash
./dujiao-next admin list-admins
./dujiao-next admin reset-password
./dujiao-next admin reset-2fa
```

## Frontend Notes

Two independent SPAs, both built with Vite and embedded at release time.

**Mount points.** The storefront is served at `/`; the admin panel at `web.admin_path`
(default `/admin`). `/api`, `/uploads`, and `/health` are reserved prefixes — an unmatched
path under them returns 404 instead of falling through to the SPA shell. Adding a new
top-level backend prefix means updating `reservedPaths` in `internal/web/handler.go`.

**The admin base path is resolved at runtime, not at build time.** Since `web.admin_path` is
configurable, `pnpm run build:fullstack` only injects a `<base href="__DJ_ADMIN_BASE__/">`
placeholder, which the server rewrites on startup. Consequences for admin code:

- native `<a href>` and `window.location` navigation must go through `adminUrl()` in
  `src/utils/adminBase.ts`
- `<router-link :to>` and `router.push()` must **not** — vue-router already carries the base,
  and prefixing again yields `/admin/admin/...`

**Storefront templates.** The customer frontend ships more than one look, selected by the
`storefront_template` site setting (`classic`, `vault`). Template pages live in
`src/templates/<name>/` and fall back to `src/views/` when a page has no template-specific
version; see `src/templates/registry.ts`. Append `?template=vault` to preview one locally.

**i18n.** Both frontends and all API responses are localized — Simplified Chinese, Traditional
Chinese, and English. Do not hard-code user-facing strings on either side.

## Quick Start (Deploy)

### Official one-click installer (Ubuntu / Debian)

On a fresh Ubuntu 22.04+ or Debian 12+ server, download and run the official
interactive installer:

```bash
curl -fsSL https://raw.githubusercontent.com/dujiao-next/dujiao-next/main/scripts/dujiao-next-manager.sh \
  -o /tmp/dujiao-next-manager.sh
sudo bash /tmp/dujiao-next-manager.sh install
```

The installer deploys the release binary with systemd, an isolated local Redis,
Nginx, SQLite, and a Let's Encrypt certificate. After installation, reopen the
management menu with:

```bash
sudo dujiao-next-manager
```

Common automation-friendly commands are also available:

```bash
sudo dujiao-next-manager status
sudo dujiao-next-manager logs app
sudo dujiao-next-manager restart
sudo dujiao-next-manager configure-domain
sudo dujiao-next-manager configure-admin-path
sudo dujiao-next-manager renew-cert
sudo dujiao-next-manager admin-reset-password
sudo dujiao-next-manager admin-reset-2fa
sudo dujiao-next-manager uninstall
```

The first release supports a single non-wildcard domain on Ubuntu/Debian only.
It does not adopt an existing manual installation. If SMTP is skipped, configure
it in the admin panel before enabling email-verification registration. Application
data lives in `/opt/dujiao-next`; installer state is stored in
`/etc/dujiao-next/install-state.json`. TLS failures leave only the ACME challenge
endpoint enabled, and `install` can be rerun after DNS or firewall repair. Safe
uninstall creates and verifies a `0600` recovery archive under
`/var/backups/dujiao-next` before deleting managed data.

### Manual binary installation

Download the latest `dujiao-next_*.tar.gz` from [Releases](https://github.com/dujiao-next/dujiao-next/releases):

```bash
tar -xzf dujiao-next_*.tar.gz
cp config.yml.example config.yml
# edit config.yml: set jwt.secret, user_jwt.secret, and web.admin_path
./dujiao-next
```

Full instructions: https://dujiao-next.com/deploy/

Or with Docker:

```bash
docker run -d -p 8080:8080 -v $PWD/config.yml:/app/config.yml:ro dujiaonext/dujiao-next:latest
```

## Quick Start (Develop)

Run the backend and the two frontends separately for hot reload:

```bash
go mod tidy && go run ./cmd/server   # :8080 — API only, no SPAs mounted

cd frontend/user  && pnpm install && pnpm run dev   # :5173
cd frontend/admin && pnpm install && pnpm run dev   # :5174
```

Both dev servers proxy `/api`, `/uploads`, `/sitemap.xml`, and `/robots.txt` to
`localhost:8080`. In production everything is same-origin, so these proxies are a
development-only concern.

> Use `pnpm` via corepack. `pnpm --dir X` does not read the `packageManager` field of the
> target directory and will pick the wrong version — `cd` into the package first.

## Building the Full-Stack Binary

```bash
goreleaser build --snapshot --single-target --clean
```

This builds both frontends, embeds them, and compiles with `-tags fullstack` — the same path
CI uses for releases. The manual equivalent:

```bash
(cd frontend/admin && pnpm run build:fullstack)   # injects the <base> placeholder
(cd frontend/user  && pnpm run build)
rm -rf internal/web/dist && mkdir -p internal/web/dist
cp -r frontend/admin/dist internal/web/dist/admin
cp -r frontend/user/dist  internal/web/dist/user
go build -tags release,fullstack -o dujiao-next ./cmd/server
```

Note that admin uses `build:fullstack`, not `build`. Plain `build` produces a bundle pinned to
`/`, which silently breaks a custom `web.admin_path`.

## Testing

```bash
go test ./...                              # full suite
go test ./internal/architecture/...        # dependency and layering guards
go test ./internal/modules/order/...       # one module

cd frontend/user  && pnpm run build        # includes vue-tsc type checking
cd frontend/admin && pnpm run build
```

Health check endpoint: `GET /health`

## Notes on Data Access

SQLite runs with `MaxOpenConns=1`. A store opens a transaction through
`WithinTransaction(func(tx contract.Transaction) error)`, and every query inside the closure
must go through that `tx` handle or a store bound to it via `WithTx(tx)`. Reaching for the
global DB handle instead asks for a second connection that will never be granted, deadlocking
the process — including indirectly, by calling a service that queries on its own. Read any
settings you need *before* opening the transaction, and keep outbound HTTP calls (payment
gateways and the like) outside it.

## Online Documentation

- https://dujiao-next.com
