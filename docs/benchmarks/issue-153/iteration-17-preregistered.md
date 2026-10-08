## 固定全段計測

iteration16のfixture SHAを保持。初回grammar→必要parentのみ詳細grammar→host validation→signed scorer→deterministic partition、fresh最大8call（初回1/詳細最大6/score1）、retry0。model/helper/profile/context/output/8192byte guard/8unit cap固定。gold/eval_intentはmodel payloadへ渡さない。全段結果は逐次保存。各stage入力/出力tokens/wall/rejectを記録。atom pair goldからexact/FM/FSを評価。拒否はnull。production baseline比較は別で今回の結果へ合算しない。
