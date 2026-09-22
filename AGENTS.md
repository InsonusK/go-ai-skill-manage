# Текущая инициатива: SkillCatalog как активный источник скилов

Этот документ фиксирует направление незавершённой переделки пайплайна
`sync`, чтобы не переигрывать уже принятые решения и не терять контекст
между сессиями. Обновлять по мере продвижения по шагам ниже.

## Зачем

`sync.go` сегодня заранее (`s.Sources.GetOrAdd`) грузит все объявленные
источники, а `Detector.Select`/`DiscoverByPath` сразу находит и собирает
все скилы, попавшие под фильтр. Для скилов, которые пакуют внутри себя
целые example-приложения (сотни/тысячи файлов), это означает лишнюю
работу ещё до того, как выяснится, нужен ли конкретный файл вообще.
Цель — сделать всю цепочку "источник → скил → файл" по-настоящему ленивой:
ничего не грузится, пока явно не запрошено.

## Итоговое видение (куда движемся)

- `SkillCatalog` переезжает из `internal/domain/model` в
  `internal/domain/services/sourcing`, получает поле `Manager`.
- Новый метод `SkillCatalog.GetByPath(sourceKey, path)`: просит у `Manager`
  `Repository` для этого `SourceKey` (тот грузит лениво, если ещё не в
  кеше), затем ищет/строит `Skill` по `path` внутри этого `Repository`
  (сегодня это делают `Detector.Rooted`/`makeSkill`, эта логика переезжает
  в `sourcing` вместе с `SkillCatalog` — см. "Принятые решения" ниже).
  `SkillCatalog` кеширует скилы, которые у него запрашивали; `Skill`
  кеширует файлы, которые у него запрашивали (последнее уже сделано —
  см. шаг 1).
- `discovery` перестаёт сам строить `Skill` — он проходит по `Source` и
  для каждого пути-кандидата (по фильтрам subpath/tags) просто просит
  `SkillCatalog.GetByPath(...)`. Чистый обходчик/фильтр, без обратной
  зависимости на `sourcing`.
- Тот же `SkillCatalog` — точка входа, через которую позже Transformers
  смогут получать скилы по ссылкам, которые не были загружены во время
  исходного Discovery (сегодня для этого используется
  `relations.Expander`+`Detector.Find`, отдельный, дублирующий путь).
- `Sync`, возможно, перестанет заранее грузить `Repository` вообще —
  просто передаёт `SkillCatalog` в `Discovery`, а тот уже сам, по мере
  обхода, дёргает `SkillCatalog`, которая лениво дёргает `Manager`.
- **Открытый вопрос, не решён**: где и в каком виде хранить `Skill` после
  того, как его прогнали через Transformers (постобработанный вид). Не
  блокирует текущие шаги, но всплывёт позже.

## План (4 шага)

**После каждого шага — стоп.** Не двигаться дальше и не коммитить самому
без явного разрешения; пользователь проверяет диф и коммитит вручную сам,
затем говорит продолжать.

1. **Доработать работу с файлами в `Skill`/`Skill.Files`.** ✅ **Готово,
   закоммичено** (`62f8b98`). См. "Что сделано (шаг 1)" ниже.
2. **Переделать `SkillCatalog`**: перенос в `sourcing`, добавление
   `Manager`, метод `GetOrAddByPath`. ✅ **Реализовано, ждёт ревью/коммита
   пользователем.** См. "Что сделано (шаг 2)" ниже — включая незапланированный,
   но важный побочный рефакторинг `SourceProvider`/`SourceCache` на
   `SourceKey` и оставленный `TODO` на живую регрессию `SkipFolders`.
3. **Переделать `Discovery`**: убрать построение `Skill` из `discovery`,
   сделать чистым обходчиком поверх `SkillCatalog.GetOrAddByPath`.
   **Обязательно** заодно закрыть `TODO` про `SkipFolders` (см. ниже) —
   иначе `examples`-исключение из `nested-skill` останется сломанным.
   ✅ **Реализовано, ждёт ревью/коммита пользователем.** См. "Что сделано
   (шаг 3)" ниже — включая пересмотр владения `SkillCatalog` (передаётся
   в `Select` параметром, `sync.go` строит и владеет) и перенос
   `SourceSelector`-порта из `interfaces` в `services`.
