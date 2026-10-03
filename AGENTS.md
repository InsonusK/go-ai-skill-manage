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
- Критерий готовности шага: `go build ./...`, `go vet ./...` и `go test`
  затронутых пакетов зелёные.
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
  - `DefaultExcludeFromChecks = ["examples", "templates"]` — единственное место
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
  `Document()` — не поле, а разбор **текущего** содержимого файла-маркера
  (один источник правды); `SetDocument` кодирует документ обратно в него
  (ключи frontmatter сортируются, YAML-комментарии теряются — только у
  реально изменённых скилов); `AddFile` — файл без источника
  (`Origin() == nil`);
  `TargetFile.Changed()` — менял ли содержимое этот слой или нижний;
  `TargetFile.Mode()`/`SetMode` — режим сквозь слои: исходный
  (`File.Mode()`, лениво `fs.Stat`, кэш), у добавленного файла
  `DefaultTargetFileMode` (0644);
  `Applied()`/`AddApplied` — применённые трансформеры (клон начинает со
  списка базового).
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
  `handler/sync.go`: `SyncService{Sources, State, Writer}.Run(ctx, req)
  (SyncResult{Skills, Plans []entity.TargetPlan, DryRun}, error)`:
  `FetchAndValidateSkills` (проблемы → `issues.SkillIssues`, стоп) →
  базовый `TargetSkillCatalog` + `[flat]` → на каждый target `Clone()` +
  `[claude-when-to-use` при `ClaudeAdapter`, `managed-marker]` → снимок →
  `planning.Plan` → если у всех target нет проблем
  (`issues.TargetIssues` вместе) и не `dry_run` — запись по порядку
  target. Проблема где угодно — ни один target не тронут; сбой диска при
  записи может остановиться после части target. Тесты — фейковые
  `StateReader`/`PlanWriter` в памяти (`handler/test/sync_steps_test.go`).
- `services/links`: `parser` (markdown, wikilink → `ParsedLink`),
  `content_excluder` (inline code, example-fence; `CodeFences`),
  `LinkFactory` (реализует `entity.LinkSearcher`).
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
  → `a/b/c.md`, если он есть, даже когда рядом папка `a/b/c` (как
  Obsidian; решение пользователя вместо ошибки неоднозначности); `a/b/c/`
  — папка. Расширение в результатах всегда явное. Нет цели → `missing-link-target`, вне репозитория →
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
  `["examples", "templates"]` + info в лог; `[]`/пусто — ничего не исключать. Старые
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
- Трансформеры: `services/transform` — `Transformer{Name, Transform(ctx,
  *TargetSkillCatalog) error}` и `Pipeline` (`NewPipeline` отказывает
  повтору имени; `Run` по порядку, после каждого `AddApplied`, стоп на
  первой ошибке — каталог тогда писать нельзя) — и
  `services/transform/transformers`. Ошибка трансформера — сбой работы
  (чтение файла); невалидный скил/ссылка после валидации — panic. Свой
  конвейер, а не `validator.Manager`: трансформеры меняют каталог и
  останавливаются на ошибке.
- `FlatTransformer` — всегда и первым, в базовом слое: скил →
  `{target}/{name}/`, маркер (любого формата) → `{name}/SKILL.md`,
  остальные файлы — `{name}/<путь от папки скила>`. Ссылки в `.md`
  берутся из исходного файла (`Origin().Links()`, уже разобраны при
  валидации): Flat первый, содержимое ещё исходное — файл с `Changed()`
  → panic. Путь цели → новое место (`places.of`: маркер →
  `{name}/SKILL.md`, папка/файл скила → `{name}/...`) → относительный
  путь от нового места файла (`./x`, `../x`). В markdown меняется только
  `(path#fragment)`, и только если отличается; wikilink'и → markdown
  (`[[p]]` без текста → текст = имя файла, `[[#a]]` → `[a](#a)`);
  web-ссылки и якоря не трогаются; в папках `exclude_from_checks` и не-
  `.md` файлах ссылки не переписываются (их не проверяли). Картинка
  внутри текста ссылки `[![i](x)](y)` отвергается уже разбором
  (`link-overlap`). Общая папка `files/` (старый код) не нужна.
- `ClaudeWhenToUseTransformer` (адаптер `claude-property-adapter`):
  `whenToUse` → нативное `when_to_use` (Claude Code дописывает его к
  `description`, лимит 1536 символов на оба; незнакомые поля молча
  игнорирует). Список → через `", "`; если оба поля есть — остаётся
  `when_to_use`, warning, `whenToUse` не трогается.
