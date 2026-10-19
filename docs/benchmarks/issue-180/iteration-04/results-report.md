## 要約

前回改善済みiteration 03のD-Tをbaselineとして、共通prefix KV準備の層別評価を一括評価へ変更した。変更は`for entry in cache { eval(entry.state) }`から`eval(cache)`への1行だけで、②不変grammarデータ保持・③同期/解放頻度変更は行っていない。2ケース・計6要求で入力と最終JSONはすべて前回と一致したが、KV準備は両ケースで長くなり、今回の一巡では速度改善を確認できなかった。性能改善としての通常経路への採用は見送る。

## 検証結果

| 条件 | KV準備 旧→新 秒 | Category＋Text 旧→新 秒 | 全体 旧→新 秒 | MLX peak 旧→新 GiB |
| --- | ---: | ---: | ---: | ---: |
| C01-D-T | 4.69 → 7.59 | 44.97 → 193.18 | 78.86 → 234.09 | 4.80 → 4.80 |
| C02-D-T | 66.47 → 104.13 | 129.34 → 197.85 | 248.48 → 371.92 | 5.20 → 5.09 |

- C01-D-T: 入力要求bytes一致=[True, True, True]、各phaseの最終JSON一致=[True, True, True]、Validatorエラー=''、Text再利用token=2221。
  最終plan: `{"schema_version": 1, "commits": [{"type": "fix", "scope": "AssetHandler", "breaking": false, "summary": "静的アセットの解決処理を導入し、画像表示の404エラーを修正", "file_ids": ["F001", "F002", "F003"]}]}`
- C02-D-T: 入力要求bytes一致=[True, True, True]、各phaseの最終JSON一致=[True, True, True]、Validatorエラー=''、Text再利用token=17971。
  最終plan: `{"schema_version": 1, "commits": [{"type": "feat", "scope": "gitstate", "breaking": false, "summary": "Issue #167: ファイルごとの所有権検証を実装", "file_ids": ["F001", "F002", "F003", "F004"]}]}`

- 前回baselineを再推論せず、その保存済み結果を比較に使用した。C01-D-T→C02-D-Tを各1回、実要求数6、上限6要求。timeoutなし、retry/repair/追加judgeなし。
- 通常経路の未commit採用コードは所在不明のため、保存済みiteration 03の検証経路を使用。source/runner/helper/Metalとfixtureを推論前にhash固定し、完了後も不変を確認した。前回と同じMetalライブラリを実行に使用した。
- 固定モデル/revision、prompt、Gold、seed144、temperature0、thinking予算512/384/384、生成予算、chunk512、同期とメモリ解放、KV保存/copy方法を維持。共通prefixをC01=2221、C02=17971tokenに維持。
- Go全test/vet/build成功。Swift buildは初回Metal Toolchain不足で失敗し、ユーザー承認によるMetal Toolchain 27A266a導入後に成功。Swift既存37テスト（protocol10＋bounded generation27）が成功し、結果をswift-tests.logに保存。git diff --check成功。
- 初回失敗log・初回報告・導入環境情報も保持した。実モデル実行前の候補helperにMetalの加算smokeを実施し、成功を確認した。

## 考察

各条件1回、baselineは過去測定で、同時paired比較ではない。SDKや開発環境・実行時負荷の差も統制していない。全体時間の差をそのまま一括評価の因果効果としない。特にGroupingのsourceに処置はないため、Groupingの時間変動は本変更の効果から区別する。

MLX peakはallocatorのcycle内最大であり、OS RSSやシステム全体のメモリ上限を証明しない。実GroupingはC01が参照と同じ1group、C02は参照1groupに対して3groupsだった。記録・妥当性確認後に固定参照グループを注入し、後段を比較した。最終Validator受理を通常の一貫実行の成功や意味品質の改善とは扱わない。出力一致は本fixtureの観測であり、一般的な同一性や意味品質の保証ではない。旧D-Tに残る意味品質課題もそのまま残る。

## Next Steps

事前固定した一巡で停止する。結果に合わせたprompt/Gold変更、追加生成、retry/repairによる救済は行わない。通常経路への接続・採用と②③の検証は別作業とし、今回の測定のみで自動的に進めない。コードと全raw/失敗/監査/報告は専用worktreeに保存した。

commit/push・PR作成・Issue更新は未実施。初回のMetal不足と承認後の解消を含め、証跡はローカルに保存している。
