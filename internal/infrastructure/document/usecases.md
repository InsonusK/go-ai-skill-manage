# Use cases: internal/infrastructure/document

- internal/infrastructure/document
    - Codec.Decode, Codec.Encode
        - Contract
            - WHEN_Round_trip_preserves_properties_and_body_THEN_declared_result
                - description: [Round trip preserves properties and body](features/document.feature)
                - input: a document; I round trip the document; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``the document result is``
            - WHEN_Documents_without_frontmatter_remain_text_THEN_declared_result
                - description: [Documents without frontmatter remain text](features/document.feature)
                - input: a document; I round trip the document; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``the document result is``
            - WHEN_Malformed_frontmatter_is_rejected_THEN_declared_result
                - description: [Malformed frontmatter is rejected](features/document.feature)
                - input: a document; I round trip the document; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``the document error contains "frontmatter"``