- Маркер `.ai-skills-managed` (`model.Marker`, имя не менять — по нему
  распознаются уже синхронизированные папки) — трансформер, последний в
  каждом target (`ManagedMarkerTransformer`, `managed-marker`): JSON
  `model.ManagedState{source (SourceKey объектом), commit (если известен),
  skill_path (DirOrMarkerPath), transformers (применённые до него, без
  себя), version (model.TransformVersion)}`;
  маркер, уже лежащий в исходном скиле (папка target как источник),
  заменяется, а не дублируется.
  Запись (`filesystem.Store`) маркер **не пишет**, только проверяет, что
  он есть у папки, которую заменяет или удаляет. **Hash отложен** (был
  для пропуска неизменённых скилов; сначала замерить скорость, возможно
  хватит распараллеливания).
- Запись, домен: `planning.Plan(target, catalog, state, removeOrphans)
  (entity.TargetPlan, issues.TargetIssues)` — чистая функция (без ФС и
  `ctx`), подробное описание — в её doc-комментарии. По снимку папки
  (`model.Managed{Exists, Managed}` по имени): нет папки → `create`;
  папка с маркером → `update` (целиком, без hash пропуска нет); что-то
  без маркера (чужой скил, файл, symlink) → `unmanaged-target`, операции
  нет; управляемая папка, которой нет в каталоге → `remove` (по имени) при
  `remove_orphans`; немаркированное лишнее не трогается. Все планы
  строятся до первой записи; `dry_run` — только планы. Типы плана —
  `entity.TargetPlan{Target, Operations}`, `entity.TargetOperation{Action,
  Name, Skill *TargetSkill}` (содержимое читается только при записи).
  Ошибки — `issues.TargetIssue{Code, Target, Skill, Message}` (`Where`:
  target → skill).
- Запись, инфраструктура: `filesystem.Store` (`interfaces.StateReader`/
  `PlanWriter`). `Snapshot`: маркер — обычный файл, symlink/файл/папка без
  маркера — не управляемые. `Apply` исполняет план, но сам охраняет папку:
  план проверяется до записи (имя папки — один элемент пути, пути файлов
  внутри неё, **у каждого записываемого скила есть маркер**); заменяет и
  удаляет только папки с маркером; пишет во временную папку и меняет
  переименованием; target через symlink — отказ. Режим файла — точно как у
  источника (`Chmod` после записи, мимо umask).
- Устаревшие настройки (warning при подключении конфига): `force` (без
  hash ничего не меняет), `on_conflict: last_wins` (дубли имён — всегда
  ошибка валидации), адаптер `link-adapter`.
- Старые `services/transform` и `services/planning` заменены новыми.

Шаги (после каждого — стоп):
1. ✅ `TargetSkillCatalog`/`TargetSkill`/`TargetFile` в `entity`
   (слои, `Clone`) + тесты.
2. ✅ Конвейер трансформеров + `FlatTransformer`.
3. ✅ `ClaudeWhenToUseTransformer`; `Document()` из файла-маркера.
4. ✅ Маркер `.ai-skills-managed`.
5. ✅ Режимы файлов (`File.Mode()`, `TargetFile.Mode()`).
6. ✅ Запись: `planning.Plan` + `filesystem.Store` на новой модели.
7. ✅ `handler/sync.go` целиком; `sync.feature` на новых шагах.
8. ✅ Уборка: удалены `relations`, `links.Resolve`/`InSkippedFolder`,
   порт `interfaces.RepositoryLookup` и `Manager.LookupId`,
   `SkillCatalog.Owner`/`Destination`/`relativePath` (правило `ownsPath`
   проверяется через `GetByPathUp`), `model.NestedRepoPath`, типы
   `model.TargetPlan`/`Operation`/`OutputFile`/`Result`.


## `command` и `cmd` ✅

