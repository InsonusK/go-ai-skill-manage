# Проверки

## Команды

```sh
make unit-test WITH_CODE_COVERAGE=true
make mutation-test
make test-report
# Все три стадии:
make test-and-report
```

`unit-test` всегда собирает coverage; `WITH_CODE_COVERAGE=true` добавляет
HTML, нормализованную метрику и проверку порога 80%.
`mutation-test` устанавливает gremlins v0.6.0 в `bin/`, если его там нет.
Можно передать `GREMLINS=/path/to/gremlins`.
Для изменений относительно Git ref:
`make mutation-test ONLY_DELTA=true DELTA_BASE=origin/master`.

Текущий baseline: 133 сценария, 133 passed, покрытие production statements 82,7%.
Поле `linePct` в общем формате отчёта содержит именно метрику Go statements.
Бизнес-слой `internal/domain` покрыт на 89,9%.
Актуальные цифры после изменений смотрите в отчёте, а не в этом baseline.

## Где находится проверка конкретного правила

[Индекс возможностей](features/sync.md) ведёт в пакет.
Внутри пакета:

| Файл | Что читать |
| --- | --- |
| `doc.go` / package comment | Ответственность |
| `usecases.md` | Метод → условие → результат |
| `TESTS.md` | Сценарий → реализация → проверяемое утверждение |
| `features/*.feature` | Входные данные, ожидаемый результат, таблицы граничных случаев |
| `test/*_steps_test.go` | Вызов реального кода и сравнение |
| `test/runner_test.go` | Единственный TestFeatures пакета |

Пример: `go test -v ./internal/domain/services/links/test`.
Для вывода зарегистрированных шагов: `GODOG_STEPS=1 go test -v ./internal/command/test`.
Каждый шаг принимает `context.Context`; входы и результаты видны в stdout.

## Изоляция

- Доменные вычисления проверяются через `fstest.MapFS` и небольшие fake-порты.
- Оркестратор проверяет количество вызовов writer/cleanup, включая dry-run и
  ошибку второй цели: первая цель тоже не должна записываться.
- Файловый адаптер работает во временной директории: проверяются реальные
  содержимое, marker, пропуск, замена устаревших файлов, защита личных объектов.
- GitCloner проверяется записью argv; GitProcess — локальной командой Git,
  ошибкой и отменённым context.
- ArchiveFetcher получает tar.gz от `httptest.Server`; проверяются распаковка
  и выход за корень. Fetcher тестирует clone/fallback и cleanup.
- Командные сценарии собирают реальные адаптеры во временном проекте.
- Архитектурный сценарий разбирает импорты Go через AST и запрещает
  инфраструктурные зависимости домена и проверяет `context.Context` первым
  аргументом публичных методов сервисов.
- Тесты не меняют `.agents/skills`, `.claude/skills` и пользовательский YAML.

Отдельные проверки инструментов: `make lint`, `go test -race ./...`,
`make build`. Ручной smoke выполняется на временных проектах.

## Coverage и mutation testing

Coverage scope — `cmd/` и `internal/` без тестовых пакетов;
`tools/` и код сторонних скилов в scope не входят.
Недостающее покрытие включает composition root и часть ветвей I/O ошибок.

Gremlins работает с `--integration`, потому что сценарии находятся в соседних
пакетах `test/`. Без него изменение production package могло бы проверяться
пакетом без тестов. У v0.6.0 `--test-cpu` передаёт Go некорректный аргумент;
этот флаг намеренно не используется. Число workers ограничено четырьмя.

Итог полного прогона: **84.2%**; killed — 416, survived — 57,
timed out — 7, no coverage — 14. Это измеренный результат,
а не утверждение о полном покрытии всех ошибок.

Mutation score считается как `killed / (killed + survived + timedout + noCoverage)`.
Survived — повод проверить недостающее утверждение или эквивалентность мутации;
timeout не считается killed. Нулевые пороги gremlins оставляют этот прогон
отчётным: успешный exit code не означает отсутствие survived.
Нативный отчёт содержит файл, строку и статус каждой мутации.
Известные пробелы: часть возвратов I/O-ошибок и cleanup, альтернативные
неправильные frontmatter/архивы; их нельзя считать проверенными по coverage alone.

## Артефакты

| Путь | Содержимое |
| --- | --- |
| `tmp/result/unit-test.json` | total / passed / failed, без повторного подсчёта godog parents |
| `tmp/result/coverage-test.json` | linePct |
| `tmp/result/mutation-test.json` | killed / survived / timedout / noCoverage / score |
| `tmp/report/tests/` | Go JSON log и читаемая таблица сценариев |
| `tmp/report/coverage/` | coverage.out и HTML с покрытием строк исходника |
| `tmp/report/mutation/` | gremlins.json и HTML summary |
| `public/` | Сайт отчёта, копии native reports и badge JSON |

`test-report` использует нормализованные JSON для метрик.
Он не запускает тесты; при отсутствии предыдущего прогона соответствующих
артефактов не будет. Для полного сайта используйте `test-and-report`.
Отчёты, бинарники и профили исключены из Git.

## Python baseline и совместимость

```sh
PYTHONPATH=deprecated/ai-skill-manager/src .venv/bin/python -m pytest deprecated/ai-skill-manager/src -q
```

Baseline Python: 386 passed. Зависимости Python устанавливаются отдельно
только для сравнений. Методика и подтверждённые отличия:
[совместимость](architecture/compatibility.md).

## Диаграмма модулей

`make diagrams` запускает локальный `tools/diagram-renderer`.
Он читает `depends_on` в индексе возможности, извлекает реальные Go imports
и записывает JSON Canvas и SVG. Внешний одноимённый CLI в окружении отсутствует;
репозиторий содержит собственный воспроизводимый renderer.
