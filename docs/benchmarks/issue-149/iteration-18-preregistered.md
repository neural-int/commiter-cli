# Iteration18事前登録: 副作用なし限定Go AST observer

任意コードを実行せず、AST内の明示対応関数/式だけを評価する。対応はsingle return値・named scalar params・return/if(no init/else)、literal/local params、bool短絡・比較、bounded整数/float算術、string連結、selected packageの一意関数、int/int64/float64変換、math.Floor/Round、strings.ToLower/TrimSpace。整数絶対値2^26以下、string4096bytes以下、step1024/depth16。未知型/参照、mutation/loop/external call/曖昧関数、budget超過はunknown。環境/ファイル/process/network APIや任意Goコード実行は提供しない。Go一般の完全な意味解析とは扱わない。

Iteration17の24実行値との一致を全行確認し、unknown/budget拒否も検証する。新たなGo実行/model推論/holdout試験は行わない。成功した場合のみ、対応範囲とunknownを明記したsoft observational evidenceをglobal入力へ追加する次の試験を事前登録する。値の一致をgrouping成功や全入力の証明と混同しない。
