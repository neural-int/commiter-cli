"""Lossless LLM display projection; canonical host JSON remains unchanged."""
import argparse
import copy
import json
import os
from pathlib import Path
import subprocess
import sys


def multiline_view(compact_json):
    json.loads(compact_json)
    result = []
    in_string = False
    index = 0
    while index < len(compact_json):
        char = compact_json[index]
        if in_string and char == "\\":
            escape = compact_json[index + 1]
            result.append("\n" if escape == "n" else char + escape)
            index += 2
            continue
        if char == '"':
            in_string = not in_string
        result.append(char)
        index += 1
    rendered = "".join(result)
    assert json.loads(rendered, strict=False) == json.loads(compact_json)
    assert restore_view(rendered) == compact_json
    return rendered


def restore_view(rendered):
    result = []
    in_string = False
    index = 0
    while index < len(rendered):
        char = rendered[index]
        if in_string and char == "\\":
            result.append(rendered[index:index + 2])
            index += 2
            continue
        if char == '"':
            in_string = not in_string
        result.append("\\n" if in_string and char == "\n" else char)
        index += 1
    return "".join(result)


def json_literal(value):
    return (json.dumps(value, ensure_ascii=False).replace("&", r"\u0026")
            .replace("<", r"\u003c").replace(">", r"\u003e")
            .replace("\u2028", r"\u2028").replace("\u2029", r"\u2029"))


def without_requests(content):
    payload = json.loads(content)
    output = content
    for context in payload["agent_history_contexts"]:
        if not context["user_requests"]:
            continue
        assert len(context["user_requests"]) == 1
        old = '"user_requests":[' + json_literal(context["user_requests"][0]) + ']'
        assert output.count(old) == 1
        output = output.replace(old, '"user_requests":[]', 1)
        context["user_requests"] = []
    assert json.loads(output) == payload
    return output


def write_json(path, value):
    with path.open("x") as stream:
        json.dump(value, stream, ensure_ascii=False, indent=2)


def select_display_arm(configuration, environment):
    """A registered fixed display cannot be changed by a stale study arm."""
    if "display_format" in configuration:
        assert configuration["display_format"] == "literal_newlines"
        assert configuration.get("representation", "raw") == "raw"
        return "B"
    structured = configuration.get("representation") == "structured"
    arm = environment["STUDY_ARM"]
    assert arm in (("C", "D") if structured else ("A", "B"))
    return arm


def proxy(bundle):
    configuration = json.loads((bundle / "paths.json").read_text())
    request = json.loads(sys.stdin.buffer.readline())
    destination = Path(os.environ["COMMITER_STUDY_OUT"])
    arm = select_display_arm(configuration, os.environ)
    structured = configuration.get("representation") == "structured"
    phase = request["generation_profile"]
    index = {"bounded-grouping": "01", "bounded-category": "02", "bounded-text": "03"}[phase]
    assert request["context_tokens"] == 32768 and request["output_tokens"] == 4096
    canonical = copy.deepcopy(request)
    canonical["messages"][1]["content"] = without_requests(canonical["messages"][1]["content"])
    if phase == "bounded-category":
        canonical["messages"][0]["content"] += "\nUse docs for documentation-only changes."
    if phase == "bounded-text":
        canonical["messages"][0]["content"] += "\nNever use feat, fix, docs, style, refactor, perf, test, build, ci, chore or revert as scope. Scope must be 1-16 characters. Summary must be 1-48 characters; count characters, not tokens or bytes."
    write_json(destination / (index + "-canonical-request.json"), canonical)
    actual = copy.deepcopy(canonical)
    content = canonical["messages"][1]["content"]
    if structured:
        from structured_diff import project_content
        content, audit = project_content(content)
        write_json(destination / (index + "-code-projection-audit.json"), audit)
    actual["messages"][1]["content"] = multiline_view(content) if arm in ("B", "D") else content
    write_json(destination / (index + "-actual-request.json"), actual)
    write_json(destination / (index + "-display-audit.json"), {
        "arm": arm, "phase": phase, "host_payload_preserved": True,
        "display_roundtrip_byte_identical": restore_view(actual["messages"][1]["content"]) == content,
        "actual_newlines": actual["messages"][1]["content"].count("\n"),
    })
    environment = dict(os.environ, COMMITER_182_NEUTRAL_EXAMPLES="1",
                       COMMITER_184_SCOPE_CONSTRAINT="1",
                       COMMITER_184_SCOPE_POLICY=str(bundle / "scope-policy.json"))
    data = json.dumps(actual, ensure_ascii=False).encode() + b"\n"
    native = configuration["helper"]
    probe = subprocess.run([native, "--benchmark-output-contract"], input=data,
                           env=environment, capture_output=True, check=True)
    wire = json.loads(probe.stdout)
    assert wire["generation_calls"] == 0 and wire["thinking"] is False
    assert wire["messages"][1]["content"] == actual["messages"][1]["content"]
    write_json(destination / (phase + "-wire.json"), wire)
    return subprocess.run([native, "--benchmark-single"], input=data, env=environment).returncode


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=["proxy"])
    parser.add_argument("--bundle", required=True, type=Path)
    arguments = parser.parse_args()
    os.umask(0o077)
    sys.exit(proxy(arguments.bundle))