4. **Переделать `Sync`**: убрать заранее-загрузку `Repository`, передать
   `SkillCatalog` напрямую в `Discovery`. Существенная часть уже сделана
   в шаге 3 (см. ниже) — что реально осталось, см. "Открыто после шага 3"
   в конце этого документа. ⏳ Не начато.

## Что сделано (шаг 1) — важные детали реализации

Всё в `internal/domain/model/skill.go`, **только этот файл и его
test/features** — `discovery`/`catalog.go`/`relations`/`planning`/`sync.go`
сознательно не тронуты (будут меняться в шагах 2–4).

- `Skill.Files []File` и `Skill.FileData(i int)` (индексный, старый способ)
  — **оставлены как есть**, их продолжает заполнять `discovery.Rooted`
  (внешний пакет). Не трогать до шага 3.
- Новые, самодостаточные методы на `Skill`:
  - `FilesByPath(p string) ([]*File, error)` — список файлов скила на и
    ниже skill-relative подпути `p` (`""` = весь скил). Кеш — по каждому
    конкретному `p` отдельно (`map[string][]*File`); **известное
    ограничение**: если попросить пересекающиеся `p` (например `""` и
    `"docs"`), получатся два разных `*File` на один физический файл —
    загрузка данных через один не видна через другой. Приемлемо для
    текущего шага, не решать заранее.
  - `Find(p string, re *regexp.Regexp) ([]*File, error)` — то же самое,
    отфильтрованное по regex над `Path`.
  - Обнаружение `nested-skill` (скил внутри скила) переехало внутрь обхода
    в `FilesByPath` — та же проверка, что раньше была в
    `Detector.Rooted`, но теперь работает только в рамках запрошенного
    `p` (маркер вне текущего скоупа просто не будет увиден этим вызовом).
    Подключение этой проверки к реальному пайплайну (когда именно она
    должна сработать при `sync`) — задача шага 2/3 (`SkillCatalog`,
    предположительно в `GetByPath`), не решено окончательно.
    **⚠️ Пересмотрено в шаге 3** (по замечанию пользователя после ревью):
    `FilesByPath` — **чистый листер**, без какой-либо валидации;
    обнаружение `nested-skill` переехало в `sourcing.nestedSkillPath`
    (скан уже полученного списка файлов), см. "Что сделано (шаг 3)".
    Текст выше — историческая запись состояния на момент коммита `62f8b98`,
    не текущее поведение.
- `File` получил неэкспортируемое поле `skill *Skill` (обратная ссылка).
  - `func (f *File) Content() ([]byte, error)` — **канонический** способ
    получить содержимое файла, кеширует в `f.Data`. Требует, чтобы
    `f.skill` был проставлен (делает это `FilesByPath` для своих файлов).
  - `func (s *Skill) Data(f *File) ([]byte, error)` — **Deprecated**-адаптер:
    если `f.skill == nil` (случай `s.Files`, которые строит
    `discovery.Rooted` и не знает про `skill`), патчит `f.skill = s` на
    лету и делегирует в `f.Content()`. Держится только ради `FileData(i)`.
    **Убрать вместе с `Skill.Data`, когда `discovery` (шаг 3) начнёт сам
    проставлять `skill` при постройке `File`/`MainFile`, либо когда
    `s.Files`/`FileData` будут полностью вытеснены `FilesByPath`/`Content`.**
  - `MainFile.skill` сегодня **не проставлен** (`discovery.makeSkill` вне
    пакета `model`, поле неэкспортируемое) — безопасно, потому что
    `MainFile.Data` всегда уже загружен на момент постройки скила, поэтому
    проверка `if f.Data == nil` в `Content()` никогда не доходит до
    `f.skill`. Но это то же самое ограничение, что и выше — почини заодно
    с `discovery` в шаге 3.
