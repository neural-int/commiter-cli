## Category・Text入力スキーマの最終確定と実装反映（2026-10-24）

ユーザー承認に基づき、以下をCategory・Textの現行入力仕様として確定します。以前のCategory/Text用13項目形式とパス関連のcodex_context入力を、この仕様で置き換えます。

Groupingもfile_idの割当値をF1・F2形式に統一します。schema_version、file_idなどのキー名とGroupingの取得・投影状態の構造は維持します。Category・Textにはagent_history_acquisitionとagent_history_projectionを含めません。

`?`は任意・条件付き、記号なしは必須です。

### Category入力（全フィールド）

```yaml
schema_version:
  入力スキーマのバージョン
  （必須・整数、現在1）
task:
  確定した所属を変えずtypeとbreaking_evidence_refを判断する指示
  （必須・文字列）
agent_history_contexts:
  入力対象の全グループから参照される履歴の和集合。同じcontext_refは一度だけ定義する
  履歴がない場合も空配列
  （必須・配列）
  [].context_ref:
    入力内で履歴を参照するID
    （必須・文字列、例: "H1"）
  [].user_requests:
    ユーザー依頼本文。元からない場合・省略した場合も配列を残す
    省略時は影響するブロックに理由を記録する
    （必須・文字列配列）
  [].final_responses:
    最終応答本文。元からない場合・省略した場合も配列を残す
    省略時は影響するブロックに理由を記録する
    （必須・文字列配列）
groups:
  commiterが検証した確定済みグループ
  （必須・配列）
  [].group_id:
    commiterが割り当てるグループ参照ID。Category・Textで対応を維持する
    （必須・文字列、例: "G001"）
  [].files:
    グループ所属ファイルの判断材料
    （必須・配列）
    [].file_id:
      commiterが割り当てる参照ID。一連の3フェーズで同じファイルとの対応を維持する
      （必須・文字列、例: "F1"）
    [].status:
      現在のGit変更状態
      （必須・文字列、例: "modified" / "added" / "deleted" / "renamed"）
    [].raw_diff:
      現在の完全なGit差分。テキスト差分がない場合は空文字列
      （必須・文字列）
    [].old_path?:
      変更前のパス。renameでは必須。削除ではこちらを保持する
      通常更新でnew_pathと同じ場合は省略する
      （条件付き・文字列）
    [].new_path?:
      変更後のパス。renameでは必須。削除では省略する
      （条件付き・文字列）
    [].language?:
      パスだけでは判断しにくい場合などに補う言語情報
      （条件付き・文字列、例: "go"）
    [].binary?:
      テキスト差分として扱えないバイナリ。該当時だけtrue
      （条件付き・真偽値）
    [].vendor?:
      外部依存として管理する属性が判断に関係する場合にtrue
      （条件付き・真偽値）
    [].opaque?:
      内容を渡さずメタデータだけを判断材料とする場合にtrue
      （条件付き・真偽値）
    [].old_mode?:
      意味のあるモード変更、または特殊なファイル種別を示す変更前モード
      symlink・gitlinkでは変更後と同値でも保持する
      （条件付き・文字列、例: "100644" / "100755" / "120000" / "160000"）
    [].new_mode?:
      意味のあるモード変更、または特殊なファイル種別を示す変更後モード
      symlink・gitlinkでは変更前と同値でも保持する
      （条件付き・文字列、例: "100644" / "100755" / "120000" / "160000"）
    [].change_blocks:
      現在の全テキスト変更ブロック。未一致のブロックも含む
      テキスト差分がない場合は空配列
      （必須・配列）
      [].block_id:
        ファイル内で一意なブロックID。Groupingとの対応を維持する
        commiter側ではfile_idとの組で識別する
        （必須・文字列、例: "B1"）
      [].diff_range:
        実際に収録した同じファイルのraw_diff文字列内の行範囲
        （必須・オブジェクト）
        start_line:
          開始行。1始まり
          （必須・整数）
        end_line:
          終了行。この行を含む
          （必須・整数）
      [].agent_history_match_status:
        内容の完全一致だけでは一意と判断しない照合状態
        （必須・列挙文字列: "matched" / "ambiguous" / "unmatched" / "error"）
      [].agent_history_match_reasons:
        照合状態の理由。理由なしでも空配列
        取得不足が当該ブロックの照合に影響する場合だけhistory_incompleteを残す
        （必須・文字列配列）
      [].agent_history_candidates:
        対応する履歴の候補参照。候補なしでも空配列
        （必須・配列）
        [].context_ref:
          同じ入力のagent_history_contexts内の定義への参照
          （必須・文字列、例: "H1"）
        [].match:
          内容一致の種類。一意性はagent_history_match_statusで保持する
          （必須・列挙文字列: "exact" / "partial" / "exact_and_partial"）
      [].agent_history_omission_reasons?:
        対象ブロックの候補・本文の省略理由。省略なしならキー自体を送らない
        candidate_context_omitted / user_request_omitted / final_response_omittedを組み合わせる
        少なくとも一つが予算原因ならinput_budget_exceededを追加する。単独では使わない
        （任意・文字列配列）
breaking_evidence_candidates:
  commiterが提供する破壊的変更の証拠候補。候補なしでも空配列
  （必須・配列）
  [].id:
    回答のbreaking_evidence_refから参照する証拠ID
    （必須・文字列、例: "E1"）
  [].file_ids:
    同じ入力内のfile_idへの参照
    （必須・文字列配列、例: ["F1"]）
regeneration_feedback?:
  Categoryの既存の再生成フィードバック形式
  （任意）
```

