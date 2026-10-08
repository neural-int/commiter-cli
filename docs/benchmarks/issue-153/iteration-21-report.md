## 要約

対応実装・変更assertionの判断基準を明示しても、固定C scorerの結果は改善しなかった。元system/追加policyとも既存6file exact失敗・FS3、新規合成3case exact1/3・FM0/FS4。各caseの全scoreとmembershipが両armで完全一致し、全scoreは-1だった。このpolicy候補はNo-Go。同候補のwording/model/閾値反復を終了する。prompt一般の効果不存在や、全モデルの不可能性は主張しない。

## 検証結果

条件/code/fixture/preflightをa78a63b9で結果確認前に固定・pushし、#153コメント6054301917と#150コメント6054302340へ登録した。新規専用worktree issue-153-policy-ablation、同じQwen3-8B revision/helper hash/grammar/profile/予算/schema/payload/機械的上流acceptance/solverでsystem追加のみ比較。全8call completed/accepted/complete、retry0/repair0、unresolved/timeout/invalid0。

| ケース | 出所 | 元system exact / FM / FS | 追加policy exact / FM / FS |
| --- | --- | --- | --- |
| 既存6file/3intent | 使用済み失敗診断 | false / 0 / 3 | false / 0 / 3 |
| Fee切上げとWrap括弧＋対応test | モデル未使用合成 | false / 0 / 2 | false / 0 / 2 |
| Count負値処理＋無関係test診断文 | モデル未使用合成 | true / 0 / 0 | true / 0 / 0 |
| 同一TestBothの独立assertion | モデル未使用合成 | false / 0 / 2 | false / 0 / 2 |

FM/FSはunit-pair数。元/追加とも全4case exact1/4・FM0/FS7、新規3case exact1/3・FM0/FS4。使用済み6fileもiteration17のunit FS3と同じ失敗を再現。過去のatom FS523とは尺度が異なるため直接合算しない。各arm28pairの全scoreが-1、paired membership完全一致。入力payload/schema SHAはcase内両arm同値。

元4call wall59.826秒/input4723/output436tokens、追加4call wall61.561秒/input5091/output436tokens。追加指示で各call input92tokens増、計368増。各条件各入力1回なので時間差の安定性/反復変動/順序効果を推定しない。model-only scorer比較で、全段latencyやproduction改善を計測した結果ではない。

事前の正解intent全subset20状態のGo testとstaging成功、構造10testとgit diff --check成功。gold/要求注釈は評価側のみ。機械的親acceptは両armで同一、goldでrefine/edgeを決めていない。新規例はエージェント作成の少数合成であり、別作者/実repository独立評価として扱わない。production/default/Git mutation経路は変更なし。

## 考察

今回支持されなかったのは、このmodel/helper/profile/input contractで、明示的source/assertion判断基準を一つ追加すれば誤分割を改善する仮説。可視入力に対応する変更前後codeがあり、正解境界はA unitで表現可能でも、今回の処置では正しいpositive pairを返さなかった。

モデルが指示を無視した、意味を理解できない、入力が絶対十分、など内部原因は観測していない。input92token増は処置入力の差を確認するが、内容理解の証明ではない。過去Bのmetadata-only改善は消さず、prompt/形式が常に無関係という結論にしない。過去の別positive controlの+2も保持し、scorerがすべての入力で常に-1とは言わない。

唯一exactのログ反例は全negativeが正解の入力なので、これだけでsource/testの選択的な意味判別を証明しない。総FM0も同様に、必要なpositiveを切るFSを伴っている。独立目的の分離と対応変更の統合が同時に成立することが必要。

事前継続gateの既存6file成功、新規3case全成功、新規exact改善を満たさない。文言を結果依存で増やしたり、同じ失敗への例示/閾値/mandatory source-test edgeを足したりしない。Cの構造成立とscorer判断の未達を区別し、正式C/B/production gateは維持する。iteration-21-completion-audit.jsonでは過去19要件の未達/partialを保持して今回の限定証拠を追記した。Goal未達であり最終全project gate実行条件は未成立。

## Next Steps

- この一候補のprompt反復を終了し、元/追加の全結果・事前条件を保存する。固定比較で改善作用が得られなかったため。
- 再開には、現在のscorer契約と区別できる入力情報または責務変更と、positive/negativeを同時に判別できる独立能力証拠を先に具体化する。未測定のwordingやmodel名だけで探索を増やさないため。
- その根拠が成立した場合だけ別作者/実repository未使用入力、全段、#149回帰、同条件production比較、metadata/Validate/costの正式評価へ進む。今回の合成scorer診断をproduction採用にしないため。
- A bounded GO/B No-Go/C現候補No-Go/D未正当化/production4fileを保持する。promptの不合格だけでDのevidence/partition妥当性を証明できないため。
