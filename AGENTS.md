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

## `context.Context`: где нужен

`ctx` нужен там, где отмена даёт выигрыш: ввод-вывод и внешние
провайдеры (получение репозитория), циклы по заранее неизвестному
объёму данных (обход репозитория, проход по всем скилам и источникам),
логирование через `slog.*Context`. Мелким функциям в памяти (`Name()`,
геттеры, разбор/маскирование содержимого одного файла, проверка пути)
`ctx` не нужен: отмену проверяет цикл над ними, между единицами работы
(`fs.FS` всё равно не прерывается посреди чтения). Если `ctx` есть — он
первый параметр; это проверяет `services/test` (architecture.feature),
обязательность `ctx` — на ревью.

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
- `entity` (target): `TargetSkillCatalog`, `TargetSkill`, `TargetFile` —
  скилы в том виде, в каком их запишут в target. Каждый — **слой** над
  нижним: исходным `Skill`/`File` или (в клоне) базовым `TargetSkill`/
  `TargetFile`; хранит только изменённое (поля — указатели, `nil` = как
  внизу), остальное читает сквозь слои геттерами. `NewTargetSkillCatalog(
  []*Skill)` ничего не читает; `Clone()` — новый слой на target;
  `Document()` отдаёт копию `Properties` (меняется только через
  `SetDocument`); `AddFile` — файл без источника (`Origin() == nil`).
  Клон листает файлы базового слоя при первом обращении — базовый слой
  доделать до `Clone`. Тесты — `entity/features/target_catalog.feature`.
- `services/sourcing`: `Manager` (acquire по `SourceKey`, кеш, `TempDir`
  задаётся в `NewManager`), `SkillCatalog` (см. ниже).
- `services/selector`: `SkillSelector{SkillCatalog}.Select(ctx, spec)
  ([]*Skill, issues.SkillIssues)` — наполняет каталог: `GetOrFetchByPath`
  по `spec.Subpaths` (по умолчанию `"."`). **Теги пока не применяются**
  (см. «Отложено: фильтр по тегам»). Проблема не
  останавливает остальные subpath, кроме `source-acquire` и отмены
  контекста (`canceled`) — они прекращают источник. У каждой проблемы
  заполнен `Source`. `unsafe-subpath` — panic (его отсекает валидатор
  конфига); `ErrSkillNotCached` — panic (Select ссылки не разрешает).
- `handler`: обработчики команд — только оркестрация.
  `FetchAndValidateSkills(ctx, *sourcing.Manager, req) (*SkillCatalog,
  issues.SkillIssues)`: каталог (`AddRelations`, `ExcludeFromChecks` =
  глобальные + источника по `SourceKey`) → `Select` по **всем**
  источникам → `[link-validator, skill-name-validator]` (даже после
  ошибок выбора) → каталог и все проблемы. `req` уже прошёл
  `config/validator.Validate`.
  `handler/sync.go`: `SyncService{Sources}.Run` — заглушка: вызывает
  `FetchAndValidateSkills`, при проблемах возвращает их, иначе
  `panic("not implemented")`. Старый конвейер — в истории git; его
  `sync.feature` лежит в `handler/features` под `@todo` как спецификация.
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

