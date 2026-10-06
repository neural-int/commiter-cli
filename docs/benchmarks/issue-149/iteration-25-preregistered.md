# Iteration25事前登録: bounded provisional purpose discovery

H18はIteration24 Next Stepsに従い、host record/finite contrast全体を一度読み、暫定purpose候補を抽出してから元観測全件と候補によるglobal G-ID assignmentへ進む。候補はfile membershipや不可逆boundaryではない。後段は候補を無視/分割/結合/追加でき、最終G-ID数はfile数まで。#140のraw自由形式intent probe、H11のentity単位意味抽出とは入力と責務が異なる。

discovery schemaは候補1〜16件、各purpose1〜120文字、supporting E-ID1〜32件とunresolved boolean。hostは空/重複purpose、同候補内duplicate E-ID、unknown E-ID、範囲/文字数/required/unresolvedを拒否。同E-IDが別候補を支持することは許す。全file/E-IDの候補coverageは要求せず、元観測を保持。これは根拠参照の構造検証であり意味正しさの証明ではない。後段assignmentは既存strict partition gate。

fixed Qwen3-8B revision545dc4251c05440727734bcd94334791f6ab0192、helper SHA016b706cacbbe39315b80fc68f4e82fa29e732afc3e371652c670d24ae3550c4、全stage bounded-routed-grouping/native0/output1536/context16K/call120秒/whole600秒、temp0/top_p1/top_k0/seed144、2calls/fixture、retry/repair0。weak16/cross12/guardrail各1回順次。全通過のみfresh protocol8へ。goldは入力外、raw思考/prompt/候補text保存なし。discovery候補数/参照数と各phase metricsを記録し、失敗段階を区別。追加取得/依存/production変更なし。

失敗時は抽出・assignmentの責務別の観測を整理し、同taskのwording/budget sweepを避けて、残余のarchitecture根拠または再開prerequisiteを監査する。