- Раскладка: `command/command.go` — корень: глобальные флаги (`Global`:
  `--debug`, `--color`, `--profile*`, `--version`, `-h`; до и после
  команды), поиск команды (до неё — только глобальные флаги), общий help,
  `Execute(ctx, *common.App, Invocation, cwd)`. Каждая команда — свой файл
  и тип с методами `Name`/`Summary`/`Parse`/`Help`/`Run`
  (`common.Command`): `sync.go`, `validate.go`, `feedback.go` (действия
  draft/show/send/decline, флаги зависят от действия), `mcp.go` (serve,
  install/uninstall; `.mcp.json` — `mcp_install.go`). `command/common` —
  общее: таблица флагов `Flag` (по ней и разбор, и help — help не может
  разойтись с разбором), `App` (порты), `Source` (`-c/-t/-p/--subpath`),
  `Request`/`LoadRequest` (конфиг → `config/validator.Validate` → печать
  проблем), `PrintIssues`, `FeedbackService`. Команда принимает только свои
  флаги и глобальные, чужой — код 2. С `--help` ошибки разбора команды не
  важны — печатается её help. `CLAUDE_PROJECT_DIR` читает `mcp` serve
  (`App.Getenv`). `sourcing.NewManager` создаётся в `Run` команды, после
  разбора запроса; конфиг читается один раз.
- Проблемы любого вида печатает `PrintIssues` (`command/format.go`) по
  `Report()`: строки группируются по месту (стабильная сортировка по
  `Where`) в дерево с `├──`/`└──`, код и сообщение — отдельные строки,
  место печатается один раз, затем `Found N problem(s)`; в stderr.
  `PrintResult` — операции по target и итог в stdout.
- `--color auto|always|never` (default `auto`) управляет ANSI-цветами логов
  и проблем; `auto` красит только stderr-терминал и учитывает `NO_COLOR`.
  Уровни `slog`: DEBUG серый, INFO голубой, WARN жёлтый, ERROR красный;
- Устаревшее (warning через `slog`): флаг `-f/--force` (`Sync.Force`),
  `settings.on_conflict` (ключ задан), адаптер `link-adapter` (только если
  указан явно; из умолчаний убран, в `Target.Adapters` не попадает). Поля
  `Request.Force`/`Conflict`, `Overrides.Force` удалены.
- `main.go`: `command.Parse` → логи/профиль по `Global` →
  `command.Execute`; `SetDefaultLinkSearcher`, провайдеры `local`/`github`,
  `filesystem.Store` как `State` и `Writer`.
- `go build ./...` и `go vet ./...` собираются целиком.
- Поле источника `name` разбирается, но **игнорируется** (документировано);
  при желании — отказывать им в валидаторе конфига, как `tags`.
- `docs/architecture/` удалён пользователем. README и
  `docs/api/reference.md` актуальны; `docs/features/sync.md` (индекс
  модулей) и `docs/skills/...` ещё описывают старые пакеты.

## Текущая задача: команды CLI, `tree`, наборы ⏳

Шаги (после каждого — стоп):
1. ✅ `command` по командам (см. «`command` и `cmd`»), help на всех
   уровнях, строгие флаги.
2. `tree` — ветка, тег **или коммит**: SHA (7–40 hex) клонировать сразу по
   нему (`git init` → `fetch --depth 1 origin <sha>` → `checkout
   FETCH_HEAD`), без попытки `--branch`; сейчас коммит работает только у
   github.com через запасной архив. Имя `tree` не меняется. Документация и
   сценарии на ветку/тег/коммит.
3. Документация: разные наборы sources → targets — отдельные конфиги
   (`-c`), каждый набор отдельной командой (решение пользователя). Описать:
   пути от папки конфига; общий target у двух наборов при `remove_orphans`
   — наборы удаляют скилы друг друга (маркер не знает конфиг); это решаем
   документацией, не кодом.

## Профилирование (`profiling/`)

`profiling/Makefile` (`make -C profiling …`): `validate`, `sync`,
`profile`, `report` (`RUN=cold|warm`), `web`, `python` (старый CLI на том
же конфиге, dry run), `clean`; `MODE=local|github`.
`make profile [MODE=local|github]` → `profiling/run.sh`: сборка, `sync`
дважды (cold в пустые target, warm поверх) с `--profile` (CPU +
heap, `--mem-profile-output`; в лог `msg=memory total_alloc_mb sys_mb
num_gc`), время, `pprof -top`. `local` — `ai-skill.local.yaml` на клоне
`profiling/sources/ai-skills` (клонируется один раз); `github` —
`ai-skill.yaml` пользователя (не менять без него). `profiling/out/`,
`profiling/sources/` — в `.gitignore`.

