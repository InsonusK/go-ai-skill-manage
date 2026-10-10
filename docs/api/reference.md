# CLI и конфигурация

## Команды и exit codes

`aism` и `ai-skill-manager` — один исполняемый файл с четырьмя командами:

- `sync` — загрузить скилы источников, проверить и записать во все target;
- `validate` — проверить конфигурацию и скилы, ничего не записывая;
- `feedback` — сообщить об ошибке или предложить улучшение скила issue в
  его GitHub-источник; отправляет только пользователь (см.
  [feedback](#feedback));
- `mcp` — те же `feedback`-действия как MCP-инструменты для агента (stdio),
  подтверждение — диалогом клиента (см. [docs/feedback.md](../feedback.md#mcp-сервер-для-агента)).

| Код | Значение |
| --- | --- |
| 0 | Успех, dry-run, справка или версия; `feedback send`, на который ответили не `y` |
| 1 | Проблемы конфигурации, скилов или target; ошибка источника, записи или профилирования |
| 2 | Ошибка аргументов |

План и итог выводятся в stdout; проблемы (деревом «источник → скил → файл →
ссылка» или «target → скил»), ошибки и структурированные логи — в stderr.
Вывод предназначен для человека; стабильный JSON-протокол пока не заявлен.

## Флаги

`aism --help` — список команд и глобальные флаги; `aism <команда> --help`
(и `aism feedback <действие> --help`, `aism mcp <действие> --help`) — все
флаги команды. Справка строится из той же таблицы, по которой разбираются
аргументы.

Глобальные флаги принимаются до и после команды; флаги команды — только
после неё. Команда принимает **только свои** флаги: чужой (`aism validate
--dry-run`, `aism feedback send ID --title …`) — ошибка аргументов, код 2.
Значения можно передать через `=`.

Глобальные:

| Флаг | Тип / default | Назначение |
| --- | --- | --- |
| `--debug` | bool, false | Подробные логи этапов |
| `--color` | `auto` / `always` / `never`, `auto` | Цвет логов и проблем; `auto` красит только терминал и учитывает `NO_COLOR` |
| `--profile` | bool, false | Записать Go CPU и heap профили |
| `--profile-output` | путь, `ai-skill-manager.prof` | Файл CPU-профиля |
| `--mem-profile-output` | путь, `ai-skill-manager.mem.prof` | Файл heap-профиля |
| `--version` | bool | Версия приложения (`internal/version/version.go`) |
| `-h, --help` | bool | Справка (команды, если она указана) |

`sync` и `validate`:

| Флаг | Команды | Тип / default | Назначение |
| --- | --- | --- | --- |
| `-c, --config` | обе | путь, `ai-skills.yaml` | YAML или JSON; другой конфиг — другой набор скилов (см. [наборы](#несколько-наборов-скилов)) |
| `-t, --type` | обе | `local` / `github` | Прямой режим без конфигурации |
| `-p, --path` | обе | строка | Источник; GitHub: URL или `"URL tree"` (ветка, тег или коммит) |
| `--subpath` | обе | повторяемый путь | Для прямого GitHub-режима; default `skills` |
| `--add-relations` | обе | bool | Добавить связанные скилы; можно `=false` |
| `--target` | `sync` | путь | Заменить все цели одной, без адаптеров |
| `--dry-run` | `sync` | bool, false | Проверить и вывести план без записи целей |
| `--remove-orphans` | `sync` | bool | Включить удаление управляемых orphan-скилов |
| `--keep-orphans` | `sync` | bool | Выключить удаление; при обоих флагах побеждает remove |
| `-f, --force` | `sync` | bool, false | Устарел, ни на что не влияет (warning): управляемые папки перезаписываются всегда |

`feedback` и `mcp` берут конфигурацию только из файла (`-c`); их флаги —
в разделе [feedback](#feedback).

Устаревшие типы `auto`, `flat`, `directory` работают как `local`.
Приоритет источников: явный `--config` → `--type` + `--path` → файл по умолчанию.
CLI overrides имеют приоритет над настройками. Dry-run из конфигурации
нельзя отключить флагом `--dry-run=false`.

```sh
aism sync -t github -p "https://github.com/InsonusK/ai-skills.git master" --subpath skills/go --add-relations --dry-run
aism validate -c project/ai-skills.yaml
aism sync -c project/ai-skills.yaml --keep-orphans
aism --profile --profile-output /tmp/aism.prof sync --dry-run
go tool pprof /tmp/aism.prof
```

## feedback

Отзыв о скиле проходит три шага: агент пишет черновик, пользователь его
проверяет и сам отправляет. Зачем так и как настроить доступ к GitHub —
[docs/feedback.md](../feedback.md).

```sh
aism feedback draft --skill NAME --kind bug|improvement --title TEXT (--body TEXT | --body-file FILE|-)
aism feedback show ID
aism feedback send ID|all
aism feedback decline ID
```

| Действие / флаг | Значение |
| --- | --- |
| `draft` | Записать черновик в `.ai-skills/feedback/<id>.md` рядом с конфигурацией. Ничего не отправляет; печатает итоговый issue и команду отправки |
| `--skill` | Имя скила, как папка в target. Обязателен. Источник берётся из маркера `.ai-skills-managed` первой target с этим скилом |
| `--kind` | `bug` (метка `bug`) или `improvement` (метка `enhancement`). Обязателен |
| `--title` | Заголовок issue, одна строка. Обязателен |
| `--body` / `--body-file` | Текст issue: строкой или из файла (`-` — stdin). Ровно один из двух |
| `show ID` | Статус черновика (`draft`, `sent`, `declined`), ссылка на issue и итоговый issue |
| `send ID` | Показать итоговый issue и открыть его после ответа `y`. Работает только в терминале: без него — код 1, ничего не отправлено |
| `send all` | То же для всех черновиков в статусе `draft`, по порядку id: каждый показывается и отправляется только после своего `y`. Сбой одного не останавливает остальные, код тогда 1; черновиков нет — `No feedback drafts to send.`, код 0 |
| `decline ID` | Закрыть черновик без отправки; файл остаётся |

`aism mcp [-c FILE]` отдаёт инструменты `feedback_draft` и
`feedback_submit`; конфигурация ищется от `CLAUDE_PROJECT_DIR`, если он
задан, и перечитывается на каждый вызов.

`aism mcp install [-c FILE] [--name NAME] [--replace]` добавляет сервер в
`.mcp.json` рядом с конфигурацией (Claude Code): `command` — имя из `PATH`
или абсолютный путь с предупреждением, `args` — `["mcp"]`, плюс `-c` для
конфига не с именем `ai-skills.yaml`. `--name` (default `ai-skills`) — ключ
в `mcpServers`, `--replace` — перезаписать другую запись под этим именем.
`aism mcp uninstall [--name NAME]` убирает запись. Остальное содержимое
`.mcp.json` сохраняется.

`ID` — имя черновика, его файл или путь к нему. `-c` указывает конфигурацию:
от её папки считаются target и папка черновиков. Токен GitHub:
`GH_TOKEN`, `GITHUB_TOKEN`, затем `gh auth token`.

## Схема YAML / JSON

```yaml
sources:
  - type: github
    path: https://github.com/InsonusK/ai-skills.git
    tree: master
    subpath: [skills/go]
    tags: ["stack/go & !deprecated"]
    exclude_from_checks: [demo]
target:
  default:
    path: .agents/skills
  claude:
    path: .claude/skills
    adapters: [claude-property-adapter]
settings:
  temp_dir: ./.tmp/ai-skill-manager
  dry_run: false
  remove_orphans: true
  add_relations: true
  validation:
    exclude_from_checks: [examples, templates]
```

### Источники

| Поле | Default | Правило |
| --- | --- | --- |
| `type` | `local` | local / github / legacy aliases |
| `path` | обязательное | Локальный путь или Git URL |
| `tree` | `master` | Ветка, тег или полный хеш коммита (40 или 64 hex-символа). Короткий хеш git-сервер не отдаёт: он работает только для github.com, через архив после неудачного клонирования |
| `subpath` | GitHub: `skills`; local: корень | Строка или список; отсутствующий путь — проблема `missing-subpath`, путь за пределы источника — `unsafe-subpath` |
| `tags` | без фильтра | Строка или список выражений; список объединяется AND. **Пока не поддерживается**: источник с `tags` — ошибка конфигурации `unsupported-tags` |
| `exclude_from_checks` | нет | Папки первого уровня скилов этого источника, исключённые из проверок; дополняют `settings.validation.exclude_from_checks` |

Папки из `exclude_from_checks` **загружаются и копируются вместе со скилом**,
но не проверяются: ни на вложенные скилы, ни на битые ссылки. Обычно там
лежит код примеров, ссылки в котором часто ведут «в никуда».

Итоговый список для источника — объединение
`settings.validation.exclude_from_checks` и `exclude_from_checks` источника.
Если `settings.validation.exclude_from_checks` не задан, действует `[examples, templates]`
(с info-сообщением в логе); `[]` или пустое значение отключает исключение.

Устаревшие имена читаются с warning в логе: `skip_folder` источника →
`exclude_from_checks`, `settings.validation.rules.link.skip_folder` →
`settings.validation.exclude_from_checks`. Старое и новое имя на одном уровне
одновременно — ошибка конфигурации.

### Цели и настройки

Корневой `target` принимает строку пути либо mapping именованных целей.
Legacy-форма `settings.target` принимается только при отсутствии корневого ключа;
одновременное указание обеих форм вызывает ошибку.
Для `default` путь по умолчанию `.agents/skills`, для `claude` —
`.claude/skills`; остальные имена требуют `path`.
`for_each.adapters` объединяется с adapters каждой цели без дублей.
Единственный адаптер — `claude-property-adapter`: переименовывает `whenToUse`
во frontmatter в `when_to_use`, которое понимает Claude Code. Раскладка
`{name}/SKILL.md` с переписанными ссылками делается всегда; устаревший
`link-adapter` принимается с warning и ничего не меняет. Неизвестный адаптер
вызывает ошибку.

Defaults: `dry_run=false`, `remove_orphans=true`, `add_relations=false`.
По умолчанию `temp_dir` пуст, поэтому используется системный временный каталог. Относительный `temp_dir` разрешается от каталога с `ai-skills.yaml`; указанный каталог должен уже существовать и быть доступен для записи.
Одноимённые скилы — всегда проблема `duplicate-name`; устаревший
`settings.on_conflict` принимается с warning и ничего не меняет.
Пересекающиеся каталоги целей — проблема `target-overlap`.

## Несколько наборов скилов

Набор — это отдельный конфиг: свои `sources`, свои `target`, свои
`settings`. Команды работают с одним конфигом за раз (`-c`), поэтому
наборы запускаются отдельными командами:

```sh
aism validate -c ai-skills.backend.yaml && aism sync -c ai-skills.backend.yaml
aism validate -c ai-skills.docs.yaml && aism sync -c ai-skills.docs.yaml
```

**Где лежит конфиг.** От папки конфига считаются относительные пути
target, `temp_dir`, папка черновиков `.ai-skills/feedback/` и `.mcp.json`
(`aism mcp install`). Держите конфиги наборов в корне проекта и называйте их
по набору (`ai-skills.<набор>.yaml`). Конфиг в подпапке (`sets/a.yaml`) с
target `.claude/skills` пишет в `sets/.claude/skills`; поднимайтесь явно:
`path: ../.claude/skills`.

**Общий target.** Наборы не знают друг о друге: маркер `.ai-skills-managed`
не хранит, из какого конфига пришёл скил, а `target-overlap` проверяется
только внутри одного конфига. Если два набора пишут в одну папку:

| Ситуация | Что произойдёт | Как быть |
| --- | --- | --- |
| `remove_orphans: true` (умолчание) | `sync` набора A удалит все управляемые скилы набора B как «лишние», и наоборот | Разные target у наборов; или `remove_orphans: false` во **всех** наборах с общим target |
| Скил с одним именем в двух наборах | Побеждает последний `sync`: папка с маркером перезаписывается без ошибки `duplicate-name` (она проверяется только внутри конфига) | Не пересекать наборы по именам скилов |
| Скил убрали из набора при `remove_orphans: false` | Его папка остаётся в target | Удалить папку вручную |

**feedback и MCP.** `aism feedback` ищет скил в target **своего** конфига —
передайте тот же `-c`, что и при `sync` этого набора. Для MCP-сервера на
каждый набор нужна своя запись в `.mcp.json`:
`aism mcp install -c ai-skills.docs.yaml --name ai-skills-docs`. Если
конфиги лежат в одной папке, черновики всех наборов попадают в одну
`.ai-skills/feedback/`.

## Форматы скилов

| Формат | Главный файл | Выход |
| --- | --- | --- |
| Agent | `some-dir/SKILL.md` | `{name}/SKILL.md` + вложения |
| HumanDir | `topic.skill/topic.skill.md` | `{name}/SKILL.md` + вложения |
| HumanFlat | `topic.skill.md` | `{name}/SKILL.md` |

YAML frontmatter должен содержать `name`: lowercase буквы, цифры,
одинарные или двойные дефисы между сегментами. Имя берётся из frontmatter.
Конфликт главных файлов и вложенные скилы дают ошибку.

## Теги

Приоритет: `!` → `&` → `|`; скобки меняют порядок.
Поддерживаются wildcard `*` (один сегмент) и `**` (несколько).
Для совместимости с Python запрос `a/b/c` совпадает также с тегом `b`
или `b/c`; обратного расширения тегов нет.
Запросы `a/*` и `a/**` включают сам `a`.
Таблица проверяемых примеров: [tags.feature](../../internal/domain/services/tags/features/tags.feature).

## Ссылки и вложения

Поддерживаются Markdown, изображения, `[[path#anchor|label]]`.
`./` и `../` считаются от файла; bare path — от корня источника.
Ссылка без `.md` ведёт на заметку `X.md`, если она есть, даже когда рядом
лежит папка `X` (как wikilink в Obsidian); на папку ссылаются как `X/`.
Исключаются web-ссылки, anchors, inline code, блоки `example`
и настроенные папки. Inline code в подписи ссылку не исключает:
``[`x`](y)`` и ``[[y|`x`]]`` — обычные ссылки на `y` (в отличие от Python,
который такую ссылку пропускает).

Ссылка должна вести в файл или папку загруженного скила или в **общий файл**
источника, который не лежит ни в одном скиле (например, registry-запись
каталога). На скил, который не выбран, — проблема `unselected-skill` (или он
догружается при `add_relations`); на папку вне скилов — `external-folder`; на
несуществующий путь — `missing-link-target`; за пределы источника —
`path-escape`. Общий файл копируется в первый (по порядку загрузки) скил,
который на него ссылается, как `{name}/files/<путь в источнике>`; все ссылки
на него ведут туда, его собственные ссылки копируются как есть.
При записи ссылки переписываются относительно нового места файла в target
(`./x`, `../other/SKILL.md`); ссылка, которая уже ведёт куда нужно, остаётся
как написана. Wikilink'и становятся markdown-ссылками. В папках
`exclude_from_checks` и не-`.md` файлах ссылки не переписываются.

## Диагностика

Проблемы печатаются деревом по месту: источник → скил → файл → ссылка,
настройка конфига или target → скил. Частые коды:

- конфиг: `unsupported-tags`, `invalid-tags`, `unsafe-subpath`,
  `duplicate-source`, `conflicting-exclude`, `target-overlap`;
- источник: `source-acquire`, `missing-subpath`;
- скил: `invalid-name`, `invalid-skill`, `pattern-conflict`, `nested-skill`,
  `duplicate-name`;
- ссылки: `missing-link-target`, `missing-anchor`, `path-escape`,
  `unselected-skill`;
- target: `unmanaged-target` (одноимённая папка без `.ai-skills-managed`).

Исправьте источник или настройки и повторите `aism validate`.

Проверки всех целей выполняются до записи. I/O-сбой при применении может
оставить часть целей обновлёнными; общий rollback между целями отсутствует.
Не запускайте два писателя для одной цели одновременно.