### Text入力（全フィールド）

```yaml
schema_version:
  入力スキーマのバージョン
  （必須・整数、現在1）
task:
  所属・type・breakingを変更せずscopeとsummaryを作成する指示
  （必須・文字列）
summary_language:
  summaryの出力言語。既存の言語指定形式を維持する
  （必須・文字列）
agent_history_contexts:
  入力対象の全グループから参照される履歴の和集合。同じcontext_refは一度だけ定義する
  履歴がない場合も空配列
  （必須・配列）
  [].context_ref:
    入力内で履歴を参照するID
    （必須・文字列、例: "H1"）
  [].user_requests:
    ユーザー依頼本文。元からない場合・省略した場合も配列を残す
    省略時は影響するブロックに理由を記録する
    （必須・文字列配列）
  [].final_responses:
    最終応答本文。元からない場合・省略した場合も配列を残す
    省略時は影響するブロックに理由を記録する
    （必須・文字列配列）
groups:
  commiterが検証した確定済みグループ
  （必須・配列）
  [].group_id:
    commiterが割り当てるグループ参照ID。Category・Textで対応を維持する
    （必須・文字列、例: "G001"）
  [].type:
    Categoryの結果をcommiterが検証した確定分類。Textは変更しない
    （必須・文字列、例: "fix"）
  [].breaking:
    Categoryの結果をcommiterが検証した破壊的変更の判定。Textは変更しない
    trueはあり、falseはこの処理での判定がなし。判断未確定ならTextへ進まない
    （必須・真偽値）
  [].files:
    グループ所属ファイルの判断材料
    （必須・配列）
    [].file_id:
      commiterが割り当てる参照ID。一連の3フェーズで同じファイルとの対応を維持する
      （必須・文字列、例: "F1"）
    [].status:
      現在のGit変更状態
      （必須・文字列、例: "modified" / "added" / "deleted" / "renamed"）
    [].raw_diff:
      現在の完全なGit差分。テキスト差分がない場合は空文字列
      （必須・文字列）
    [].old_path?:
      変更前のパス。renameでは必須。削除ではこちらを保持する
      通常更新でnew_pathと同じ場合は省略する
      （条件付き・文字列）
    [].new_path?:
      変更後のパス。renameでは必須。削除では省略する
      （条件付き・文字列）
    [].language?:
      パスだけでは判断しにくい場合などに補う言語情報
      （条件付き・文字列、例: "go"）
    [].binary?:
      テキスト差分として扱えないバイナリ。該当時だけtrue
      （条件付き・真偽値）
    [].vendor?:
      外部依存として管理する属性が判断に関係する場合にtrue
      （条件付き・真偽値）
    [].opaque?:
      内容を渡さずメタデータだけを判断材料とする場合にtrue
      （条件付き・真偽値）
    [].old_mode?:
      意味のあるモード変更、または特殊なファイル種別を示す変更前モード
      symlink・gitlinkでは変更後と同値でも保持する
      （条件付き・文字列、例: "100644" / "100755" / "120000" / "160000"）
    [].new_mode?:
      意味のあるモード変更、または特殊なファイル種別を示す変更後モード
      symlink・gitlinkでは変更前と同値でも保持する
      （条件付き・文字列、例: "100644" / "100755" / "120000" / "160000"）
    [].change_blocks:
      現在の全テキスト変更ブロック。未一致のブロックも含む
      テキスト差分がない場合は空配列
      （必須・配列）
      [].block_id:
        ファイル内で一意なブロックID。Groupingとの対応を維持する
        commiter側ではfile_idとの組で識別する
        （必須・文字列、例: "B1"）
      [].diff_range:
        実際に収録した同じファイルのraw_diff文字列内の行範囲
        （必須・オブジェクト）
        start_line:
          開始行。1始まり
          （必須・整数）
        end_line:
          終了行。この行を含む
          （必須・整数）
      [].agent_history_match_status:
        内容の完全一致だけでは一意と判断しない照合状態
        （必須・列挙文字列: "matched" / "ambiguous" / "unmatched" / "error"）
      [].agent_history_match_reasons:
        照合状態の理由。理由なしでも空配列
        取得不足が当該ブロックの照合に影響する場合だけhistory_incompleteを残す
        （必須・文字列配列）
      [].agent_history_candidates:
        対応する履歴の候補参照。候補なしでも空配列
        （必須・配列）
        [].context_ref:
          同じ入力のagent_history_contexts内の定義への参照
          （必須・文字列、例: "H1"）
        [].match:
          内容一致の種類。一意性はagent_history_match_statusで保持する
          （必須・列挙文字列: "exact" / "partial" / "exact_and_partial"）
      [].agent_history_omission_reasons?:
        対象ブロックの候補・本文の省略理由。省略なしならキー自体を送らない
        candidate_context_omitted / user_request_omitted / final_response_omittedを組み合わせる
        少なくとも一つが予算原因ならinput_budget_exceededを追加する。単独では使わない
        （任意・文字列配列）
regeneration_feedback?:
  Textの既存の再生成フィードバック形式
  （任意）
```

