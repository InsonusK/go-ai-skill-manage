# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Acquiring the same source twice reuses the cached repository](features/sourcing.feature) | `Manager.Acquire` | ``the provider was called "1" times; both acquisitions returned the same repository`` | passed |
| [Acquiring different sources calls the provider for each](features/sourcing.feature) | `Manager.Acquire` | ``the provider was called "2" times`` | passed |
| [Unknown source type is rejected](features/sourcing.feature) | `Manager.Acquire` | ``acquiring fails with "unknown source type"`` | passed |
| [Close runs every acquired repository's Close in reverse order](features/sourcing.feature) | `Manager.Close` | ``repositories were closed in order "b,a"`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../../docs/testing.md)
