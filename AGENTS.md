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
  - `DefaultExcludeFromChecks = ["examples"]` — единственное место
    умолчания; `SourceSpec.ExcludeFromChecks` и `Request.ExcludeFromChecks`
    (глобальный) дополняют друг друга.
- `model/issues`: интерфейс `Reportable{Report() IssueReportRow}` —
  `IssueReportRow{Code, Message, Where []Location}` (одна проблема, `Where`
  от общего к частному; `Location{Kind, Value}`, kinds: source, skill,
  file, link, setting). Реализации сами выставляют `Where`:
  `SkillIssue{Code, Source, Skill, SkillPath, File, Link, Message}` →
  source → skill («имя (путь)») → file → link; `ConfigIssue{Code, Source,
  Setting, Message}` → source → setting. Списки `SkillIssues`,
  `ConfigIssues`. Печать — будущий общий сервис по `Report()`. Локальные
  переменные-списки называть `problems`, не `issues` (перекрывают пакет).
- `entity`: `Repository`, `Skill` (`FilesByPath` — чистый листер,
  `DirOrMarkerPath()`), `File` (`Content`, `Path(kind)`, `Links()`), `Link`
  (`MakeLink`, `Path(ctx, kind, resolver)`, `Skill(ctx, resolver)`),
  интерфейс `SkillResolver` и `ErrSkillNotCached` (объявлены в `entity`,
  потому что `sourcing` импортирует `entity`).
- `services/sourcing`: `Manager` (acquire по `SourceKey`, кеш, `TempDir`
  задаётся в `NewManager`), `SkillCatalog` (см. ниже).
- `services/selector`: `SkillSelector{SkillCatalog}.Select(ctx, spec)
  ([]*Skill, issues.SkillIssues)` — зовёт `GetOrFetchByPath` по
  `spec.Subpaths` (по умолчанию `"."`) и фильтрует по тегам. Проблема не
  останавливает остальные subpath, кроме: `invalid-tags` (ничего не
  грузится), `source-acquire` и отмены контекста (`canceled`) — они
  прекращают источник. У каждой проблемы заполнен `Source`. Паникует на `entity.ErrSkillNotCached`: `Select` —
  первичная загрузка, ссылки в нём разрешаться не должны (panic = нарушен
  инвариант, а не бизнес-ошибка).
- `services/links`: `parser` (markdown, wikilink → `ParsedLink`),
  `content_excluder` (inline code, example-fence; `CodeFences`),
  `LinkFactory` (реализует `entity.LinkSearcher`).
  `links.Resolve`/`InSkippedFolder` — старое разрешение путей для
  `relations`, дублирует `Link.Path`; убрать вместе с `relations`.
- `services/validator`: см. раздел ниже.

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
  `nested-skill`. Коды пути: `source-acquire` (провайдер не отдал
  репозиторий), `missing-subpath`, `unsafe-subpath`. `Source` заполнен.
- `ExcludeFromChecks map[SourceKey][]string` — папки верхнего уровня
  скилов источника: загружаются и копируются со скилом, но не проверяются
  (`nested-skill` здесь, ссылки в `LinkValidator`). Правило одно —
  `IsExcludedFromChecks(skill, rel)`. Своего умолчания нет: источника нет
  в карте — ничего не исключено (умолчание подставляет конфиг).
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

## Валидаторы (`services/validator`)

- `validator/validator.go`: `Validator{Name, DependsOn, Validate(ctx,
  *sourcing.SkillCatalog) model.Issues}`, `Dependency{Name, IsRequired}`,
  `Manager`. `NewManager` **не сортирует** (порядок задаёт человек) и
  отказывает всеми нарушениями сразу: повтор имени, обязательная
  зависимость не зарегистрирована, зарегистрированная зависимость стоит
  после зависящего. `Validate` запускает все валидаторы по порядку, даже
  после найденных ошибок, и возвращает все `Issue`.
