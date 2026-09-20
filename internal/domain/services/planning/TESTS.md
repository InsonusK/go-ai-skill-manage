# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Managed state determines actions](features/planning.feature) | `Plan, Fingerprint` | ``plan actions are "<actions>"; planned main text is`` | passed |
| [External files are named deterministically](features/planning.feature) | `BuildLayout, Plan` | ``shared files are`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../../docs/testing.md)
