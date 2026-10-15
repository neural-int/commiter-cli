## 要約

指定されたCategory短縮Promptを既存25条件で各1回検証した。改善仮説は不支持。本番採用は行わない。

## 検証結果

| 対象 | 前回候補 | 短縮版 |
| --- | ---: | ---: |
| 既存Gold一致 | 0/6 | 0/6 |
| 境界Gold一致 | 11/20 | 13/20 |

- Category Schema適合 25/25。入出力token件数は未取得。
- 最初の25送信はIPCの末尾改行欠落により推論前にmalformed_requestで拒否された。原記録はiteration-3に保存。送信形式だけを修正し、同一Prompt・入力・Goldの別記録iteration-3-framedで各条件1回推論した。helper送信計50件、分類用推論25件。失敗した送信を分類失敗として集計していない。
- Raw Diff、関連テスト、Schema（update/changeを含むenum）、breaking evidence契約、安全指示、モデル・samplingは固定。Grouping/Text再推論0、本番コード変更0。
- モデル・helper hashを推論前に照合し、入力・Gold・履歴・実行スクリプトのhashを前後照合。過去結果とGoldは未変更。
- 前回候補の履歴との比較であり、今回同時に実行した対照群ではない。既知のエージェント作成synthetic Goldであり、独立した人間評価や未使用holdoutではない。

| 条件 | Gold | 前回候補 | 短縮版 |
| --- | --- | --- | --- |
| A/G001 | fix | update | update |
| B/G001 | fix | update | update |
| C/G001 | fix | feat | feat |
| C/G002 | fix | feat | feat |
| D/G001 | fix | update | update |
| E/G001 | fix | update | update |
| F01/G001 | feat | feat | feat |
| F02/G001 | fix | update | update |
| F03/G001 | update | update | update |
| F04/G001 | fix | update | fix |
| F05/G001 | fix | feat | update |
| F06/G001 | refactor | refactor | refactor |
| F07/G001 | perf | refactor | refactor |
| F08/G001 | docs | docs | docs |
| F09/G001 | style | style | style |
| F10/G001 | update | style | style |
| F11/G001 | test | test | test |
| F12/G001 | build | chore | chore |
| F13/G001 | ci | chore | ci |
| F14/G001 | chore | chore | chore |
| F15/G001 | revert | revert | revert |
| F16/G001 | update | update | update |
| F17/G001 | change | change | change |
| F18/G001 | change | update | update |
| F19/G001 | feat | feat | feat |
| F20/G001 | fix | update | update |

fallback詳細：

```json
{
  "existing": {
    "update": {
      "selected": 4,
      "correct": 0,
      "false": 4,
      "missed": 0
    },
    "change": {
      "selected": 0,
      "correct": 0,
      "false": 0,
      "missed": 0
    }
  },
  "boundary": {
    "update": {
      "selected": 6,
      "correct": 2,
      "false": 4,
      "missed": 1
    },
    "change": {
      "selected": 1,
      "correct": 1,
      "false": 0,
      "missed": 1
    }
  }
}
```

Category wall中央値 23.566秒、最大 77.719秒。helper peak RSS最大 3.702 GiB。RSSはMetal/GPU総ピークではない。

## 考察

正答が増えたのはF04（update→fix）とF13（chore→ci）の2件。F05（feat→update）は誤断定を回避したがGoldのfixには一致しない。既存6グループの分類とfallbackの正当選択・誤選択件数は変わっていない。

今回の120秒上限はCategory単独call用であり、製品の共通120秒deadlineや完成planを検証したものではない。F11のtest判定は既存のtest-only Schemaで強制される。

Promptの意味判断部分だけを変更した歴史的比較である。条件ごとの変化と誤断定・fallbackの正誤を別に評価し、断定回避を正答増加と数えない。今回の結果から未使用入力への一般化や完成planの改善は主張しない。Textと共通deadlineは今回検証していない。

## Next Steps

- 今回の結果を候補の検証結果として保持し、mainへ反映しない。
- 次の検証を行う場合は別の有限契約を固定する。既存Goldや過去測定を修正せず、既知fixtureの反復による改善を一般化しない。
