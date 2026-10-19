## 要約

追加した6条件×履歴あり2ケースを各1回、12cycle/36生成要求で完了した。Category/Textのthinking設定を揃えるとKV prefix共有が成立し、全D条件でTextのcache再利用を確認できた。日本語指定・最終Validatorは12/12で通過したが、目的や具体性の不足とC02のGrouping不一致が残り、候補全体はproduction採用NO-GOとする。

## 検証結果

採用したText指示は `Write one concrete, single-line summary in Japanese describing the change and its purpose where supported by the input.`。検証専用経路で適用し、通常の製品prompt/timeoutは変更していない。TはCategory/Textの両方thinkingあり（native各384）、Nは両方通常（native0）。Groupingはnative512で維持。Bはphaseごとにload、CはGroupingから保持、Dは保持＋共通prefix KV再利用。timeoutなし、retry/repair/judgeなし。

**全体時間（秒）**

| 条件 | C01：404修正 | C02：一時index構造検証 | 最終Validator |
| --- | ---: | ---: | --- |
| B-T | 113.14 | 293.68 | 両方受理 |
| C-T | 119.10 | 252.89 | 両方受理 |
| D-T | 121.32 | 277.65 | 両方受理 |
| B-N | 75.68 | 219.01 | 両方受理 |
| C-N | 80.44 | 223.32 | 両方受理 |
| D-N | 66.96 | 237.64 | 両方受理 |

C02は全条件で実Groupingが参照1groupに対して3groups。妥当性確認後に参照1groupを注入してCategory/Textを比較した。最終Validator12/12はこの固定グループ経路の受理であり、通常の一貫実行が12/12成功したという意味ではない。固定グループに対するtype/breakingは12/12で参照一致（C01=fix、C02=feat、none）。

**KVの費用を含めたCategory＋Text（秒）とメモリ**

| ケース・mode | C：保持のみ | D：保持＋KV | Dの共通prefix作成秒 | Text再利用token | MLX peak GiB C→D |
| --- | ---: | ---: | ---: | ---: | --- |
| C01-T | 68.64 | 73.72 | 13.39 | 2,221 | 4.80→4.80 |
| C01-N | 22.40 | 22.61 | 12.60 | 2,219 | 4.80→4.80 |
| C02-T | 173.37 | 178.15 | 115.55 | 17,971 | 4.91→5.20 |
| C02-N | 120.86 | 132.42 | 119.83 | 17,969 | 4.91→5.20 |

DのTextではC01=2,221/2,219token、C02=17,971/17,969tokenをT/Nで再利用した。実token列の共通prefixは共通facts末尾まで到達し、D実行時にもprefix一致を検証した。共通prefixだけをCategoryの推論前にcacheへ保存し、独立したcopyをTextへ渡す。固定Gemma4 text modelに限定し、cache各層のoffsetも確認する。Categoryの思考や回答は継承しない。

runtimePromptSecondsは依存runtimeの入力処理＋初回tokenの値で、純prefillの独立計測ではない。Dの共通prefix作成費用はprefixPrefillSecondsへ別記し、wallには両方含めた。MLX peakはallocatorのphase/cycle最大で、OS RSSやシステム全体最大ではない。

**文章品質（変更前から固定した探索的Goldとagentが照合）**

- C01 B-T/C-T: 「静的アセットのオブジェクト参照をURL文字列に正規化し、404エラーを解消」
- C01 B-N/C-N: 「静的アセットのインポートを文字列URLに正規化し、404エラーを解消」
- C01 D-T: 「静的アセットの解決処理を導入し、画像表示の404エラーを修正」―404目的は含むが、URL正規化との関係は曖昧で部分充足。
- C01 D-N: 「静的アセットのインポートを文字列URLに正規化するヘルパーを導入」―404目的が欠落。
- C02 B-T/C-T: 「ファイル単位の変更を検証するPhase1を実装」
- C02 B-N/C-N/D-N: 「ファイルごとの完全な変更を検証するPhase1を実装」
- C02 D-T: 「Issue #167: ファイルごとの所有権検証を実装」
- C02は全て日本語だが、一時indexを用いる構造/replay検証の説明がないため意味条件は部分充足。Textで参照意味を満たす4/12、部分充足8/12。これはagentによる探索的評価で、人手の独立Goldではない。Goldの事後変更は行っていない。

同一mode内のB/C/Dは全3要求がbytes一致。B/Cは全phaseの最終JSONも4組全て一致。C/DはC02-Nのみ全JSON一致し、ほか3組ではTextが異なった。KV利用時の出力同一性・一般的な意味品質の非劣性を証明したとは扱わない。

**検証・実行監査**

- Go全test/vet/buildとSwift protocol10＋bounded generation27testsが成功。初回Swift compileエラーと修正後成功logを保存した。
- 生成前capture準備をGo binary完成前に呼んだ初回失敗も、生成なしの準備エラーとして記録。binary完成後のcaptureは成功。
- 管理scriptの要求数ガードがoriginal-requestと実要求の両方を数え、C02-B-T終了後に誤停止した。実際は21要求。失敗logと元scriptを保持し、原本を除く集計へ修正した別driverで、完了7cycleをskipし残り5cycleのみ続行した。モデル/source/prompt/budgetは変更しておらず、追加・重複生成はない。合計36要求。
- 完了監査で固定175source・helper・Metal・実行binary・overlay・元driverのhash一致、fixture2件のHEAD/index/対象ファイルbytes不変を確認した。実効native予算はcontract最上位のnative_budgetsと事前報告に記載した値。contract内generation.native_budgetsは旧contractから継承した旧値であり、本実行では使っていない。この記載の不整合は原本を改変せず注記する。

## 考察

KV再利用の成立は確認できたが、今回のprefix生成・snapshot実装ではその費用が大きく、Category＋Textの合計はC→Dで4組とも短縮しなかった。C01-Nの全体時間はDが短いが、後段合計は22.40→22.61秒で、差は主としてGrouping側にある。GroupingにDのcache処置はないため、この全体差をKV効果とは呼ばない。

モデル保持は再ロード合計を約5〜6秒減らしたが、全体はB→Cで3組が遅く、C02-Tだけ速かった。C02-Tの大幅差もGrouping変動で説明され、一般的な保持効果の推定値にはならない。TはNよりCategory/Textの時間が長いが、各1回・固定順序であり、ばらつきや一般的な速度/品質保証は評価していない。

明示的なJapanese指定を含む新指示では英語による拒否が12条件で出なかった。ただし要約指示/mode等も変更しているため言語名指定だけの因果効果ではない。C01の4条件は目的を表現できたがD-Nでは再び欠落し、C02の具体性も不足した。grammar/最終Validatorの成功と意味品質は別の結果である。

## Next Steps

- 事前登録の一巡を終了。追加生成・prompt修正・Gold修正による救済は行わない。
- 今回の構成全体は採用を見送る。KV共有の技術的成立と、採用可能な速度・品質の成立を区別して#180を未完了のまま維持する。
- prefix作成/copy費用とKV時の出力差を調べる場合は、新しい有限な検証として条件を固定する。thinkingを揃えれば自動的に高速化・品質維持できるとは扱わない。
- source変更と全raw/失敗/監査/報告は専用worktreeに保存。今回の依頼ではcommit/pushは実施しておらず、ローカル未commit。Issueの報告コメントは公開済み。
