## 要約

H14としてcausal contrast情報を固定Qwen3-8Bへgrouping-onlyで渡した。weak16/guardrailは一致したがcross12は全12fileを単独groupへ分けFS30。新情報を追加したrouteも資格不成立。

## 検証結果

H13の観測入力/contrast/host gateを固定し、modelだけmlx-community/Qwen3-8B-4bit@545dc4251c05440727734bcd94334791f6ab0192へ変更。追加取得/dependency/production変更なし。

| fixture | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | total wall秒 | observer probes/samples/unknown |
|---|---|---:|---:|---|---|---:|---:|---:|---:|---|
| weak-edges-independent-16 | true | 0 | 0 | true | false | 1 | 2856 | 210 | 30.002 | 0/0/0 |
| contract-cross-boundary-12 | false | 0 | 30 | true | false | 1 | 4942 | 158 | 42.255 | 4/16/0 |
| shared-callee-independent-6 | true | 0 | 0 | true | false | 1 | 2234 | 61 | 17.353 | 2/8/0 |

各1回、全call completed、timeout/context overflow/不完全出力なし。native0/output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0。observer budgetは前iterationと同じ。任意code実行0。資格不成立のためfresh protocol8/metadata追加測定は未実行。

## 考察

8BはH9でもcross12 FS30、今回のH13入力でもFS30。GemmaはH9/H10/H11/H12/H13でcross12 FS24が残った。新情報とcheckpointの組合せからsemantic改善の根拠を得なかった。budgetや形状の失敗ではなく、complete assignmentでも意味的partitionが誤る。

H8以降のglobal outputは、具体的な観測subject/contractまたはE-IDをglobal groupの代表にする。複数のentity contractを変更する共通purposeと、各entityのcontract identityを別々の概念として出力していない。root schemaがgold partitionを数学的に禁止しているという主張ではない。観測のIDを意味的group identityとしてモデルに選ばせるtaskが、entity/test単位の細分化へ誘導する可能性は未検証の仮説として残る。

初期H5の自由G-ID assignmentはforward cross12一致だったが、weak16 FM120とreverse cross12 FS24だった。その構成には、今回の具体的entity記録・causal contrastを用いる責務分割がなかった。したがって既否定の同構成を再試行するのではなく、観測identityとpurpose identityを分離するoutput contractを検証する余地がある。

## Next Steps

- H15として、entity contract記録/E-IDを入力根拠に限定し、global出力は複数entityの変更を含められるpurposeのG-ID assignmentへ分離する。具体的な一つのE-root選択と共通purpose判断を混同していないかを検証するため。
- causal観測/contrast/全fileを維持し、purposeはdiff/codeからのみ判断、未知/重複/欠落ID拒否とcomplete/fail-closed/metadata後段を維持する。latent author intentやsoft evidenceのhard unionを導入しないため。
- 事前登録したfixed Gemma/native0/output1536等のbudgetでweak16/cross12/guardrailを各1回測定する。初期自由assignmentとは異なる観測入力・task責務を評価し、全資格通過時のみ未測定protocol8へ進むため。
