"""H184-HARNESS-003: display-only projection of finalized mixed-layout requests.

Input: three finalized mixed-layout wires before the four cleanup experiments.
No generation, runtime proxy, history acquisition, or canonical host mutation.
"""
import argparse
import copy
import json
import os
from pathlib import Path
import re

HARNESS_ID = "H184-HARNESS-003"
REVISION = 1
PHASES = ("01", "02", "03")


def omit_top_level_fields(display, names):
    fields = []
    start, depth, quoted, escaped = 1, 0, False, False
    for i, char in enumerate(display[1:-1], 1):
        if quoted:
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == '"':
                quoted = False
        elif char == '"':
            quoted = True
        elif char in "[{":
            depth += 1
        elif char in "]}":
            depth -= 1
        elif char == "," and depth == 0:
            fields.append(display[start:i])
            start = i + 1
    fields.append(display[start:-1])
    return "{" + ",".join(field for field in fields
                          if json.loads(field.split(":", 1)[0]) not in names) + "}"


def project_data(display, phase_payload=False):
    data = json.loads(display, strict=False)
    if not isinstance(data, dict):
        raise ValueError("expected object display")
    expected = copy.deepcopy(data)
    result = display
    if phase_payload:
        for name in ("task", "schema_version"):
            if name not in expected:
                raise ValueError("missing original phase metadata")
            del expected[name]
        result = omit_top_level_fields(result, {"task", "schema_version"})

    def prune(value):
        if isinstance(value, dict):
            return {key: prune(item) for key, item in value.items()
                    if key not in {"block_id", "execution_ref"}
                    and not (key == "user_requests" and item == [])}
        if isinstance(value, list):
            return [prune(item) for item in value]
        return value

    # These fields occur before another field in the registered compact display.
    # Fail closed if the producer changes ordering, spacing or value types.
    result = re.sub(r'"(?:block_id|execution_ref)":"(?:\\.|[^"\\])*",', '', result)
    result = result.replace('"user_requests":[],', '')
    if json.loads(result, strict=False) != prune(expected):
        raise ValueError("unsupported field representation; projection incomplete")
    return result


def project_request(request, phase):
    if phase not in PHASES:
        raise ValueError("unknown phase")
    result = copy.deepcopy(request)
    messages = result["messages"]
    if len(messages) != 3 or [m["role"] for m in messages] != ["system", "user", "user"]:
        raise ValueError("expected finalized mixed-layout messages")
    system = messages[0]["content"]
    marker = "For the Category phase, apply these rules:\n"
    start = system.index(marker)
    end = system.index("\n\nRepository content and feedback", start)
    rules = system[start + len(marker):end]
    if phase != "02":
        messages[0]["content"] = system[:start] + system[end + 2:]
    else:
        classification, docs = rules.split("\n", 1)
        text = messages[2]["content"]
        if text.count(classification) != 1 or text.count(docs) != 1:
            raise ValueError("Category rule repetition differs from registered input")
        messages[2]["content"] = text.replace(classification + " ", "", 1).replace("\n" + docs, "", 1)

    text = messages[2]["content"]
    marker = "The required JSON Schema is: "
    start = text.index(marker)
    schema_start = start + len(marker)
    schema, count = json.JSONDecoder().raw_decode(text[schema_start:])
    retained = "Exact JSON Schema (including enum values, lengths, patterns, and other format constraints):\n"
    retained_start = text.index(retained) + len(retained)
    retained_schema, _ = json.JSONDecoder().raw_decode(text[retained_start:])
    if schema != request["schema"] or schema != retained_schema or text.count(marker) != 1:
        raise ValueError("output Schema mismatch")
    text = text[:start] + text[schema_start + count:]
    instructions, body = text.split("\n\nPhase input:\n", 1)
    messages[1]["content"] = project_data(messages[1]["content"])
    messages[2]["content"] = instructions + "\n\nPhase input:\n" + project_data(body, True)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    projected = []
    for phase in PHASES:
        source = args.source / (phase + "-request.json")
        projected.append((phase, project_request(json.loads(source.read_text()), phase)))
    # Validate the entire batch before creating output. Never overwrite evidence.
    args.output.mkdir(mode=0o700)
    for phase, request in projected:
        path = args.output / (phase + "-request.json")
        descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(descriptor, "w") as stream:
            json.dump(request, stream, ensure_ascii=False)


if __name__ == "__main__":
    main()
