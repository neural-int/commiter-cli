## 要約
Iteration 13で残る3自然table行のcolumn guard付き引数source bindingを検証した。取得済み18行のafter assertion source対応は累計18/18。ただし使用済み監査データのsource coverageであり、実装効果・独立意味能力・commit groupingのGOではない。モデルcall0。

## 検証結果
- 新専用worktree/branch: codex/issue-155-column-guard。
- 192a0217の2行とa6b8dcdeの1行について、列数==3のguard、追加引数の宣言/初期値/代入、callの2引数、result!=expectedをAST照合。
- 192a0217の対応するnatural before行ではuint8引数64、afterはstring .@。新規after行はstring .-。a6b8dcdeのbefore/afterはuint8 64。natural before行がない場合は別区分のまま。
- stringは空文字初期化後にrow[2]を代入、uint8はzero初期値後にrow[2][0]を代入する限定形。byteはUTF-8先頭byteとして扱い、Unicode codepointへ変換しない。空stringへのbyte indexは拒否。
- failure時の診断文字列は別の局所変数と同じcolumn guardによる純粋な文字列式だけを受理。resultやcall引数を変更する診断処理、別receiverのErrorfは拒否。
- row/入力/期待値/call/比較/guard/追加引数宣言/代入のbyte span/textを元sourceと照合。版ごとの引数が過去natural auditの実引数と一致。
- Python3tests、Go build/vet、artifact再確認、diff check pass。string/byte初期値、UTF-8先頭byte、誤guard・診断でのresult変更・偽testingの拒否を確認。
- source binding coverage内訳はvalidator5、straight loop6、error guard4、column guard3。すべて過去にsource監査した18after行。モデルcall0、任意repository実行0、依存追加0、production変更0。
- whole-repository typecheck/実行PASS/actual値/実装attribution/独立意味goldは未証明。型/closureに関する限定parserのsource対応を一般的なprogram slicingと主張しない。

## 考察
自然なtable assertionについて、値の実行をせずに入力と期待値、control条件を保持する準備ができた。単一型へ引数を正規化すると前後の実callが変わるため、型とsource式を版ごとに保存する必要がある。

18/18はこの既知小標本へのsource抽出能力であり、モデルに変更目的の判別能力があるという根拠ではない。既存6履歴はsource対応規則の監査に使用済みで、fresh holdoutへ昇格しない。実装が観測条件を変えるかは別の独立評価に残す。

## Next Steps
- 現小標本に対するsource binding拡張を一区切りとし、検証済み根拠を含む入力契約を固定する。fixture固有の新規規則追加を能力GOへ数えない。
- 未使用positive/negative/unknownとcross-boundaryの評価入力を確保し、識別可能性とgold根拠をモデルなしで監査する。使用済み18行は診断対照に保持。
- 独立データ・gold・受入閾値・call/token/time・baselineとnative wireが推論前に固定できた場合のみ残余意味判断candidateを計測する。未達なら計測せず不足を記録する。
- #155の意味能力GOと元#150の全完了条件は未達のまま保持し、B/C・productionへ進めない。
