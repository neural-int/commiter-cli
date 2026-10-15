"""Run the frozen Iteration 2 helper calls once. No product deadline and no prompt edits."""
import argparse, hashlib, json, os, re, signal, subprocess, time
from pathlib import Path

SRC = Path(__file__).resolve().parents[2]
DATA = SRC / "docs/benchmarks/issue-173/iteration-2"
PRIOR = SRC / "docs/benchmarks/issue-173"


def h(b):
    return hashlib.sha256(b).hexdigest()


def save(path, obj):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(obj, indent=2, ensure_ascii=False) + "\n")


def resources():
    pressure = int(subprocess.check_output(["/usr/sbin/sysctl", "-n", "kern.memorystatus_vm_pressure_level"]))
    swap = subprocess.check_output(["/usr/sbin/sysctl", "-n", "vm.swapusage"], text=True)
    match = re.search(r"used = ([0-9.]+)([MG])", swap)
    return {"pressure_level": pressure, "swap_used_mib": float(match[1]) * (1024 if match[2] == "G" else 1)}


def critical(before, after, rss):
    return after["pressure_level"] >= 4 or after["swap_used_mib"] - before["swap_used_mib"] > 2048 or (rss is not None and rss > 12 * 1024**3)


def unique(pairs):
    decoded = {}
    for key, value in pairs:
        if key in decoded:
            raise ValueError("duplicate JSON key")
        decoded[key] = value
    return decoded


def validate_category(text, request):
    result = json.loads(text, object_pairs_hook=unique)
    properties = request["schema"]["properties"]
    if not isinstance(result, dict) or set(result) != set(properties):
        raise ValueError("group keys")
    for key, value in result.items():
        if not isinstance(value, dict) or set(value) != {"type", "breaking_evidence_ref"}:
            raise ValueError("category fields")
        for field in value:
            if value[field] not in properties[key]["properties"][field]["enum"]:
                raise ValueError("category enum")
    return result


def validate_labels(text, request):
    result = json.loads(text, object_pairs_hook=unique)
    properties = request["schema"]["properties"]
    if not isinstance(result, dict) or set(result) != set(properties):
        raise ValueError("label keys")
    for key, value in result.items():
        if value not in properties[key]["enum"]:
            raise ValueError("label enum")
    return result


def validate_text(text, request):
    result = json.loads(text, object_pairs_hook=unique)
    properties = request["schema"]["properties"]
    if not isinstance(result, dict) or set(result) != set(properties):
        raise ValueError("text keys")
    for key, value in result.items():
        if not isinstance(value, dict) or set(value) != {"scope", "summary"}:
            raise ValueError("text fields")
        scope, summary = value["scope"], value["summary"]
        if "\n" in scope or "\n" in summary or "\r" in scope or "\r" in summary:
            raise ValueError("text newline")
        if not (1 <= len(scope) <= 16 and 1 <= len(summary) <= 48):
            raise ValueError("text length")
    return result


def load_request(item):
    if item["phase"] == "category":
        path = DATA / "inputs" / f"{item['case']}-{item['arm']}.json"
    else:
        path = DATA / "inputs" / f"{item['case']}-{item['phase']}.json"
    return json.loads(path.read_text()), path


def check_freeze(preregistration, runtime):
    lock = json.loads((DATA / "execution-lock.json").read_text())
    if h((DATA / "preregistration.json").read_bytes()) != lock["preregistration_sha256"]:
        raise ValueError("preregistration drift")
    for name, digest in lock["scripts"].items():
        if h((SRC / "tools/benchmark173" / name).read_bytes()) != digest:
            raise ValueError("harness drift: " + name)
    for name, digest in preregistration["input_sha256"].items():
        if h((DATA / name).read_bytes()) != digest:
            raise ValueError("input drift: " + name)
    if h((PRIOR / "gold.json").read_bytes()) != preregistration["iteration1_gold_sha256"]:
        raise ValueError("gold drift")
    if h((PRIOR / "category-results.jsonl").read_bytes()) != preregistration["iteration1_category_results_sha256"]:
        raise ValueError("historical result drift")
    if h(Path(runtime["helper"]).read_bytes()) != preregistration["helper_sha256"]:
        raise ValueError("helper drift")
    prior = json.loads((PRIOR / "preregistration.json").read_text())
    for name in (
        "internal/planning/three_phase.go",
        "internal/planning/schema.go",
        "internal/planning/category_input.go",
        "internal/planning/validate.go",
    ):
        if h((SRC / name).read_bytes()) != prior["source_fixture_request_sha256"][name]:
            raise ValueError("production source drift: " + name)
    if preregistration["safety_timeout_seconds"] != 600 or preregistration["shared_deadline_enforced"] is not False:
        raise ValueError("deadline contract drift")
    if preregistration["max_generation_calls"] != len(preregistration["order"]):
        raise ValueError("call budget drift")


