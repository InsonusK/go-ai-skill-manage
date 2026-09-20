# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Skill formats and structural validation](features/discovery.feature) | `Detector.Discover, Rooted, ValidName` | ``discovered names are "<names>" and discovery error contains "<error>"`` | passed |
| [A flat example belongs to its ancestor skill](features/discovery.feature) | `Detector.Find, Rooted` | ``discovered names are "guide" and discovery error contains ""`` | passed |
| [Canceled discovery stops before reading the tree](features/discovery.feature) | `Detector.Discover` | ``Empty result and context canceled`` | passed |
| [Select resolves subpaths, tags and name override](features/discovery.feature) | `Detector.Select, scanPaths` | ``discovered names are "<names>" and discovery error contains "<error>"`` | passed |
| [A single-file source ignores a configured subpath](features/discovery.feature) | `Detector.Select, scanPaths` | ``discovered names are "a" and discovery error contains ""`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../../docs/testing.md)
