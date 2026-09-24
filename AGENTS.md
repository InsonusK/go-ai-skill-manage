# Переделка пайплайна `sync`: состояние и принятые решения

Документ фиксирует текущее состояние незавершённой переделки и решения,
которые уже приняты, чтобы не переигрывать их между сессиями. Обновлять по
мере продвижения. История промежуточных вариантов сюда не пишется — только
то, что верно для кода сейчас.

## Как работаем

- Работа идёт шагами; каждый шаг — самодостаточная правка со своими
  тестами. **После каждого шага — стоп**: не коммитить без прямого
  разрешения, пользователь сам смотрит диф и коммитит.
- Решение, которое меняет уже принятое (цикл импортов, «кто кому
  принадлежит», новые имена), — сначала обсудить, потом код.
- Весь `go build ./...` сейчас **не собирается** (см. «Старые пакеты» ниже).
  Критерий готовности шага: `go vet` и `go test` зелёные во всех
  затронутых пакетах, список несобираемых пакетов не вырос.
  `internal/infrastructure/repository/test` падает и на чистом `HEAD`
  (сценарий про имя временной папки) — не наша регрессия.
- Тесты — godog-сценарии (`features/*.feature` + `test/*_steps_test.go`).
  Для новых сценариев проверять, что они ловят поломку (временно сломать
  код → сценарий падает → вернуть).

## Зачем переделка

Раньше `sync` заранее загружал все источники и собирал все скилы, попавшие
под фильтр, — для скилов с большими example-приложениями это лишняя работа.
Цель — ленивая цепочка «источник → скил → файл → ссылка»: ничего не
грузится, пока явно не запрошено. Точка входа — `sourcing.SkillCatalog`.

## Текущая архитектура (новый код)

- `model`:
  - `SourceKey` — идентичность источника (acquire работает только через
    неё, не через `SourceSpec`).
  - `PathKind` (`OsAbsolute` `/a/b.md`, `RepoAbsolute` `a/b.md` без `./`,
    `FileRelative` всегда `./`/`../`, `SkillRelative`), `DetectPathKind`,
    `PathInRepo` — единственное место перевода пути между видами, без ФС.
  - `ParsedLink` — что парсер прочитал из ссылки (start, end, text, path,
    fragment, format, image).
  - `Issue{Code, Source, Skill, SkillPath, File, Link, Message}` — одна
    проблема; хватает для вывода «скил (имя+путь) → файл → ссылка +
    ошибка» (как `LinkValidationError`/`formatters.py` в `deprecated/`).
- `entity`: `Repository`, `Skill` (`FilesByPath` — чистый листер,
  `DirOrMarkerPath()`), `File` (`Content`, `Path(kind)`, `Links()`), `Link`
  (`MakeLink`, `Path(ctx, kind, resolver)`, `Skill(ctx, resolver)`),
  интерфейс `SkillResolver` и `ErrSkillNotCached` (объявлены в `entity`,
  потому что `sourcing` импортирует `entity`).
- `services/sourcing`: `Manager` (acquire по `SourceKey`, кеш, `TempDir`
  задаётся в `NewManager`), `SkillCatalog` (см. ниже).
- `services/selector`: `SkillSelector{SkillCatalog}.Select(ctx, spec)` —
  зовёт `GetOrFetchByPath` по `spec.Subpaths` (по умолчанию `"."`) и
  фильтрует по тегам. Паникует на `entity.ErrSkillNotCached`: `Select` —
  первичная загрузка, ссылки в нём разрешаться не должны (panic = нарушен
  инвариант, а не бизнес-ошибка).
- `services/links`: `parser` (markdown, wikilink → `ParsedLink`),
  `content_excluder` (inline code, example-fence; `CodeFences`),
  `LinkFactory` (реализует `entity.LinkSearcher`).
  `links.Resolve`/`InSkippedFolder` — старое разрешение путей для
  `relations`, дублирует `Link.Path`; убрать вместе с `relations`.
- `services/validators`: см. раздел ниже.

## `SkillCatalog`

