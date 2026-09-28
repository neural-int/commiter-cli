# Issue #141: small candidate cluster boundary 検証の事前条件

## 仮説と比較

局所 pair 判定は guardrail の偽 relation 4/4 を分離した一方、主比較で exact 6/12、false merge 4 pair、false split 214 pair、22 calls だった。今回の仮説は、最大4ファイルの候補集合内で分割を許し、その結果できた小さな cluster 間の変更目的を比較すれば、単一 pair の判定から推移的結合を作る方式より grouping が改善する、というもの。前回との比較では prompt、候補生成、判断単位が同時に変わるため、改善時も単一要素の因果効果とは扱わない。

前回と同じ6主 fixture と2 guardrail、同じ正解 grouping、MLX model revision `a962dcb09eee4169c890e544c9eb938f1113fdee`、Release helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`、1024 output tokens/request、2分/試行、2反復を使う。Pass 2 は実行しない。モデルや helper は既存のローカルキャッシュを使い、ダウンロードしない。

## 候補生成と判定

- 4ファイル以下は全ファイルを1つの候補 block に入れる。5ファイル以上は basename から拡張子と末尾の数字を除いた stem ごとに入力 file ID 順で集め、最大4ファイルに分割する。block は provisional であり、Go は結合を確定しない。正解ラベルは候補生成に使わない。
- 2 block ずつ1 request にまとめ、file ID を key とする割当をモデルから受け取る。ラベルは block 内だけ有効とする。Go は raw duplicate key、欠落、未知 ID、不正ラベルを検査し、候補 block 内の分割を保持する。
- 得られた cluster 間の候補 pair は、同じ stem を共有する cluster の隣接 pair、同じ block から分割された cluster の pair、relation edge がまたぐ cluster の pair を和集合として固定する。cluster が2つだけならその pair も候補にする。relation 種別は prompt に渡さず、候補化だけに用いる。
- 最大4 cluster pair を1 request にまとめ、各 cluster に属する全ファイルの path、language、raw diff、summary を提示する。固定 boolean key に対して keep / separate を返す。Go は raw duplicate key、欠落・未知 key、値型を検査し、肯定 pair の connected component を最終 grouping とする。孤立 cluster も全て残す。否定 pair の再結合件数を監査する。
- 失敗時に部分的な割当や推測で補完せず、構造的失敗を記録する。生の prompt、応答、生成 summary は保存しない。

## 評価と判断

候補 block / cluster / pair の件数、候補の正解連結性、候補内部に固定されてしまった false merge、cluster 判定の TP/TN/FP/FN、最終 exact grouping、false merge / false split、全 file ID 完全割当、構造的失敗、calls、wall、prompt/output bytes を測る。output token 数を backend が返さない場合は `unavailable` とする。

主判定は、guardrail と `multi_commit` の false merge を0に抑え、`mixed_24` の104 false split pair/反復を減らし、`holdout_join` の正しい結合を回復できるか。主比較 calls は局所 pair 判定の22より少ないことを期待するが、段数と prompt bytes も併記する。候補連結性が欠ける場合と判定誤りを区別する。小標本と合成ラベル、とくに `mixed_24` の命名規則依存を一般化しない。

この実装は benchmark 専用とし、production planner、SRS、通常 CLI、新しい依存は変更しない。
