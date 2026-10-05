# Iteration 1 事前登録

2026-10-05。#139/#140のover-merge、#141のhard縮約によるguardrail誤結合と局所判断の誤り波及、#142の候補欠落・非gold選択、#143の暫定4-file基準、#146の整合したFM12とcontext依存矛盾を確認した。

仮説H1: raw diffを1ファイルずつbefore/after/changed_contract/symbolsへ抽出し、全件のprovisional IRからglobal membershipを一度決定する。局所groupを固定しない。goldはmodelへ渡さない。まず実行可能な独立3修正と対応testのcontract-independent-6で、exact/FM/FS/complete/unresolvedと実token/call/wallを測る。#146既使用fixtureは探索用でholdoutではない。

固定条件: 既存Gemma E4B 4bit revision 475b9088d29754a3379866cf5aeb6b41acd313c2、16K、bounded-grouping（native512、出力768）、temperature0/top_p1/top_k0/seed144。モデル取得なし。各callのhelper上限120秒、fixture全体600秒、N<=16でN+1calls以内、retry/repairなし。prompt/生成本文/native thoughtをartifactへ保存しない。IRはRAMのみ。metadata前の探索probeでありplan成功とは扱わない。

最初のprobeがexactでなければ採用評価を拡大せず、H2としてraw-globalを同一fixtureで比較する。情報圧縮の寄与を切り分ける診断であり、既存raw global方式を新規採用候補として再主張しない。H1が有望なら4/12file、独立holdout、ordering、bounded raw verificationおよびmetadata/Validate接続へ進む。
