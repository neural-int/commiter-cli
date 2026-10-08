"""Lossless structural proposals; parents never impose semantic must-link."""
from evaluate import extract as line_extract
from inline import extract as atom_extract

def extract(before,after,file_id):
 parents=line_extract(before,after,file_id)
 atoms=atom_extract(before,after,file_id)
 children={p['id']:[] for p in parents}
 for atom in atoms:
  matches=[p for p in parents if all(p[k][0]<=atom[k][0]<=atom[k][1]<=p[k][1] for k in ('old_span','new_span'))]
  if len(matches)!=1:raise ValueError('ambiguous_parent_mapping')
  children[matches[0]['id']].append(atom['id'])
 if any(not c for c in children.values()):raise ValueError('empty_parent')
 return dict(atoms=atoms,operations=[dict(p,children=children[p['id']],splittable=True) for p in parents])
