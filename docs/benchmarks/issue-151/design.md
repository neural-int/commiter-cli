# ChangeUnit representation と staging contract

## Representation

最小content unitはfileのline edit。idはfile ID、before/after blob digest、old/new byte spanから決定する。before/after factsはbyte-safe base64、old/new line rangeは0-based半開区間、symbol identityはstandard Go ASTのsource spanとの交差注釈。ASTがparseできない場合は空annotationとし、raw変更を失わない。line-diff opcodeを位置順のline operationsへ分解し、semantic intentやgoldを使わない。

全unit sequenceと両blobへのsource mappingはimmutable。IDの意味は同一source snapshot内の安定性であり、編集後/rebase後の永続identityではない。source snapshot変更時は再抽出する。reconstruct(before, units, selected)は選択unitのafterを採用し、未選択unitのbeforeとunchanged spansを保存する。任意subsetで中間fileを生成する。全unitsでafter byte一致が必須。

## Git operation contract

partial content stageは、現在のindex blobを検証し、source snapshot上の選択unit集合からtarget blobを再構築する。git diff --binaryによりcurrent index→target差分を作り、git apply --cachedしてindex blobがtargetと一致することを確認する。検証prototypeはtemporary repositoryのみで実行。実CLIへの統合は未実施。

create/delete/rename/mode/symlink/gitlink等metadata操作は独立したwhole-file operationとする。metadataとcontentの共存にはvalid path/kind/modeとoperation順序の検証が必要。renameのcontent unitsは論理file IDを保ち、metadata適用前後のpath mappingを持つ。binary/nontext blobはwhole-blob unitへ退避し、binaryを意味的に細分化しない。file deletionは全残存contentが消去されるgroupでのみmetadata deletionを許可する。partial line以下の同一line複数intentはatomic lineとして保持し、表現不能を明示する。

## Safety / boundedness

per file 1MiB / 20,000lines / 256units。上限超過で停止しsilent truncationしない。unit IDは重複不可、source spansは重複不可、old bytes一致、complete assignment、unknown/duplicate IDs、empty/unknown group、metadataの矛盾、source snapshot差し替えをGit mutation前に拒否する。任意subsetの中間stageは意図したpartial operationであり、最終planのcomplete assignmentとは異なる。final commit planでは全対象unitsとwhole operationsをexactly onceへ割り当てる必要がある。

## Evaluation boundaries

#149のgoldはfile単位で、各fileの複数line unitsを同じgold intentへ対応付ければ表現できる。これはgoldを抽出器へ渡すことでも、semantic intentを推定したことでもない。Aのgold feasibilityはrepresentation能力の評価で、FM/FS改善の測定ではない。production4file上限やfile単位のcommit policyの変更はこのbenchmarkから行わない。以下のB/Cで品質と費用を別途検証する。
