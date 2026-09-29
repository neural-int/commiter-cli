# Issue #142: ranking と verifier の実測結果

## 条件

[事前条件](issue-142-ranking-verifier-preregistered-2026-09-29.md)と benchmark 実装をコミット `586f03e` で固定後、前回の worktree で測定した。新規の合成 fixture 5件はすべて候補2件で、gold 候補あり3件、なし2件だった。想定と異なる候補包含はなかった。

固定 Louvain generator、MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`を使用した。各 fixture の2反復目は候補提示順のみ反転した。ranking と verifier はそれぞれ2048 output tokens、call ごとに2分上限、repair/retry 0とした。gold 候補ありの3 fixture では、1反復目に verifier へ gold 候補を直接渡す対照 call を1件ずつ追加した。新規依存とモデルの取得はなかった。

## 結果

| fixture | gold 候補 | ranking 選択（1回目 / 2回目） | gold 一致 | 選択候補 verifier 完了 | gold 対照 verifier 完了 | wall 合計 |
| --- | :---: | --- | ---: | ---: | ---: | ---: |
| `verify_join_present` | C001 | C001 / C002 | 1/2 | 0/2 | 0/1 | 183.3s |
| `verify_split_present` | C001 | C001 / C001 | 2/2 | 0/2 | 0/1 | 212.1s |
| `verify_join_absent` | なし | C001 / C002 | 0/2 | 0/2 | 対象外 | 142.0s |
| `verify_split_absent` | なし | C001 / C002 | 0/2 | 0/2 | 対象外 | 152.4s |
| `verify_misleading_relation` | C002 | C001 / C002 | 1/2 | 0/2 | 0/1 | 253.0s |

ranking は10/10件で `completed`、有効な候補ID、完全割当を得た。合成 gold との一致は全試行で4/10件、gold 候補ありの6試行で4/6件だった。gold 候補なしの4試行では、選択候補が合成 gold と一致することはない。ranking で選択した10 grouping の pair 指標の合計は false merge 5、false split 5。これは verifier 後の最終出力ではない。

選択候補への verifier 10件はすべて `max_tokens` で未完了だった。うち選択候補が合成 gold と一致した4件と、不一致の6件の双方に `accept` / `reject` の有効応答はなかった。gold 候補を直接渡した対照 verifier 3件もすべて `max_tokens`。verifier の `deadline_exceeded`、`backend_error`、不正 JSON/schema は観測されなかったが、これらは完了応答の妥当性を測る分母を与えない。`reject` は0件であり、`none` への最終判定も0件。最終候補IDまたは `none` が確定した試行は0/10件で、最終 exact grouping や verifier accept/reject accuracy は算出できない。

backend calls は ranking 10、選択候補 verifier 10、gold 対照 verifier 3、計23件。各段階の wall 合計は43.5秒、691.5秒、207.8秒で、総計942.8秒。output token telemetry は全23件で `unavailable`。10行の `(fixture, run)` は一意だった。

## 記録と範囲

[候補生成の記録](issue-142-ranking-verifier-candidates.jsonl)と[実モデル結果](issue-142-ranking-verifier-mlx.jsonl)を保存した。JSONL には数値、stop reason、候補ID、prompt/schema hash のみを残し、生の prompt、response、repository content は保存していない。production planner、Pass 2、通常 CLI、SRS は測定・変更していない。合成 gold と実際の開発者判断との一致は測っていない。

```sh
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe verify -describe -fixture all
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe verify -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <上記 SHA-256 と一致する helper> -fixture all
```
