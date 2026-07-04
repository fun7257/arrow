# Arrow Skills

可分享给其他开发者 / AI 助手的 Arrow 框架 Skill 集合。当前包含：

| Skill | 目录 | 说明 |
|-------|------|------|
| `arrow` | [`arrow/`](arrow/) | HTTP 穿透中间件、路由、`target` 响应、开发与排错 |

人类用户文档见仓库根目录 [`README.md`](../README.md)。

## 安装（Grok）

### 仅本仓库生效（推荐协作者）

在仓库根目录执行：

```bash
./skills/install.sh
# 或显式：./skills/install.sh project
```

将 `skills/arrow/` 复制到 `.grok/skills/arrow/`。Grok 会自动加载，斜杠命令：`/arrow`。

### 全局生效（所有项目）

```bash
./skills/install.sh user
```

安装到 `~/.grok/skills/arrow/`。

### 手动安装

```bash
# 项目级
mkdir -p .grok/skills
cp -R skills/arrow .grok/skills/arrow

# 用户级
mkdir -p ~/.grok/skills
cp -R skills/arrow ~/.grok/skills/arrow
```

## 其他 AI 工具

将 `skills/arrow/SKILL.md` 与 `skills/arrow/references/` 一并作为上下文提供即可；详细 API 在 `references/` 中按需引用。

## 维护

修改 Arrow 公共 API、中间件语义或 `target` 时，请同步更新：

1. `skills/arrow/SKILL.md` 与 `skills/arrow/references/`
2. 仓库根目录 `README.md`

安装目录（`.grok/skills/`）为本地生成物，**不要**直接编辑；改源文件后重新运行 `./skills/install.sh`。