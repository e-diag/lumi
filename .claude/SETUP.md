# FreeWay VPN — Установка и запуск через Claude Code

## Шаг 1: Установи Claude Code

```bash
npm install -g @anthropic-ai/claude-code
```

Проверь:
```bash
claude --version
# должно быть v2.1.0 или выше
```

## Шаг 2: Установи everything-claude-code

```bash
# Клонируй репозиторий
git clone https://github.com/affaan-m/everything-claude-code.git
cd everything-claude-code
npm install

# Установи Go-правила (linux/mac)
./install.sh golang

# Windows
.\install.ps1 golang
```

Это установит Go-специфичные rules в `~/.claude/rules/` — Claude Code будет следовать
лучшим практикам Go автоматически в любом проекте.

## Шаг 3: Создай проект FreeWay

```bash
# Создай директорию проекта
mkdir freeway && cd freeway

# Скопируй наши файлы конфигурации
# (файлы из архива freeway-claude-config.zip)
cp -r /path/to/downloaded/freeway/. .

# Убедись что структура правильная:
ls .claude/
# rules/  agents/  commands/
```

## Шаг 4: Настрой Claude Code для проекта

В директории проекта запусти Claude Code:
```bash
claude
```

Claude Code автоматически прочитает:
- `CLAUDE.md` — контекст проекта
- `.claude/rules/` — правила разработки
- `.claude/agents/` — специализированные агенты
- `.claude/commands/` — slash-команды

## Шаг 5: Установи плагин everything-claude-code в Claude Code

Внутри Claude Code:
```
/plugin marketplace add affaan-m/everything-claude-code
/plugin install everything-claude-code@everything-claude-code
```

Теперь доступны все команды: /plan, /go-review, /go-test, /go-build, /tdd и др.

## Шаг 6: Запусти разработку Фазы 1

В Claude Code выполни:
```
/phase1-kickoff
```

Или вручную:
```
Прочитай CLAUDE.md и .claude/commands/phase1-kickoff.md и начни выполнение.
Реализуй все 8 шагов по порядку, объясняя каждое решение.
```

## Полезные команды в процессе разработки

```bash
# Внутри Claude Code:

/go-review         # Проверить текущий код перед коммитом
/go-test           # Запустить TDD workflow
/go-build          # Исправить ошибки сборки
/plan "задача"     # Спланировать реализацию новой функции

# Обратиться к агентам:
Use vpn-expert to explain how subscription URL format works
Use go-architect to design the payment webhook handler
Use go-reviewer to review internal/usecase/config_usecase.go
```

## Порядок работы с Claude Code

**Каждая сессия начинается так:**
1. Открой терминал в директории `freeway/`
2. Запусти `claude`
3. Claude прочитает CLAUDE.md автоматически
4. Скажи что делаем сегодня: "Продолжаем Фазу 2, нужно реализовать ЮKassa webhook"

**Экономия токенов (важно):**
- Используй `/compact` после завершения каждой фазы
- Используй `/clear` при переключении между несвязанными задачами
- Model по умолчанию: sonnet (достаточно для 80% задач)
- Для архитектурных решений: `/model opus`

## Структура проекта после Шага 6

```
freeway/
├── CLAUDE.md                    ← контекст для Claude Code
├── .claude/
│   ├── rules/
│   │   ├── common.md            ← общие правила
│   │   └── golang.md            ← Go-специфичные правила
│   ├── agents/
│   │   ├── go-architect.md      ← архитектурные решения
│   │   ├── go-reviewer.md       ← ревью кода
│   │   └── vpn-expert.md        ← VPN/Xray вопросы
│   └── commands/
│       └── phase1-kickoff.md    ← запуск Фазы 1
├── cmd/
│   ├── api/main.go
│   ├── bot/main.go
│   ├── web/main.go
│   └── migrator/main.go         ← реализуется в Фазе 6
├── internal/
│   ├── domain/
│   ├── usecase/
│   ├── repository/
│   ├── handler/
│   ├── worker/
│   └── infrastructure/
├── config.yaml
├── .env.example
├── docker-compose.yml
└── Dockerfile
```

## Частые вопросы

**Q: Claude Code не видит CLAUDE.md**
A: Убедись что запускаешь `claude` из директории `freeway/`, а не из родительской.

**Q: Как проверить что всё работает?**
A: После Фазы 1: `go run ./cmd/api` → `curl http://localhost:8080/sub/test-token`
   Должен вернуть base64 строку.

**Q: Claude начинает делать что-то не то**
A: Напомни: "Прочитай CLAUDE.md. Мы в Фазе X, текущая задача: Y"

**Q: Закончились токены в середине задачи**
A: В новой сессии скажи: "Продолжаем. Прочитай CLAUDE.md. 
   Мы реализовывали [что]. Вот что уже сделано: [список файлов]"
