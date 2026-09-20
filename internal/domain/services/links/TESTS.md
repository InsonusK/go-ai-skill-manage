# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Markdown and wiki links preserve labels and fragments](features/links.feature) | `Extract, Excluded` | ``references are`` | passed |
| [Inline code and example fences are excluded](features/links.feature) | `Extract, Excluded` | ``references are`` | passed |
| [Folder exclusions apply to documents](features/links.feature) | `Extract, Excluded` | ``references are`` | passed |
| [Raw paths follow repository conventions](features/links.feature) | `Resolve` | ``the resolved path is "<path>" and error contains "<error>"`` | passed |
| [Inline code in a link label excludes the reference for Python compatibility](features/links.feature) | `Extract, Excluded` | ``references are`` | passed |
| [Missing path inside a selected skill follows Python ownership resolution](features/links.feature) | `Resolve` | ``the resolved path is "guide/templates/example.md" and error contains ""`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../../docs/testing.md)
