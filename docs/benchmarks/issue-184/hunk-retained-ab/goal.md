# H184-005 / hunk-retained-ab

## Goal

raw_diffの@@ hunkヘッダーを保持し、diff --git・index・---/+++のみ除く条件を、元diffと比較する。ホストの元データは保持し、LLM表示だけを変換する。

## Conditions

指定4-HARNESS-001 / revision 1、登録H184-HARNESS-001 revision 1、profile B（Text scope制約ON）。C184-02、Qwen3.5-4B affine4bit/group64、Swift Release、Thinking OFF、json_schema、native KV、入力32768/出力4096、temperature0.6/top_p0.95/top_k20、repetition_penalty1、seed144、eager。日本語・user_requestsは両群全phaseで空、final_responsesを保持。固定fixture H184-KIMDEL-REPAIRED-002。各群3phaseを新規生成、実際の先行結果を後続へ渡す。時間制限なし、warmup/retry/repair=0。

A=元raw_diffの一行JSON。B=同じ一行JSONでファイルヘッダー20行（5ファイル×4行）だけ削除し、@@全文、行頭+/-、コード、履歴本文、プロンプト、スキーマを保持。diff_rangeは新raw_diff上の行番号へ再計算するため、この数値変更も実験条件に含む。

## Validation

全6実リクエストが各群の想定変換と一致。source hash不変、コード内容と+/-同一、hunkヘッダー保持を監査。全phase completed、2セル/6生成、監視異常なし。非推論wireプローブは生成0回。投影の単体テスト10件成功。

## Results

| 条件 | Grouping | Category | docs | UI追加（正解feat） | メール修正（正解fix） | 最終validator | host秒 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A | 3/3 | 3/3 | docs | feat | fix | 通過 | 72.477 |
| B | 3/3 | 2/3 | docs | refactor | fix | 通過 | 58.816 |

| 条件 | phase | 入力tokens | 出力tokens | prefill秒 | decode秒 | decode tok/s | MLX peak GiB |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A | 01 | 5148 | 52 | 20.098 | 1.962 | 26.50 | 3.123 |
| A | 02 | 4052 | 71 | 13.520 | 2.894 | 24.53 | 3.148 |
| A | 03 | 3936 | 85 | 13.469 | 5.149 | 16.51 | 3.120 |
| B | 01 | 4800 | 52 | 15.935 | 1.946 | 26.72 | 3.125 |
| B | 02 | 3704 | 72 | 13.081 | 2.626 | 27.42 | 3.113 |
| B | 03 | 3589 | 78 | 12.573 | 4.310 | 18.10 | 3.147 |

## Text review

A/BともJSONと最終validatorは通過。Aのメール要約は空文字列をNoneへ正規化するという表現が広すぎる。BのUI要約は「期限日のツールチップ計算ロジックを修正」で機能追加を修正として記述し、scopeも実ファイル名widget_view.pyとは異なるwidget.pyを含む。Bのメール要約「空の角度文字列正規化を拡張」も具体性が低い。validatorは内容正確性の全保証ではない。

## Interpretation

直前の全ヘッダー削除ではメール修正がfeat、今回@@保持ではfix。メール誤分類についてhunk情報・残存構造が関係する仮説を支持する観測が得られた。ただし今回もCategory合計2/3で、別のUI追加がrefactor。@@だけを普遍的な原因と断定できない。別実行との比較であり、入力長・配置・diff_range数値も変化する。単一fixture/seed各1回。元diffは今回3/3を維持しており、ヘッダー削減を採用する根拠はまだ不十分。

B host時間は短いがA→B固定順・warmupなしでload/prefillも変わるため、短縮率を一般化しない。native KV dtype/bytesは未export。GPU/CPU/RSS/swap/thermalは監視原本と集計に保存。プロセスRSS、MLX peak、システムswapは別指標。

## Resources

```json
{
  "A": {
    "samples": 37,
    "max_concurrent_helpers": 1,
    "max_helper_rss_kib": 1667328,
    "max_helper_cpu_percent": 146.9,
    "system_swap_used_mib_min": 2793.56,
    "system_swap_used_mib_max": 4303.94,
    "thermal_states": [
      "nominal"
    ],
    "gpu_available_samples": 37
  },
  "B": {
    "samples": 29,
    "max_concurrent_helpers": 1,
    "max_helper_rss_kib": 2324992,
    "max_helper_cpu_percent": 468.7,
    "system_swap_used_mib_min": 2793.56,
    "system_swap_used_mib_max": 3407.94,
    "thermal_states": [
      "fair",
      "nominal"
    ],
    "gpu_available_samples": 29
  }
}
```

## Record notes

元plan.jsonのText説明に前実験由来のrequest ablationという表記が残る。実リクエスト監査で両群user_requestsは空であり、今回依頼本文の有無の比較は行っていない。原本planは改変せず、この補足を残す。追加推論・再試行なし。生の会話・Thinking・個人パスは本報告へ収録しない。
