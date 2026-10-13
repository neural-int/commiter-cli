# R6/R7指示だけを加える有限診断

対象: [追加検証コメント](https://github.com/neural-int/commiter-cli/issues/171#issuecomment-6139165124)。人間ユーザーが実行を明示指示した。run3の品質NO-GOと既存記録は保持する。

既知F01〜F12 × 通常/逆順 × A/B = 48新規call、各cell1回、retry0、timeout120秒。48で終了し自動継続しない。入力不足や失敗を採点後に除外しない。F12は既存Goldのundeterminedとして生成前から採点外、確定11fixtureの分母は各22。今回Bは旧run3のRaw Diff arm Bとは異なり、既存入力にR6/R7を追加した処置を指す。

Aのrequestファイルはrun3 Aの全文bytesを再利用。BはAのSystem Prompt内の安全文の直前に指定6行と改行のみを挿入し、user message、schema、ID、model/runtime設定を維持する。全request/hashを生成前に固定する。指定英語段落は `r67_compare.py` のRULEおよび各Bリクエストに全文を保存。AST入力をRaw Diffへ変更しない。

前回出力を見た同じケースを新holdoutとは呼ばない。Aも全24cellを新規再測定するため、環境変動がある場合も旧Aとの違いを別記できる。model/helper/decoderは前回のpinを検証し維持する。#170の意味判断プロンプトへの置換はしない。

スコアは凍結済み `evaluate_process_memory.py` のpartition/scoreを再利用し、Exact/FM/FS/正しいFallback/全件merge・singleton/順序一致を算出する。必要統合の対照F01/F04/F09について両順のExactとFSを確認し、独立目的F02/F03/F08/F10も含む全確定ケースを報告する。旧Aの24partition一致と差分も算出する。FM/FSはペア数、発生cell、unique fixtureを混同しない。

診断上の改善信号は、完全実行・全出力構造適合・資源とGit保持に加えて、BのFallback隔離がAより増え、Exact>=A、FM<=A、FS<=A、順序一致>=A、必要統合F01/F04/F09の両順がExactかつFS0の場合だけとする。これは新規採用Gateではない。未使用fixtureは今回追加せず、採用GOは未評価のままにする。前回D用Gateは再利用・変更しない。

F05/F06/F07の既存入力は推論前監査でinsufficient。改善なしでもR6/R7指示自体が無効とは断定しない。改善しても入力不足、既知自己作成合成ケース、n=1/順、同じモデルの限界を保持する。全singletonへの退避と必要統合の悪化は別記する。

MLX Gemma E4B 4bit revision `475b9088d29754a3379866cf5aeb6b41acd313c2`、seed144、temperature0、top_p1、top_k0、context16384/output768/native thought512、bounded-groupingを固定。helper RSSとMLX peakそれぞれ8GiB上限、timeout・critical pressure・観測欠落・hash/Git driftで停止。全体swapはユーザー指示により停止・合否条件から除外しraw記録だけ残す。プロセスメモリはRSSとMLX allocationを別々に報告する。

既存runner・tokenize-only preflight・validatorを再利用。後続Category/Text、新Planner、追加LLM stage、新依存、Production、既定4file上限は変更しない。旧成果物を上書きしない。途中停止はintent journalとterminal記録を残し、そのrunを再開しない。

実行: `PYTHONDONTWRITEBYTECODE=1 python3 tools/benchmark171/r67_compare.py run --runtime /private/tmp/commiter-171-runtime.json`。評価は同じコマンドのmodeを `evaluate` とする。ローカルmodel pathを持つruntimeファイルはcommitしない。
