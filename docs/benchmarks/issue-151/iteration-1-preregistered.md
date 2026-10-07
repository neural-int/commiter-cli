# A / Iteration 1 事前登録

仮説: semantic推定を行わない連続したline edit単位でも、同一file内の離れた変更を別commitへ割り当てる構造的自由度を得られる。Go ASTはsymbol位置の注釈だけに用い、goldで抽出境界を決めない。

固定比較: #149 controlled/contract/guardrail/既使用holdout全fixture（freshではなく回帰）。新syntheticは同一file複数intent、adjacent変更、同一line変更、追加/削除、空file、no-final-newline、非UTF8 byte、CRLF。選択済みunitsから中間内容を構築し、隔離したtemporary Git repositoryのindexへ実際にapplyし、index blob byte一致を確認する。原repositoryのindexには触れない。

測定: extraction coverage、determinism、overlap、gold表現可能性、file baselineとの差、units数/serialized byte数/latency、reconstructionとforward/reverse staging。LLM calls/tokensは0。exact/FM/FSはsemantic predictionが存在しないためN/A、gold representation feasibilityとは区別。

GO: 全changed bytesが保存され、全unit適用でafterと一致、独立hunkを別stageでき、同一file複数intentの表現能力がfile baselineより改善。既存multi-file regressionのsemantic ambiguity解消はGOの証拠にしない。部分化不能な同一line/隣接editはopaque atomic blockとして保持し、推測で分割しない。mode/rename/binaryのmetadataはwhole-file operationの契約を記述する。bounded上限超過はsilent truncationせず停止。

制約: synthetic bytesだけ、追加依存/モデル取得/production変更なし。A成功だけではsemantic品質、production staging実装、4file上限拡大を主張しない。