- `validator/validator.go` — общий для каталога и конфига (одни правила,
  не заводить второй менеджер): дженерики `Validator[T, I]{Name,
  DependsOn, Validate(ctx, T) []I}`, `Manager[T, I]`, `Dependency{Name,
  IsRequired}`. Для каталога `T=*sourcing.SkillCatalog, I=issues.SkillIssue`
  (алиас `validators.CatalogValidator`). Тип-аргументы `NewManager` из
  конкретных валидаторов не выводятся — передавать `[]Validator[T, I]`
  или указывать явно. `NewManager` **не сортирует** (порядок задаёт человек) и
  отказывает всеми нарушениями сразу: повтор имени, обязательная
  зависимость не зарегистрирована, зарегистрированная зависимость стоит
  после зависящего. `Validate` запускает все валидаторы по порядку, даже
  после найденных ошибок, и возвращает все проблемы.
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
- **Одно место проверки конфига.** `config.Parse` — только форма и
  типы (граница пользовательского ввода, возвращает ошибки) →
  `config.Resolve` — только преобразования (пути абсолютные), ничего не
  проверяет → `config/validator.Validate(ctx, req) issues.ConfigIssues` —
  **единственное** место смысловых проверок. Всё после него считает
  запрос валидным: попасть туда с невалидным конфигом — ошибка в коде,
  **panic**, а не Issue (сейчас: `invalid-tags` и `unsafe-subpath` в
  selector). Вызывается в `command` между `Resolve` и обработчиком (домен
  не импортирует `config`).
