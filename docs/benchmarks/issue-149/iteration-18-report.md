## 要約

任意Goコードを実行しない限定AST observerを追加し、Iteration17の24実行値すべてとの一致を確認した。対応外/評価予算超過はunknown。これは追加観測方法の前提検証で、global grouping品質やproduction candidateは未立証。

## 検証結果

保存済み24観測をoracleとして、同じcallee/callerの4snapshotとliteral引数をAST内で評価した。24/24一致、mismatch0。新規Go fixture実行0、model calls0。focused observer testsとbenchmark build成功。

拒否を確認したケースはmutation、外部関数、再帰、整数範囲外、未解決参照、Goの任意精度を要する浮動定数式、不正な暗黙float→整数return、空budget。Goを完全に型検査する実装ではない。支持範囲は明示したscalar関数/式とselected package参照、限定math/strings操作だけ。整数絶対値2^26以下、string4096bytes以下、step1024/depth16。file/process/environment/network APIは提供しない。

fresh protocol8、globalモデル推論、metadata、production変更は未実行。対応範囲/検証結果はiteration-18-observer-check.jsonと事前登録に記録。

## 考察

この有限のoracleでは、前回得た相互作用情報を任意コード実行なしで再現できた。未知構文・参照・型を推測せず、限定fragmentのobservational evidenceとして利用する仮説を試す前提が得られた。

24値との一致を一般Go semanticsや全入力の証明とは扱わない。特に型alias、global state、method、loop、多値返却、未対応標準関数等の対応を今回追加していない。任意精度の浮動定数式をfloat64で近似しない。Global groupingの採用条件は別途測定が必要で、依存・joint effectをhard unionにする根拠にもならない。

## Next Steps

- H12 global prototypeへ、観測caller/callee/test引数から得た4snapshot値とunknownをsoft evidenceとして加える。syntax/意味抽出IRにはなかった組合せ情報がsemantic判断を改善するか検証するため。
- file/evidence/従来観測を維持し、fixture名/gold/期待groupはpayloadへ渡さず、snapshot値から決定的groupを作らない。観測情報だけを根拠とする条件と不可逆な局所boundaryの禁止を維持するため。
- whole600秒、observer step/depth/string/整数制限とprobe数上限、global native0/output1536/context16K/call120秒を事前登録し、weak16/cross12/shared-callee guardrailを各1回測定する。品質とunknown/費用を独立評価し、全資格通過時のみ未測定protocol8へ進むため。
