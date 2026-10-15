"""Derive Iteration 2 metrics. Does not edit Gold, requests, or result rows."""
import collections, json, statistics
from pathlib import Path

SRC = Path(__file__).resolve().parents[2]
DATA = SRC / "docs/benchmarks/issue-173/iteration-2"
PRIOR = SRC / "docs/benchmarks/issue-173"


def save(path, obj):
    path.write_text(json.dumps(obj, indent=2, ensure_ascii=False) + "\n")


def load_jsonl(path):
    return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []


def median(values):
    return statistics.median(values) if values else None


def category_metrics(rows):
    groups = [group for row in rows for group in row.get("groups", [])]
    counts = collections.Counter(group["actual"] for group in groups)
    return {
        "requests": len(rows),
        "schema_valid": sum(bool(row["schema_valid"]) for row in rows),
        "timeouts": sum(row.get("stop_reason") == "timeout" for row in rows),
        "groups": len(groups),
        "gold_exact": sum(group["expected"] == group["actual"] for group in groups),
        "fix_selected": sum(group["actual"] == "fix" for group in groups),
        "feat_selected": sum(group["actual"] == "feat" for group in groups),
        "update_selected": sum(group["actual"] == "update" for group in groups),
        "change_selected": sum(group["actual"] == "change" for group in groups),
        "false_definite_feat_fix": sum(group["actual"] in ("feat", "fix") and group["actual"] != group["expected"] for group in groups),
        "type_counts": dict(counts),
        "confusion": dict(collections.Counter(f"{group['expected']}->{group['actual']}" for group in groups)),
        "wall_seconds": {"median": median([row["wall_seconds"] for row in rows]), "max": max([row["wall_seconds"] for row in rows], default=None)},
        "helper_peak_rss_max_bytes": max([row.get("helper_peak_rss_bytes") or 0 for row in rows], default=None),
        "swap_delta_max_mib": max([row["resources_after"]["swap_used_mib"] - row["resources_before"]["swap_used_mib"] for row in rows], default=None),
        "input_tokens": None,
        "output_tokens": None,
    }


def by_type(rows, expected_type):
    groups = [group for row in rows for group in row.get("groups", []) if group["expected"] == expected_type]
    selected = [group for row in rows for group in row.get("groups", []) if group["actual"] == expected_type]
    return {
        "gold_groups": len(groups),
        "correct": sum(group["actual"] == expected_type for group in groups),
        "missed": sum(group["actual"] != expected_type for group in groups),
        "selected": len(selected),
        "false_selection": sum(group["expected"] != expected_type for group in selected),
    }


def historical_candidate():
    rows = {}
    for row in load_jsonl(PRIOR / "category-results.jsonl"):
        if row["arm"] == "candidate":
            rows[row["case"]] = {group["group_id"]: group["actual"] for group in row["groups"]}
    return rows


