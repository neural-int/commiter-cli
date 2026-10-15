"""Freeze the Iteration 2 category arm without generation or historical edits.

Arm A reuses the Iteration 1 candidate requests byte for byte. Arm B removes only
the category type `update` and the two fallback sentences that mention it.
User diffs, grouping requests, text requests, Gold, and Iteration 1 results stay
untouched. The product 120s cycle deadline is not modified.
"""
import argparse, hashlib, json, shutil
from pathlib import Path

SRC = Path(__file__).resolve().parents[2]
PRIOR = SRC / "docs/benchmarks/issue-173"
DATA = PRIOR / "iteration-2"
SCHEMA_PREFIX = " The required JSON Schema is: "
UPDATE_LINE = "- update: Behavior changed; only feat vs fix remains ambiguous.\n"
OLD_RULES = (
    "- If only feat vs fix is ambiguous, use update.\n"
    "- Otherwise use change only if no specific type is supported.\n"
)
NEW_RULES = (
    "- If behavior changed and only feat versus fix remains possible, choose feat or fix. Do not use change to avoid that choice.\n"
    "- Use change only when no specific type is supported.\n"
)
FORMER_UPDATE = ["A/G001", "B/G001", "D/G001", "E/G001"]
AMBIGUOUS = ["F03", "F10", "F16"]
CATEGORY_AMBIGUOUS = ["F17", "F18"]


def h(b):
    return hashlib.sha256(b).hexdigest()


def save(path, obj):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(obj, indent=2, ensure_ascii=False) + "\n")


def go_marshal(obj):
    raw = json.dumps(obj, ensure_ascii=True, separators=(",", ":"), sort_keys=True)
    return raw.replace("&", "\\u0026").replace("<", "\\u003c").replace(">", "\\u003e")


def cases():
    return [* "ABCDE", *[f"F{i:02d}" for i in range(1, 21)]]


def strip_update(schema):
    changed = []
    for group_id, spec in schema["properties"].items():
        enum = spec["properties"]["type"]["enum"]
        if enum == ["test"]:
            if "update" in enum:
                raise ValueError("test-only enum must stay [test]")
            continue
        if "update" not in enum:
            raise ValueError("missing update enum: " + group_id)
        spec["properties"]["type"]["enum"] = [item for item in enum if item != "update"]
        changed.append(group_id)
    return changed


def arm_b(request):
    system = request["messages"][0]["content"]
    if system.count(SCHEMA_PREFIX) != 1:
        raise ValueError("schema prefix count")
    body, suffix = system.split(SCHEMA_PREFIX, 1)
    if json.loads(suffix) != request["schema"] or suffix != go_marshal(request["schema"]):
        raise ValueError("embedded schema drifted from the schema field")
    if body.count(UPDATE_LINE) != 1 or body.count(OLD_RULES) != 1:
        raise ValueError("fallback text is not the frozen candidate prompt")
    body = body.replace(UPDATE_LINE, "", 1).replace(OLD_RULES, NEW_RULES, 1)
    if "update" in body:
        raise ValueError("arm B prompt still mentions update")
    strip_update(request["schema"])
    request["messages"][0]["content"] = body + SCHEMA_PREFIX + go_marshal(request["schema"])
    return request


def assert_arm_difference(source, transformed):
    if source["messages"][1] != transformed["messages"][1]:
        raise ValueError("user diff changed")
    for key in ("context_tokens", "generation_profile", "model", "output_tokens"):
        if source[key] != transformed[key]:
            raise ValueError("inference parameter changed: " + key)
    if [m["role"] for m in source["messages"]] != [m["role"] for m in transformed["messages"]]:
        raise ValueError("message roles changed")
    original_body = source["messages"][0]["content"].split(SCHEMA_PREFIX, 1)[0]
    new_body = transformed["messages"][0]["content"].split(SCHEMA_PREFIX, 1)[0]
    expected = original_body.replace(UPDATE_LINE, "", 1).replace(OLD_RULES, NEW_RULES, 1)
    if new_body != expected:
        raise ValueError("prompt edit was not limited to the update fallback")
    if json.loads(transformed["messages"][0]["content"].split(SCHEMA_PREFIX, 1)[1]) != transformed["schema"]:
        raise ValueError("arm B embedded schema mismatch")


def historical_update_groups():
    found = []
    for line in (PRIOR / "category-results.jsonl").read_text().splitlines():
        row = json.loads(line)
        if row["arm"] != "candidate":
            continue
        for group in row["groups"]:
            if group["actual"] == "update" and row["case"] in "ABCDE":
                found.append(f"{row['case']}/{group['group_id']}")
    if found != FORMER_UPDATE:
        raise ValueError("historical update groups drifted: " + ",".join(found))
    return found


