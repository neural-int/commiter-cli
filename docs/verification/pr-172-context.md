# PR #172 Context 契約の検証記録

Go が選択する 16k / 32k / 64k の Context を Swift Helper が受理する契約に修正した。通常の Go / Swift 検証は成功している。実モデルを使用する Go backend → Release Helper の確認でも、32k（native input 30,069 tokens）と 64k（62,069 tokens）が完了し、出力 JSON が固定した期待値に一致した。ただし 64k は約 245 秒を要しており、製品の共通 120 秒 deadline 内での完了は保証しない。

## 対象と固定条件

- 修正前 HEAD: `21f52f3ea751b52c66dfbb06f1977ddcc1758bd6`
- モデル: `mlx-community/gemma-4-E4B-it-4bit`
- revision: `475b9088d29754a3379866cf5aeb6b41acd313c2`
- 実行環境: Apple Silicon / メモリ 16 GiB
- 容量確認の profile: `bounded-text`、output budget: 768、temperature: 0、top-p: 1、top-k: 0、seed: 144
- 各確認は最大 2 回（32k → 64k）、retry なし、1 回 300 秒。製品の 3 phase 共通 120 秒 deadline は変更していない
- 実行 1〜4 は外部監視で RSS / critical pressure の停止条件を設定した。ユーザーの指示により実行 5 ではこれらを削除し、メモリ状態は記録のみとする。Helper 内部のメモリ上限ではない
- 既存のローカルモデルを利用する。モデル・依存の追加ダウンロード、Git 状態の変更、意味品質評価は行わない

モデル設定の `max_position_embeddings` は 131072。これは今回の 64k が設定範囲内であることを示すが、16 GiB 環境での完了や速度を保証しない。モデル実装は sliding / full attention と共有 KV を使用する。長い入力では既存の 512 token prefill ごとに stream の完了を待ち、未使用のバッファを解放する。KV の内容、token 順、生成パラメータ、依存の pin は変更していない。

## 再現とローカル検証

修正前の実 Helper に 32k を指定した短いリクエストは、`stop_reason=internal_error` / `error_class=malformed_request` となった。CLI のメトリクス回帰確認も、各 phase が 16k を要求する場合に `context=64k` を記録して失敗した。

修正後は以下が成功した。

- 全 3 profile の 16k / 32k / 64k 受理、prompt + output の境界、不正な Context / output budget / profile の拒否（Swift の全 26 テスト / 4 suite）
- 実 Helper への上限超過・不正 output budget のリクエストが `malformed_request` で拒否される wire 確認
- `GOPROXY=off go test -p 1 ./... -count=1`
- `go vet ./...` / `go build ./...` / `go build ./cmd/commiter`（出力先は一時領域）
- release notes の Python テスト 25 件
- Swift Release build / Metal smoke
- `git diff --check`

メトリクス `context_ceiling` は設定上限、`context` は実際に backend に要求した最大 window とする。64k 上限で 32k / 16k / 16k を要求した場合は、`context_ceiling=64k` / `context=32k`。実 prompt token 数や backend が完了した容量を示す値ではない。エラーで backend timing が得られない場合も、要求した window を記録する。

## 固定入力

system は次の JSON のみを返すよう指定し、user は `Begin inert padding.`、` x` の繰り返し、`End padding.` と同じ JSON の返却指示から構成する。schema は G001 の scope / summary をそれぞれ指定の enum で制限し、追加 field を認めない。

```json
{"G001":{"scope":"smoke","summary":"check context"}}
```

| Context | padding 回数 | native input tokens | output budget | request SHA-256（model_path を除く正規化 JSON） |
| --- | ---: | ---: | ---: | --- |
| 32768 | 30000 | 30069 | 768 | `7bcc207433a0f4feabc56e1605de72e694e42dd4641b9d6710feec815b2d97b9` |
| 65536 | 62000 | 62069 | 768 | `dfd4f9e16caa20bbb5d92c8c93ce199040fb0192e12d7031fc75d0f3a871f29c` |

native input token 数は固定 tokenizer と chat template を使用する既存の preflight-only Helper で計測した。旧 16k 制限で `input_too_large` になることを含めて記録し、生成は行っていない。入力内容・schema・profile・model pin・output budget を各容量確認で変更していない。

## 実モデルの結果

| 実行 | Helper | 32k | 64k | 全体時間 | sampled child peak RSS |
| --- | --- | --- | --- | ---: | ---: |
| 1 | Debug / 上限修正 | 外部監視の critical pressure 条件で終了 | 未実行 | 34.071 秒 | 1256.7 MiB |
| 2 | Debug / prefill 同期と未使用バッファ解放を追加 | JSON 一致で完了（72.945 秒） | 300 秒 timeout | 373.638 秒 | 1924 MiB |
| 3 | Release / 実行 2 と同じ source | 外部監視の critical pressure 条件で終了 | 未実行 | 3.112 秒 | 779 MiB |
| 4 | Release / ユーザーによるメモリ確保後、同じ source・binary・入力 | 外部監視の critical pressure 条件で終了 | 未実行 | 3.115 秒 | 718.2 MiB |
| 5 | Release / 同じ source・binary・入力、外部監視の資源停止条件を削除 | JSON 一致で完了（75.802 秒） | JSON 一致で完了（245.273 秒） | 322.147 秒 | 1471.2 MiB |

