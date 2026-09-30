"""Recompute the 384-call fixed candidate encoding comparison (stdlib only)."""
import collections
import json
import statistics
import sys
from pathlib import Path

root = Path(sys.argv[1])
folder = root / "docs/benchmarks"
rows = [json.loads(line) for line in (folder / "issue-143-contract-comparison-2026-09-30.jsonl").read_text().splitlines()]
manifest = json.loads((folder / "issue-143-contract-manifest-2026-09-30.json").read_text())
assert len(rows) == 384
assert [row["sequence"] for row in rows] == list(range(1, 385))
assert len({(r["model_index"], r["fixture"], r["arm"], r["run"]) for r in rows}) == 384
hashes = {(h["fixture"], h["arm"], h["reverse"]): h for h in manifest["hashes"]}
contracts = {c["fixture"]: c for c in manifest["parent"]["contracts"]}
for row in rows:
    h = hashes[row["fixture"], row["arm"], row["reverse"]]
    assert row["prompt_sha256"] == h["prompt_sha256"]
    assert row["schema_sha256"] == h["schema_sha256"]
    assert row["reverse"] == manifest["parent"]["reverse"][row["run"] - 1]
    assert row["helper_sha256"] == manifest["parent"]["helper_sha256"]
    assert row["calls"] == 1 and row["output_budget"] == 2048 and row["context_tokens"] == 8192
    assert row["candidate_count"] == 2 and row["model_index"] in manifest["model_indices"]
    spec = manifest["parent"]["models"][row["model_index"]]
    assert row["model"] == spec["repo"] + "@" + spec["revision"]
    assert not any(k in row for k in ("prompt", "response", "raw_response", "summary", "path", "diff"))
    assert row["valid_candidate"] or ("false_merge" not in row and "false_split" not in row)

def aggregate(rs):
    valid = [r for r in rs if r["valid_candidate"]]
    join = [r for r in rs if r["category"] == "join"]
    guard = [r for r in rs if r["guardrail"]]
    guard_valid = [r for r in guard if r["valid_candidate"]]
    statuses = collections.Counter()
    for fixture in sorted({r["fixture"] for r in rs}):
        fr = sorted([r for r in rs if r["fixture"] == fixture], key=lambda r: r["run"])
        assert len(fr) == 4
        if not all(r["valid_candidate"] and r["stop_reason"] == "completed" for r in fr):
            statuses["incomplete"] += 1
        elif fr[0]["selected_id"] != fr[3]["selected_id"] or fr[1]["selected_id"] != fr[2]["selected_id"]:
            statuses["repeat_variation"] += 1
        elif fr[0]["selected_id"] != fr[1]["selected_id"]:
            statuses["direction_disagreement"] += 1
        else:
            statuses["stable"] += 1
    first = 0
    for r in valid:
        candidates = contracts[r["fixture"]]["candidates"]
        first += r["selected_id"] == candidates[int(r["reverse"])]["id"]
    return dict(n=len(rs), gold=sum(r["correct_selection"] for r in rs), valid=len(valid),
        completed=sum(r["stop_reason"] == "completed" for r in rs),
        false_merge=sum(r["false_merge"] for r in valid) if valid else None,
        false_split=sum(r["false_split"] for r in valid) if valid else None,
        join_gold=sum(r["correct_selection"] for r in join), join_n=len(join),
        guardrail_gold=sum(r["correct_selection"] for r in guard), guardrail_n=len(guard),
        guardrail_valid=len(guard_valid), guardrail_false_merge=sum(r["false_merge"] for r in guard_valid) if guard_valid else None,
        structural_invalid=sum(r.get("failure") in {"invalid_json", "invalid_schema", "unknown_candidate_id", "duplicate_key", "forbidden_none", "grammar_failure"} or (r["stop_reason"] == "completed" and not r["valid_candidate"]) for r in rs),
        statuses=dict(statuses), first_presented=first,
        stops=dict(collections.Counter(r["stop_reason"] for r in rs)),
        failures=dict(collections.Counter(r.get("failure", "") for r in rs)),
        wall_ms=sum(r["wall_ms"] for r in rs), median_wall_ms=statistics.median(r["wall_ms"] for r in rs),
        prompt_bytes=sum(r["prompt_bytes"] for r in rs), output_bytes=sum(r["output_bytes"] for r in rs),
        output_tokens=dict(collections.Counter(str(r["output_tokens"]) for r in rs)))
result = {}
for mi in manifest["model_indices"]:
    result[str(mi)] = {}
    for arm in manifest["arms"]:
        rs = [r for r in rows if r["model_index"] == mi and r["arm"] == arm]
        assert len(rs) == 48
        known = [r for r in rs if r["set"] == "known"]
        hold = [r for r in rs if r["set"] == "holdout"]
        assert len(known) == 16 and len(hold) == 32
        result[str(mi)][arm] = dict(all=aggregate(rs), known=aggregate(known), original_holdout=aggregate(hold),
            fixtures={f: aggregate([r for r in rs if r["fixture"] == f]) for f in contracts})
base = result["0"]["partition-list"]["original_holdout"]
for mi in manifest["model_indices"]:
    arms = result[str(mi)]
    h = arms["file-membership"]["original_holdout"]
    control = arms["partition-list"]["original_holdout"]
    gates = dict(gold_at_least_28=h["gold"] >= 28,
        gain_at_least_4_over_ministral_control=h["gold"] >= base["gold"] + 4,
        direction_decrease_at_least_2=control["statuses"].get("direction_disagreement", 0) - h["statuses"].get("direction_disagreement", 0) >= 2,
        guardrail_false_merge_zero=h["guardrail_valid"] == 8 and h["guardrail_false_merge"] == 0,
        join_no_worse=h["join_gold"] >= base["join_gold"],
        false_split_no_worse=h["false_split"] is not None and h["false_split"] <= base["false_split"],
        completion_no_worse=h["valid"] >= base["valid"], structural_zero=h["structural_invalid"] == 0,
        wall_median_at_most_2x=h["median_wall_ms"] <= base["median_wall_ms"] * 2,
        repeat_variation_no_worse=h["statuses"].get("repeat_variation", 0) <= control["statuses"].get("repeat_variation", 0))
    arms["file-membership"]["gates"] = gates
    arms["file-membership"]["eligible"] = all(gates.values())
summary = dict(calls=384, models=result)
(folder / "issue-143-contract-summary-2026-09-30.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n")
for mi, arms in result.items():
    for arm, obs in arms.items():
        print(json.dumps(dict(model_index=mi, arm=arm, known=obs["known"]["gold"],
            original_holdout=obs["original_holdout"], eligible=obs.get("eligible")), ensure_ascii=False))
