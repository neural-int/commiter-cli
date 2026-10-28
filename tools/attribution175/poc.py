"""Conservative content correspondence experiment, never a production adapter.
Exact directional edit content is evidence of correspondence, not causality.
No history reads, LLM calls, Git writes or third-party dependencies.
"""
import collections,json,time,tracemalloc

def classify(current, edits, worktree='W'):
 candidates=[e['turn'] for e in edits if e['worktree']==worktree and e['path']==current['path'] and e['status']=='completed' and e['before']==current['before'] and e['after']==current['after']]
 ids=sorted(set(candidates))
 return {'status':'unique_content_correspondence' if len(ids)==1 else ('ambiguous' if ids else 'unmatched'),'turns':ids}

def edit(turn='T1',before='A',after='B',path='file',worktree='W',status='completed'):
 return dict(turn=turn,before=before,after=after,path=path,worktree=worktree,status=status)

def cases():
 return [
 ('single',edit(),[edit()],['T1']),
 ('separate_files',edit(),[edit(),edit('T2',path='other')],['T1']),
 ('repeat_file_distinct',edit(),[edit(),edit('T2',before='X',after='Y')],['T1']),
 ('sequential_composition',edit(after='C'),[edit(),edit('T2',before='B',after='C')],['T1','T2']),
 ('cancelled',edit(before='X',after='Y'),[edit(),edit('T2',before='B',after='A')],[]),
 ('overwrite',edit(after='C'),[edit(),edit('T2',before='B',after='C')],['T1','T2']),
 ('same_line_repeated',edit(),[edit(),edit('T2',before='B',after='A'),edit('T3')],['T3']),
 ('multiple_sessions_same_change',edit(),[edit(),edit('T2')],['T1','T2']),
 ('other_worktree',edit(),[edit(worktree='OTHER')],[]),
 ('manual_change',edit(after='C'),[edit()],[]),
 ('rename_path_unresolved',edit(path='renamed'),[edit()],['T1']),
 ('missing_log',edit(),[],[]),
 ('failed_edit',edit(),[edit(status='failed')],[]),
 ('addition',edit(before='',after='new'),[edit(before='',after='new')],['T1']),
 ('deletion',edit(before='old',after=''),[edit(before='old',after='')],['T1']),
 ]

def evaluate():
 started=time.perf_counter();tracemalloc.start();rows=[]
 for name,current,edits,gold in cases():
  got=classify(current,edits);baseline=sorted(set(e['turn'] for e in edits if e['worktree']=='W' and e['path']==current['path'] and e['status']=='completed'))
  strong=set(got['turns']) if got['status']=='unique_content_correspondence' else set()
  rows.append(dict(case=name,reference=gold,path_candidates=baseline,content=got,path_false_candidates=len(set(baseline)-set(gold)),strong_false_candidates=len(strong-set(gold)),strong_missed_turns=len(set(gold)-strong)))
 _,peak=tracemalloc.get_traced_memory();tracemalloc.stop()
 return dict(cases=rows,elapsed_ms=(time.perf_counter()-started)*1000,python_tracemalloc_peak_bytes=peak,limits='synthetic known references; no hunk parser, composition, rename tracking or causal proof; ambiguous candidates are not strong attribution')

if __name__=='__main__':print(json.dumps(evaluate(),indent=2))
