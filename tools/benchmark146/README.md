# Issue #146 の bounded planning 検証

このハーネスは最大4ファイルの局所 membership 判断から、Go で全体の grouping を決める方式を比較する。通常 CLI の4ファイル上限、planner、model、backend の既定値は変更しない。

## 測定対象と参照 grouping

`controlled` suite は4 / 5 / 8 / 9 / 12 / 16ファイルの固定した合成入力で、理想化した識別子の chain、欠落した edge、弱い path hint を分離する。graph は既存 `relation.BuildGraph()` で作り、expected grouping はモデルへの入力に含めない。定数を同じ方向へ変更する入力を含むため、exact / FM / FS は固定参照との一致であり、通常の変更の意味的精度を推定する標本ではない。

`contract` suite は、期限境界の修正、金額の丸め、zero capacity の修正と、それぞれの対応テストを入力にする。4 / 6 / 12ファイルの参照を推論前に固定する。graph は既存の syntax parser と `relation.Extract()` / `BuildGraph()` で構築し、正解に合わせて edge を供給しない。12ファイルでは2つの intent が各6ファイルに跨がる。

`oracle` は局所判断に参照 grouping を返して window / reconciliation / metadata の契約を確認する。モデルの測定ではなく、backend calls は0、実 token 数は `null` と記録する。`mlx` は保存済みの固定 Gemma を実際に呼び出す。Oracle の成功をモデルの精度として数えない。

## 比較する方式

`graph-only` は候補 edge を覆う window と、残るファイルを覆う window だけを判断する。4ファイル以内なら既存と同じ全件を1つの window で提示する。5ファイル以上では、未処理の edge を ID 順に選び、window 内の未処理 edge を最も多く覆うファイルを追加する。最大4ファイルを守り、重複を許可する。edge を持たない残りのファイルも最大4ファイルずつ提示する。path hint は接続として利用しない。

`bridge` は `graph-only` の結果を統合した後、関係が未確認の group 同士について代表ファイルを最大4つ追加判断する。代表は group 内の最小 file ID とする。未確認の group 対を多く覆う代表を選び、同点では ID 順にする。代表の提示によって group の関係を判断することの意味的な限界も評価対象にする。

局所判断は、通常の `ThreePhaseGenerator` と同じ prompt、path ID、source-first 表示、JSON schema、generation profile、全件・一意割当の検証を再利用する。局所 category の呼び出し直前を捕捉し、その時点までに検証済みの membership だけを取り出す。局所 type / summary は生成しない。

## 統合と停止条件

各 window で同じラベルなら same、違うラベルなら different の制約を作る。same の推移閉包から group を作り、different が同じ group 内に入った場合は矛盾として停止する。同一の file 対に same / different が両方観測された場合も停止する。投票、repair、再試行、graph の soft edge による union は行わない。

group 間に different の観測がなく、same とも決まっていない場合は未確認とする。graph に edge がないことを different とみなさない。未確認が残る `graph-only` は `unresolved` で終了する。`bridge` も window 数、cycle 時間、入力 context を超えた場合は停止し、部分 plan を返さない。

全体の grouping が確定し、全 selected file ID の一意割当を確認した後にのみ Stage 2 / 3 を開始する。最大4ファイルの制約は Stage 1 の membership 判断に適用する。metadata は確定済み group の全ファイルを保持し、最大4 group の packet ごとに実 prompt と output 枠が固定16K context 内に収まるか確認する。group は packet 境界によって統合・分割しない。全 category が成功してから summary を生成し、最後に `planning.Validate()` を実行する。

category / summary の system instruction と schema の prototype は既存 generator の検証済み1ファイル経路から取得し、変更しない。空の host evidence pool は互換性の証明と扱わず、`unresolved` は停止する。packet の duplicate / unknown group ID、未知 field、過長 scope / summary、不正な最終 plan も拒否する。

## 指標の定義

- `exact` はラベル名・group 順・group 内順序を除いた参照 partition との完全一致。
- FM は別の参照 group の file 対を同じ group にした対数。FS は同じ参照 group の file 対を分けた対数。
- `complete_assignment` は全 selected ID の全件・一意割当。正しい目的別 grouping を意味しない。
- `unresolved` は grouping の矛盾、未確認、局所構造違反、予算停止。全体 grouping がない場合、exact / FM / FS は `null` とする。
- `plan_succeeded` / `plan_failure` は metadata と最終 plan の検証結果。grouping の一致や未解決率と別に記録する。
- `backend_calls` は実際の helper 呼び出し数。window ごとの `calls` は phase、wall time、prompt bytes、input/output tokens、生成枠、stop reason を記録する。
- wall time はハーネスの window 構築開始から局所判断、全体統合、metadata、最終検証まで。fixture / graph の事前構築、Go / Swift build、モデル準備、Git の入力収集は含まない。helper は呼び出しごとに起動し、load を含む。

`max-windows=48` と `timeout=10m` は測定を停止できる実験用 guard であり、採用済みの production 予算ではない。現行4ファイル baseline は通常 generator の3呼び出し・120秒をそのまま使用する。採用する規模と予算は測定結果と別途照合する。

## 再実行

契約と fixture をモデルなしで確認する。

```sh
go test ./tools/benchmark146
go run ./tools/benchmark146 -mode oracle -metadata
go run ./tools/benchmark146 -suite contract -mode oracle -metadata
```

実 token 数の取得には計測専用 helper を使用する。既存 helper のコピーに `input.text.tokens.size` と `info.generationTokenCount` の数値だけを加える。native thought、prompt、生成テキストは測定 artifact に保存しない。output tokens は runtime が報告した生成数で、native thought と channel の token を含む。visible JSON の token 数ではない。

```sh
python3 tools/benchmark146/prepare_helper.py mlx-helper /tmp/commiter-issue146-helper-new
# 必要なら既存の .build を APFS clone して、固定済み依存の再取得を避ける。
# clone は上の計測専用ディレクトリ内の .build に作成する。
swift build --package-path /tmp/commiter-issue146-helper-new -c release --disable-automatic-resolution
```

固定モデルを既に保存した cache と、上で build した helper を明示する。モデルは取得しない。

```sh
go run ./tools/benchmark146 -mode mlx -fixture baseline-4 -strategy baseline \
  -helper /tmp/commiter-issue146-helper-new/.build/release/commiter-mlx-helper \
  -cache /path/to/existing/commiter/mlx-models

go run ./tools/benchmark146 -mode mlx -strategy bridge -metadata \
  -helper /tmp/commiter-issue146-helper-new/.build/release/commiter-mlx-helper \
  -cache /path/to/existing/commiter/mlx-models
```

`-suite contract` で別の固定 suite を選ぶ。比較対象を変える場合は新しい fixture / run として記録し、既存の測定結果や参照を上書きしない。
