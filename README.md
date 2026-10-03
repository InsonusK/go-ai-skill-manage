# AI Skill Manager

[![Pull request](https://github.com/InsonusK/go-ai-skill-manage/actions/workflows/pull-request.yml/badge.svg)](https://github.com/InsonusK/go-ai-skill-manage/actions/workflows/pull-request.yml)
[![Tests](https://img.shields.io/endpoint?url=https://insonusk.github.io/go-ai-skill-manage/tests-badge.json)](https://insonusk.github.io/go-ai-skill-manage/tests/)
[![Coverage](https://img.shields.io/endpoint?url=https://insonusk.github.io/go-ai-skill-manage/coverage-badge.json)](https://insonusk.github.io/go-ai-skill-manage/coverage/)
[![Mutation score](https://img.shields.io/endpoint?url=https://insonusk.github.io/go-ai-skill-manage/mutation-score-badge.json)](https://insonusk.github.io/go-ai-skill-manage/mutation/)

CLI на Go, который собирает AI-скилы из локальных папок и GitHub-репозиториев,
проверяет их и раскладывает в папки скилов агентов проекта —
`.agents/skills`, `.claude/skills` и любые другие.

## Установка

Готовые исполняемые файлы для Linux, macOS (amd64, arm64) и Windows (amd64) — на странице
[Releases](https://github.com/InsonusK/go-ai-skill-manage/releases):
`ai-skill-manager_<версия>_<os>_<arch>` (для Windows — с `.exe`) и
`ai-skill-manager_<версия>_checksums.txt` для проверки.

Linux (скрипт установит `aism` в `/usr/local/bin`, при необходимости через
`sudo`):

```sh
curl -fsSL https://raw.githubusercontent.com/InsonusK/go-ai-skill-manage/master/scripts/install.sh | sh
aism --version
```

Windows PowerShell (скрипт установит `aism.exe` в профиль пользователя и
добавит его в `PATH`):

```powershell
irm https://raw.githubusercontent.com/InsonusK/go-ai-skill-manage/master/scripts/install.ps1 | iex
aism --version
```

Оба скрипта сами находят последний GitHub Release и проверяют SHA-256 перед
установкой. Другую папку можно задать переменной окружения `AISM_INSTALL_DIR`.
Исходники скриптов: [Linux](scripts/install.sh), [Windows](scripts/install.ps1).
Для macOS скачайте соответствующий исполняемый файл со страницы Releases.

Git нужен для GitHub-источников; если `git clone` не удался, репозиторий
скачивается архивом. Собрать самому можно из исходников — нужен Go 1.26+:

Через `go install` (исполняемый файл — `ai-skill-manager`):

```sh
go install github.com/InsonusK/go-ai-skill-manage/cmd/ai-skill-manager@latest
ai-skill-manager --help
```

Из исходников (нужен Make):

```sh
git clone git@github.com:InsonusK/go-ai-skill-manage.git
cd go-ai-skill-manage
make build            # bin/ai-skill-manager и его копия bin/aism
./bin/aism --version
```

Положите `bin/aism` в любую папку из `PATH`. `aism` и `ai-skill-manager` —
один и тот же исполняемый файл. Python не нужен.

## Базовая настройка

Создайте в корне проекта `ai-skills.yaml`:

```yaml
sources:
  - path: ./my-skills                  # локальная папка со скилами
  - type: github                       # или репозиторий GitHub
    path: https://github.com/InsonusK/ai-skills.git
    tree: master                       # ветка, тег или полный хеш коммита
    subpath: [skills/go]
target:
  default: {}                          # .agents/skills
  claude:                              # .claude/skills
    adapters: [claude-property-adapter]
settings:
  remove_orphans: true                 # удалять скилы, которых больше нет в источниках
  add_relations: true                  # подтягивать скилы, на которые ссылаются выбранные
```

Пути считаются от папки конфигурации. Без `--config` берётся `ai-skills.yaml`
из текущей папки. Скил — это папка с `SKILL.md`, папка `{name}.skill` с
`{name}.skill.md` или отдельный файл `{name}.skill.md`; имя берётся из
frontmatter:

```markdown
---
name: code-review
description: Review a change before merge
whenToUse: on pull request
---
See [the checklist](./docs/checklist.md).
```

## Запуск

```sh
aism validate            # проверить конфиг и скилы, ничего не записывая
aism sync --dry-run      # показать, что будет создано, обновлено и удалено
aism sync                # записать скилы во все target
```

Без конфигурации — один источник и одна папка:

```sh
aism sync --type local --path ./my-skills --target .agents/skills
aism sync --type github --path "https://github.com/InsonusK/ai-skills.git master" \
  --subpath skills/go --target .agents/skills --dry-run
```

При проблемах `validate` и `sync` печатают их деревом «источник → скил → файл
→ ссылка» и завершаются с кодом 1, ничего не записав:

```text
source local:/project/my-skills@master
└── skill code-review (code-review)
    └── file SKILL.md
        └── link [the checklist](./docs/checklist.md)
            └── missing-link-target
                code-review/docs/checklist.md: link target does not exist
Found 1 problem(s)
```

Логи и коды проблем окрашиваются по умолчанию только при выводе в терминал.
`--color always` принудительно включает ANSI-цвета, `--color never` отключает
их; переменная окружения `NO_COLOR` отключает автоматический режим.

## Несколько наборов скилов

Чтобы разные источники уходили в разные target, заведите по конфигу на набор
и запускайте каждый отдельно:

```sh
aism sync -c ai-skills.backend.yaml
aism sync -c ai-skills.docs.yaml
```

- Кладите конфиги в корень проекта: пути target, папка черновиков
  `.ai-skills/feedback/` и `.mcp.json` считаются от папки конфига.
- Наборы не должны делить target при `remove_orphans: true` — каждый
  `sync` удалит скилы другого набора как «лишние». Подробнее и как быть —
  [docs/api/reference.md](docs/api/reference.md#несколько-наборов-скилов).

## Что делает `sync`

1. Загружает скилы источников и проверяет их: имена, вложенные скилы, ссылки
   и якоря, дубли имён между источниками. Любая проблема останавливает
   `sync` до записи.
2. Раскладывает каждый скил в `{target}/{name}/`: главный файл становится
   `SKILL.md`, остальные файлы сохраняют свои пути внутри папки скила и права
   (исполняемые скрипты остаются исполняемыми). Ссылки переписываются на новые
   места, wikilink'и `[[...]]` становятся markdown-ссылками.
3. Для target с `claude-property-adapter` переименовывает поле `whenToUse` в
   `when_to_use`, которое понимает Claude Code.
4. Кладёт в папку каждого скила маркер `.ai-skills-managed`: откуда скил и
   какие преобразования применены. Папки с маркером `sync` при следующем
   запуске перезаписывает целиком, а при `remove_orphans` удаляет, если скила
   больше нет. Папки без маркера (ваши собственные скилы) не трогаются никогда:
   одноимённая папка без маркера — ошибка `unmanaged-target`.
5. Сначала строит планы для всех target и только потом пишет, поэтому проблема
   в одном target оставляет нетронутыми все.

Папки `examples` и `templates` внутри скилов копируются, но не проверяются
(там обычно код примеров и шаблоны со ссылками-заглушками); список задаётся
`settings.validation.exclude_from_checks`.

Пока не поддерживается фильтр по тегам: `tags` в источнике — ошибка
конфигурации. Скил всегда получает имя из своего frontmatter.
Устарели и выдают warning: флаг `--force`, `settings.on_conflict`, адаптер
`link-adapter`.

Полный справочник флагов и конфигурации — [docs/api/reference.md](docs/api/reference.md).

## Отзыв о скиле

Если скил ошибается или его можно улучшить, агент пишет черновик issue в
GitHub-репозиторий, откуда скил пришёл, а отправляете его вы: в диалоге
Claude Code или командой в своём терминале. Черновики лежат в
`.ai-skills/feedback/` и коммитятся вместе с проектом.

Настройка — один раз:

```sh
# 1. Вход в GitHub через GitHub CLI (https://cli.github.com/): токен хранит gh
gh auth login
gh auth status

# 2. MCP-сервер для Claude Code в этом проекте (пишет .mcp.json — закоммитьте)
aism mcp install
claude mcp list          # ai-skills: aism mcp - ✔ Connected
```

При первом запуске `claude` в проекте одобрите сервер `ai-skills`. Дальше
попросите агента написать отзыв о скиле: он вызовет `feedback_draft`, затем
`feedback_submit`, и Claude Code покажет итоговый issue с галочкой
«Open this issue» — без неё ничего не отправится.

Без MCP — те же шаги командами:

```sh
aism feedback draft --skill guide --kind bug --title "Broken anchor" --body "The anchor #setup leads nowhere."
# проверить .ai-skills/feedback/<id>.md, затем в своём терминале:
aism feedback send <id>
```

Подробно — [docs/feedback.md](docs/feedback.md): [пошаговая настройка
`gh` и MCP](docs/feedback.md#настройка) (установка `gh` на разных системах,
вход из контейнера, одобрение сервера, проверка), свой токен вместо `gh` и
где его хранить, ошибки.

## Как читать проект

1. [Справочник CLI и конфигурации](docs/api/reference.md).
2. [Отзыв о скиле: черновик, проверка, отправка, доступ к GitHub](docs/feedback.md).
3. [Индекс возможностей, модулей и тестов](docs/features/sync.md).
4. [Запуск тестов, coverage и mutation testing](docs/testing.md).
5. [Инструкция для AI-агентов](docs/skills/go/ai-skill-manager.skill/ai-skill-manager.skill.md).
6. [Состояние переделки и принятые решения](AGENTS.md).

Точка сборки зависимостей — [main.go](cmd/ai-skill-manager/main.go).
Входной адаптер — [internal/command](internal/command/sync.go).
Обработчики команд — [internal/domain/handler](internal/domain/handler/sync.go),
бизнес-правила — [internal/domain/services](internal/domain/services).
Работа с диском, Git, HTTP и YAML — [internal/infrastructure](docs/features/sync.md#модули).
Каждый тестируемый пакет содержит `features/`, `test/`, `TESTS.md` и `usecases.md`.

## Разработка

```sh
make lint
make unit-test WITH_CODE_COVERAGE=true
make mutation-test
make test-report
```

Сайт отчёта: `public/index.html`. Единая команда: `make test-and-report`.
Тесты используют память, временные каталоги и локальный HTTP-сервер.
Сеть для тестовых сценариев не нужна; первая загрузка Go-зависимостей требует сеть.

Исходная Python-реализация сохранена в git submodule
[deprecated/ai-skill-manager](deprecated/ai-skill-manager).