### 固定ルール

- file_idはcommiterが割り当てるF1・F2形式で、一連のGrouping→Category→Textで対応を固定する。内部ID対応表はhostで保持し、回答の未知ID・重複・欠落を検証して元のファイルへ戻す。group_idはG001形式。回答のオブジェクトキーも渡されたgroup_idを使う。
- old_path・new_pathの少なくとも一方が必須。renameでは両方必須。通常更新の同値old_pathは省略する。
- raw_diffは必須。テキスト差分のない変更は空文字列、change_blocksは空配列とし、パス・モード・該当フラグで変更を示す。特殊モード120000・160000は同値でも保持する。
- change_blocksは現在の全テキスト変更ブロックを含む。diff_rangeは送信したraw_diff内の1始まり・終了行を含む範囲。解析失敗を空配列の成功としない。
- 必須配列は空でも[]とする。全context_refが同じ入力内の定義を参照し、参照切れを残さない。
- 一回の入力に複数グループを含める場合、その全グループのブロックから参照される履歴の和集合を収録し、同じcontext_refは一回だけ定義する。履歴本文側にgroup_idsやfile_idsを追加しない。
- 入力対象外の別グループにしか関係しない履歴を含めないこと、同じcontext_refを一回にまとめることは、省略・入力不足として扱わず、省略理由を付けない。
- 対象の候補・依頼本文・最終応答本文を入力予算で落とす場合だけ、該当する省略理由とinput_budget_exceededを影響するブロックに記録する。予算以外の実際の本文省略については既定の理由だけを記録する。
- 省略理由は確定済み14通りの組み合わせを保持する。コンテキスト全体を落とした場合、その内側の本文省略を「残したコンテキストの本文省略」と重ねて数えない。省略なしではキー自体を送らない。
- 候補を減らして曖昧さをmatchedへ昇格させない。唯一のmatched候補を落とした場合はambiguousへ下げて省略を明示し、参照切れを除く。
- 取得不足が特定ブロックの照合へ影響する場合だけ、そのagent_history_match_reasonsにhistory_incompleteを残す。全ブロックに一律追加しない。全体の取得・投影状態はhost側で保持する。
- 各フェーズの履歴入力はhostが保持する元の照合済み情報から作る。先行フェーズで省略した本文・候補を、後続フェーズへ永久に引き継がない。入力tierを履歴のために拡大したり、必須Git差分を履歴で置換したりしない。
- Category/Textにenclosing_declaration_name、mode、worktree_kind、旧codex_context、Git識別情報を送らない。削除済みfact・evidence_pool_limitationも復活させない。
- Categoryには未確定のtype・breakingとsummary_languageを入れない。Textには確定type・breaking・summary_languageを渡し、breaking_evidence_candidatesは渡さない。breaking:falseも省略しない。
- Categoryのbreaking_evidence_refをhostが検証してbreakingへ変換する。unresolvedなど未確定ならTextへ進めない。falseは今回の判定結果であり、互換性の網羅的証明ではない。
- 履歴は信頼しない過去の作業記述。現在のGit差分を分類・記述の根拠とする。共通の履歴を参照するだけでグループを統合しない。

