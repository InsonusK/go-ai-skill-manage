# Use cases: internal/domain/model

- internal/domain/model
    - SourceMap.Put, SourceMap.Get, SourceMap.Repositories
        - Contract
            - WHEN_Registering_and_looking_up_repositories_THEN_declared_result
                - description: [Registering and looking up repositories](features/model.feature)
                - input: a source map; I put repository "a" and repository "b"
                - output: содержимое SourceMap после Put/Get
                - expected_result: ``repositories in order are "a,b"``
            - WHEN_Re-registering_a_source_key_updates_it_without_reordering_THEN_declared_result
                - description: [Re-registering a source key updates it without reordering](features/model.feature)
                - input: a source map; I put repository "a" and repository "b"; I put repository "a" again with root "/root-a2"
                - output: содержимое SourceMap после повторного Put с тем же ID
                - expected_result: ``repositories in order are "a,b"``
