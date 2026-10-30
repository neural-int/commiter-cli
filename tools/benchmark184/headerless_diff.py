"""Remove only diff headers in an LLM projection; retain +/- and valid ranges."""
import copy
import json
from structured_diff import split_code
from multiline_view import json_literal


def project_content(content, keep_hunks=False):
    host = json.loads(content)
    projected = copy.deepcopy(host)
    files = projected.get("files")
    if files is None:
        files = [f for g in projected["groups"] for f in g["files"]]
    audits = []
    rendered = content
    for file in files:
        raw = file["raw_diff"]
        before = split_code(raw, file["change_blocks"])
        parts = raw.split("\n")
        lines = [s + "\n" for s in parts[:-1]] + ([parts[-1]] if parts[-1] else [])
        kept, mapping = [], {}
        in_hunk = False
        for number, line in enumerate(lines, 1):
            if line.startswith("@@ "):
                in_hunk = True
                if keep_hunks:
                    kept.append(line)
                continue
            if line.startswith("diff --git "):
                in_hunk = False
            if in_hunk:
                if not line.startswith(("+", "-", " ", "\\ No newline at end of file")):
                    raise ValueError("Unexpected hunk line")
                kept.append(line)
                mapping[number] = len(kept)
        new_raw = "".join(kept)
        # Replace exact field fragments to preserve all non-target serialized bytes.
        old_file = next(f for f in (host.get("files") or [x for g in host["groups"] for x in g["files"]]) if f["file_id"] == file["file_id"])
        fragment = json.dumps(old_file, ensure_ascii=False, separators=(",", ":"))
        fragment = fragment.replace("&", r"\u0026").replace("<", r"\u003c").replace(">", r"\u003e").replace("\u2028", r"\u2028").replace("\u2029", r"\u2029")
        assert rendered.count(fragment) == 1
        new_fragment = fragment.replace(json_literal(raw), json_literal(new_raw), 1)
        for block in file["change_blocks"]:
            old_range = block["diff_range"]
            new_range = {"start_line": mapping[old_range["start_line"]], "end_line": mapping[old_range["end_line"]]}
            old_text = '"diff_range":' + json.dumps(old_range, separators=(",", ":"))
            new_text = '"diff_range":' + json.dumps(new_range, separators=(",", ":"))
            assert new_fragment.count(old_text) == 1
            new_fragment = new_fragment.replace(old_text, new_text, 1)
            block["diff_range"] = new_range
        file["raw_diff"] = new_raw
        # Reinsert synthetic headers ONLY for validating code extraction equivalence.
        synthetic_blocks = copy.deepcopy(file["change_blocks"])
        for block in synthetic_blocks:
            block["diff_range"] = {k: v + 1 for k, v in block["diff_range"].items()}
        after = split_code(new_raw, file["change_blocks"]) if keep_hunks else split_code("@@ synthetic @@\n" + new_raw, synthetic_blocks)
        assert before == after
        rendered = rendered.replace(fragment, new_fragment, 1)
        audits.append({"file_id": file["file_id"], "removed_header_lines": len(lines)-len(kept), "hunk_headers_retained": keep_hunks, "code_and_diff_markers_identical": True, "diff_ranges_remapped": True})
    assert json.loads(rendered) == projected
    return rendered, audits