На реальном `ai-skills` (8 subpath, ветка пользователя с исправленными
ссылками) полный `sync`: 674 скила в 2 target, cold ~2,55 с, warm ~2,86 с,
~290 МБ выделено; ~63% CPU — системные вызовы (запись ~43%: `os.Root`
разбирает путь на каждую операцию, temp-файл + chmod + rename + MkdirAll на
файл; проверка ссылок ~33%: `fs.Stat`). Было до исправлений: 231 проблема; CPU ~60% — системные вызовы под `validateLink`
(`fs.Stat` при разрешении ссылок), ~17% — regexp. Находки ждут решения:
- `skill-not-found` (113): ссылки на файлы вне скилов (`registry/*.md` и
  т. п.); старый CLI копировал их в `target/files/`, новый — нет.
- `missing-anchor` (108): ~62 — ссылка без расширения с якорем
  (`[[.../X.extend#MUST]]`), рядом папка `X.extend` и заметка
  `X.extend.md`; `Link.Path` берёт папку, а не `.md`. Остальное похоже на
  ошибки репозитория (якорь на жирный текст, slug без `_`).
- `missing-link-target` (9): шаблоны со ссылками «от будущего места».
- `link-overlap` (1): бейдж `[![alt](img)](url)` отвергается разбором.
Решения пользователя: файлы вне скилов вернуть (как — обдумать, возможно
трансформер); ссылка без расширения при наличии и `X`, и `X.md` ведёт на
заметку `X.md` (строгая ошибка дала 271 проблему на `ai-skills`); `templates`
добавлен в умолчание `exclude_from_checks` (бейдж и заглушки там).
Расхождения с Python: Python не проверяет якоря; Python, похоже, проверяет
ссылки только в файлах, достижимых по ссылкам от главного файла скила (мы —
во всех `.md`); разрешение `X` → `X.md` одинаковое. Предложено: сверочные
godog-сценарии старой и новой реализации (обсуждается).

## Сверка с Python (`test/conformance/`)

Общие сценарии (только `.feature`) + Go-шаги, запускающие CLI из
`AISM_CLI` (осознанное отступление от solution-shared-conformance-testing:
обе реализации — CLI, шаги чёрного ящика одни). Без `AISM_CLI` — skip.
`make conformance` / `conformance-python` / `conformance-compare`
(`test/conformance/compare.sh`, логи в `tmp/conformance/`). Python — `.venv`
(коммит `f89ab47` = `deprecated/`), понимает только `settings.target`.
Список расхождений — `test/conformance/README.md`; по всем решено
оставить поведение Go (бейдж вне `examples`/`templates` — ошибка; ссылка за
пределы источника — ошибка, риск ИБ).
- **Общие файлы вне скилов** (registry-записи, variability map каталога —
  так их раскладывают `delta-conflict-detection`/`variability-map-create`):
  ссылка на файл, который не лежит ни в одном скиле, допустима
  (`LinkValidator`: `FetchByPathUp` → `skill-not-found` значит «общий
  файл»; незагруженный скил → `unselected-skill`; папка → `external-folder`;
  якорь проверяется). `FlatTransformer` копирует файл в **первый** по
  порядку ссылающийся скил: `{name}/files/<путь в источнике>` (детерминированно,
  без совпадений; управляется маркером скила), все ссылки ведут туда; ссылки
  внутри самого файла не переписываются — `LinkValidator` выдаёт warning (один
  на файл) со списком таких ссылок, чтобы пользователь посмотрел реальные
  данные (на `ai-skills` — 28 файлов: 116 ссылок на скилы, 129 на прочее).

## CI и релизы (`.github/`)

По скилам `devops-github-*`: действия `check-changes` (фильтр Go; `test`
включает `**/features/**` — сценарии лежат у пакетов) и `check-version`
(версия из строки `var Version = "..."` в `internal/version/version.go`,
`sort -V`; база без числовой версии, например `"dev"`, считается отсутствующей;
`publishable=false`); `pull-request.yml`
(changes → version-check только для PR в master → `make unit-test` с
`actions/setup-go` → агрегирующий `report`, его и требовать в branch
protection); `release-info-publish.yml` (push в master при поднятом
версии или ручной запуск → Release `v{версия}` с бинарниками
`ai-skill-manager_{v}_{linux,windows,darwin}_amd64` + checksums; сборка
`./cmd/ai-skill-manager` без `-X`). Первый Release: поднять версию в PR в
master или запустить workflow вручную.
- **Версия приложения — только `internal/version/version.go`** (решение
  пользователя): `var Version = "X.Y.Z"`, поднимать там; файла `VERSION` и
  `-ldflags -X` нет, любая сборка (`go build`/`go install`/`make build`)
  печатает настоящую версию. Это отступление от ADR скила
  `devops-github-action-check-version-in-go` (он выбирает файл `VERSION`).
