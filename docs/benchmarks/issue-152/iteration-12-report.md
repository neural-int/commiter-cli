## 要約

Iteration 11の未使用3件を診断したところ、repository evidenceにはgoldと一致するsource/test対が全12組存在した。余分な対も欠落も0だった。経路抽出の修正を必要とする欠落は確認できず、既存のsoft relationを正しいpartitionへ変換できないことを特定した。B No-Goは保持する。

## 検証結果

| 条件 | 正しい対 | 余分な対 | 欠落した対 |
| --- | --- | --- | --- |
| A-only | 0 | 0 | 12 |
| repository | 12 | 0 | 0 |

3件それぞれA-only 0/4、repository 4/4。変更unitのsymbol annotation、syntax graph、最大2hopのdirected call pathから生成したunit_relationsを、評価側の既存SymbolGoldと比較した。graphとunit_relationsは前回の同じpayload関数をそのまま使用した。goldは関係生成後の評価にのみ参照し、抽出器へ渡していない。モデル追加callは0。前回のexact0/3、FM16、FS12は変更しない。

## 考察

この3ケースでは追加repository情報がsource/test対応を回復できる。ただし、構造上のcall pathの一致だけで共有intentを一般的に保証することはできない。今回のgold一致を根拠に全call relationをhard mergeへ変更すると、shared dependencyや独立intentのguardrailを満たす証拠がない。Cの最終partition最適化をBで先取りして合格扱いにはしない。

失敗はモデルに渡る以前の経路欠落ではなく、モデルがsoft relationを利用してpartitionを構成する段階にある。この診断は使用済み3syntheticに限定され、独立generalizationや実repositoryの品質を証明しない。new file/rename/sparse history、65unit拒否、memory/cacheも未達。

## Next Steps

- 次の新規専用worktreeで、call pathが存在する独立intentの反例を固定し、unit relationのfalse-positive率を評価する。正しい対の抽出成功を一般的なintent signalと誤認しないため。
- shared callee、source/testを介した共通adapter、独立したpublic API変更の依存方向を含む最小反例で、同じ2hop抽出を変更せず検査する。graphの完全性とsemantic relevanceを分けるため。
- 反例でsignalが崩れる場合、hard relation採用やC移行を行わず、Bで扱える別の独立evidenceが残るかを評価する。モデル指示文や出力予算の調整は解決策にしない。