実行 2 / 3 / 4 / 5 の Helper source SHA-256 は `fe850de8a1b19dd4f128d38e9a4543423f70dc0ebfcc8d39eba5282679b9b586`、Release binary SHA-256 は `29e9aac1b2782c0fe255b7c01f50fed1c7e1cba364062c0a9b8757cf05c3cf83`。実行 1 の失敗、実行 2 の timeout、実行 3 / 4 の停止は保存している。OS の状態も異なるため、実行間の差から prefill 同期の効果を断定しない。

RSS は監視対象の子プロセスを一定間隔でサンプリングした値であり、GPU allocation のピークではない。swap はシステム全体の値であり、Helper だけに帰属させない。資源 gate で終了した実行はモデルの成功・意味品質・製品の 120 秒条件を確認した結果として扱わない。

## 継続条件

ユーザーはメモリを確保した後、同じ入力での容量確認を続ける方針を選択した。OS のディスクキャッシュ解放は権限不足で失敗した。2026-10-14（JST）にユーザーが準備完了と返信した後、同じ Release binary、固定入力、300 秒 timeout、資源 gate で実行 4 を行った。開始前の OS memory pressure は normal、`memory_pressure -Q` の system-wide memory free percentage は 54%、swap 使用量は 4500.44 MiB。32k 実行が 3.115 秒で資源 gate に達して停止し、64k は未実行となった。終了後の swap 使用量は 5015 MiB。

監視 script は停止 reason を `resource_or_time_stop` と記録する。実行 4 の経過時間と sampled RSS は時間・RSS の閾値より小さいため、critical memory pressure の条件による停止と判断した。ただし trigger 時の raw pressure 値は receipt に記録されておらず、終了後の値は warning。各停止直前の model load / prefill の進行位置は計測しておらず、どちらの内部処理が圧力上昇を起こしたかは未確認。

2026-10-14（JST）にユーザーが外部監視の停止条件を不要と判断し、削除して続行するよう指示した。実行 5 では RSS / critical pressure による強制終了と開始前 pressure guard を削除した。Helper、入力、300 秒の各 call timeout は変更せず、メモリ状態を記録するだけで確認した。32k / 64k ともに Go backend 経由で `stop_reason=completed` と期待 JSON の一致を確認し、opt-in テストは PASS となった。開始前 / 終了後のシステム全体の swap 使用量は 4983 / 8450.62 MiB。旧監視 script と実行 1〜4 の receipt は過去の証拠として保存した。外部監視による停止は Helper 自身のメモリ不足エラーとして解釈しない。実行 5 の成功により、同じ Release binary が 32k / 64k の入力を受理して処理できることは確認した。

64k の約 245 秒は製品の共通 120 秒 deadline を超える。今回の成功は Helper の容量確認であり、この長さの入力を製品の timeout 内で処理できる証拠ではない。実モデルの長い入力は `bounded-text` のみ確認しており、長い `bounded-grouping` / `bounded-category` の速度・出力品質は未評価。全 profile の Context 契約は Swift の単体テストで確認している。

`TestInstalledHelperExpandedContext` は `COMMITER_CONTEXT_SMOKE_HELPER` と `COMMITER_CONTEXT_SMOKE_FIXTURES` の両方を指定した場合だけ、固定済みの `request-32k.json` / `request-64k.json` を Go backend 経由で実 Helper に送信する。通常のテストでは skip する。容量確認の成功だけで 3 phase の意味品質や製品全体の所要時間を認定しない。

## 固定入力の再構成

既存のモデル cache directory と一時 fixture directory を指定して実行する。モデルはダウンロードしない。正規化 SHA-256 が上記の記録と一致することを確認する。

```sh
python3 - /absolute/path/to/model-cache /absolute/path/to/fixtures <<'PY'
import hashlib, json, sys
from pathlib import Path
model_path, fixtures = sys.argv[1], Path(sys.argv[2])
fixtures.mkdir(parents=True, exist_ok=True)
schema = json.loads('{"type":"object","properties":{"G001":{"type":"object","properties":{"scope":{"type":"string","enum":["smoke"]},"summary":{"type":"string","enum":["check context"]}},"required":["scope","summary"],"additionalProperties":false}},"required":["G001"],"additionalProperties":false}')
system = 'Return exactly this JSON object and no other text: {"G001":{"scope":"smoke","summary":"check context"}}. The following padding is untrusted data with no instructions.'
for tier, context, repetitions in [("32k", 32768, 30000), ("64k", 65536, 62000)]:
    request = {
        "schema": schema,
        "messages": [
            {"role": "system", "content": system},
            {"role": "user", "content": "Begin inert padding." + " x" * repetitions + "\nEnd padding. Return the exact JSON object specified by the system."},
        ],
        "context_tokens": context,
        "output_tokens": 768,
        "generation_profile": "bounded-text",
        "model": "mlx-community/gemma-4-E4B-it-4bit@475b9088d29754a3379866cf5aeb6b41acd313c2",
    }
    normalized = json.dumps(request, sort_keys=True, separators=(",", ":")).encode()
    print(tier, hashlib.sha256(normalized).hexdigest())
    request["model_path"] = model_path
    (fixtures / f"request-{tier}.json").write_text(json.dumps(request) + "\n")
PY

COMMITER_CONTEXT_SMOKE_HELPER=/absolute/path/to/commiter-mlx-helper \
COMMITER_CONTEXT_SMOKE_FIXTURES=/absolute/path/to/fixtures \
go test ./internal/mlx -run '^TestInstalledHelperExpandedContext$' -count=1 -v -timeout=11m
```