- **Валидатор конфига** — `internal/config/validator/validator.go`
  (список и порядок валидаторов) + `validators/`
  (`T=model.Request, I=issues.ConfigIssue`, алиас `ConfigValidator`):
  - `tags-validator`: `invalid-tags`, Setting `sources[i].tags[j]`;
  - `unsupported-tags-validator`: `unsupported-tags`, Setting
    `sources[i].tags` — временно, пока нет фильтра по тегам;
  - `subpath-validator`: `unsafe-subpath` (`..`, `\`, абсолютный вне
    local-источника, абсолютный в github), без ФС;
  - `source-validator` (источники одного SourceKey, каждый позже — с
    первым): `duplicate-source` (те же subpath-множество и теги),
    `conflicting-exclude` (разные `exclude_from_checks`);
  - `target-validator`: `target-overlap` (тот же путь или вложенный) —
    перенесён из `Resolve`.
  Тесты идут через `Parse` → `Resolve("/project")` → `Validate`; фича
  валидатора сверяет только свои коды (`the config issues with codes`).
- В `main.go` нужно вызвать `entity.SetDefaultLinkSearcher(
  links.NewDefaultLinkFactory())`, иначе `File.Links()` — ошибка.

Шаги (после каждого — стоп):
1. ✅ Переименование: `services/validators` → `services/validator`,
   `validators/validator` → `validator/validators`.
2. ✅ Модель ошибок `model/issues`, `source-acquire`/`missing-subpath`,
   `Source` в ошибках selector, `ExcludeFromChecks` (константа, карта в
   каталоге, новые ключи YAML + warning).
3. ✅ Дженерик `Manager[T, I]` + валидатор конфига; `target-overlap` из
   `Resolve` в валидатор; panic в selector на `invalid-tags`/`unsafe-subpath`.
4. ✅ `handler/fetch_and_validate_skills.go` + тесты; `sync.go` →
   `handler/sync.go` (заглушка). Фильтр тегов отложен.

## Текущая задача: SkillCatalog → TargetSkillCatalog → target ⏳

Схема `sync`: `FetchAndValidateSkills` → базовый `TargetSkillCatalog`
(`NewTargetSkillCatalog(catalog.Skills())`) → общие трансформеры →
`Clone()` на каждый target (`.agents/skills`, `.claude/skills`) →
трансформеры target по его адаптерам → запись в папку target.

Принятые решения:
- Трансформеры: `services/transformer` (конвейер, пишет имена
  применённых) + `services/transformer/transformers`, по аналогии с
  валидаторами. Неразрешимая после валидации ссылка — panic.
- `FlatTransformer` — всегда и первым, в базовом слое: скил →
  `{target}/{name}/`, маркер (любого формата) → `{name}/SKILL.md`,
  остальные файлы — `{name}/<путь от папки скила>`. Ссылки в `.md`
  переписываются по **текущему** содержимому (разбор заново) на новый
  относительный путь; web-ссылки и ссылки-якоря не трогаются; в папках
  `exclude_from_checks` ссылки не переписываются (их не проверяли).
  Wikilink'и превращаются в markdown-ссылки. Общая папка `files/`
  (старый код) не нужна: ссылки ведут только в загруженные скилы.
- `ClaudeWhenToUseTransformer` (адаптер `claude-property-adapter`):
  `whenToUse` → нативное `when_to_use` (Claude Code дописывает его к
  `description`, лимит 1536 символов на оба; незнакомые поля молча
  игнорирует). Список → через `", "`; если оба поля есть — остаётся
  `when_to_use`, warning, `whenToUse` не трогается.
- Маркер `.ai-skills-managed` (`model.Marker`, имя не менять — по нему
  распознаются уже синхронизированные папки) — трансформер, последний в
  каждом target: источник, путь скила в источнике, применённые
  трансформеры. **Hash отложен** (был для пропуска неизменённых скилов;
  сначала замерить скорость, возможно хватит распараллеливания).
- `link-adapter` в конфиге теряет смысл (раскладка со ссылками всегда) —
  объявить устаревшим при подключении к конфигу.
- Старые `services/transform` и `services/planning` заменяются новыми
  пакетами, не чинятся.

Шаги (после каждого — стоп):
1. ✅ `TargetSkillCatalog`/`TargetSkill`/`TargetFile` в `entity`
   (слои, `Clone`) + тесты.
2. Конвейер трансформеров + `FlatTransformer`.
3. `ClaudeWhenToUseTransformer`.
4. Маркер `.ai-skills-managed`.
5. `sync`: каталог на target → трансформеры по адаптерам → запись.

## Старые пакеты (ещё не переведены, не собираются)

`relations`, `planning`, `transform`, `command`, `cmd/ai-skill-manager`,
`infrastructure/filesystem`. Причины: пакеты `services/discovery` и
`infrastructure/document` удалены; `relations`/`planning` работают со
старым `model.SkillCatalogImpl` и `Skill.FileData`; `transform/links.go`
обращается к `Link.Target` и `model.SkillDocument`, которых нет.
`relations.Expander` по смыслу заменяется `LinkValidator` + `SkillCatalog`.

`services/test` (проверка архитектуры домена) с переездом `sync.go`
снова собирается и проходит. Внешние библиотеки домену запрещены, кроме
`allowedExternal` в `architecture_steps_test.go`: `go.yaml.in/yaml/v3` —
только в `entity` (frontmatter скила — YAML, разбор часть сущности).

## Отложено: фильтр по тегам

Сейчас выбор только по путям, selector `SourceSpec.Tags` не применяет.
Чтобы теги не игнорировались молча, валидатор конфига отказывает
источнику с `tags` (`unsupported-tags-validator`, код `unsupported-tags`);
`tags-validator` (синтаксис) остаётся зарегистрированным. Когда фильтр
вернётся — удалить `unsupported-tags-validator`, снять `@todo` со
сценариев с тегами (selector, handler).
Требование: после `FetchAndValidateSkills` в каталоге только скилы,
прошедшие фильтры пути **и** тегов, плюс связанные при `add_relations`;
валидируются только они. Отвергнуто/проблемы:
- фильтр-поле каталога: теги у каждого источника свои (один репозиторий
  может быть в двух источниках с разными тегами), а догрузка по ссылкам
  фильтр обходит;
- фильтр в `GetOrFetchByPath`: кэш по пути не знает, какой фильтр
  применялся;
- `accept func` в каталоге: пользователю не нравится; лучше набор
  объектов-фильтров, каждый знает, как фильтровать.
Идея пользователя: два кэша — по путям (все найденные скилы) и
прошедших фильтр; в будущем поиск по тегам вместо путей, тогда кэш путей
становится контейнером всех скилов репозитория.

## Открытые вопросы

- Невалидное имя скила даёт разные коды: flat-скил — `invalid-name`,
  скил-папка — `invalid-skill` (`isSkillDir` заворачивает ошибку
  `MakeSkill` в общий код вместе с `pattern-conflict`). Тесты проверяют
  общий текст `invalid skill name`.

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
