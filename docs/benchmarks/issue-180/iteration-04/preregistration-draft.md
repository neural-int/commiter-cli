## 比較案（未実行・実行hash未固定）

- baseline: iteration03改善後D-Tの保存済み結果を再利用。baselineの再推論なし。
- 候補: 層別eval(entry.state)をeval(cache)へ置換する1行のみ。
- C01-D-T→C02-D-Tを各1回、最大6生成要求。retry/repair/judgeなし。
- prompt/model/revision/Gold/予算/chunk512/同期・解放頻度/KVコピーを維持。検証のみtimeoutなし。
- helper/validator失敗時は残り停止。出力差は保存し、追加生成で救済しない。
- source/helper/Metal/runner/fixtureを推論前にhash固定する。現在はhelper未buildのため固定未完了。
- 通常経路の未commit採用コードは所在不明。本比較は保存済み検証経路のみ。
