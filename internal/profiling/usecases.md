# Use cases: internal/profiling

- internal/profiling
    - Start
        - Contract
            - WHEN_CPU_profile_is_written_and_closed_THEN_declared_result
                - description: [CPU profile is written and closed](features/profiling.feature)
                - input: I record a CPU profile; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``the profile is a readable gzip stream``
            - WHEN_Invalid_profile_destination_reports_an_error_THEN_declared_result
                - description: [Invalid profile destination reports an error](features/profiling.feature)
                - input: I start profiling in a missing directory; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``profiling error contains "no such file"``
