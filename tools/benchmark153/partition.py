"""Exact signed partition for a bounded diagnostic; no gold or fixture rules."""
import itertools

def partition(ids,scores):
 ids=sorted(ids)
 if not 2<=len(ids)<=8 or len(set(ids))!=len(ids):raise ValueError('unit_budget_or_duplicate')
 pairs=list(itertools.combinations(ids,2))
 if set(scores)!={a+'__'+b for a,b in pairs}:raise ValueError('invalid_score_coverage')
 if any(type(v) is not int or v not in (-2,-1,0,1,2) for v in scores.values()):raise ValueError('invalid_score')
 best=None;winner=None;ties=0;states=0
 def search(labels):
  nonlocal best,winner,ties,states
  if len(labels)==len(ids):
   states+=1
   objective=sum(scores[a+'__'+b]*(1 if labels[i]==labels[j] else -1) for i,a in enumerate(ids) for j,b in enumerate(ids) if i<j)
   if best is None or objective>best:best,winner,ties=objective,tuple(labels),1
   elif objective==best:ties+=1
   return
  for label in range(max(labels)+2):search(labels+[label])
 search([0])
 if ties!=1:raise ValueError('ambiguous_optimum')
 result={u:f'G{winner[i]+1:03}' for i,u in enumerate(ids)}
 return result,dict(objective=best,states=states,optimum_count=ties)
