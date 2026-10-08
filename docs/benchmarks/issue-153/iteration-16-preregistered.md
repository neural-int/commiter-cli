## 固定preflight

未使用6file/3intent：ゼロ除算ガード、label前後空白除去、上限100補正。それぞれ実装＋回帰test。全8目的部分状態のGo test/staging、atom budgets/初回message byte guardをモデル前に検証。eval_intentと目的状態は評価側だけで、model payloadへ含めない。

現production4file内の比較対象は先頭4file/2intent（division＋label）に事前固定。full6fileは上限外でありbaseline拒否をFM/FS=0成功にしない。baselineの実モデル/production経路の品質比較は次の計測で別途固定・実施が必要で、今回未実施。初回proposalは<=8を要求。構造不成立ならsetup失敗を保存してmodelへ進まない。model calls0。
