"""Contiguous edit proposals with lossless, splittable atom children."""
import base64,hashlib
import hierarchy

def extract(before,after,file_id):
 h=hierarchy.extract(before,after,file_id)
 groups=[]
 for p in h['operations']:
  if groups and all(groups[-1][-1][k][1]==p[k][0] for k in ('old_span','new_span')):groups[-1].append(p)
  else:groups.append([p])
 out=[]
 for group in groups:
  p=dict(group[0]);old=[group[0]['old_span'][0],group[-1]['old_span'][1]];new=[group[0]['new_span'][0],group[-1]['new_span'][1]]
  p.update(old_span=old,new_span=new,before=base64.b64encode(before[slice(*old)]).decode(),after=base64.b64encode(after[slice(*new)]).decode(),children=[c for g in group for c in g['children']],line_children=[g['id'] for g in group],old_lines=[group[0]['old_lines'][0],group[-1]['old_lines'][1]],new_lines=[group[0]['new_lines'][0],group[-1]['new_lines'][1]],kind='contiguous-edit-proposal')
  p['id']=hashlib.sha256(f"{file_id}:{p['source_digest']}:{old}:{new}:contiguous".encode()).hexdigest()[:24];out.append(p)
 return dict(atoms=h['atoms'],operations=out)