- Тесты: `internal/domain/model/features/skill.feature`, 25 сценариев —
  покрывают старый `FileData`, новый `FilesByPath`/`Find`/`Content`,
  кеширование (без завязки на точное число открытий файла — только "не
  выросло"), скоуп `nested-skill` (внутри/вне запрошенного пути), и
  отдельно — что `Content()`, вызванный через `Find`, виден через
  последующий `FilesByPath` с тем же `p` (это была реальная баг, уже
  починенная: `Find` раньше копировал `File` по значению).

## Что сделано (шаг 2) — важные детали реализации

Новый файл `internal/domain/services/sourcing/catalog.go` +
`features/catalog.feature`/`test/catalog_steps_test.go`. Плюс —
незапланированный, но сделанный по прямому запросу в этой же сессии
рефакторинг портов источников (см. ниже, он расходится с исходным
пунктом плана "`Manager` не трогаем").

- `SkillCatalog{Manager, Codec, Skills, Conflict}` (в `sourcing`, не в
  `model` — `model` не может зависеть от `sourcing`).
  `GetOrAddByPath(ctx, key model.SourceKey, path string, options) ([]*model.Skill, error)`
  — грузит `Repository` через `Manager` (лениво, кеш по `SourceKey`),
  рекурсивно ищет скилы (та же семантика, что у `Detector.DiscoverByPath`
  сегодня), валидирует (`pattern-conflict`/`invalid-name`/`nested-skill`
  — последний ловится вызовом `skill.FilesByPath("")` сразу после
  постройки скила, это же прогревает его файловый кеш), добавляет с
  дедупом по имени (`Conflict`: `error`/`last_wins`).
  `Owner`/`Destination` — без предвычисленного индекса, сканом через
  `OwnsPath`/`RelativePath` (осознанно, см. ниже).
  Своя копия `rooted`/`makeSkill`/`validName` — **не** импортирует
  `discovery` (цикл `discovery` ↔ `sourcing` не создан).
  `model.SkillCatalog` (старый тип в `model/catalog.go`) **не тронут**,
  всё ещё используется `sync.go`/`relations`/`planning` как раньше.
- **Мемоизация по пути**: `bySource map[string]*model.Skill` (ключ —
  уже существующий `model.OriginalKey(repoID, path)`, переиспользован,
  не изобретали вложенную карту), пишется в `remember()` (вызывается из
  `finish()` после успешного `add()`) под ключами `Main` и — для
  directory-скила — `Root`. `scan(p)` в самом начале проверяет `bySource`
  и, если скил уже разрешён по этому пути, возвращает его без единого
  чтения — не только избегает повторного `fs.ReadDir`/`fs.ReadFile`, но и
  повторного `FilesByPath("")`-обхода (который иначе был бы новым
  объектом `*Skill` с пустым кешем, даже для физически того же файла).
  `add()`'s `last_wins`-ветка зовёт `forget()` для заменяемого скила —
  без этого в `bySource` остался бы указатель на скил, которого уже нет
  в `c.Skills`. Тест: "A repeated GetOrAddByPath call for the same path
  reuses the already-resolved skill" (считает открытия файлов до/после
  повторного вызова, ожидает 0 новых).
- **`GetOrAdd` в старом смысле — удалён из дизайна нового `SkillCatalog`
  целиком.** Никакого отдельного "просто резолвнуть, не добавляя" —
  `GetOrAddByPath` всегда добавляет всё найденное и валидное.
  `SkillCatalog` **не знает** про теги/exclude-path — это забота
  `discovery` (шаг 3): он сам, получив `[]*model.Skill` от
  `GetOrAddByPath`, решает, что отсеять.
- **Предвычисленный индекс destinations (`dest`/`index()`/`deindex()`)
  убран.** `Destination` всегда сканит `c.Skills` через
  `OwnsPath`/`RelativePath` — тот же fallback, что и в старом
  `model.SkillCatalog.Destination`, просто теперь единственный путь, не
  fallback. O(n) вместо O(1), осознанный трейд-офф под "не делать
  безусловную работу".
