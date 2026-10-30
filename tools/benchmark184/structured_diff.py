"""Project zero-context Git change blocks for LLM display, not host storage."""
import copy
import hashlib
import json


def split_code(raw_diff, blocks):
    parts = raw_diff.split("\n")
    lines = [part + "\n" for part in parts[:-1]]
    if parts[-1]:
        lines.append(parts[-1])
    changed = set()
    no_newline = set()
    in_hunk = False
    for index, line in enumerate(lines):
        if line.startswith("@@ "):
            in_hunk = True
            continue
        if line.startswith("diff --git "):
            in_hunk = False
        if not in_hunk:
            continue
        if line.startswith("\\ No newline at end of file"):
            if index == 0 or index - 1 not in changed:
                raise ValueError("Unexpected no-newline marker")
            no_newline.add(index - 1)
        elif line.startswith(("+", "-")):
            changed.add(index)
        elif line.startswith(" "):
            raise ValueError("Context lines require a separate representation; refusing to discard")
        else:
            raise ValueError("Unsupported hunk content")
    coverage = set()
    output = []
    for block in blocks:
        start = block["diff_range"]["start_line"] - 1
        end = block["diff_range"]["end_line"]
        if not 0 <= start < end <= len(lines):
            raise ValueError("Invalid block range")
        added, deleted = [], []
        for index in range(start, end):
            line = lines[index]
            if line.startswith("\\ No newline at end of file"):
                continue
            if index not in changed or index in coverage:
                raise ValueError("Block contains metadata, context, or overlaps")
            coverage.add(index)
            code = line[1:]
            if index in no_newline and code.endswith("\n"):
                code = code[:-1]
            (added if line[0] == "+" else deleted).append(code)
        if not added and not deleted:
            raise ValueError("Empty change block")
        projected = copy.deepcopy(block)
        del projected["diff_range"]
        projected["deleted_code"] = "".join(deleted)
        projected["added_code"] = "".join(added)
        output.append(projected)
    if coverage != changed:
        raise ValueError("Changed lines lost from projection")
    return output


def project_payload(payload):
    result = copy.deepcopy(payload)
    files = result.get("files")
    if files is None:
        files = [file for group in result["groups"] for file in group["files"]]
    audit = []
    for file in files:
        raw = file.pop("raw_diff")
        blocks = file["change_blocks"]
        projected = split_code(raw, blocks)
        file["change_blocks"] = projected
        for original, new in zip(blocks, projected):
            audit.append({
                "file_id": file["file_id"], "block_id": new["block_id"],
                "host_diff_range": original["diff_range"],
                "host_raw_diff_sha256": hashlib.sha256(raw.encode()).hexdigest(),
                "added_code_sha256": hashlib.sha256(new["added_code"].encode()).hexdigest(),
                "deleted_code_sha256": hashlib.sha256(new["deleted_code"].encode()).hexdigest(),
            })
    return result, audit


def project_content(content):
    payload, audit = project_payload(json.loads(content))
    rendered = json.dumps(payload, ensure_ascii=False, separators=(",", ":"))
    rendered = rendered.replace("&", r"\u0026").replace("<", r"\u003c").replace(">", r"\u003e")
    rendered = rendered.replace("\u2028", r"\u2028").replace("\u2029", r"\u2029")
    return rendered, audit
