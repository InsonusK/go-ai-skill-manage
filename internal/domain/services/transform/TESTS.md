# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [References are rewritten to Markdown](features/transform.feature) | `Rewrite` | ``transformed text is`` | passed |
| [Claude moves custom metadata and normalizes whenToUse](features/transform.feature) | `Claude, Codec.Encode` | ``transformed properties are; transformed text contains "## Metadata"; transformed text contains "tags:"`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../../docs/testing.md)
