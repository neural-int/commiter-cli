"""Validate local refinement before scoring; no Git mutation or gold input."""
import hashlib,pathlib,sys
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent.parent/'benchmark151'))
import contiguous
from evaluate import reconstruct

def validate(snapshots,plan):
 operations={};atoms={};before={}
 for fid,(old,new) in sorted(snapshots.items()):
  h=contiguous.extract(old,new,fid);before[fid]=old
  for a in h['atoms']:
   if a['id'] in atoms:raise ValueError('duplicate_atom')
   atoms[a['id']]=a
  for p in h['operations']:operations[p['id']]=p
  if reconstruct(old,h['atoms'],[a['id'] for a in h['atoms']])!=new:raise ValueError('snapshot_reconstruction')
 if not isinstance(plan,dict) or set(plan)!=set(operations):raise ValueError('parent_coverage')
 groups=[]
 for pid,p in sorted(operations.items()):
  decision=plan[pid]
  if not isinstance(decision,dict):raise ValueError('invalid_decision')
  action=decision.get('action')
  if action=='unresolved':raise ValueError('unresolved')
  if action=='accept':
   if set(decision)!={'action'}:raise ValueError('invalid_accept')
   local=[p['children']]
  elif action=='refine':
   if set(decision)!={'action','groups'}:raise ValueError('invalid_refine')
   local=decision['groups']
   if not isinstance(local,list) or len(local)<2 or any(not isinstance(g,list) or not g or any(type(a) is not str for a in g) for g in local):raise ValueError('invalid_groups')
   flat=[a for g in local for a in g]
   if len(flat)!=len(set(flat)) or set(flat)!=set(p['children']):raise ValueError('atom_coverage')
  else:raise ValueError('invalid_action')
  for group in local:
   ids=sorted(group);units=[a for a in atoms.values() if a['id'] in set(ids)]
   # Source mapping and byte payloads come from re-extraction, never model output.
   reconstruct(before[p['file']],units,ids)
   groups.append(dict(id=hashlib.sha256(('|'.join(ids)).encode()).hexdigest()[:24],parent=pid,file=p['file'],atoms=ids))
 if not 1<=len(groups)<=8:raise ValueError('refined_unit_budget')
 return sorted(groups,key=lambda g:g['id'])
