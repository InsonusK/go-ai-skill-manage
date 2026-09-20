# Use cases: internal/domain/services/tags

- internal/domain/services/tags
    - Match
        - Contract
            - WHEN_Operator_and_hierarchical_matching_THEN_declared_result
                - description: [Operator and hierarchical matching](features/tags.feature)
                - input: tags "<tags>"; I evaluate "<expression>"; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``the match is "<match>" and error is "<error>"``