- Реализации — подпакет `validator/validators`:
  - `LinkValidator{}` (`link-validator`): идёт по
    `catalog.Skills()` по индексу, поэтому догруженные по ссылкам скилы
    (`AddRelations=true`) проверяются тоже. Только `.md`-файлы, без
    web-ссылок и без файлов, исключённых из проверок
    (`catalog.IsExcludedFromChecks`). Коды:
    `missing-link-target`, `path-escape`, `unselected-skill`, ошибки
    каталога как есть, `missing-anchor`.
  - Якоря (`anchors.go`): `#`-заголовки вне fenced-блоков (GitHub-slug с
    `-1`, `-2` для повторов, или текст заголовка без учёта регистра),
    `^block-id`, HTML `id`/`name`. Якорь в не-`.md` → `missing-anchor`.
  - `SkillNameValidator` (`skill-name-validator`): `duplicate-name` на
    каждый скил с неуникальным именем по всем источникам. Зависит от
    `link-validator` (`IsRequired=false`).

## Текущая задача: загрузка и проверка скилов по конфигу ⏳

Цель: обработчик, который по `model.Request` наполняет `SkillCatalog` и
проверяет его; на выходе — каталог и все ошибки. Потом он становится
первой частью `sync`, и появляется отдельная команда `validate` (вывод
ошибок деревом, код выхода 1 при ошибках) — имя/флаги ещё обсудить.

Принятые решения:
- **Обработчики команд** — `internal/domain/handler`: только оркестрация
  вызовов services/entity, минимум логики. Сюда переезжают `sync.go` и
  новый `FetchAndValidateSkills` (не «Validate…»: это не чистый валидатор).
- **`SourceSelector`** остаётся отдельным классом: ищет скилы по конфигу и
  наполняет `SkillCatalog`. Обработчик логики поиска не содержит — зовёт
  selector по **всем** источникам, затем `validator.Manager`
  (`[link-validator, skill-name-validator]`).
- **Папки, исключённые из проверок** — конфиг:
  `settings.validation.exclude_from_checks` (глобально) и
  `sources[].exclude_from_checks`, дополняют друг друга (обработчик кладёт
  объединение в `catalog.ExcludeFromChecks[key]`). Глобальный не задан →
  `["examples"]` + info в лог; `[]`/пусто — ничего не исключать. Старые
  имена (`sources[].skip_folder`, `settings.validation.rules.link.
  skip_folder`) читаются с warning; старое и новое на одном уровне —
  ошибка.
- **Валидатор конфига** — `internal/config/validator` (как доменный:
  `validator.go` + `validators/`), проверяет `model.Request` (значит, и
  запрос из CLI-флагов). Проверки: `invalid-tags`, `duplicate-source`
  (одинаковые SourceKey+Subpaths+Tags), `conflicting-exclude` (один
  SourceKey с разными исключениями), `unsafe-subpath` (`..`, `\`),
  `duplicate-target`. Запускается в `command` до обработчика (домен не
  импортирует `config`). Логику `Manager` не дублировать — общий
  дженерик `Manager[T]` для конфига и каталога.
- **Проверки конфига**: `target paths overlap` уже ловится в
  `config.Resolve` (ошибкой) — учесть при `duplicate-target`.
- В `main.go` нужно вызвать `entity.SetDefaultLinkSearcher(
  links.NewDefaultLinkFactory())`, иначе `File.Links()` — ошибка.

Шаги (после каждого — стоп):
1. ✅ Переименование: `services/validators` → `services/validator`,
   `validators/validator` → `validator/validators`.
2. ✅ Модель ошибок `model/issues`, `source-acquire`/`missing-subpath`,
   `Source` в ошибках selector, `ExcludeFromChecks` (константа, карта в
   каталоге, новые ключи YAML + warning).
3. Дженерик `Manager[T]` + валидатор конфига с проверками.
4. `handler/fetch_and_validate_skills.go` (`FetchAndValidateSkills`) +
   тесты (всё валидно; ошибки из нескольких источников; add_relations
   on/off; дубли имён между источниками); перенос `sync.go` в
   `handler/sync.go` (отдельный файл, не часть обработчика).

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
