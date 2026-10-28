# 初回取得の本文RPC・応答量・並列競合診断

## Goal / Conditions

同じ対象・期間・保存Git入力、一覧中本文取得、queue16、保護条件で本文1worker→4workerを各1回。アプリ側永続cacheなし。ただしOS/App Serverのcacheをflushしておらず厳密なcold比較ではない。要求引数のSHA256で同じRPCを対応付け、開始終了時刻から本文RPCの重なりを計算する。生ID/本文は追加計測に保存しない。公開summaryには要求キーも含めない。

## Results / Decision

全セルcomplete、同じ513本文要求を対応付け。全体10.933→4.235秒、約61.3%短縮。対応要求の所要時間比の中央値は1.335倍、513中467要求が4worker側で遅い。4並列は個々の要求に遅延を伴うが全体を短縮する。固定順の単発比較のため競合だけを因果として特定しない。

応答bytesと時間のPearson相関は1worker -0.057、4worker -0.064、順位相関は0.081/0.031で、量と時間の単純な単調関係は弱い。4workerの遅い上位4要求は約183〜185ms、約12KB〜144KBであり、共通の時間区間に重なる。大応答だけが遅いという説明は支持されない。server処理/IO/CPUスケジューリング等の共通遅延は候補だが未特定。

要求キーは一致するが応答bytes完全一致は192/513。差の最大は208bytes。外枠RPC IDの桁数の違いに加えlive応答の変化もあり得るため、差の全原因は未特定。生のwire bytesを本文量の完全一致とみなさない。source/照合結果はlive probeで一致。対象編集0件なので実編集精度は未評価。

## Validation / Next gate

関連3パッケージtest、race、vet成功。pressureは両方正常、実同時数1/4。次は遅延区間をサーバーの読込・JSON化・CPU/IO待ちへ切り分けるのが候補。追加取得/再試行/条件変更は行わない。既定変更なし。私有原本はtmpのissue182-rpc-concurrency-15。公開用は本goalと数値summaryのみ。commit/push・Issue投稿なし。