- **`SourceProvider`/`SourceCache` переведены на `model.SourceKey`**
  (было `model.SourceSpec`) — не по исходному плану шага 2, а по
  дополнительному запросу в этой же сессии, т.к. `SourceSpec` для
  ACQUIRE (не для discovery-фильтрации) был не нужен уже ничем, кроме
  `SkipFolders`. Затронуты (за пределами `sourcing`, с явным
  подтверждением пользователя, что это ок):
  `internal/domain/interfaces/source.go`,
  `internal/infrastructure/repository/local.go`, `fetch.go`, `sync.go`
  (вызов `s.Sources.GetOrAdd(ctx, spec.Key(), ...)` вместо `spec`).
  - **⚠️ TODO / известная регрессия, оставлена по решению пользователя,
    закрыть в шаге 3**: `Repository.SkipFolders` теперь **никогда и
    никем не проставляется** — `Local.Acquire`/`Fetcher.Acquire` больше
    не получают `SourceSpec.SkipFolders` (только `SourceKey`). Это не
    "пробел в новом неиспользуемом коде" — это ломает **уже боевой**
    `discovery.Rooted` (`internal/domain/services/discovery/detector.go:149`,
    `for _, skip := range repo.SkipFolders`), который прямо сейчас
    используется реальным `sync` (например чтобы `examples/` со
    вложенным flat-скилом не считался `nested-skill`). Отмечено `TODO`
    в `interfaces/source.go` и `repository/local.go`. **При переделке
    `Discovery` в шаге 3 обязательно решить, как `SkipFolders` реально
    доходит до места, где он нужен** (скорее всего — явным параметром в
    `Skill.FilesByPath`/`GetOrAddByPath`, а не через `Repository`).
- Тесты: `sourcing/features/catalog.feature`, 14 новых сценариев (+7
  старых `sourcing.feature` — не задеты). Есть отдельный сценарий,
  явно документирующий вышеописанный пробел ("Known gap -- SkipFolders
  isn't threaded through yet...").

## Что сделано (шаг 3) — важные детали реализации

Прошёл через **два раунда ревью и пересмотра** после первой реализации:
раунд 1 — skipFolders/repo-acquisition/TempDir (Правки 1–3 ниже); раунд 2
— `FilesByPath`/`accept`'s nested-skill responsibility split (Правка 4
ниже, `model/skill.go`'s `FilesByPath`/`Find` **действительно поменялись
второй раз** здесь, вопреки более ранней пометке в "Что сделано (шаг 1)").
Этот раздел описывает **финальное** состояние, не промежуточные.

Изменены: `model/skill.go` (`FilesByPath`/`Find` — оба раунда, см. Правку
4), `sourcing/manager.go` (`NewManager`/`GetOrAdd`), `sourcing/catalog.go`
(`SkipFolders`-поле, `GetOrAddByPath` без `options`/`skipFolders`-
параметров, `normalizePath` — бывший `discovery.scanPaths`, перенесён
сюда, `nestedSkillPath` — новый, см. Правку 4), `discovery/source.go`
(`Select` переписан, `scanPaths` удалён), `sync.go` (`SourceSelector`
переехал сюда, `SyncService`, `Run`), `interfaces/discovery.go` (удалён),
`interfaces/source.go` (`SourceCache` удалён, doc-комментарии поправлены),
`repository/local.go` (doc-комментарий поправлен), `command/
configuration.go` (`request` → экспортирован как `Request`),
`command/sync.go`, `main.go` (+`Codec`, TempDir резолвится до `Manager`),
плюс все связанные тесты. `discovery.Detector`/`DiscoverByPath`/
`Rooted`/`makeSkill`/`Find`/`ValidName` — **не тронуты**, всё ещё
используются `relations.Expander` (сознательно отложено) и собственными
тестами `Find`.

- **Правка 1 — `SkipFolders`: НЕ per-call параметр, а поле `SkillCatalog`,
  константа на сейчас.** Пользователь поправил моё понимание:
  `SkipFolders` — это НЕ "какие директории исключить из поиска" (это было
  моё ошибочное предположение), а "какие директории освобождены от
  проверки на `nested-skill`, но при этом остаются частью скила и
  копируются вместе с ним". Раз это так тесно завязано на то, как скил
  строит свой файловый список — это осталось задачей `SkillCatalog`, не
  `discovery`. `GetOrAddByPath` **не принимает** `skipFolders` параметром
  вообще — `SkillCatalog.SkipFolders []string`, поле, читаемое `accept()`
  при вызове `skill.FilesByPath("", c.SkipFolders)`. Значение — **пока
  константа** (`sourcing.DefaultSkipFolders = []string{"examples"}`,
  пакетная переменная), передаётся при инициализации `SkillCatalog`
  (`sync.go`: `&sourcing.SkillCatalog{..., SkipFolders: sourcing.
  DefaultSkipFolders}`). **Открытый вопрос, не решён**:
  `SourceSpec.SkipFolders` (per-source, из конфига `skip_folder`) сейчас
  **никак не используется** — реальное значение всегда константа. Сделать
  это настраиваемым per-source — отдельная, не запланированная задача.
