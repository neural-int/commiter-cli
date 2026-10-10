# Issue #167: File-first の Gate 検証

## 要約

Phase 1 の対象は、既存 `gitstate.Collect()` が選択した 1〜16 個の通常ファイルを、ファイルごとの完全な `HEAD → working tree` 変更として保持する構造経路である。`gitstate.PreviewFileFirst()` を追加し、完全所有権、バイナリを含む byte 復元、一時 index による正逆 replay、最終 tree の一致を検証する。

この API は構造検証の結果だけを返す。CommitPlan、metadata、CLI flag、commit 実行への接続は次の Phase の対象である。既存 Three-phase のデフォルト、4-file 上限、validator、確認・検証・commit の契約は変更していない。

## 先行資産の監査と再利用

- [#151](https://github.com/neural-int/commiter-cli/issues/151) の byte 所有権と再構築は、bounded representation の構造能力である。意味判断や production 導入の証明ではない。
- [#157 の A2](https://github.com/neural-int/commiter-cli/issues/157#issuecomment-6072026513) は、coarse 所有権を先に確定し、不要な inline atom 分割による停止を避けた。成果物は `codex/issue-156-adaptive-planner` の `tools/benchmark156/coarse.py` と `planner.py` に保存されている。監査した revision は `82a179a1`。
- [#160 の独立評価](https://github.com/neural-int/commiter-cli/issues/160#issuecomment-6072416177) は exact 4/12、計画 coverage 15/15、production NO-GO。構造 coverage を意味品質へ読み替えない。

今回再利用するのは「refinement 前に全 file の所有権を保持する」「欠落・重複・範囲逸脱は停止する」「独立 target と正逆 replay を照合する」という契約である。Python の研究 pipeline、inline/CU 抽出、目的未確定 `chore` metadata を production package へ複製しない。初期版は file-singleton なので、既存 Go の collector と Git blob/tree を直接使う方が実 CLI のファイル境界と整合する。

## Git 境界と対応範囲

`PreviewFileFirst(snapshot, options)` の `options` は、元の `Collect()` と同じ pathspec/include/exclude/sensitive approval を指定する。元の repository lock を保持したまま、HEAD・branch・index identity・selected Changes・Excluded・Untracked を前後で再照合する。選択に含まれる追加・欠落・ID/hash/path の改変は、部分結果を返さず停止する。

既存 executor は、選択したファイルの全 working-tree 変更を `git add` する。新 API もこの契約に従う。選択ファイルに staged と unstaged の両方がある場合、最終 target は working-tree bytes であり、元の staged blob だけをコミットする新しいモードは導入しない。対象外の staged content と unstaged content は保持する。

| 対象 | Phase 1 の扱い |
|---|---|
| 通常ファイルの追加・変更・削除 | 対応 |
| rename と rename 後の内容変更 | old/new の両 path を一つの所有権に含める |
| 部分ステージ・staged-only・untracked | 既存の file-level 選択契約で対応 |
| バイナリ・non-UTF8 content・CRLF・末尾改行なし・空ファイル | raw bytes を保持 |
| 通常ファイルの実行権限変更 | 100644 / 100755 を保持 |
| 空入力・17+ files | 停止 |
| symlink・gitlink | 停止。status の submodule 再帰を避けるため、index に対象外 gitlink がある repository も停止 |
| 所有 path の重複・file/directory の祖先衝突 | 停止 |
| merge/rebase 等の進行中操作・index lock・snapshot drift | 既存 collector の検査または再照合で停止 |
| selected skip-worktree / assume-unchanged 等 | 停止 |
| clean/process filter・fsmonitor 設定 | 再収集前に停止。対象外ファイルの status でも外部コマンドが実行され得るため repository 全体を拒否 |
| text/eol/working-tree-encoding/ident の変換指定・core.autocrlf=true/input・core.filemode=false | raw bytes と後続 `git add` の差を避けるため停止 |
| 1 MiB/file・20,000 lines/file 超過 | 停止。working-tree 読み込みにも byte 上限を適用 |

元の collector が契約に従って除外した機密候補などは `Excluded` として前後照合する。新経路の unsupported selected change を `Excluded` へ移して継続することはない。

## 構造検証

一時 index と一時 object directory を用い、元の object database は読み取り用 alternate にする。新しい blob/tree は repository の object database に追加しない。実 index・worktree・refs は書き換えない。一時 index 操作で `post-index-change` hook を実行しないよう、一時処理だけに空の hooks path を指定する。実 commit の hook 設定は変更しない。

各ファイルの before/after blob・mode・path から独立した最終 target tree を構成する。正順と逆順でファイル変更を適用し、各段階を binary patch として逆適用・再適用する。各中間 tree、両順の最終 tree、全変更を戻した base tree を照合する。結果の tree ID は監査用の値であり、一時 object directory の削除後に実 Git 操作へ使えるハンドルではない。

受入テストは、1〜16 files の全 file 数、対象外の部分ステージ保持、入力順反転、rename/delete/untracked、binary/mode/空ファイル、失敗時の部分結果禁止を確認する。実 index bytes・worktree bytes/mode・refs・保存済み object の前後一致を判定する。安全性ケースは、drift・不正所有権・上限・変換・外部 Git コマンドを検査する。

検証コマンドと実測結果は `docs/benchmarks/issue-167/phase-1-validation.json` に記録する。研究 fixture の再実験を成果として数えず、今回の Go API の接続境界を検証する。LLM calls/tokens は 0。metadata 品質、semantic exact/FM/FS、paired p50/p95、TTFT、memory pressure/swap は未評価である。

Phase 1 の対応範囲では Gate 1 を満たした。1〜16 files の全16ケースで完全割当・byte/tree・一時 index replay・原状態保持を確認し、混在 Git 状態、入力順反転、replay 中の別プロセスによる変更も検証した。`go test ./... -count=1`、`go vet ./...`、`go build ./...`、release notes の25テスト、差分整合は成功した。これは Issue 全体の最終 Gate、意味品質 GO、production GO を意味しない。

## 考察

最終 tree 一致は、全変更が失われずに stage/replay できる根拠である。中間 commit がビルド可能であることや、file-singleton が意味的に正しい partition であることは証明しない。source と test、複数ディレクトリにまたがる同一目的は過剰分割され得る。同一ファイル内の複数目的も分割されない。

## Next Steps

1. Phase 1 Gate の検証結果を固定した後、既存 `bounded-category` / `bounded-text` の metadata 生成を再利用する。最大16 groups を既存の4-group packetで処理し、LLM に file ID の join/split を許さない。失敗時に架空の目的を持つ fallback を作らない。
2. 同一1〜4-file snapshot、4→5-file 対照、5〜8 / 9〜16 の workload と cold/warm・提示順・反復数・資源上限を推論前に固定する。metadata calls/tokens と構造 replay の時間を分離し、既存 Three-phase と paired 比較する。
3. `planning.Validate()` と前後の snapshot 無変更を確認する。Gate 2 が成立してから明示 opt-in CLI と既存 executor を接続する。Gate 3 の独立品質評価なしに default 化しない。

Phase 1 だけでは Issue 全体の検証は完了しない。以下に Phase 2 の接続・実測と停止判断を記録する。

## Phase 2 — Metadata 接続と事前登録

`planning.FileFirstGenerator` は ID 順の file-singleton を固定し、既存 Three-phase の metadata・host evidence・bounded invocation を共有する。最大4 groups の packet ごとに category/text を生成し、最大16 files / 8 calls / 合計 output budget 5,120 tokens / 共通120秒の上限を持つ。新しい stage・model・decoder・fallback は追加しない。各 packet の validator と最後の全 ID を対象とする `planning.Validate()` を通過した完全 plan だけを返す。

metadata の failure・unknown property・join/split 試行・重複 key・不正 type/scope/summary・unresolved breaking・host witness 無視・sensitive output・context 超過・deadline では停止し、先行 packet を含む部分 plan を返さない。入力順の反転でも ID 順の境界を維持する。既存 default の4-file拒否境界と generation profiles は保持する。

1〜16 の metadata 契約、後続 packet 失敗、既存 Three-phase の回帰テストを確認した。repository 全体の Go tests、vet/build、release notes の25テストは成功した。これは実モデル metadata の成功率・費用・意味品質の証明ではない。

実測の条件は `docs/benchmarks/issue-167/phase-2-preregistration.md`、固定 fixture は `phase-2-fixtures.json` に保存する。`tools/benchmark167` は合成 fixture repository を収集し、同じ Prepared を両 planner に入力する。4→5 の最初の4変更は同じ bytes で、追加の対応テストによる増分と方式差を分ける。model 入力はローカル helper の stdin のみで、測定 artifact に prompt・生成内容・stderr を保存しない。

## Phase 2 — 実測結果と停止判断

事前登録した36 runを完了した。1〜4は同じ snapshot / Prepared の両方式を各3反復、5/8/9/16はFile-first各3反復である。完成plan 0/36、planner停止33、cycle timeout3（8 files全反復）だった。元index/worktree/refs/objectsのdigestは36/36で同一。Gate 2 は未達であり、Phase 3 は gated-out とする。

| files | Three-phase p50 / p95秒 | File-first p50 / p95秒 | 主計測のplan |
|---:|---:|---:|---|
| 1 | 37.91 / 50.67 | 29.21 / 35.76 | 両方式0/3 |
| 2 | 76.63 / 99.18 | 48.24 / 54.18 | 両方式0/3 |
| 3 | 82.12 / 89.88 | 55.79 / 58.11 | 両方式0/3 |
| 4 | 89.48 / 95.16 | 68.24 / 71.13 | 両方式0/3 |
| 5 | 対象外 | 83.59 / 100.69 | 0/3 |
| 8 | 対象外 | 120.14 / 120.20 | timeout 3/3 |
| 9 | 対象外 | 81.90 / 89.31 | 0/3 |
| 16 | 対象外 | 81.39 / 81.95 | 0/3 |

これは失敗までの時間であり、有効planを得る速度ではない。n=3のp95は最大値である。1〜4のcallsはThree-phase3 / File-first2、8fileは3 calls目でdeadline、5/9/16は2 callsで停止した。後続packetへ未到達のrunを、全16 groupsのmetadata費用測定として数えない。call/stage別時間・tokens・runtime TTFT・load・RSS/MLX peak・pressure/swap・diff量は raw と summary に残す。swap増分256 MiB超過は7件だが、共存負荷下の前後差を当該モデルだけの原因と断定しない。

同じ基底の4→5は、反復ごとのFile-first 5−4差のp50 +18.49秒 / p95 +29.55秒。Three-phase4→File-first5はp50 −11.58秒 / p95 +11.21秒だった。後者には方式差が含まれ、前者にも追加した対応テスト・packet内容・時期が含まれる。どちらも成功planの比較ではなく、ファイル数だけの因果効果とは解釈しない。

主計測に詳細failure codesを保存しなかった不足を、設定を変えない各1反復の追加診断8 runで補った。全8 runが `invalid_schema`。typed JSON、欠落/未知group0を確認した上で、以下の文字数超過を観測した。型付きdecodeは重複keyを含む全validator検査の証明ではないが、文字数超過だけで既存metadata規約への不適合が確定する。主計測33件すべての原因をこの8件から遡及確定しない。

| files | Three-phase scope / summary最大文字数 | File-first scope / summary最大文字数 |
|---:|---:|---:|
| 1 | 18 / 42 | 18 / 42 |
| 2 | 7 / 49 | 22 / 46 |
| 3 | 9 / 52 | 10 / 50 |
| 4 | 8 / 54 | 18 / 42 |

scope上限16、summary上限48を緩めない。生成文の切り詰めや架空の目的に置換しない。既存Three-phase metadataには有効な完全metadataを保証する決定的fallbackがなく、この初期接続では失敗時停止を維持する。scopeだけの代替候補があり得ることと、summaryを含む有効な全metadata fallbackが実装・検証済みであることは別である。

追加診断のmetadata前partitionでは、1〜3のThree-phaseは参照一致、4fileは独立変更を全てまとめFM6だった。File-firstは1/4fileで参照一致、2fileでFS1、3fileでFS3。これらは有効metadataを含む最終planの品質ではない。全8合成fixtureのLLMなし構造確認は8/8成功、singletonの静的参照比較はexact3/8・FM0・FS45だった。独立holdoutの精度や#160の値を置き換える結果ではない。

## Phase 3 — Gated-out と未評価範囲

Gate 2 は完成plan・資源・walltime条件を満たさない。明示CLI opt-in、実commit、partial failure、同一ファイルmixed-intent、new API dependency、弱い/ない根拠、実中間state・revert妥当性を含む独立holdoutは実行していない。これは到達不能による未評価であり、成功と記録しない。既存defaultの1〜4 Three-phase、5+拒否、local-only、validatorと確認・Git操作契約を維持する。production GOは出さない。

主計測・診断・構造確認・source/環境の証拠は `docs/benchmarks/issue-167/` に保存する。現行制限を緩めたり、新たなmodel/decoder/stage・意味merge研究へ拡げたりしてGateを通すことは、この検証の成果に含めない。
