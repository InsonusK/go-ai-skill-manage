# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Registering and looking up repositories](features/model.feature) | `SourceMap.Put, SourceMap.Get, SourceMap.Repositories` | ``repositories in order are "a,b"`` | passed |
| [Re-registering a source key updates it without reordering](features/model.feature) | `SourceMap.Put, SourceMap.Get, SourceMap.Repositories` | ``repositories in order are "a,b"`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../docs/testing.md)
