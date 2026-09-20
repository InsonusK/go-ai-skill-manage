# Use cases: internal/logging

- internal/logging
    - Init
        - Contract
            - WHEN_Debug_is_opt_in_THEN_declared_result
                - description: [Debug is opt-in](features/logging.feature)
                - input: I log with debug "<debug>"; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``log contains debug "<present>" and info "true"``
