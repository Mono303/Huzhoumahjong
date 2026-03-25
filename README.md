# Huzhoumahjong

## 现有项目技术结构分析

原仓库几乎是一个单文件网页游戏：

- 前端与“联机”逻辑都堆在 [public/index.html](/d:/huzhoumajiang/Huzhoumahjong/public/index.html)
- `package.json` 仍保留了早期 Node/Express/Socket.IO 痕迹，但当前仓库里没有对应的 `server.js`
- 原始“在线房间”实际上使用 `BroadcastChannel` 做同浏览器/同设备广播，不是真实服务端联机
- 旧代码可复用的核心资产主要是：
  - 牌堆生成、排序、胡牌判定、吃碰杠逻辑
  - 基础番型与积分计算
  - 绿色台面 + 金色描边的视觉语言

## 重构方案总览

本次重构将项目拆分成现代 Web 游戏系统：

- 前端：`frontend/`，Vue 3 + Vite + TypeScript + Pinia + Vue Router
- 后端：`backend/`，Go + Gin + 标准 WebSocket
- 数据库：PostgreSQL，初始化脚本在 [001_init.sql](/d:/huzhoumajiang/Huzhoumahjong/migrations/001_init.sql)
- 缓存：Redis，保存会话、在线状态、心跳、房间快照、房间玩家缓存
- 反向代理：Nginx，配置在 [nginx.conf](/d:/huzhoumajiang/Huzhoumahjong/deployments/nginx/nginx.conf)
- 部署：Docker Compose，入口在 [docker-compose.yml](/d:/huzhoumajiang/Huzhoumahjong/docker-compose.yml)

## 分阶段实施结果

### 第一阶段：旧仓库分析

- 已确认旧仓库是“单 HTML + 原地状态机 + BroadcastChannel 假联机”结构
- 已把旧版页面归档到 [legacy-index.html](/d:/huzhoumajiang/Huzhoumahjong/legacy/legacy-index.html) 作为迁移参考

### 第二阶段：基础骨架

- 新建 `frontend/`、`backend/`、`migrations/`、`deployments/`、`legacy/`
- 前端路由与页面已具备：
  - 登录页
  - 大厅页
  - 房间页
  - 游戏页
- 后端已具备：
  - 游客登录
  - 获取当前用户
  - 创建房间 / 加入房间 / 离开房间 / 获取房间
  - 获取历史战绩
  - 健康检查
  - WebSocket 实时通信

### 第三阶段：联机基础链路

- 前端通过 HTTP 创建/加入房间
- 进入房间后通过 WebSocket 建立实时连接
- 支持：
  - 房间成员状态同步
  - 玩家准备
  - 房主开始
  - 在线/断线状态展示
  - 心跳与基础重连

### 第四阶段：服务端权威牌局

- 将旧版核心牌局逻辑迁移到 Go 侧，服务端持有权威状态
- 已实现 MVP 级联机牌局流程：
  - 开局
  - 发牌
  - 出牌
  - 吃 / 碰 / 杠 / 胡
  - AI 补位到 4 人
  - 基础积分结算
  - 牌局日志写入 `match_events`

### 第五阶段：部署与文档

- Docker Compose、Nginx、环境变量示例、数据库迁移脚本已补齐

## 实际代码改动

主要新增目录：

- [frontend](/d:/huzhoumajiang/Huzhoumahjong/frontend)
- [backend](/d:/huzhoumajiang/Huzhoumahjong/backend)
- [migrations](/d:/huzhoumajiang/Huzhoumahjong/migrations)
- [deployments](/d:/huzhoumajiang/Huzhoumahjong/deployments)
- [legacy](/d:/huzhoumajiang/Huzhoumahjong/legacy)

关键入口文件：

- 前端入口：[main.ts](/d:/huzhoumajiang/Huzhoumahjong/frontend/src/main.ts)
- 前端房间状态仓库：[room.ts](/d:/huzhoumajiang/Huzhoumahjong/frontend/src/stores/room.ts)
- 前端实时通信客户端：[client.ts](/d:/huzhoumajiang/Huzhoumahjong/frontend/src/ws/client.ts)
- 后端入口：[main.go](/d:/huzhoumajiang/Huzhoumahjong/backend/cmd/api/main.go)
- 房间服务：[room_service.go](/d:/huzhoumajiang/Huzhoumahjong/backend/internal/service/room_service.go)
- 游戏服务：[game_service.go](/d:/huzhoumajiang/Huzhoumahjong/backend/internal/service/game_service.go)
- 麻将判定逻辑：[mahjong_logic.go](/d:/huzhoumajiang/Huzhoumahjong/backend/internal/service/mahjong_logic.go)
- WebSocket Hub：[hub.go](/d:/huzhoumajiang/Huzhoumahjong/backend/internal/ws/hub.go)

## 本地启动

### 方式一：Docker Compose

1. 在仓库根目录执行：

```bash
docker compose up --build
```

2. 浏览器访问：

```text
http://localhost
```

3. 后端健康检查：

```text
http://localhost/healthz
```

### 方式二：手动开发

后端：

```bash
cd backend
go mod tidy
go run ./cmd/api
```

前端：

```bash
cd frontend
npm install
npm run dev
```

## 当前 MVP 已迁移与暂时简化的部分

### 已迁移

- 牌堆生成与洗牌
- 牌排序与手牌展示
- 胡牌判定
- 吃碰杠流程
- 番型/积分基础计算
- 房间系统与实时同步
- 游戏结果弹窗

### 暂时简化

- 当前反应优先级为“单一声明者处理”，没有完整实现多人同时抢胡裁决
- 当前牌局运行态以内存为主，Redis 负责快照、在线状态和会话，尚未实现服务端重启后的完整对局恢复
- 规则实现以原仓库逻辑为基础做了工程化迁移，不代表完整覆盖全部正宗湖州麻将细则
- 前端视觉风格保留了旧版的台面感与配色，但没有逐像素复刻旧 HTML

## 说明

- 如果你要继续补规则，优先从 [game_rules.go](/d:/huzhoumajiang/Huzhoumahjong/backend/internal/service/game_rules.go) 和 [mahjong_logic.go](/d:/huzhoumajiang/Huzhoumahjong/backend/internal/service/mahjong_logic.go) 扩展
- 如果你要继续补房间/重连/恢复能力，优先从 [room_service.go](/d:/huzhoumajiang/Huzhoumahjong/backend/internal/service/room_service.go) 和 [repository.go](/d:/huzhoumajiang/Huzhoumahjong/backend/internal/repository/redis/repository.go) 扩展