- Скилы в `.claude/skills` **не править**: их источник — другой
  репозиторий; найденные расхождения пользователь заводит issue там.
  Отступления этого проекта от примеров скилов: `**/features/**` в
  check-changes; шаг, который роняет version-check при неподнятой версии
  (в примере action только выставляет `bumped`); `main` в
  `./cmd/ai-skill-manager` в сборке релиза; версия из `version.go`.

## Обратная связь в источник скила (`feedback`) ✅

Цель: агент, пользуясь скилом, может сообщить в его источник о баге или
предложить улучшение; CLI-команда `feedback` и MCP-сервер поверх неё.

Принятые решения:
- **Обратная связь — маркер `.ai-skills-managed`** (`model.ManagedState`):
  `source` — `SourceKey` объектом (`type`, `path`, `tree`), `commit` —
  точный коммит, из которого собран скил (git: `rev-parse HEAD`; архив
  GitHub: pax global header `comment`; `local` — пусто), `skill_path`.
  Агент источник не знает — на вход **имя скила**, источник берётся из
  маркера его папки в target.
- **Три этапа, отправляет только пользователь.** У агента нет
  инструмента, который отправляет без человека (от случайной утечки
  данных проекта в публичный issue; от злонамеренного агента с shell это
  не защищает, и не цель):
  1. черновик — файл `.ai-skills/feedback/<id>.md` в репозитории
     проекта (не во временной папке пользователя); **коммитится** вместе
     с проектом, чтобы оставался след (в `.gitignore` не добавляем),
     frontmatter: skill, kind (`bug`/`improvement`),
     repo, commit; дальше title/body. Файл — единственный источник
     правды, пользователь может править его в редакторе;
  2. подтверждение — MCP elicitation (клиент показывает итоговый текст,
     Accept/Decline); клиент без elicitation → агенту ответ «попросите
     пользователя выполнить `ai-skill-manager feedback send <id>`»:
     показ текста и `y/N`, **только в TTY**;
  3. отправка — из файла, со сверкой хеша показанного текста (правка
     после подтверждения его отменяет); к телу дописывается блок
     контекста (скил, путь в источнике, коммит, версия CLI); `kind` →
     метка; черновик помечается отправленным со ссылкой на issue.
- Корень проекта — папка `ai-skill.yaml`: черновики в
  `{папка конфига}/.ai-skills/feedback/`, как и пути target считаются от
  неё.
- Проверки дублей нет — на совести пользователя.
- `local`-источник → ошибка «локальный источник, правьте вручную».
- Команда — `feedback` (не только баги, но и предложения).
- MCP — подкоманда того же бинарника (stdio), инструменты — обёртки
  над теми же обработчиками, что и CLI.

Шаги (после каждого — стоп):
1. ✅ `ManagedState` → `model`, `source` объектом, `commit` в маркере
   (`Repository.Commit` от провайдера `github`); заодно архив GitHub:
   pax global header больше не роняет распаковку.
2. ✅ Черновик и отправка без CLI:
   - `model.FeedbackDraft` (статус `draft` → `sent`/`declined`, дальше
     не меняется), `model.NewIssue`;
   - `services/feedback` — чистые правила: `ValidKind`, `CheckText`,
     `DraftID` (`<дата>-<скил>-<slug заголовка>`, slug только латиница и
     цифры), `Compose` (тело + блок контекста, `bug`→`bug`,
     `improvement`→`enhancement`), `Hash` (source + итоговый issue);
   - порты `interfaces.MarkerReader` (`ErrSkillNotManaged`),
     `FeedbackDrafts`, `IssueTracker`;
   - `handler.FeedbackService{Markers, Drafts, Trackers по типу source,
     Now, Version}`: `Draft(ctx, targets, in)` (скил ищется по target по
     порядку, первый с маркером; `local` и тип без трекера — ошибка до
     записи черновика), `Preview`, `Send(ctx, id, hash)` (только
     `draft`, текст непустой, хеш совпадает; сбой трекера — черновик
     остаётся `draft`), `Decline`;
   - infra: `filesystem.Store.ReadMarker` (только настоящая папка с
     обычным файлом-маркером, как `Snapshot`; маркер старого формата —
     «run sync»), `filesystem.FeedbackDrafts{Dir}`
     (`FeedbackDraftsDir = .ai-skills/feedback`, файл `<id>.md`:
     YAML-frontmatter с отступом 2, `# <title>`, тело; занятый id →
     `-2`, `-3`…; id только `[a-z0-9-]`), `tracker.GitHub{Client,
     BaseURL, Token func}` (REST `POST /repos/{o}/{r}/issues`, 201 →
     `html_url`; нет токена — подсказка про `GITHUB_TOKEN`/`gh auth
     login`). Откуда брать токен — шаг 3.
