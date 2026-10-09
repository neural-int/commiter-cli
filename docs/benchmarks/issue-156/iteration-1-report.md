## 要約

Stage Aは**unknownからfile fallbackを返す構造能力に限定してGO**。24/24ケース・56file・145CUで完全割当、byte再構築、temp-index正逆順stage後の最終tree一致を確認した。ただし意味診断18件はexact0/18、FM211、FS20であり、Aを意味品質GOまたは80%達成と扱わない。mixed-intentの自動検出は未実装で、全fileの目的をunknown、mixed riskをunresolvedとして明記する。

## 検証結果

- #151固定revision c5acdcb0f9f5fa87d47f070146902333f75e165aのextract/reconstructを変更せず複製。fixtureのmanifest hashを確認した。
- 既存18診断と新規6構造ケースを測定。完全割当/再構築/stagingは24/24。764file-stage手順を一時repoの独立indexに適用した。16file32CUの正逆順replayは最終tree一致、測定wall52.848秒。これは検証器の時間で、LLMまたはruntime planner latencyではない。
- 18使用済み診断: file fallback / naive file singletonはexact0/18、FM211、FS20、24commit。naive CU singletonはexact8/18、FM0、FS216、84commit。goldは評価器のみが参照し、planner入力へ渡していない。
- semantic unknown継続は24/24、semantic停止対照のcoverage0に対しfallbackの構造coverage24/24。独立holdoutではなく、実ユーザー分布のcoverageではない。
- 現行authoritative planning.Validateはfile fallback24件すべて受理。compose候補はinvalid_typeで拒否。allowedTypesにcomposeはない。commitlint設定はsource検索で確認できなかった。releaseはPRのRelease category/Breaking change等を読む経路であり、任意type追加がrelease互換性を保証するわけではない。
- 本候補はchore(changes)の目的未確定summaryと別metadataのpurpose_status=unknown/mixed_intent_risk=unresolvedを用いる。混在がないという断定・架空目的・一律composeは行わない。既存production Plan schemaには追加metadataを挿入せず、研究成果物側に保存する。
- 欠落/重複/未知unit、source mapping改変、選択path逸脱、unsupported mode、17file、257atom、4097byte inlineの停止、UTF8行内partial replayを検証する3テストはPASS。新依存0、LLM call/token0、Skill不使用、外部repoコードの実行0。
- 初回validatorビルドはSensitiveValuesへのnil引数の型不一致で失敗。空structへ修正後にビルド成功。ビルド失敗から続いていた重複評価を中断し、完了した一方の測定だけを保存した。失敗や中断を成功件数へ含めない。

## 考察

file convergenceはCU-singletonの過剰FSを抑制するが、mixed-fileをファイル単位で保持すると同一file内の異目的pairをmergeするためFMが増える。metadataでunknown/mixed riskを示すことは意味的な境界修正ではない。Aのexact0%をfallback成功率100%へ読み替えない。

unknown metadataは意味を捏造しないための最小表現であり、実際の単一/複数目的分類能力は未成立。singleは1目的の根拠がある場合、mixedは2以上の独立目的と根拠がある場合、unknownは判断不能として扱う契約とする。mixedかつ分離不能の場合のcomposeは研究用の候補に留め、現行pipelineへ渡す計画には互換性のあるtypeと混在を明記する別metadataが必要。現時点ではcomposeの直接production採用はNO-GO。

stage可能な部分stateは中間stateで対象repoのtestsが成功する保証ではない。今回の新規16fileはsynthetic構造検証であり、16file独立意味性能の証拠ではない。regular-fileの内容作成/削除・編集のみが成立範囲で、rename/symlink/submodule/mode-only/競合は本候補の非対応範囲である。

## Next Steps

- Aの構造GOと意味限界を固定し、Bで限定候補rankingとgroup全体の帰属coverageを評価する。file fallbackが失うcross-file同一目的を回収する必要があるため。
- Bの入力に未使用repository/作者の公開履歴を事前固定する。既知18診断だけの改善を独立GOと誤認しないため。
- 判定・閾値・decoder・予算・gold・入力hashを推論前に保存する。評価結果に合わせた調整を防ぐため。
- confirmed mixedの分類・metadata能力はB/Cの意味評価で監査する。Aに未実装の意味能力を推定しないため。

本iterationは中間段階であり、#156全完了条件と最終品質gateは未達。production・4file上限・既定modelは変更していない。
