# Use cases: internal/domain/services/relations

- internal/domain/services/relations
    - Expander.Expand
        - Contract
            - WHEN_Dependency_cycles_terminate_and_policy_is_enforced_THEN_declared_result
                - description: [Dependency cycles terminate and policy is enforced](features/relations.feature)
                - input: relation policy "<policy>"; I expand from "a.skill.md"; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``relation names are "<names>" and error contains "<error>"``
            - WHEN_Errors_across_files_are_collected_THEN_declared_result
                - description: [Errors across files are collected](features/relations.feature)
                - input: relation policy "enabled"; I expand from "a"; значения и таблицы в сценарии
                - output: результат вызова и наблюдаемое состояние
                - expected_result: ``relation names are "a" and error contains "missing-one"; relation names are "a" and error contains "missing-two"``
