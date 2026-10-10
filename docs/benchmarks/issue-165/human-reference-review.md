# Issue #165 基準事実19項目の人間レビュー

以下は生成前に固定した source-audit.json の audit_facts を、そのまま読みやすく表示したもの。F番号はcase内の番号。関係labelやcommit境界の承認ではなく、claimが引用sourceから正しく確認できるかを確認する。Codexのsource照合は済んでいるが、人間レビューは未確認。

修正が必要ならcase名・F番号・正しい内容を指定する。修正なしなら「19項目を確認済み・修正なし」と回答する。元の事前登録とrawは変更せず保持する。

## nil-slice-corresponding-assertion

**F1**: Clone(nil) before returns a non-nil empty slice; after explicitly returns nil for nil input.

- `EA:before:L3` — `return append([]int{}, v...)`
- `EA:after:L3` — `if v == nil { return nil }`

**F2**: TestNil calls Clone(nil); before fails on nil, after fails on non-nil.

- `EB:before:L4` — `if Clone(nil) == nil`
- `EB:after:L4` — `if Clone(nil) != nil`

## shared-bound-independent-ui-and-batch

**F1**: DisplayWidth adds 2 to Bound(n,80) after the change; its Bound arguments stay the same.

- `EA:before:L3` — `return Bound(n, 80)`
- `EA:after:L3` — `return Bound(n, 80) + 2`

**F2**: BatchSize changes the Bound maximum argument from 32 to 16.

- `EB:before:L3` — `Bound(n, 32)`
- `EB:after:L3` — `Bound(n, 16)`

**F3**: Bound returns max when n > max, otherwise n; this helper is unchanged.

- `context:bound.go:L3` — `if n > max { return max }; return n`

## same-package-independent-time-and-permission

**F1**: Expired changes > to >=, making equality return true after the change.

- `EA:before:L3` — `now > deadline`
- `EA:after:L3` — `now >= deadline`

**F2**: CanEdit adds editor alongside owner to accepted roles.

- `EB:before:L3` — `role == "owner"`
- `EB:after:L3` — `role == "owner" || role == "editor"`

## shared-import-independent-search-and-sort

**F1**: Contains lowercases both arguments before strings.Contains after the change.

- `EA:before:L4` — `strings.Contains(s, q)`
- `EA:after:L4` — `strings.Contains(strings.ToLower(s), strings.ToLower(q))`

**F2**: Before changes strings.Compare(a,b) < 0 to > 0.

- `EB:before:L4` — `strings.Compare(a, b) < 0`
- `EB:after:L4` — `strings.Compare(a, b) > 0`

## boolean-polarity-compensation

**F1**: EncodeFlag maps true/false to 1/0 before and 0/1 after.

- `EA:before:L3` — `if v { return 1 }; return 0`
- `EA:after:L3` — `if v { return 0 }; return 1`

**F2**: DecodeFlag changes the true condition from n == 1 to n == 0.

- `EB:before:L3` — `n == 1`
- `EB:after:L3` — `n == 0`

**F3**: The source requires FlagRoundtrip for both booleans; FlagRoundtrip compares DecodeFlag(EncodeFlag(v)) with v.

- `context:contract.go:L3` — `// The local flag codec contract requires`
- `context:contract.go:L4` — `DecodeFlag(EncodeFlag(v)) == v`

## new-round-provider-eb

**F1**: Total changes from returning n to calling RoundToTen(n).

- `EA:before:L3` — `return n`
- `EA:after:L3` — `return RoundToTen(n)`

**F2**: RoundToTen is absent before and added in EB after; it returns (n+9)/10*10.

- `EB:before:L3` — `func Unchanged()`
- `EB:after:L4` — `func RoundToTen(n int) int { return (n + 9) / 10 * 10 }`

## cache-expiry-multiple-boundaries

**F1**: ImageTTL changes 60 to 120.

- `EA:before:L3` — `ImageTTL = 60`
- `EA:after:L3` — `ImageTTL = 120`

**F2**: DocumentTTL changes 90 to 180.

- `EB:before:L3` — `DocumentTTL = 90`
- `EB:after:L3` — `DocumentTTL = 180`

## unknown-admission-contract

**F1**: CPUReservation changes 2 to 3.

- `EA:before:L3` — `return 2`
- `EA:after:L3` — `return 3`

**F2**: MemoryReservation changes 4 to 6.

- `EB:before:L3` — `return 4`
- `EB:after:L3` — `return 6`

**F3**: Admitted passes both reservations to a client-supplied Admission function; its cross-resource constraints are unavailable.

- `context:admission.go:L3` — `// Admission is supplied by clients; its cross-resource constraints are unavailable.`
- `context:admission.go:L5` — `check(CPUReservation(), MemoryReservation())`