def invoke(helper, request, timeout):
    process = subprocess.Popen(
        ["/usr/bin/time", "-l", helper],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        start_new_session=True,
    )
    try:
        out, err = process.communicate(json.dumps(request).encode() + b"\n", timeout=timeout)
        status = "returned"
    except subprocess.TimeoutExpired:
        os.killpg(process.pid, signal.SIGTERM)
        try:
            out, err = process.communicate(timeout=5)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGKILL)
            out, err = process.communicate()
        status = "timeout"
    rss = None
    match = re.search(rb"(?m)^\s*(\d+)\s+maximum resident set size\s*$", err or b"")
    if match:
        rss = int(match[1])
    return out, err, status, process.returncode, rss


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, required=True)
    args = parser.parse_args()
    root = args.root.resolve()
    preregistration = json.loads((DATA / "preregistration.json").read_text())
    runtime = json.loads((root / "runtime.json").read_text())
    gold = json.loads((PRIOR / "gold.json").read_text())
    check_freeze(preregistration, runtime)
    if (DATA / "completion.json").exists():
        raise ValueError("suite already completed")
    if (DATA / "resource-stop.json").exists():
        raise ValueError("resource stop already recorded; not resuming generation")
    journal = DATA / "attempts.jsonl"
    results = DATA / "results.jsonl"
    done = {}
    if results.exists():
        for line in results.read_text().splitlines():
            row = json.loads(line)
            done[row["number"]] = row
    attempted = set()
    if journal.exists():
        for line in journal.read_text().splitlines():
            attempted.add(json.loads(line)["number"])
    lock_path = root / "run.lock"
    try:
        fd = os.open(lock_path, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
    except FileExistsError:
        raise ValueError("run lock exists; not starting a second generation")
    os.write(fd, str(os.getpid()).encode())
    os.close(fd)
    timeout = preregistration["safety_timeout_seconds"]
    try:
        for number, item in enumerate(preregistration["order"], 1):
            if number in done:
                continue
            check_freeze(preregistration, runtime)
            before = resources()
            if before["pressure_level"] >= 4:
                raise ValueError("critical resource stop before generation")
            request, _ = load_request(item)
            request["model_path"] = runtime["model_path"]
            attempt = {"number": number, "case": item["case"], "phase": item["phase"], "arm": item.get("arm"), "resumed": number in attempted}
            with journal.open("a") as handle:
                handle.write(json.dumps(attempt) + "\n")
                handle.flush()
                os.fsync(handle.fileno())
            started = time.monotonic()
            print("START " + json.dumps(attempt), flush=True)
            out, err, status, code, rss = invoke(runtime["helper"], request, timeout)
            row = {
                **attempt,
                "stop_reason": status,
                "schema_valid": False,
                "response": None,
                "exit_code": code,
                "repository_not_opened": True,
            }
            try:
                decoded = json.loads(out)
                row.update(stop_reason=decoded.get("stop_reason"), error_class=decoded.get("error_class"), helper_ok=decoded.get("ok"))
                if status != "timeout" and decoded.get("ok") and decoded.get("stop_reason") == "completed":
                    if decoded.get("model") != preregistration["model"] or decoded.get("generation_profile") != request["generation_profile"]:
                        raise ValueError("protocol drift")
                    generated = decoded["generated_json"]
                    if item["phase"] == "category":
                        row["response"] = validate_category(generated, request)
                    elif item["phase"] == "grouping":
                        row["response"] = validate_labels(generated, request)
                    else:
                        row["response"] = validate_text(generated, request)
                    row["schema_valid"] = True
                if status == "timeout":
                    row["stop_reason"] = "timeout"
            except (ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
                if str(exc) == "protocol drift":
                    raise
                row["validation_error"] = "invalid_helper_or_schema"
            if item["phase"] == "category":
                row["groups"] = [
                    {
                        "group_id": group_id,
                        "expected": expected,
                        "actual": (row["response"] or {}).get(group_id, {}).get("type"),
                        "identifiability": gold[item["case"]]["identifiability"],
                    }
                    for group_id, expected in gold[item["case"]]["groups"].items()
                ]
                row["input_bytes"] = sum(len(message["content"].encode()) for message in request["messages"])
            row["input_tokens"] = None
            row["output_tokens"] = None
            after = resources()
            row.update(
                wall_seconds=round(time.monotonic() - started, 3),
                resources_before=before,
                resources_after=after,
                helper_peak_rss_bytes=rss,
                safety_timeout_seconds=timeout,
                product_deadline_enforced=False,
            )
            directory = root / "runs" / f"{number:02d}-{item['case']}-{item['phase']}-{item.get('arm') or 'na'}"
            directory.mkdir(parents=True, exist_ok=True)
            (directory / "helper.stdout.json").write_bytes(out or b"")
            (directory / "helper.stderr.txt").write_bytes(err or b"")
            save(directory / "result.json", row)
            with results.open("a") as handle:
                handle.write(json.dumps(row, ensure_ascii=False) + "\n")
                handle.flush()
                os.fsync(handle.fileno())
            print("RESULT " + json.dumps({"number": number, "case": item["case"], "phase": item["phase"], "arm": item.get("arm"), "stop_reason": row["stop_reason"], "schema_valid": row["schema_valid"], "wall_seconds": row["wall_seconds"]}), flush=True)
            if critical(before, after, rss):
                save(DATA / "resource-stop.json", {"number": number, "case": item["case"], "phase": item["phase"], "arm": item.get("arm")})
                raise ValueError("resource stop; receipt retained")
        check_freeze(preregistration, runtime)
        rows = [json.loads(line) for line in results.read_text().splitlines()]
        if len(rows) != len(preregistration["order"]):
            raise ValueError("incomplete results")
        save(
            DATA / "completion.json",
            {
                "status": "completed",
                "attempts": len(rows),
                "generation_calls": len(rows),
                "preregistration_sha256": h((DATA / "preregistration.json").read_bytes()),
            },
        )
    finally:
        lock_path.unlink(missing_ok=True)


if __name__ == "__main__":
    main()
