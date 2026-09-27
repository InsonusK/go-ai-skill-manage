# CLI и конфигурация

## Команды и exit codes

`aism` и `ai-skill-manager` — один исполняемый файл с двумя командами:

- `sync` — загрузить скилы источников, проверить и записать во все target;
- `validate` — проверить конфигурацию и скилы, ничего не записывая.

| Код | Значение |
| --- | --- |
| 0 | Успех, dry-run, справка или версия |
| 1 | Проблемы конфигурации, скилов или target; ошибка источника, записи или профилирования |
| 2 | Ошибка аргументов |

План и итог выводятся в stdout; проблемы (деревом «источник → скил → файл →
ссылка» или «target → скил»), ошибки и структурированные логи — в stderr.
Вывод предназначен для человека; стабильный JSON-протокол пока не заявлен.

## Флаги

| Флаг | Тип / default | Назначение |
| --- | --- | --- |
| `-c, --config` | путь, `ai-skills.yaml` | YAML или JSON |
| `-t, --type` | `local` / `github` | Прямой режим без конфигурации |
| `-p, --path` | строка | Источник; GitHub: URL или `"URL branch"` |
| `--subpath` | повторяемый путь | Для прямого GitHub-режима; default `skills` |
| `--target` | путь | Заменить все цели одной, без адаптеров |
| `--dry-run` | bool, false | Проверить и вывести план без записи целей |
| `-f, --force` | bool, false | Устарел, ни на что не влияет (warning): управляемые папки перезаписываются всегда |
| `--remove-orphans` | bool | Включить удаление управляемых orphan-скилов |
| `--keep-orphans` | bool | Выключить удаление; при обоих флагах побеждает remove |
| `--add-relations` | bool | Добавить связанные скилы; можно `=false` |
| `--debug` | bool, false | Подробные логи этапов |
| `--profile` | bool, false | Записать Go CPU profile |
| `--profile-output` | путь, `ai-skill-manager.prof` | Файл профиля |
| `--version` | bool | Версия из VERSION при make build |
| `-h, --help` | bool | Справка |

Флаги принимаются до и после команды; значения можно передать через `=`.
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
| `tree` | `master` | Ветка или тег Git |
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
и настроенные папки. Как в Python, inline code даже внутри подписи исключает ссылку.

Ссылка должна вести в файл или папку загруженного скила: на скил, который не
выбран, — проблема `unselected-skill` (или он догружается при
`add_relations`), на файл вне всех скилов — `missing-link-target`/`path-escape`.
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
