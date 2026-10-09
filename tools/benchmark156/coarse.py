"""A2: line ownership first; bounded inline refinement is an explicit request."""
import base64
import pathlib
from planner import coverage, MAX_FILES, MAX_TOTAL_UNITS
from evaluate import extract as line_extract
from inline import extract as inline_extract


def prepare(files):
    if not 1 <= len(files) <= MAX_FILES:
        raise ValueError('file_budget')
    ids, paths, units = set(), set(), []
    for f in files:
        path = pathlib.PurePosixPath(f['path'])
        if (f['id'] in ids or f['path'] in paths or path.is_absolute() or
                any(p in ('..', '.git') for p in path.parts) or str(path) != f['path']):
            raise ValueError('invalid_file_mapping')
        ids.add(f['id']); paths.add(f['path'])
        if f.get('mode', '100644') not in ('100644', '100755'):
            raise ValueError('unsupported_mode')
        before = base64.b64decode(f['before_b64'], validate=True)
        after = base64.b64decode(f['after_b64'], validate=True)
        own = line_extract(before, after, f['id'])
        if not own:
            raise ValueError('no_changed_units')
        coverage(before, after, own)
        units.extend(own)
    if len(units) > MAX_TOTAL_UNITS:
        raise ValueError('total_unit_budget')
    return units


def refine(file):
    """No semantic inference. A budget rejection retains existing coarse ownership."""
    before = base64.b64decode(file['before_b64'], validate=True)
    after = base64.b64decode(file['after_b64'], validate=True)
    coarse = prepare([file])
    try:
        fine = inline_extract(before, after, file['id'])
    except ValueError as exc:
        if str(exc) not in ('unit_budget', 'inline_byte_budget'):
            raise
        return coarse, str(exc)
    coverage(before, after, fine)
    return fine, None
