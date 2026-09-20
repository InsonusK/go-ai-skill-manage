# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Round trip preserves properties and body](features/document.feature) | `Codec.Decode, Codec.Encode` | ``the document result is`` | passed |
| [Documents without frontmatter remain text](features/document.feature) | `Codec.Decode, Codec.Encode` | ``the document result is`` | passed |
| [Malformed frontmatter is rejected](features/document.feature) | `Codec.Decode, Codec.Encode` | ``the document error contains "frontmatter"`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../docs/testing.md)