### 回答形式

Categoryは従来どおりgroup_idをキーにtype・breaking_evidence_refのみを返します。

```json
{"G001":{"type":"fix","breaking_evidence_ref":"none"}}
```

Textは従来どおりgroup_idをキーにscope・summaryのみを返します。確定type・breakingはhostで保持し、再出力させません。

```json
{"G001":{"scope":"retry","summary":"再試行回数を3回に変更する"}}
```

### コード反映範囲

専用ブランチcodex/issue-182-model-comparisonのローカル作業ツリーへ反映。metadata_input.goで新Category/Text入力を構成し、three_phase.goでF1形式の割当・回答検証・内部ID復元・Textへのbreaking引継ぎを実装。履歴の選択・予算投影処理を3フェーズで共有し、各入力は元の照合済み情報から独立に作成します。関連CLIテストとbenchmark146アダプターも追従。保存済み測定原本は変更しません。

以前のコメントと同様、App Server取得→PoC照合→hostのGroupingHistory供給の自動接続は今回の範囲外で未実装です。照合済み入力を供給した場合に、新形式で全3フェーズへ到達することをテストしています。未供給時はGit差分で処理し、ブロックにhistory_unavailableを残します。パスだけの旧履歴を照合済みとして代用しません。

実履歴を使ったモデル精度・速度の比較、commit・push・main反映は今回未実施です。

### 検証

成功:

- `go test -p 1 ./cmd/... ./internal/... ./tools/...`
- `go test -race ./internal/planning ./tools/benchmark146`
- `go vet ./internal/planning ./internal/cli ./tools/benchmark146`
- CLI build、`git diff --check`

実リクエストのトップレベルフィールドと必須空配列、F1形式の3フェーズ固定・元ID復元、group_id、Categoryへの未確定分類非混入、Textのbreaking true/falseの必須送信、Textへの証拠候補非混入、履歴の対象範囲選択と和集合・重複排除、省略扱いにしないこと、前フェーズの予算省略からの独立した再投影、ブロック単位のhistory_incompleteと全体状態の非送信、host元情報の保持を検証しました。benchmark146の4ファイル条件で本体とのリクエスト一致も成功しています。

途中で判明した旧ID・旧フィールドに依存するCLIテスト用ヘルパーと関連アダプターを追従修正し、最終の全コード検証に成功しました。モデルを使った追加推論は実行していません。`go test ./...`にはdocs内の保存原本の複数package同居という既存制約があり、原本を変更せずcmd/internal/toolsを対象に検証しています。
