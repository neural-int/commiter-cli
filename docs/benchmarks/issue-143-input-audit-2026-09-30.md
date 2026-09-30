# #143 入力監査の観測結果（2026-09-30）

指定考察コメントの数値は前回記録と一致した。判断境界が結合側へ動いたという原因説明、過適合・一般化の成否、Phiの内部failure mode、容量不足の支持/否定は測定されていないため、観測パターンの記述に修正した。Gemmaを含む比較を一律に3〜4B級とは記述しない。進行基準はproduction採用基準と区別した。

## 実施した検証

起点は前回結果commit a2a36029c7981c3a41f13cf68f0d3e725b8ac762。
入力監査手順とコードを c078e70 に固定した後、次のコマンドを実行し成功した。

```sh
go run ./tools/benchmark110 -issue143-input-audit > docs/benchmarks/issue-143-input-audit-2026-09-30.json
```

issue143Prepareで再構築した全12例のcontractは前回manifestと一致した。対象holdout8例では16方向分のprompt/schema hashが記録値と一致した。モデル/helperを呼ばないコード経路で実行し、model call 0、受領した人間の独立判断0。

| 観測項目 | 結果 |
| --- | ---: |
| holdout fixture | 8 |
| prompt-visible file | 20 |
| raw_diff mode | 20/20 |
| structural / metadata_only mode | 0 / 0 |
| evidence reduction | 0 |
| 元diffとのraw_diff完全byte一致 | 0/20 |
| 合成hunk headerを除いた差分本文の完全一致 | 20/20 |
| 正逆でsystem/schema/repository_inputが同一 | 8/8 |

元diffの前に `@@ -1,1 +1,2 @@\n` が追加されており、完全byte一致ではない。このheaderのみを除くと全20fileで元diff本文と一致した。diff本文の欠落、構造evidenceへの置換、metadataのみへの縮退は観測されなかった。file IDとpathの対応、元diff SHA-256も照合した。

## 可視情報の具体的な観測

下表は可視コード・docsの記述についての静的観測。goldの独立承認や入力十分性の判定ではない。

| fixture | 可視情報の観測 |
| --- | --- |
| h143_source_test_independent | F001はHideArchivedResultsの追加。F002はlegacySnapshotTitleの文字列変更で、F001の関数への参照はない。source_test構造relationは可視入力にある。 |
| h143_stem_docs_independent | F001はMaxUploadBytesの1024→2048。F002はsupport mailboxの変更。docsに上限値変更の説明はない。 |
| h143_shared_directory_independent | CompressionLevelの1→6とDefaultLocaleのen→ja。差分内に両関数の相互参照はない。 |
| h143_two_features_interleaved | F003はF001のApplyMemberDiscountを直接呼ぶ。F002はHealthStatusがreadyを返す関数で、F004はreadiness endpointと受付可能状態を説明する。可視差分にendpointへの接続処理・受付可能状態の条件はない。 |
| h143_atomic_validation | F001はlen(s)が3〜20かを返す。F002はab/abcを検査。F003は3〜20 charactersと述べる。1 rune・3 bytesの「あ」ではこの条件式がtrueとなることをGoで実測した。 |
| h143_crossdir_documentation | F001はageDays>14を返すpredicate。F002はstorage cleanup jobによる自動削除を述べる。可視コードにjob起動や削除処理はない。 |
| h143_paraphrased_feature | F001はRecoveryLinkExpires(minutes)のminutes>=30。F002はcredential reset URLがhalf an hourで使用不能になると述べる。両者を結び付ける呼出し・参照は可視差分にない。 |
| h143_crosscomponent_feature | F001はDefaultPageSize=25、F002はapi.DefaultPageSizeを直接参照、F003はAPI/web pagesの25件を述べる。 |

handleの観測はfixture条件式を直接評価したもので、production関数のテストではない。次の標準ライブラリだけのGoコードで再現できる。

```go
package main
import ("encoding/json"; "os"; "unicode/utf8")
func main() {
 s := "あ"
 json.NewEncoder(os.Stdout).Encode(map[string]any{
  "byte_length": len(s), "rune_count": utf8.RuneCountInString(s),
  "fixture_valid_handle": len(s)>=3 && len(s)<=20,
 })
}
```

観測値: byte_length=3、rune_count=1、fixture_valid_handle=true。

## 人間確認との境界

人間確認用資料には可視file evidence・relation context・2候補を収録し、gold/rationale/前回モデル結果は入れていない。全文system/user promptや生成responseは保存していない。

AI監査者は元gold/rationale/結果を既読であり、盲検ではない。人間判断は未受領。したがって、8例についてgoldが独立に支持された、入力が十分/不足と確定した、という結果は得ていない。差分本文保持の検証とsemantic十分性の判定は分ける。後続contractの推論比較、入力追加実験、Nemotron互換性診断は実施していない。前回240callの再集計・基準・gold・fixture・モデルpinは変更していない。

## 検証と記録

- go test ./... 成功
- go vet ./... 成功
- 元diff・manifest・file ID/path・20file本文の照合成功
- git diff --check 成功
- 依存追加、model download、production変更なし

資料:
- issue-143-input-audit-protocol-2026-09-30.md
- issue-143-input-audit-2026-09-30.json
- issue-143-input-audit-counts-2026-09-30.json
- issue-143-human-review-packet-2026-09-30.md
- tools/benchmark110/issue143_input_audit.go
