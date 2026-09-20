# Use cases: internal/domain/services/planning

- internal/domain/services/planning
    - Plan, Fingerprint
        - Contract
            - WHEN_Managed_state_determines_actions_THEN_declared_result
                - description: [Managed state determines actions](features/planning.feature)
                - input: planning input with state "<state>" and force "<force>"; I plan a sync; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``plan actions are "<actions>"; planned main text is``
    - BuildLayout, Plan
        - Contract
            - WHEN_External_files_are_named_deterministically_THEN_declared_result
                - description: [External files are named deterministically](features/planning.feature)
                - input: a planning tree; I plan a sync; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``shared files are``
