# Iteration 02: 共有期限の有限比較

本番の120秒は変更せず、Go build overlayでCandidateCycleTimeoutを360秒にした比較。入力、Gold、モデル、helper、profile、出力予算はIteration 01と同じ。各2ケース×A/B×1回、最大12要求、retry/repair/追加judgeなし。

C02 Groupingが約107秒だった条件で、各phaseへ従来の120秒を個別に与えるのではなく、共有cycle期限を延ばした場合を観測した。長い期限で意味不一致やschema不合格が解決するとは仮定しない。スコアと制約は[結果報告](report.md)に記録する。

実行時のpatch、置換ソース、capture、request/response、ログは公開せず、ローカルまたはCI Artifactsで管理する。再現に必要な条件は[実行契約](execution-contract.json)に記録する。
