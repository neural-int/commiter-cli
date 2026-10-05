# Iteration 8 事前登録: canonical entity/contract IR

H6はH5の観測IRをsource-first/path順に固定し、model IDを同順のS001…へ写像する。selected IDへ戻して全件一意性を検証する。soft edges、caller/callee、test assertionsも同じIDへ写像し順序を固定する。groupのmerge/splitは行わない。

Go ASTのdeclaration kind/name/before-after observed valueを追加する。関数は観測function literal、const/varは名前と値の対応が一意のValueSpecだけを抽出する。非対応の形はliteral before/afterに残す。goldや任意の目的labelを用いない。

incoming presentationを逆転してもFiles/edges/call/assertion/declarationのserialized IRが同一になるhost testを通過した。実モデルではまずcontract12/weak16を各1回測る。grouping instruction/model/profile/context/samplingはH5と同じ。両方exact/FM0/FS0/completeでなければcandidateではない。成績を見てgoldやcanonical順を変更しない。