- `Get*` — только кэш, поиск **по ключу**; промах → `ErrSkillNotCached`.
  `FetchByPath`/`FetchByPathUp` (публичные) — найти и провалидировать в
  `Repository`, ничего не запоминают. `addPath` — запомнить.
  `GetOrFetch*` = `Get*` + fetch + `addPath`; `TryGetOrFetch*` = `Get*` при
  `AddRelations=false`, иначе `GetOrFetch*`.
- `*ByPath` — скилы на/ниже пути; `*ByPathUp` — скил, в папке которого
  лежит путь (подъём по папкам вверх, соседние скилы не грузятся).
- Кэш `cachedByPath`: ключ `GetSkillKey(repo, путь)` → скилы. `addPath`
  кладёт результат под **запрошенным путём** и каждый скил под его
  `DirOrMarkerPath()`. `GetByPath` отвечает только на ранее запрошенный путь
  или место скила (иначе `GetOrFetchByPath(".")` после `"a"` вернул бы
  неполный ответ). Загрузка с `Issues` под запрошенным путём не
  запоминается — повторный вызов снова сообщит о тех же проблемах.
- `Skills()` — все загруженные скилы по одному, в порядке загрузки;
  догруженные позже идут в конец.
- `AddRelations` — поле каталога (зеркало настройки `add_relations`).
- Валидация при fetch: `pattern-conflict`, `invalid-name`/`invalid-skill`,
  `nested-skill`. Папки из `SetSkipFoldersInNestedChecker` (по умолчанию
  `examples`) освобождены от проверки `nested-skill`, но остаются частью
  скила и копируются с ним.
- Примеры преобразований — в doc-комментариях `catalog.go`; тесты —
  `catalog.feature`, `catalog.fetchByPath.feature`,
  `catalog.fetchByPathUp.feature`.

## Ссылки (`entity.Link`)

- Парсеры и `LinkFactory` возвращают `model.ParsedLink`; `MakeLink(file,
  parsed)` сам вычисляет `Raw` и `External` (web-префиксы). `File.Links()`
  строит ссылки через `MakeLink`. Требует `entity.SetDefaultLinkSearcher`
  при старте.
- Путь «как записан» — `Link.WrittenPath` (поле и метод `Path` не могут
  называться одинаково).
- `Link.Path`: resolver нужен только для `SkillRelative`. Web-ссылка →
  `web-link`. Пустой путь (`[x](#part)`) — сам файл. Ссылка без `.md`
  (`a/b/c`, есть только `a/b/c.md`) → `a/b/c.md`: расширение в результатах
  всегда явное. Нет цели → `missing-link-target`, вне репозитория →
  `path-escape`.
- `Link.Skill` → `resolver.TryGetOrFetchByPathUp`.

## Валидаторы (`services/validators`)

- `validators/validator.go`: `Validator{Name, DependsOn, Validate(ctx,
  *sourcing.SkillCatalog) model.Issues}`, `Dependency{Name, IsRequired}`,
  `Manager`. `NewManager` **не сортирует** (порядок задаёт человек) и
  отказывает всеми нарушениями сразу: повтор имени, обязательная
  зависимость не зарегистрирована, зарегистрированная зависимость стоит
  после зависящего. `Validate` запускает все валидаторы по порядку, даже
  после найденных ошибок, и возвращает все `Issue`.
- Реализации — подпакет `validators/validator`:
  - `LinkValidator{SkipFolders}` (`link-validator`): идёт по
    `catalog.Skills()` по индексу, поэтому догруженные по ссылкам скилы
    (`AddRelations=true`) проверяются тоже. Только `.md`-файлы, без
    web-ссылок и без файлов в `SkipFolders` верхнего уровня. Коды:
    `missing-link-target`, `path-escape`, `unselected-skill`, ошибки
    каталога как есть, `missing-anchor`.
  - Якоря (`anchors.go`): `#`-заголовки вне fenced-блоков (GitHub-slug с
    `-1`, `-2` для повторов, или текст заголовка без учёта регистра),
    `^block-id`, HTML `id`/`name`. Якорь в не-`.md` → `missing-anchor`.
  - `SkillNameValidator` (`skill-name-validator`): `duplicate-name` на
    каждый скил с неуникальным именем по всем источникам. Зависит от
    `link-validator` (`IsRequired=false`).

