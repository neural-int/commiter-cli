## 要約

C iteration1〜3の固定候補はNO CANDIDATE。局所solverの責務分離・再現性は確認できたが、semantic品質は改善せず、正式Cの10条件は全範囲で未達または未立証。親Goal完了、C正式GO、D開始の根拠はない。これは現候補の終了判定であり全architectureの不可能性ではない。

## 検証結果

最新#153本文10項目をiteration-4-contract-audit.jsonで個別照合。checkboxは変更しない。iteration1は2ケース両方式exact0、FM24→12/FS8、同一A-only入力でhidden routing/gold変更が可能。iteration2は3ケースdirect exact1/3/FM48に対しcandidate exact0/3/FM49。全score+2、oracle3/3は診断のみ。iteration3は明示purpose診断2/3、独立目的はunresolved。

solverは2〜8unit、全pair、n8は4140partitionを厳密列挙しtie拒否。局所反転再現は確認済みだが、矛盾辺検出・source mapping・実Git入口は正式統合されていない。production/#149品質比較、selected B evidence ablation、scorerのみablation、全rangeコスト/資源/最終holdoutは未実施。

#149 fixtures.go/holdout.goは明示author-defined synthetic。C visible-fixtures.jsonもindependent_synthetic。B fixtures.pyの「real repository evidence」というコメントは固定合成adapter本体を指し、実運用repository由来のprovenanceを証明しない。今回調べた評価入力に実commit由来の未使用gold付きデータは確認できなかった。

一方、実履歴fd042894にはimport診断重複修正＋回帰test（2file）、df4297d8には宣言/ルートtest対応修正（3file）が存在する。これは将来の候補資料であり、独立goldや未使用holdoutとして未認定。commit境界のみからintent正解を決めない。モデル追加callは0。

## 考察

同じscore/objectiveで再計測・閾値調整を続ける根拠はない。C先行探索は入力識別性とscorerの誤りを切り分けたが、Bの不足を解消したわけではない。Dはrepresentation/evidence/partition妥当性と汎用scorer支配の証明を必要とし、oracle成功や単発診断では不足する。

実repository資料の不足は新しいIssue要件ではなく、次候補の評価を合成データだけに依存させないための監査結果。履歴資料は存在するため、直ちに入力不足で全Goalをblockせず、次の最小限のprovenance/意味妥当性検証に進める。

## Next Steps

- 新規C専用worktreeで実履歴fd042894のbefore/afterと回帰assertionを抽出し、変更の目的・source/test対応・再構築を検証する。未使用の実データとして成立するかをモデル実行前に判定するため。
- 複数commitの混合をgoldへ直結せず、相互依存と同一目的の重複を監査する。履歴をそのままsemantic正解として使う誤りを避けるため。
- 固定候補の再試行はせず、入力監査から新たな因果情報または責務境界が得られた場合のみ別候補を事前固定する。B停止・D条件・production4file上限を保持する。
