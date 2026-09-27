# AI Skill Manager

CLI на Go, который собирает AI-скилы из локальных папок и GitHub-репозиториев,
проверяет их и раскладывает в папки скилов агентов проекта —
`.agents/skills`, `.claude/skills` и любые другие.

## Установка

Готовые исполняемые файлы для Linux, macOS и Windows (amd64) — на странице
[Releases](https://github.com/InsonusK/go-ai-skill-manage/releases):
`ai-skill-manager_<версия>_<os>_amd64` (для Windows — с `.exe`) и
`ai-skill-manager_<версия>_checksums.txt` для проверки.

```sh
v=2.0.0
curl -fsSLo aism "https://github.com/InsonusK/go-ai-skill-manage/releases/download/v$v/ai-skill-manager_${v}_linux_amd64"
chmod +x aism && sudo mv aism /usr/local/bin/
aism --version
```

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
    tree: master
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
  skill code-review (code-review)
    file SKILL.md
      link [the checklist](./docs/checklist.md)
        missing-link-target: code-review/docs/checklist.md: link target does not exist
Found 1 problem(s)
```

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
репозиторий, откуда скил пришёл, а вы проверяете его и отправляете сами:

```sh
aism feedback draft --skill guide --kind bug --title "Broken anchor" --body "The anchor #setup leads nowhere."
# проверить .ai-skills/feedback/<id>.md, затем в своём терминале:
aism feedback send <id>
```

Нужен вход в GitHub (`gh auth login`) или токен в `GH_TOKEN`. Как это
устроено, какой токен выбрать и где его хранить — [docs/feedback.md](docs/feedback.md).

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