def build_order():
    order = []
    for index, case in enumerate(cases()):
        if len(case) == 1:
            order.append({"case": case, "phase": "grouping"})
        arms = ["A", "B"] if index % 2 == 0 else ["B", "A"]
        for arm in arms:
            order.append({"case": case, "phase": "category", "arm": arm})
        if len(case) == 1:
            order.append({"case": case, "phase": "text"})
    if sum(1 for item in order if item["phase"] == "category") != 50:
        raise ValueError("category call count")
    if len(order) != 60:
        raise ValueError("total call count")
    return order


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--helper", type=Path, required=True)
    parser.add_argument("--model", type=Path, required=True)
    args = parser.parse_args()
    if (DATA / "preregistration.json").exists():
        raise ValueError("never overwrite a frozen iteration-2 suite")
    if (DATA / "inputs").exists():
        shutil.rmtree(DATA / "inputs")
    root = args.root.resolve()
    helper = args.helper.resolve()
    model = args.model.resolve()
    root.mkdir(parents=True, exist_ok=True)
    historical_update_groups()
    gold = json.loads((PRIOR / "gold.json").read_text())
    ambiguous = [case for case, row in gold.items() if row["identifiability"] == "feat_fix_ambiguous"]
    category_ambiguous = [case for case, row in gold.items() if row["identifiability"] == "category_ambiguous"]
    if sorted(ambiguous) != AMBIGUOUS or sorted(category_ambiguous) != CATEGORY_AMBIGUOUS:
        raise ValueError("gold identifiability partition drifted")
    pins = json.loads((PRIOR / "model-artifacts.json").read_text())
    for item in pins:
        blob = (model / item["name"]).read_bytes()
        if len(blob) != item["bytes"] or h(blob) != item["sha256"]:
            raise ValueError("model artifact drift: " + item["name"])
    prior_prereg = json.loads((PRIOR / "preregistration.json").read_text())
    helper_bytes = helper.read_bytes()
    if h(helper_bytes) != prior_prereg["helper_sha256"]:
        raise ValueError("helper is not the iteration-1 production helper")
    for name in (
        "internal/planning/three_phase.go",
        "internal/planning/schema.go",
        "internal/planning/category_input.go",
        "internal/planning/validate.go",
    ):
        if h((SRC / name).read_bytes()) != prior_prereg["source_fixture_request_sha256"][name]:
            raise ValueError("production planning source drifted: " + name)
    manifest = []
    for case in cases():
        source_path = PRIOR / "inputs" / f"{case}-candidate.json"
        source = json.loads(source_path.read_text())
        user = source["messages"][1]["content"]
        transformed = arm_b(json.loads(source_path.read_text()))
        assert_arm_difference(source, transformed)
        if transformed["messages"][1]["content"] != user:
            raise ValueError("user bytes changed after transform")
        destination_a = DATA / "inputs" / f"{case}-A.json"
        destination_a.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source_path, destination_a)
        if destination_a.read_bytes() != source_path.read_bytes():
            raise ValueError("arm A copy drifted")
        save(DATA / "inputs" / f"{case}-B.json", transformed)
        if len(case) == 1:
            for phase, filename in (("grouping", "01"), ("text", "03")):
                phase_source = PRIOR / "integrated-inputs" / f"{case}-{filename}.json"
                phase_destination = DATA / "inputs" / f"{case}-{phase}.json"
                shutil.copyfile(phase_source, phase_destination)
                if phase_destination.read_bytes() != phase_source.read_bytes():
                    raise ValueError("phase copy drifted")
        test_only = [
            group_id
            for group_id, spec in source["schema"]["properties"].items()
            if spec["properties"]["type"]["enum"] == ["test"]
        ]
        manifest.append({"case": case, "test_only_groups": test_only})
    for item in manifest:
        saved = json.loads((DATA / "inputs" / f"{item['case']}-B.json").read_text())
        item["arm_b_type_enums"] = {
            group_id: spec["properties"]["type"]["enum"] for group_id, spec in saved["schema"]["properties"].items()
        }
    runtime = {"helper": str(helper), "model_path": str(model), "model": prior_prereg["model"]}
    save(root / "runtime.json", runtime)
    order = build_order()
    input_hashes = {str(path.relative_to(DATA)): h(path.read_bytes()) for path in sorted((DATA / "inputs").glob("*.json"))}
    preregistration = {
        "experiment": "issue-173 iteration 2: exclude update from category choices",
        "source_iteration": "docs/benchmarks/issue-173",
        "iteration1_preregistration_sha256": h((PRIOR / "preregistration.json").read_bytes()),
        "iteration1_category_results_sha256": h((PRIOR / "category-results.jsonl").read_bytes()),
        "iteration1_gold_sha256": h((PRIOR / "gold.json").read_bytes()),
        "model": prior_prereg["model"],
        "helper_sha256": prior_prereg["helper_sha256"],
        "model_artifacts_sha256": h((PRIOR / "model-artifacts.json").read_bytes()),
        "production_planning_source_unchanged": True,
        "product_cycle_timeout_seconds_unchanged": 120,
        "shared_deadline_enforced": False,
        "safety_timeout_seconds": 600,
        "safety_timeout_reason": "Finite hang guard. The longest Iteration 1 phase was grouping E at 115.540s and category E at 102.313s. The same 600s cap applies to every phase and both arms.",
        "temperature": 0,
        "top_p": 1,
        "top_k": 0,
        "seed": 144,
        "category_native_thought_limit": 384,
        "output_budgets": {"grouping": 768, "category": 512, "text": 768},
        "retries": 0,
        "repairs": 0,
        "additional_quality_calls": 0,
        "order": order,
        "max_generation_calls": 60,
        "former_update_groups": FORMER_UPDATE,
        "feat_fix_ambiguous_cases": AMBIGUOUS,
        "category_ambiguous_cases": CATEGORY_AMBIGUOUS,
        "input_sha256": input_hashes,
        "manifest": manifest,
        "comparison": {
            "A": "Iteration 1 candidate: 11 conventional types plus update and change. Frozen request bytes.",
            "B": "Same user content, model, and sampling. Type enum omits update. Fallback text requires feat or fix when only those two remain, and forbids change as that escape.",
            "unchanged": "Group-scoped raw diff, related tests, grouping requests, text requests, membership, test-only schema, Gold, Iteration 1 results.",
        },
        "decision_rule": {
            "primary": "Compare this run's arm A with arm B. Iteration 1 candidate outputs are a replication check for arm A, not the paired control.",
            "existing_groups": "For the four historical update groups, record B's type individually. Report gold matches and fix selections on all six groups separately from false feat/fix assertions.",
            "ambiguous_boundaries": "On gold identifiability feat_fix_ambiguous, a B output of feat or fix is a forced definite assertion. It is not gold agreement and is not an accuracy gain.",
            "change_escape": "change on a feat/fix-ambiguous group is an escape from the required binary choice.",
            "other_categories": "Report gold matches and false selections for the other boundary classes separately.",
            "plans": "Sum of grouping, category, and text wall times is a deadline observation. A completed text schema does not improve the category conclusion and is not compared with Iteration 1 accepted-plan rate.",
            "adoption": "Do not edit product types or the 120s deadline. Do not overturn the Iteration 1 NO-GO. Record whether the update-exclusion hypothesis is supported.",
            "hypothesis_not_supported": "Arm B gold matches on the six existing fix groups are not strictly greater than arm A, or none of the four historical update groups becomes fix.",
            "hypothesis_not_specific": "Fix selections rise while false fix selections on identifiable non-fix boundaries also rise.",
        },
        "stop": "No edits to Gold, fixtures, prompts, or Iteration 1 files after this freeze. No second generation of a completed call. An interrupted call without a result may be continued once and must be marked resumed. Stop on memory pressure >= 4, helper RSS > 12 GiB, or per-call swap growth > 2 GiB.",
    }
    save(DATA / "preregistration.json", preregistration)
    watched = [
        DATA / "preregistration.json",
        Path(__file__),
        SRC / "tools/benchmark173/iteration2_run.py",
        SRC / "tools/benchmark173/iteration2_analyze.py",
    ]
    # The preregistration hash cannot include itself stably if we also store the runner hashes inside it.
    # Runner hashes are stored beside it and checked before generation.
    lock = {
        "preregistration_sha256": h((DATA / "preregistration.json").read_bytes()),
        "scripts": {path.name: h(path.read_bytes()) for path in watched if path.name != "preregistration.json"},
        "input_sha256": input_hashes,
    }
    save(DATA / "execution-lock.json", lock)
    print(json.dumps({"cases": len(cases()), "calls": len(order), "preregistration_sha256": lock["preregistration_sha256"]}, sort_keys=True))


if __name__ == "__main__":
    main()
