# CLI и конфигурация

## Команды и exit codes

`aism sync [options]` и `ai-skill-manager sync [options]` запускают один use case.

| Код | Значение |
| --- | --- |
| 0 | Успех, dry-run, справка или версия |
| 1 | Ошибка конфигурации, источника, проверки, записи или профилирования |
| 2 | Ошибка аргументов |

План и итог выводятся в stdout; ошибки и структурированные логи — в stderr.
Вывод предназначен для человека; стабильный JSON-протокол пока не заявлен.

## Флаги

| Флаг | Тип / default | Назначение |
| --- | --- | --- |
| `-c, --config` | путь, `ai-skills.yaml` | YAML или JSON |
| `-t, --type` | `local` / `github` | Прямой режим без конфигурации |
| `-p, --path` | строка | Источник; GitHub: URL или `"URL branch"` |
| `--subpath` | повторяемый путь | Для прямого GitHub-режима; default `skills` |
| `--target` | путь | Заменить все цели одной с link-adapter |
| `--dry-run` | bool, false | Проверить и вывести план без записи целей |
| `-f, --force` | bool, false | Переписать неизменившиеся скилы |
| `--remove-orphans` | bool | Включить удаление управляемых orphan-скилов |
| `--keep-orphans` | bool | Выключить удаление; при обоих флагах побеждает remove |
| `--add-relations` | bool | Добавить связанные скилы; можно `=false` |
| `--debug` | bool, false | Подробные логи этапов |
| `--profile` | bool, false | Записать Go CPU profile |
| `--profile-output` | путь, `ai-skill-manager.prof` | Файл профиля |
| `--version` | bool | Версия из VERSION при make build |
| `-h, --help` | bool | Справка |

Флаги принимаются до и после `sync`; значения можно передать через `=`.
Устаревшие типы `auto`, `flat`, `directory` работают как `local`.
Приоритет источников: явный `--config` → `--type` + `--path` → файл по умолчанию.
CLI overrides имеют приоритет над настройками. Dry-run из конфигурации
нельзя отключить флагом `--dry-run=false`.

```sh
aism sync -t github -p "https://github.com/InsonusK/ai-skills.git master" --subpath skills/go --add-relations --dry-run
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
  for_each:
    adapters: [link-adapter]
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
  on_conflict: error
  validation:
    exclude_from_checks: [examples]
```

### Источники

| Поле | Default | Правило |
| --- | --- | --- |
| `type` | `local` | local / github / legacy aliases |
| `path` | обязательное | Локальный путь или Git URL |
| `tree` | `master` | Ветка или тег Git |
| `subpath` | GitHub: `skills`; local: корень | Строка или список; отсутствующий путь даёт пустой выбор |
| `tags` | без фильтра | Строка или список выражений; список объединяется AND |
| `exclude_from_checks` | нет | Папки первого уровня скилов этого источника, исключённые из проверок; дополняют `settings.validation.exclude_from_checks` |
| `name` | исходное имя | Override допустим при ровно одном выбранном скиле |

Папки из `exclude_from_checks` **загружаются и копируются вместе со скилом**,
но не проверяются: ни на вложенные скилы, ни на битые ссылки. Обычно там
лежит код примеров, ссылки в котором часто ведут «в никуда».

Итоговый список для источника — объединение
`settings.validation.exclude_from_checks` и `exclude_from_checks` источника.
Если `settings.validation.exclude_from_checks` не задан, действует `[examples]`
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
Если список пуст, включается `link-adapter`. Claude-преобразование включается
явным `claude-property-adapter`. Неизвестный адаптер вызывает ошибку.

Defaults: `dry_run=false`, `remove_orphans=true`, `add_relations=false`,
`on_conflict=error`. По умолчанию `temp_dir` пуст, поэтому используется системный временный каталог. Относительный `temp_dir` разрешается от каталога с `ai-skills.yaml`; указанный каталог должен уже существовать и быть доступен для записи. При `last_wins` одноимённый скил последнего источника
заменяет предыдущий. Пересекающиеся каталоги целей запрещены.

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
Если путь не найден, проверяется вариант с `.md`.
Исключаются web-ссылки, anchors, inline code, блоки `example`
и настроенные папки. Как в Python, inline code даже внутри подписи исключает ссылку.

Ссылки переписываются относительно каталога конфигурации, а не выходного Markdown.
Внешние файлы/папки копируются в `target/files/`; совпадающие basename
получают суффиксы `_1`, `_2`.
Ссылки внутри внешних вложений не обходятся рекурсивно.
Имя скила `files` конфликтует с этим каталогом при наличии внешних вложений.

## Диагностика

Ошибка содержит код, скил, файл и исходную ссылку, если этот контекст доступен.
Частые коды: `invalid-name`, `duplicate-name`, `missing-link`,
`unselected-skill`, `path-escape`, `unmanaged-target`, `output-collision`.
Исправьте источник или настройки и повторите dry-run.

Проверки всех целей выполняются до записи. I/O-сбой при применении может
оставить часть целей обновлёнными; общий rollback между целями отсутствует.
Не запускайте два писателя для одной цели одновременно.