- **Правка 2 — `Select` не акклайрит `Repository` сам, `scanPaths`
  переехал в `sourcing`.** Раньше `Select` дёргал `catalog.Manager.
  GetOrAdd(...)` отдельно (чтобы получить `repo` для `scanPaths`'
  `SingleFile`/`Root`-логики), затем сам резолвил список путей. Теперь
  `Select` **не знает про `Repository` вообще** — просто идёт по
  `spec.Subpaths` (или `["."]` по умолчанию) и на каждый путь зовёт
  `catalog.GetOrAddByPath(ctx, spec.Key(), p)`. Вся нормализация пути
  (single-file override, абсолютный путь → repo-relative, `unsafe
  subpath`-проверка — бывшая `discovery.scanPaths`, дословно) переехала
  внутрь `GetOrAddByPath` как приватная `normalizePath(repo, start)`,
  вызывается сразу после `c.Manager.GetOrAdd`, поскольку `repo` там уже
  есть. `discovery/source.go` лишился импортов `io/fs`/`path/filepath`
  (`scanPaths`-функция удалена целиком).
- **Правка 3 — `Manager` держит `TempDir`, не принимает его per-call.**
  `AcquisitionOptions` (сейчас — только `TempDir`) вместо того чтобы
  таскаться параметром через `GetOrAddByPath`/`Select`/`Run`, теперь
  живёт только в `Manager`: `NewManager(providers, tempDir string) *Manager`,
  `Manager.GetOrAdd(ctx, key)` (без `options`) сам строит
  `model.AcquisitionOptions{TempDir: m.tempDir}` при вызове
  `provider.Acquire`. `interfaces.SourceProvider.Acquire` — не менялся
  (всё ещё принимает `AcquisitionOptions`, только теперь единственный
  вызывающий — сам `Manager`).
  **Побочный эффект, вызвавший реальный рефакторинг `command`-пакета**:
  `main.go` строит `Manager` **до** того, как известен `req.TempDir`
  (тот резолвится из конфига только внутри `App.Execute`). Решение
  (подтверждено пользователем — "Move config resolution earlier"):
  `App.request` экспортирован в `App.Request` (`command/configuration.go`);
  `main.go` теперь сам зовёт `command.App{...}.Request(opts, cwd)` **до**
  постройки `Manager`, чтобы узнать `TempDir` заранее — `App.Execute`
  всё ещё резолвит `req` самостоятельно внутри себя (второй, избыточный,
  но дешёвый парсинг конфига) и её собственный контракт/сигнатура не
  поменялись. Ошибка из этого предварительного вызова в `main.go`
  **молча игнорируется** — `Execute` увидит и корректно отрепортит её
  сама, когда до неё дойдёт очередь.
