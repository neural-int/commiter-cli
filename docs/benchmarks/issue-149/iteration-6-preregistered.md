# Iteration 6 事前登録: observed test-contract IR

H5はH4へGo test assertionのbefore/after観測を追加する。直接t.Fatal/t.Fatalfで失敗するif分岐の比較、boolean call、否定boolean callから、callee ID/function/arguments/expected/comparison/versionを抽出する。生成した目的説明やgoldは含めない。比較式は観測sourceに基づき、!=の失敗条件をequalの期待条件へ、==はnot_equalへ反転する。runtime観測値やテスト全面検証とは混同しない。

解析非対応のcodeやassertionはliteral before/afterに残す。unique bindingがない場合はcallee IDを空にし、soft call evidenceと全件割当を維持する。grouping prompt/model/profile/context/samplingはH4と同じ。まず6/12fileを各1回。両方exact/FM0/FS0/completeが成立しなければcandidateではない。成立した場合だけ4/8/9/16、独立評価、共通callee独立guardrail、順序、metadata/Validateへ進む。

抽出ロジックを12fileの関数名・パス・goldへ合わせない。sourceのassertion shapesに対する一般の構文prototypeとし、production導入には別child Issueで観測範囲・module binding・非対応時停止を設計する。
