"""Prepare H184-005 display comparisons without generation or live history access."""
import argparse
import copy
import hashlib
import json
import os
import re
from pathlib import Path
import shlex
import shutil
import subprocess
import sys

from multiline_view import multiline_view, restore_view, write_json


def digest(path):
    hasher = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(4 * 1024 * 1024), b""):
            hasher.update(chunk)
    return hasher.hexdigest()


def prepare(source, scope_study, research_root, output, harness_manifest, representation="raw"):
    structured = representation == "structured"
    arms = ("C", "D") if structured else ("A", "B")
    study_key = "H184-005/structured-diff-cd" if structured else "H184-005/multiline-ab"
    harness = json.loads(harness_manifest.read_text())
    assert harness["harness_id"] == "H184-HARNESS-001" and harness["revision"] == 1
    for name, expected in harness["measured_artifact_sha256"].items():
        assert digest(scope_study / "runtime" / name) == expected
    case = json.loads((source / "case.json").read_text())
    assert case["model"] == harness["model"]
    assert case["thinking"] == harness["settings"]["thinking"]
    assert case["output_constraint"] == harness["settings"]["output_constraint"]
    for source_key, setting in (("temperature", "temperature"), ("topP", "top_p"), ("topK", "top_k")):
        assert case[source_key] == harness["settings"][setting]
    assert json.loads((scope_study / "scope-policy.json").read_text()) == harness["scope_policy"]
    assert not output.exists(), "Never overwrite a prepared study"
    output.mkdir(mode=0o700)
    shutil.copyfile(harness_manifest, output / "registered-harness.json")
    shutil.copytree(source / "fixture", output / "fixture")
    shutil.copytree(source / "build", output / "build")
    for name in ("case.json", "history.en.json"):
        shutil.copyfile(source / name, output / name)
    shutil.copytree(scope_study / "runtime", output / "runtime")
    shutil.copyfile(scope_study / "scope-policy.json", output / "scope-policy.json")
    module = Path(__file__).with_name("multiline_view.py")
    shutil.copyfile(module, output / "multiline_view.py")
    if structured:
        shutil.copyfile(Path(__file__).with_name("structured_diff.py"), output / "structured_diff.py")
    (output / "proxy.py").write_text(
        "from pathlib import Path\nfrom multiline_view import proxy\n"
        "raise SystemExit(proxy(Path(__file__).parent))\n")
    launcher = output / "proxy-launcher"
    launcher.write_text("#!/bin/sh\nexec " + shlex.quote(sys.executable)
                        + " -B " + shlex.quote(str(output / "proxy.py")) + "\n")
    launcher.chmod(0o700)
    write_json(output / "paths.json", {"helper": str(output / "runtime/commiter-mlx-helper"), "representation": representation})
    fixture_before = digest(source / "fixture/SHA256SUMS.txt")
    checks = []
    for index, phase in (("01", "grouping"), ("02", "category"), ("03", "text")):
        original = json.loads((source / "execution/final-only" / (index + "-actual-request.json")).read_text())
        canonical = original["messages"][1]["content"]
        assert all(not c["user_requests"] for c in json.loads(canonical)["agent_history_contexts"])
        if structured:
            from structured_diff import project_content
            write_json(output / (phase + ".host-request.json"), original)
            canonical, audit = project_content(canonical)
            write_json(output / (phase + ".code-projection-audit.json"), audit)
        wires = []
        for arm in arms:
            directory = output / "preview" / arm
            directory.mkdir(parents=True, exist_ok=True)
            request = copy.deepcopy(original)
            request["messages"][1]["content"] = canonical if arm == arms[0] else multiline_view(canonical)
            displayed = request["messages"][1]["content"]
            assert restore_view(displayed) == canonical
            write_json(directory / (phase + ".request.json"), request)
            (directory / (phase + ".user.txt")).write_text(displayed)
            environment = {k: v for k, v in os.environ.items() if not k.startswith(("COMMITER_180_", "COMMITER_182_", "COMMITER_184_"))}
            environment.update(COMMITER_182_CASE_CONFIG=str(output / "case.json"), COMMITER_182_NEUTRAL_EXAMPLES="1")
            probe = subprocess.run([str(output / "runtime/commiter-mlx-helper"), "--benchmark-output-contract"],
                                   input=json.dumps(request, ensure_ascii=False).encode() + b"\n",
                                   capture_output=True, env=environment, check=True)
            wire = json.loads(probe.stdout)
            assert wire["generation_calls"] == 0 and wire["thinking"] is False and wire["output_constraint"] == "json_schema"
            assert wire["messages"][1]["content"] == displayed
            write_json(directory / (phase + ".wire.json"), wire)
            wires.append(wire)
            checks.append({"arm": arm, "phase": phase, "actual_newlines": displayed.count("\n"),
                           "user_bytes": len(displayed.encode()), "roundtrip_byte_identical": True, "generation_calls": 0})
        assert wires[0]["messages"][0] == wires[1]["messages"][0] and wires[0]["schema"] == wires[1]["schema"]
    assert digest(source / "fixture/SHA256SUMS.txt") == fixture_before
    subprocess.run([sys.executable, "-B", str(output / "fixture/verify.py")], check=True,
                   stdout=(output / "fixture-verification.json").open("w"))
    runner = (source / "run.py").read_text()
    runner, root_replacements = re.subn(r"ROOT=pathlib.Path\('[^']+'\)",
                                      lambda _: "ROOT=pathlib.Path(" + repr(str(research_root)) + ")", runner)
    assert root_replacements == 1
    assert "HELPER=ROOT/'mlx-helper/.build/out/Products/Release/commiter-mlx-helper'" in runner
    runner = runner.replace("HELPER=ROOT/'mlx-helper/.build/out/Products/Release/commiter-mlx-helper'", "HELPER=B/'runtime/commiter-mlx-helper'")
    runner = runner.replace("assert not OUT.exists();OUT.mkdir(mode=0o700)", "if sys.argv[1:] != ['--execute']:\n print('Prepared only: zero generation. Use --execute only after measurement authorization.');sys.exit(0)\nassert not OUT.exists();OUT.mkdir(mode=0o700)")
    runner = runner.replace("id='H184-004'", "id='H184-005',study_key='H184-005/multiline-ab',harness_id='H184-HARNESS-001',requested_harness_id='4-HARNESS-001',harness_revision=1,harness_profile='B'").replace("['with-request','final-only']", "['A','B']")
    runner = runner.replace("translation='none; Japanese only; final-only empties user_requests in all three phases'", "translation='none; both arms Japanese final responses only; A compact JSON display, B literal newline display; host JSON unchanged'")
    runner = runner.replace("files += [p for p in (B/'fixture').rglob('*') if p.is_file()]", "files += [p for p in (B/'fixture').rglob('*') if p.is_file()]\nfiles += [B/'multiline_view.py',B/'paths.json',B/'scope-policy.json',B/'registered-harness.json',ROOT/'tools/benchmark182/kimdel_inference.py',ROOT/'tools/benchmark182/monitor_v2.py']")
    if structured:
        runner = runner.replace("['A','B']", "['C','D']").replace("H184-005/multiline-ab", study_key)
        runner = runner.replace("A compact JSON display, B literal newline display; host JSON unchanged", "C compact JSON and D literal newline display; added_code/deleted_code projected from host raw_diff; host unchanged")
        runner = runner.replace("hashes={str(p):sha(p) for p in files}", "files += [B/'structured_diff.py']\nhashes={str(p):sha(p) for p in files}")
    (output / "run.py").write_text(runner)
    write_json(output / "preparation.json", {
        "id": "H184-005", "study_key": study_key,
        "harness_id": "H184-HARNESS-001", "requested_harness_id": "4-HARNESS-001",
        "harness_revision": 1, "harness_profile": "B", "runtime_artifact_hashes_verified": True,
        "state": "prepared_not_executed", "model_generation_calls": 0,
        "contract_probes": 6, "case": "C184-02", "arms": list(arms), "phases": 3,
        "maximum_generation_calls_when_authorized": 6, "fresh_grouping_category_text": True,
        "host_fixture": "H184-KIMDEL-REPAIRED-002", "user_requests": "empty arrays in both arms",
        "scope_constraint": "registered harness profile B enabled in Text for BOTH display arms " + "/".join(arms),
        "raw_diff": "host unchanged; LLM receives added_code/deleted_code per block" if structured else "unchanged; no deleted_code/added_code conversion",
        "multiline_format": "display-only JSON-like text with literal LF inside strings; not strict JSON",
        "checks": checks, "source_fixture_unchanged": True,
        "bundle_hashes": {str(p.relative_to(output)): digest(p) for p in output.rglob("*") if p.is_file()},
    })
    print(json.dumps({"prepared": str(output), "generation_calls": 0, "checks": checks}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    for flag in ("source", "scope-study", "research-root", "output", "harness-manifest"):
        parser.add_argument("--" + flag, required=True, type=Path)
    parser.add_argument("--representation", choices=["raw", "structured"], default="raw")
    args = parser.parse_args()
    os.umask(0o077)
    prepare(args.source.resolve(), args.scope_study.resolve(), args.research_root.resolve(), args.output.resolve(), args.harness_manifest.resolve(), args.representation)
