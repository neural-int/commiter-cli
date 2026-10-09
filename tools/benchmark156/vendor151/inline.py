"""Bounded UTF-8 edit atoms; no semantic labels, paths or model input."""
import base64
import difflib
import hashlib
from evaluate import extract as line_extract, MAX_UNITS

MAX_INLINE_BYTES = 4096


def offsets(chars):
    result = [0]
    for c in chars:
        result.append(result[-1] + len(c.encode('utf-8')))
    return result


def extract(before, after, file_id, symbols=()):
    result = []
    for parent in line_extract(before, after, file_id, symbols):
        old = base64.b64decode(parent['before'])
        new = base64.b64decode(parent['after'])
        if not old or not new:
            result.append(parent)
            continue
        try:
            a, b = old.decode('utf-8'), new.decode('utf-8')
        except UnicodeDecodeError:
            result.append(parent)
            continue
        if max(len(old), len(new)) > MAX_INLINE_BYTES:
            raise ValueError('inline_byte_budget')
        ao, bo = offsets(a), offsets(b)
        for tag, i, j, k, l in difflib.SequenceMatcher(None, a, b, autojunk=False).get_opcodes():
            if tag == 'equal':
                continue
            for index in range(max(j-i, l-k)):
                oi, ni = min(i+index, j), min(k+index, l)
                oe, ne = min(oi+1, j), min(ni+1, l)
                u = dict(parent)
                u['old_span'] = [parent['old_span'][0]+ao[oi], parent['old_span'][0]+ao[oe]]
                u['new_span'] = [parent['new_span'][0]+bo[ni], parent['new_span'][0]+bo[ne]]
                u['before'] = base64.b64encode(old[ao[oi]:ao[oe]]).decode()
                u['after'] = base64.b64encode(new[bo[ni]:bo[ne]]).decode()
                u['kind'] = tag
                u['granularity'] = 'utf8-edit-atom'
                u['id'] = hashlib.sha256(f"{file_id}:{u['source_digest']}:{u['old_span']}:{u['new_span']}:inline".encode()).hexdigest()[:24]
                result.append(u)
                if len(result) > MAX_UNITS:
                    raise ValueError('unit_budget')
    if len(result) > MAX_UNITS:
        raise ValueError('unit_budget')
    return result
