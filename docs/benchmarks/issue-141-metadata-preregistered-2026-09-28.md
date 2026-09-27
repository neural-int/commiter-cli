# Issue #141: metadata 呼び出し形態の補助比較

主比較の Pass 1 では、正解の2 group が得られなかった。そこで Pass 2 の1バッチ request と group ごとの request の call 数・時間・validation を比較するため、正解 group を Go 側から固定して入力する**補助試験**を事前に定義する。これは Pass 1 を通過した end-to-end 成功として扱わず、grouping quality の採否判断にも加算しない。

使用する正解 group は既存ラベルの multi_commit、mixed_24、holdout_split（各2 group）と、既存の日本語 fixture japanese（1 group）。各 fixture の batch と per-group を1回ずつ、同じローカル MLX model revision、helper SHA-256、1024 output token/request、plan cycle 2分で実行する。repository input、metadata schema、planning.Validate() は両方式で共通とする。per-group の出力も Go 側で正解 group に結合した1つの plan として検証する。

計測対象は metadata の schema/言語/型/割当検証成功、total calls、wall time、各 request の prompt/output bytes、output tokens の可用性。per-group は group 数だけ request が増えるため、合計 prompt bytes と合計 wall time を比較する。日本語 fixture は summary language の検証に使う。MLX の output tokens が取得できない場合は unavailable と記録する。
