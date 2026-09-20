# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Dependency cycles terminate and policy is enforced](features/relations.feature) | `Expander.Expand` | ``relation names are "<names>" and error contains "<error>"`` | passed |
| [Errors across files are collected](features/relations.feature) | `Expander.Expand` | ``relation names are "a" and error contains "missing-one"; relation names are "a" and error contains "missing-two"`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../../docs/testing.md)
