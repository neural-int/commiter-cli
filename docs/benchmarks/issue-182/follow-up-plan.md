# Issue #182 追加検証の順序

2026-10-20のユーザー指示を正本とする。過去の結果・実行契約は上書きしない。

1. Gemma 4 E4B: 現行プロンプトと簡潔化案を、他の条件を固定して比較する。
2. Ministral 3 3B Instruct / Reasoning 2512: 公式mistral-common tokenizer、chat serialization、Thinking/EOSをSwiftハーネスへ適合させ、H182-002のC01/C02を各1回再測定する。モデルweight revision、sampling、seed144、KV8bit/group64、schema、validator、固定partitionによる後段比較、出力・Thinking予算無制限、120秒制限なしを維持する。公式system prompt適合などの変更は別途明記する。
3. Qwen3.5-4B ON / Qwen3.5-2B OFF: 現行と簡潔化案を同条件で比較する。2B ONは検証対象外。
4. プロンプト採用後、Gemma / Qwen4BのThinking ONで無制限と1024 tokensを比較する。Qwen2BはOFFのみ。

共通プロンプトを採用できれば共通化し、差が出る場合はモデル別ハーネスを含む比較として報告する。プロンプト比較とThinking予算比較を同時に変更しない。

Ministral再検証は、公式tokenizerによる入力token列・語彙対応・EOS・THINK境界を推論前に監査する。新しい公式tokenizer依存は検証専用環境へ分離する。監視と中断は生成予算から独立させ、中断を成功に含めない。新規実行記録はH182-002と別の保存先を使う。
