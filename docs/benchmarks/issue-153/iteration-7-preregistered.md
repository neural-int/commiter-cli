## 固定評価

未使用synthetic2件：同file連続行のMaxUploadMiB 2→4とRetryAttempts 3→5は独立仕様（2local groups）、RetryAttempts 3→5単独は1local group。gold名/要件/期待数はmodelに渡さない。実例fd042894の3proposalも評価するがgold未認定でexact/FM/FS=null。

3call再試行なし、既存pinned Qwen3-8B/revision/context16384/output1536/timeout120。新責務はproposal内部のaccept/refine/unresolvedのみで、全体commit groupingは対象外。host validate後の局所境界と拒否・costを記録する。単純2atomの期待group数は局所gold一致を判定できるが一般化/正式C品質ではない。実fixtureのauthor-defined独立目的がruntimeで識別できるかは未立証。