## Следующий шаг: `services/validate.go` ⏳

Задача пользователя: рядом с `sync.go` собрать `validate.go`, который по
настройкам строит `SkillCatalog`, загружает скилы и валидирует их; на
выходе — собранный каталог и список ошибок. Затем `validate.go` становится
первой частью `sync.go`, и появляется отдельная команда для проверки
конфигов без формирования итогового каталога скилов.

Что уже известно:
- Вход — `model.Request` (`command.App.Request` резолвит его из опций и
  конфига). Нужное из него: `Sources []SourceSpec` (`Subpaths`, `Tags`),
  `AddRelations` → `SkillCatalog.AddRelations`, `LinkSkipFolders`
  (по умолчанию `["examples"]`) → `LinkValidator.SkipFolders`.
- Порядок: для каждого `SourceSpec` — `selector.Select` (первичная
  загрузка), затем `validators.Manager` с `[link-validator,
  skill-name-validator]`. `Select` должен отработать по **всем**
  источникам до валидации (ссылки между источниками, дубли имён).
- `SourceSelector` в `sync.go` объявлен как `Select(ctx, catalog, spec)`,
  а реальный `selector.SkillSelector` — `Select(ctx, spec)` с каталогом в
  поле: сигнатуры разошлись, привести к одной — часть шага.
- В `main.go` нужно вызвать `entity.SetDefaultLinkSearcher(
  links.NewDefaultLinkFactory())`, иначе `File.Links()` — ошибка.
- Открыто: `SourceSpec.SkipFolders` (per-source, конфиг `skip_folder`) не
  используется нигде; `sourcing.SetSkipFoldersInNestedChecker` —
  глобальная переменная пакета, в `main.go` не вызывается (действует
  значение по умолчанию `["examples"]`). Решить, откуда её брать.

## Старые пакеты (ещё не переведены, не собираются)

`relations`, `planning`, `transform`, `services` (`sync.go`),
`services/test`, `command`, `cmd/ai-skill-manager`,
`infrastructure/filesystem`. Причины: пакеты `services/discovery` и
`infrastructure/document` удалены; `relations`/`planning` работают со
старым `model.SkillCatalogImpl` и `Skill.FileData`; `transform/links.go`
обращается к `Link.Target` и `model.SkillDocument`, которых нет.
`relations.Expander` по смыслу заменяется `LinkValidator` + `SkillCatalog`.

## Открытые вопросы

- Невалидное имя скила даёт разные коды: flat-скил — `invalid-name`,
  скил-папка — `invalid-skill` (`isSkillDir` заворачивает ошибку
  `MakeSkill` в общий код вместе с `pattern-conflict`). Тесты проверяют
  общий текст `invalid skill name`.
- Где хранить скил после Transformers (постобработанный вид) — не решено.

## Имена: абстрактные термины, переименовать по ходу

Имена вида root/main/owner не говорят, корень *чего*, главный *в чём*,
владелец *чего*. Массово не переименовываем — правим, когда трогаем
соответствующий код, или по вопросу пользователя. В новом коде такие имена
не вводить.

| Сейчас | Что значит на самом деле | Переименовать в |
|---|---|---|
| `Repository.RootPath` | абсолютный путь в ОС до папки, где лежит/куда скачан репозиторий | `RepoDirOsPath` |
| `Skill.MainFilePath` / `Skill.MainFile` | файл-маркер (`SKILL.md` / `{name}.skill.md`), по которому папка распознаётся как скил | `MarkerFilePath` / `MarkerFile` |
| `SkillCatalog.Owner` / `ownsPath` | найти скил, в папке которого лежит путь / лежит ли путь в папке скила | `GetSkillContainingPath` / `skillContainsPath` |

`Skill.SkillDirPath` — конкретное, оставить.
