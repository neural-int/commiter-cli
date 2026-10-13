# Iteration 2 資源停止

元条件を維持したrun2もF01/A/通常順1回で停止した。Exact1、FM0、FS0、Schema/完全割当PASS。swap前後差は8092.62→12276.12 MiB（+4183.50 MiB）、RSS718.75 MiB、MLX peak4.796 GiB、wall35.597秒。残り95cellは未実行、paired比較0。品質の要因効果は未判定。

ユーザーからシステム全体swapを停止条件から除外し、commiterプロセスのメモリを観測する指示を受けた。旧run2を再開・上書きせず、新しいrun3の事前登録でその指示を適用する。入力/Gold/prompt/schema/model/decoder/Exact/FM/FS基準は変更しない。旧停止は記録に残す。
