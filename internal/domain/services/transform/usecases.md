# Use cases: internal/domain/services/transform

- internal/domain/services/transform
    - Rewrite
        - Contract
            - WHEN_References_are_rewritten_to_Markdown_THEN_declared_result
                - description: [References are rewritten to Markdown](features/transform.feature)
                - input: transformation document; I rewrite links to ".agents/skills/new/SKILL.md"; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``transformed text is``
    - Claude, Codec.Encode
        - Contract
            - WHEN_Claude_moves_custom_metadata_and_normalizes_whenToUse_THEN_declared_result
                - description: [Claude moves custom metadata and normalizes whenToUse](features/transform.feature)
                - input: transformation document; I transform Claude properties; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``transformed properties are; transformed text contains "## Metadata"; transformed text contains "tags:"``
