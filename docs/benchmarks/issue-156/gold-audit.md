# 独立goldの監査記録

モデルへ渡す入力にgold、作者の目的ラベル、単一目的stateを含めない。評価の正解は各preregistered JSONに先に固定し、モデル結果に応じて変更しない。

## C

新規controlled7件は、独立編集それぞれのafter stateを別に作り、全編集のunionからlossless atomの一意な所有権を評価器だけで求めた。同じ行・symbol・shared configにあっても、目的別stateが別に復元できるかでgold feasibilityを監査した。

公開afero mixed履歴では、UnionFile Closeの実装修正と対応test、別のCacheOnReadFs WriteReader testをsourceから区別できる。しかし反復する括弧・空行には複数source witnessがあり、二解上限のDPでは目的別stateの同時完全所有を確定できなかった。推論前にunique-gold primaryから分離した。source-span参照境界は診断に限り、そのpair数を独立exactの成功へ算入しない。

## D

- mock sentinel: missing argumentをtyped sentinelで扱い、matcherへ渡す前にargument数の違いを記録する。対応testはmissing/Anything/matcher/literal sentinelを検証する。同じ失敗条件の実装・assertionであり1purposeと固定。
- NotSubset format: `%q`から`%#v`へ変え、非string要素のメッセージと対応assertionを修正する。1purpose。performance変更という根拠はない。
- Formatter: build条件と同値なgo:build表記を追加し、godocの見出し・箇条書き・indentを正規化する。機械的source正規化として1purposeを固定。commit messageは補助根拠であり、編集内容も確認した。
- README badge: GoReportCard badgeを削除するだけ。CI coverage変更とは独立。
- CI: Go1.25 matrix coverageを残す変更。mock/assertionの動作修正とは独立。
- require selected file: CollectT型の参照をassert packageへ直す2つのgodoc例。元履歴の_codegenテンプレートcleanupはselected filesに含めず、元commit全体の評価とは呼ばない。
- 16file public workloadは以上のdisjoint pathsのcontrolled superposition。自然な同時差分ではない。formatter8、他の小入力とcomponentが相関しているため、12 primaryを12独立作者の観測とは呼ばない。
- authored settingsのindependent群、source/assertion pair群、same-line、shared-test群はCと別に作成したstateを固定した。個々のnumber変更だけで目的同一とは扱わず、fixture constructionの独立stateを評価器側の根拠とする。
- opaque unknown/nonUTF8 unknown/interleaved hypothetical目的はsourceから目的境界を観測できない。15中3件を推論前にprimaryから分離し、unknown継続と所有権の監査だけを行う。同じbyteを2つの目的が同時所有するgoldは創作しない。

## 補助formatter監査

D推論開始後、固定goldを変更せずsource-dataの補助監査を追加した。Go1.27.1の標準gofmtへbefore bytesをstdinとして渡し、stdoutと公開afterを比較した。外部Goコードやtestsは実行せず、source/indexも変更しない。

9file中7fileは公開afterとbyte-exactに一致した。2fileの差はbuild directive直前の空コメント行`//`が公開afterにはあり、現在のformatter出力にはない点だけだった。現在のformatterと当時の出力の完全一致を9/9とは主張しない。結果hashはiteration-7-formatter-source-audit.jsonへ保存した。この補助結果によるcase除外、gold再分類、threshold変更は行っていない。

controlled入力の目的ラベルはfixture作者が固定した独立編集の契約であり、sourceだけから作者の意思を常に一意推定できるという証明ではない。特に同じ値を複数設定へ変更するauthored群には、共通方針変更という別の説明もあり得る。公式の事前登録12件は変更せず、公開履歴のみ7件とauthored5件の結果も分けて報告する。どちらも80%未達であり、controlledの解釈だけでproduction採否を決めない。