def main():
    preregistration = json.loads((DATA / "preregistration.json").read_text())
    gold = json.loads((PRIOR / "gold.json").read_text())
    rows = load_jsonl(DATA / "results.jsonl")
    category = [row for row in rows if row["phase"] == "category"]
    existing = {arm: category_metrics([row for row in category if row["case"] in "ABCDE" and len(row["case"]) == 1 and row["arm"] == arm]) for arm in ("A", "B")}
    boundary = {arm: category_metrics([row for row in category if row["case"].startswith("F") and row["arm"] == arm]) for arm in ("A", "B")}
    history = historical_candidate()
    transitions = []
    for key in preregistration["former_update_groups"]:
        case, group_id = key.split("/")
        actual = {}
        for arm in ("A", "B"):
            match = next(row for row in category if row["case"] == case and row["arm"] == arm)
            actual[arm] = next(group["actual"] for group in match["groups"] if group["group_id"] == group_id)
        transitions.append({"group": key, "gold": gold[case]["groups"][group_id], "iteration1": history[case][group_id], "A": actual["A"], "B": actual["B"]})
    ambiguous_rows = {arm: [row for row in category if row["case"] in preregistration["feat_fix_ambiguous_cases"] and row["arm"] == arm] for arm in ("A", "B")}
    ambiguous = {}
    for arm, arm_rows in ambiguous_rows.items():
        groups = [group for row in arm_rows for group in row["groups"]]
        ambiguous[arm] = {
            "cases": [
                {"case": row["case"], "expected": row["groups"][0]["expected"], "actual": row["groups"][0]["actual"], "identifiability": row["groups"][0]["identifiability"]}
                for row in arm_rows
            ],
            "forced_definite_feat_fix": sum(group["actual"] in ("feat", "fix") for group in groups),
            "change_escape": sum(group["actual"] == "change" for group in groups),
            "label_match_to_update": sum(group["actual"] == "update" for group in groups),
        }
    identifiable = [case for case, row in gold.items() if case.startswith("F") and row["identifiability"] == "identifiable"]
    identifiable_rows = {arm: [row for row in category if row["case"] in identifiable and row["arm"] == arm] for arm in ("A", "B")}
    other = {}
    for arm in ("A", "B"):
        other[arm] = {
            "metrics": category_metrics(identifiable_rows[arm]),
            "by_type": {typ: by_type(identifiable_rows[arm], typ) for typ in ["feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"]},
            "false_fix_on_non_fix": sum(group["actual"] == "fix" and group["expected"] != "fix" for row in identifiable_rows[arm] for group in row["groups"]),
        }
    category_ambiguous = {}
    for arm in ("A", "B"):
        arm_rows = [row for row in category if row["case"] in preregistration["category_ambiguous_cases"] and row["arm"] == arm]
        category_ambiguous[arm] = [
            {"case": row["case"], "expected": row["groups"][0]["expected"], "actual": row["groups"][0]["actual"]} for row in arm_rows
        ]
    replication = {"existing": 0, "existing_groups": 6, "boundary": 0, "boundary_groups": 20}
    for row in category:
        if row["arm"] != "A":
            continue
        for group in row["groups"]:
            same = history[row["case"]][group["group_id"]] == group["actual"]
            if len(row["case"]) == 1:
                replication["existing"] += int(same)
            else:
                replication["boundary"] += int(same)
    cycles = []
    for case in "ABCDE":
        grouping = next(row for row in rows if row["case"] == case and row["phase"] == "grouping")
        text = next(row for row in rows if row["case"] == case and row["phase"] == "text")
        for arm in ("A", "B"):
            category_row = next(row for row in category if row["case"] == case and row["arm"] == arm)
            total = round(grouping["wall_seconds"] + category_row["wall_seconds"] + text["wall_seconds"], 3)
            cycles.append(
                {
                    "case": case,
                    "arm": arm,
                    "grouping_seconds": grouping["wall_seconds"],
                    "category_seconds": category_row["wall_seconds"],
                    "text_seconds": text["wall_seconds"],
                    "total_seconds": total,
                    "any_timeout": any(item["stop_reason"] == "timeout" for item in (grouping, category_row, text)),
                    "exceeds_product_120s": total > 120,
                    "grouping_schema_valid": grouping["schema_valid"],
                    "category_schema_valid": category_row["schema_valid"],
                    "text_schema_valid": text["schema_valid"],
                    "all_phase_schemas_valid": grouping["schema_valid"] and category_row["schema_valid"] and text["schema_valid"],
                }
            )
    recovered = sum(item["B"] == "fix" for item in transitions)
    to_feat = sum(item["B"] == "feat" for item in transitions)
    false_fix_increased = other["B"]["false_fix_on_non_fix"] > other["A"]["false_fix_on_non_fix"]
    if existing["B"]["gold_exact"] <= existing["A"]["gold_exact"] or recovered == 0:
        hypothesis = "not_supported"
    elif false_fix_increased:
        hypothesis = "not_specific"
    elif recovered == 4 and to_feat == 0:
        hypothesis = "supported_on_existing_fix_groups"
    else:
        hypothesis = "partial"
    summary = {
        "existing": existing,
        "boundary_all": boundary,
        "former_update_groups": transitions,
        "former_update_destinations": dict(collections.Counter(item["B"] for item in transitions)),
        "feat_fix_ambiguous": ambiguous,
        "identifiable_boundaries": other,
        "category_ambiguous": category_ambiguous,
        "replication_arm_a_vs_iteration1": replication,
        "integrated_timing": cycles,
        "generation_calls": len(rows),
        "category_schema_valid": sum(bool(row["schema_valid"]) for row in category),
        "category_requests": len(category),
        "timeouts": sum(row.get("stop_reason") == "timeout" for row in rows),
        "repository_not_opened": all(row.get("repository_not_opened") for row in rows),
        "tokens": "unavailable; production helper response has no token fields; byte counts are not token counts",
        "hypothesis": hypothesis,
        "adoption": "not_adopted",
        "iteration1_nogo_overturned": False,
        "product_source_changed": False,
        "completed": (DATA / "completion.json").exists() and len(rows) == preregistration["max_generation_calls"],
    }
    save(DATA / "analysis.json", summary)
    print(json.dumps({"hypothesis": hypothesis, "existing": existing, "transitions": transitions, "ambiguous": ambiguous, "timeouts": summary["timeouts"], "completed": summary["completed"]}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
