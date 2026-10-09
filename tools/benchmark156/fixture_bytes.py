"""Byte-exact diagnostic fixture storage; logical paths and manifest hashes stay fixed."""
import base64
import hashlib
import json

SUFFIX = '.bytes.json'

def read(path):
    if path.is_file():
        return path.read_bytes()
    container = path.with_name(path.name + SUFFIX)
    record = json.loads(container.read_text())
    if record['logical_filename'] != path.name:
        raise ValueError('fixture_logical_path')
    data = base64.b64decode(record['bytes_b64'], validate=True)
    if hashlib.sha256(data).hexdigest() != record['source_sha256']:
        raise ValueError('fixture_source_digest')
    return data

def paths(root):
    result=[]
    for p in root.rglob('*'):
        if not p.is_file():
            continue
        path=str(p.relative_to(root))
        result.append(path[:-len(SUFFIX)] if path.endswith(SUFFIX) else path)
    if len(set(result)) != len(result):
        raise ValueError('duplicate_fixture_storage')
    return sorted(result)
