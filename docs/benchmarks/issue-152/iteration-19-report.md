## 要約

再代入がある変数について初期定義を確定的に辿る問題を再現し、unknownへ落とすhost契約を修復した。分岐内再代入、通常再代入、incrementの3件で古いValue依存を返さなくなった。B No-Goは維持する。

## 検証結果

修復前3件すべてgot!=8のcalls=[Value]、status=observed_syntactic。修復後すべてcalls=[]、status=partial_unknown。実際の到達定義は解析せず、非定義write/既存変数を含む短縮宣言/increment/address取得があるobjectを不安定として扱う。unknownがあるrootはcall集合を空にして採用不能にする。source/比較位置と旧新結果をrawへ保存した。

修復前に通常/分岐2件の失敗を確認してからコード変更した。追加回帰testは3件のunknownと空call集合を確認。旧単一定義/兄弟引数test、tool build、Python6test、diff check成功。model call0、goldや旧計測値変更なし。

## 考察

初期定義参照はflow-sensitiveな到達定義ではない。今回の保守的unknownは未来のwriteでも観測を抑制するためcoverageを減らし得るが、誤った確定情報をBへ渡すより明示的な不明として保存する。pointer alias、closure、method effect、package globalや完全なscope/type解析は未対応。これを完全なデータフロー安全性と呼ばない。

## Next Steps

- 新規専用worktreeで候補のサポート範囲を単一局所定義に固定し、未使用のsource/test対応と独立診断文、unknown例を共通containerの比較へ入れる。使用済み修復testを一般化の証明にしないため。
- model比較前にunsupportedがunknownになるcoverageを固定し、calls空を無関係の証拠として使わない。absenceからunsafe mergeを生成しないため。
- 改善とregressionを同時測定し、Bの未達条件・production4file制約・C未着手を維持する。
