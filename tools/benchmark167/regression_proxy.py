#!/usr/bin/python3
"""Local synthetic-input proxy: persist only hashes and numeric diagnostics.

static mode is a protocol double, never a model positive control. real mode
forwards the request bytes unchanged to an explicitly supplied product helper.
Generated text and helper stderr remain in memory and are never written out.
"""
import hashlib
import json
import os
import re
import subprocess
import sys
import time


def sha(data):
    return hashlib.sha256(data).hexdigest()


def encoded(value):
    return json.dumps(value, separators=(",", ":"), ensure_ascii=False).encode()


def main():
    raw = sys.stdin.buffer.read(8 * 1024 * 1024 + 1)
    if len(raw) > 8 * 1024 * 1024:
        raise ValueError("request limit")
    request = json.loads(raw)
    shape = request["schema"]
    profile = request["generation_profile"]
    if profile not in ("bounded-grouping", "bounded-category", "bounded-text"):
        raise ValueError("profile limit")
    record = {
        "profile": profile,
        "request_audit": {
            "request_bytes": len(raw.rstrip(b"\n")),
            "request_sha256": sha(raw.rstrip(b"\n")),
            "system_sha256": sha(request["messages"][0]["content"].encode()),
            "user_sha256": sha(request["messages"][1]["content"].encode()),
            "schema_sha256": sha(encoded(shape)),
        },
        "context_tokens": request["context_tokens"],
        "output_tokens": request["output_tokens"],
        "model": request["model"],
        "model_path_sha256": sha(request["model_path"].encode()),
        "model_inference": os.environ["COMMITER_167_PROXY_MODE"] == "real",
    }
    started = time.monotonic()
    if record["model_inference"]:
        proc = subprocess.run(
            ["/usr/bin/time", "-l", os.environ["COMMITER_167_PRODUCT_HELPER"]],
            input=raw, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=125,
            check=False,
        )
        match = re.search(rb"(?m)^\s*(\d+)\s+maximum resident set size\s*$", proc.stderr)
        record["helper_peak_rss_bytes"] = int(match[1]) if match else None
        if proc.returncode:
            raise ValueError("helper process failure")
        reply = json.loads(proc.stdout)
        forwarded = proc.stdout
    else:
        if profile == "bounded-grouping":
            answer = {k: "G001" for k in shape["required"]}
        elif profile == "bounded-category":
            answer = {k: {"type": "fix", "breaking_evidence_ref": "none"} for k in shape["required"]}
        else:
            answer = {k: {"scope": "sample", "summary": "correct the value"} for k in shape["required"]}
        reply = {"ok": True, "stop_reason": "completed", "generated_json": json.dumps(answer), "model": request["model"], "runtime": "mlx", "generation_profile": profile}
        if os.environ.get("COMMITER_167_PROXY_INSTRUMENTED") == "1":
            reply.update(benchmark_input_tokens=0, benchmark_output_tokens=0, benchmark_load_seconds=0, benchmark_ttft_seconds=0, benchmark_peak_bytes=0)
        forwarded = encoded(reply) + b"\n"
    record["wall_seconds"] = time.monotonic() - started
    record["stop"] = reply.get("stop_reason", "missing")
    if profile == "bounded-text":
        duplicate_keys = []
        def pairs(items):
            out = {}
            for key, val in items:
                if key in out:
                    duplicate_keys.append(True)
                out[key] = val
            return out
        try:
            result = json.loads(reply.get("generated_json", ""), object_pairs_hook=pairs)
            required = shape["required"]
            scopes = [v["scope"] for v in result.values()]
            summaries = [v["summary"] for v in result.values()]
            if not all(isinstance(s, str) for s in scopes + summaries):
                raise ValueError("text type")
            record["text_diagnostic"] = {
                "typed_decode_ok": True,
                "expected_groups": len(required), "returned_groups": len(result),
                "missing_groups": sum(k not in result for k in required),
                "unknown_groups": sum(k not in required for k in result),
                "duplicate_keys": len(duplicate_keys),
                "max_scope_characters": max(map(len, scopes), default=0),
                "max_summary_characters": max(map(len, summaries), default=0),
            }
        except (ValueError, TypeError, KeyError, AttributeError):
            record["text_diagnostic"] = {"typed_decode_ok": False, "duplicate_keys": len(duplicate_keys)}
    trace = os.environ["COMMITER_167_PROXY_TRACE"]
    with open(trace, "a", encoding="utf-8") as handle:
        handle.write(json.dumps(record, sort_keys=True) + "\n")
    sys.stdout.buffer.write(forwarded)


if __name__ == "__main__":
    try:
        main()
    except Exception:
        # No exception details: they could contain generated values.
        sys.stderr.write("regression proxy failed\n")
        sys.exit(1)
