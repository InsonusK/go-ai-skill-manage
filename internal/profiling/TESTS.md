# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [CPU profile is written and closed](features/profiling.feature) | `Start` | ``the profile is a readable gzip stream`` | passed |
| [Invalid profile destination reports an error](features/profiling.feature) | `Start` | ``profiling error contains "no such file"`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../docs/testing.md)
