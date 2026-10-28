# root権限の対象限定ファイルI/O診断

## Goal / Conditions

ユーザーがターミナルでfs_usageのみroot実行。通常ユーザーのハーネスとApp Serverは維持し、対象5PIDで25秒記録。本文workerを開始ゲートで待機させ、観測開始後に解放。最大4並列・一覧中本文取得・同じ対象/期間/保存Git入力。ゲート待機は取得全体時間に含まれるため速度比較に使わない。

## Results

fs_usage exit0、stderr空、runner PASS、本文RPC513要求でcomplete。raw約47.5MB、codex行191,368件。stat64 134,320回/合計166.7ms、open 13,310回/154.9ms、getdirentries64 13,191回/110.7ms。各最大0.054/0.090/0.131ms。pread 1,231回/45.9ms、最大0.652ms。履歴pathが明示されたイベント147,506件、記録時間合計345.7ms。

readには最大999ms、合計3.88秒があるが、path/FDの対応を確認できておらず、pipe入力待ち等を含み得る。これを履歴ファイルの読込待ちとみなさない。syscallとdiskイベントの合計は重複・並行を含みwall timeへ加算しない。

## Interpretation / Limits

履歴関連のメタデータ確認と探索が大量にあることは確認。観測した長い本文RPC約174〜204msの時間窓には、長い履歴path操作は見つからず、ストレージの単一長時間読込だけが主因という説明は支持されない。全所有プロセスをまとめた時刻相関であり、RPCとスレッドの一対一対応、未表示pathのFD、fs_usage timestampの境界/観測欠落は未検証。物理ストレージ待ちがないとは断定しない。

pressureは1/2が混在。監視とゲートの影響を含み通常性能と比較しない。CPU処理/JSON化/ロック待ちの割合はI/Oログだけでは未確定。次は対象版のソースでRolloutFile::from_path/ディレクトリ探索とRPCの関係を確認し、必要なら診断用区間計測。自動追加測定なし。

## Artifacts

私有原本はtmpのissue182-root-io-17。生ログは個人pathを含み公開しない。公開用は本goalと匿名数値summaryのみ。commit/push・Issue投稿なし。
