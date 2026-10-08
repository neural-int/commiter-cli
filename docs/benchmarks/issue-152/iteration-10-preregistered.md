# Metadata-only対照

使用済み7caseの同一input/gold/model/helper/schema/systemで7call追加する。context-onlyからunchanged_sourcesを空にし、observations空と同じsemantics metadata/containerを保持する。新規repository/test情報は追加しない。context16384/output1536/120sec、retry/repairなし。結果でmetadata/prompt/goldを調整しない。

各caseのexact/FM/FS/completeをA-only/context-only/impactと比較する。metadata-onlyでも成功するcaseは、その成功をrepository evidenceの増分へ帰属しない。使用済み診断でありB Goの独立評価ではない。結果と次の仮説を記録し、CへはB gate達成まで進まない。