3. ✅ CLI `feedback`: `draft --skill --kind --title (--body |
   --body-file FILE|-)`, `show ID`, `send ID`, `decline ID` (ID — имя,
   файл или путь черновика). Конфиг проходит `Validate`, как у `sync`;
   папка черновиков — `command.FeedbackDraftsDir` от `req.Base`. `send`
   без терминала (`App.IsTerminal`: stdin — char device) — код 1 и
   подсказка пользователю; в терминале — итоговый issue и `[y/N]`, всё
   кроме `y`/`yes` — «Not sent.», код 0. Токен —
   `tracker.TokenSource`: `GH_TOKEN` → `GITHUB_TOKEN` → `gh auth
   token` (свой токен не храним; SSH-ключ API не принимает); 401/403/404
   и отсутствие токена отсылают к `docs/feedback.md`. Документация:
   `docs/feedback.md` (шаги, черновики, выбор токена — fine-grained
   только для своих репозиториев, classic `public_repo` для чужих
   публичных; где хранить; Codespaces), раздел в README, `feedback` в
   `docs/api/reference.md`.
4. ✅ `aism mcp` (stdio, `internal/mcpserver`, Go SDK
   `github.com/modelcontextprotocol/go-sdk` v1.8.0): `feedback_draft`,
   `feedback_submit`. Подтверждение — multi round-trip (SEP-2322): первый
   вызов возвращает `InputRequests` с формой (галочка `send`, по
   умолчанию снята) и хешем показанного issue в `RequestState`, повтор
   отправляет только при `accept` **и** `send=true` и только этот хеш
   (правка во время диалога — не отправлено). Серверный `Elicit` на
   ревизии 2026-07-28 запрещён; для старых клиентов SDK сам делает те же
   раунды через него — тесты на обеих ревизиях. Клиент без формы
   (`req.ClientCapabilities()`: нет elicitation или только URL) —
   «Not sent» и `aism feedback send`. Конфиг перечитывается на каждый
   вызов (`Request` → `Validate`); корень — `CLAUDE_PROJECT_DIR`, если
   задан. stdout — только протокол. Документация: раздел MCP в
   `docs/feedback.md` (`.mcp.json`, `claude mcp add`, не ставить
   авто-ответ хуком `Elicitation`).
5. ✅ `aism mcp install|uninstall [-c] [--name ai-skills] [--replace]` —
   только Claude Code: запись в `.mcp.json` рядом с конфигом (корень
   проекта). `--name`/`--replace` — только установке, серверу не
   передаются; `-c` с именем не `ai-skills.yaml` попадает в `args`.
   `command` — имя, под которым запущен, если оно находит этот же файл в
   `PATH` (`App.Self`), иначе абсолютный путь + warning (файл
   коммитится). Прочее содержимое `.mcp.json` сохраняется (ключи
   сортируются — `encoding/json`); та же запись — ничего не делает, другая
   под тем же именем — отказ без `--replace`; невалидный JSON — отказ,
   файл не трогается. `CLAUDE_PROJECT_DIR` подменяет cwd только для
   запуска сервера. Другие клиенты (VS Code, Cursor, Codex) — позже,
   флагом `--client`.

## Идеи оптимизации (не внедрены, ждут решения)

- Запись: внутри временной папки писать файлы сразу (без temp-файла и
  `rename` на файл — папка и так подменяется целиком); не звать
  `MkdirAll` на каждый файл (помнить созданные папки); меньше операций
  через `os.Root` с глубокими путями.
- Проверка ссылок: кэшировать `fs.Stat`/разрешение цели по пути (одни и
  те же цели проверяются много раз); якоря — кэш заголовков по файлу.
- Параллельность: target'ы пишутся независимо — писать параллельно;
  проверку ссылок скилов можно параллелить (каталог — не потокобезопасен,
  нужен lock или предзагрузка).
- Hash в маркере (отложен) — пропуск неизменённых скилов при warm-прогоне.

## Старые пакеты

Не осталось: весь модуль собирается.

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
| `ownsPath` | лежит ли путь в папке скила | `skillContainsPath` |

`Skill.SkillDirPath` — конкретное, оставить.