- **Правка 4 (раунд 2 ревью) — `FilesByPath` перестаёт сам решать, что
  такое `nested-skill`.** Пользователь указал на дублирование
  ответственности: `FilesByPath`'s собственный `fs.WalkDir`-колбэк сам
  прерывался с ошибкой `"nested-skill: ..."`, как только находил маркер
  — то есть *единственное* место, которое реально ходит по файлам и
  решает, что невалидно, было `FilesByPath`, а `accept()` просто
  перекладывал эту ошибку в `Issue{Code: "nested-skill"}` — работающая,
  но нечистая связка (любая ошибка `FilesByPath`, включая настоящий I/O-
  сбой, слепо помечалась как `nested-skill`). Решение, подтверждённое
  пользователем: `Skill.FilesByPath`/`Find` (`model/skill.go`) —
  **чистые листеры**, без какой-либо валидации внутри обхода; сигнатура
  лишилась параметра `skipFolders` вообще (он ей больше не нужен —
  обход теперь безусловный, включая директории `SkipFolders`, раз они
  всё равно часть скила и копируются). Маркерные файлы (`SKILL.md`/
  `*.skill.md`), найденные при обходе, теперь **просто попадают в
  список файлов**, как любой другой файл — обход их не различает.
  Новая приватная функция `sourcing.nestedSkillPath(files []*model.File,
  skipFolders []string) string` (в `catalog.go`) сканирует уже
  *полученный* список файлов на предмет маркерного имени, чей верхний
  сегмент пути не входит в `skipFolders`, и возвращает его путь (`""`,
  если не нашла) — это единственное место, решающее "это nested-skill".
  `accept()` теперь: зовёт `skill.FilesByPath("")` (без `skipFolders`,
  просто список), реальную ошибку обхода помечает `Code: "source-read"`
  (не `"nested-skill"` — эти два случая больше не путаются), затем зовёт
  `nestedSkillPath(files, c.SkipFolders)` и только её находку помечает
  `Code: "nested-skill"`. Побочный эффект: маркер внутри `SkipFolders`
  теперь и физически остаётся в `skill.Files` (раньше обход его вообще
  не добавлял в список — прерывался раньше; теперь список полный, а
  фильтрация — только для решения "ошибка или нет", не для того, что
  возвращается).
  Тесты: `model/features/skill.feature` — сценарий "A nested skill
  marker inside the scoped subtree fails FilesByPath" переписан в
  позитивный ("...lists a nested skill marker file like any other
  file"), сценарий про `skip folders are "examples"` **удалён из
  `model`-фичи целиком** (это больше не концерн `FilesByPath`, его
  покрывает `sourcing/features/catalog.feature`'s "SkipFolders exempts a
  folder from nested-skill", не тронут).
- **Реальный баг, найденный на `go test ./...` (не в дизайне плана,
  актуален и после всех правок)**: `planning.Plan` (нетронутый, вне
  скоупа) копирует вложенные файлы скила через старое `skill.Files
  []model.File` + `FileData(i)`, а `sourcing.makeSkill` (в отличие от
  `discovery.makeSkill`/`Rooted`) **никогда не заполнял** `Files`. Раз
  `planning`/`relations` всё ещё читают `Files`, `accept()` дополнительно
  копирует уже прогретый результат `skill.FilesByPath("")` (`[]*File`) в
  `skill.Files` (`[]model.File`, по значению) сразу после успешной
  nested-skill проверки. Это отдельная копия `File`, не тот же `*File`,
  что в `s.scanned[""]` — `Content()`/`Data()` через `Files[i]` не увидит
  кеш, прогретый через `FilesByPath("")`, и наоборот (то же известное,
  документированное в шаге 1 ограничение "overlapping p's get separate
  *File's"). Не проблема для текущих потребителей.
- **Владение `SkillCatalog` — параметр, не поле `Detector`.** `sync.go`
  строит **свежий** `sourcing.SkillCatalog{Manager: s.Sources, Codec:
  s.Codec, Conflict: req.Conflict, SkipFolders: sourcing.
  DefaultSkipFolders}` **на каждый `spec` в цикле** (не один общий на весь
  `Run`) и передаёт его в `Select`. Свежий на каждый `spec` — осознанно,
  чтобы избежать двойного срабатывания дедупа по имени (`GetOrAddByPath`
  сам дедупит по имени внутри catalog, а `sync.go` следом ещё раз дедупит
  через старый `model.SkillCatalog.GetOrAdd`). `Manager`-кеширование
  (дорогая часть) не страдает — `s.Sources` общий на весь `Run`.
  `model.SkillCatalog` (старый тип) **не убран** — нужен `relations`/
  `planning` (обе — не в скоупе этого шага).
- **`SourceSelector`-порт переехал из `interfaces` в `services`.** Его
  сигнатура ссылается на `*sourcing.SkillCatalog`, а `sourcing` уже
  импортирует `interfaces` — обратный импорт зациклил бы. Единственные
  потребители (`sync.go` + тестовый `failingDetector`) уже живут в
  `services`; порт объявлен прямо в `sync.go` (один интерфейс, один
  потребитель, отдельный файл не создавался). `interfaces/discovery.go`
  удалён целиком.
- **`interfaces.SourceCache` тоже удалён** — единственный потребитель
  (`SyncService.Sources`) теперь нужен **конкретным** `*sourcing.Manager`
  (построить `SkillCatalog.Manager`). `SyncService` получил новое поле
  `Codec interfaces.DocumentCodec`.
- Тесты: `discovery/features/discovery.feature` — только шаги `I select
  from subpath`/`I select the single file` переписаны (фейковый
  `interfaces.SourceProvider`, реальный `sourcing.Manager`+`SkillCatalog`,
  `NewManager(..., "")`); `I discover skills at`/`I find the owner of`/
  `discovery is canceled` — **не тронуты**. `sourcing/features/
  catalog.feature`: шаг `skip folders are "..."` теперь пишет
  `catalog.SkipFolders` напрямую (не отдельную переменную, передаваемую
  per-call). `sync_steps_test.go`/`command_steps_test.go`: `NewManager`
  зовётся с уже известным `TempDir` (из `request.TempDir` /
  `App.Request(...)`).

## Открыто после шага 3 (для шага 4 или отдельно)

- **`SourceSpec.SkipFolders` (per-source, из конфига) не используется.**
  Реальное значение, применяемое ко всем источникам, — жёстко закодированная
  константа `sourcing.DefaultSkipFolders`. Сделать её настраиваемой
  per-source (через `SkillCatalog.SkipFolders`, видимо передаваемое из
  `spec.SkipFolders` при постройке catalog в `sync.go`, а не только на
  старте) — отдельная, не запланированная задача.
- **`relations.Expander` всё ещё не тронут** (сознательный выбор
  пользователя: "Defer relations entirely"). Использует `Detector.Find`/
  `Rooted`/`makeSkill` + старый `model.SkillCatalog.GetOrAdd`, полностью
  в стороне от `sourcing.SkillCatalog`/`SkillCatalog.SkipFolders`. Его
  `SkipFolders`-пробел (читает всегда-пустой `Repository.SkipFolders`)
  остаётся — отдельная задача.
- **`Skill.Files`/`Files`-копия в `accept()` — временный мост**, нужный
  только пока `planning.Plan`/`relations.Expand` читают `Skill.Files`
  вместо `FilesByPath`. Кандидат на удаление, когда (если) `planning`/
  `relations` тоже переведут на ленивый доступ.
- **Что реально осталось от шага 4** — владение `SkillCatalog`/передача
  параметром уже сделаны в шаге 3. Не сделано: `sync.go` всё ещё
  параллельно строит и наполняет **старый** `model.SkillCatalog` (нужен
  `relations`/`planning`) — **две** структуры каталога одновременно за
  один `Run`. Уберётся, когда (если) `relations`/`planning` тоже
  переведут на `sourcing.SkillCatalog`. Зависит от того, войдёт ли
  `relations` в объём переделки вообще.

## Принятые решения (чтобы не переигрывать на шаге 3–4)

- **Импорт-цикл `discovery` ↔ `sourcing`**: решено переносом построения
  `Skill` (`Detector.Rooted`/`makeSkill`) в `sourcing` (сделано в шаге 2,
  своя копия, без импорта `discovery`). `discovery` (шаг 3) должен звать
  `SkillCatalog.GetOrAddByPath`, обратной зависимости быть не должно.
- **`SourceKey` vs `SourceSpec`**: решено — `GetOrAddByPath` и весь путь
  acquire (`SourceProvider`/`SourceCache`/`Manager`/`Local`/`Fetcher`)
  работают через `SourceKey`. См. "Что сделано (шаг 2)" выше про
  вытекающий `TODO` на `SkipFolders`.
- **Fetch-кеш vs output-список**: один общий список — `GetOrAddByPath`
  сам добавляет всё найденное и валидное; никакого отдельного "просто
  закешированные, не добавленные" реестра. Фильтрация (теги/exclude) —
  целиком забота `discovery`, после получения списка от `GetOrAddByPath`.
  Следствие: семантику `add_relations=false` ("нашли скил по ссылке, но
  не добавили — ошибка `unselected-skill`") придётся переносить на
  сторону вызывающего в шаге 3/4.

## Общие договорённости на весь ход работы

- Работаем строго по шагам плана выше; каждый шаг — отдельная,
  самодостаточная правка с своими тестами.
- После каждого шага — `go build ./... && go vet ./... && go test ./...`
  зелёные, затем **стоп**, жду ревью и коммита от пользователя вручную,
  не коммичу сам без прямого разрешения.
- Если в процессе шага всплывает решение, которое меняет более раннее
  (например снова цикл импортов, снова вопрос "кто кому принадлежит") —
  сначала обсудить, потом код.
