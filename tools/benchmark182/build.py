"""Build a study-only test binary; production source is never modified."""
import argparse,json,pathlib,subprocess
ROOT=pathlib.Path(__file__).resolve().parents[2]
FILE_GUARD='''\tif len(ids) > CandidateMaxFiles {
\t\treturn result, exitcode.New(exitcode.LLM, "provisional three-phase planner supports at most four selected files; use single-pass for larger changes")
\t}
'''

def planner_overlay(text,unlimited=False):
    if unlimited:
        assert text.count(FILE_GUARD)==1,'planner guard changed; inspect before building'
        return text.replace(FILE_GUARD,'')
    assert text.count('const CandidateMaxFiles = 4')==1
    return text.replace('const CandidateMaxFiles = 4','const CandidateMaxFiles = 16')

def live_overlay(text):
    replacements={
        'len(paths) == 0 || len(paths) > 4 || paths[0] == ""':'len(paths) == 0 || paths[0] == ""',
        'one to four explicit paths required':'at least one explicit path required',
        'len(snapshot.Changes) == 0 || len(snapshot.Changes) > 4':'len(snapshot.Changes) == 0',
        'expected one to four real changes':'expected at least one real change',
    }
    for before,after in replacements.items():
        assert text.count(before)==1,'live guard changed; inspect before building'
        text=text.replace(before,after)
    return text

def build(destination,unlimited=False):
    destination=destination.resolve()
    destination.mkdir(parents=True,exist_ok=False)
    source=ROOT/'internal/planning/three_phase.go'
    replacement=destination/'three_phase.go'
    replacement.write_text(planner_overlay(source.read_text(),unlimited))
    mapping={str(source):str(replacement)}
    if unlimited:
        live=ROOT/'internal/cli/attributed_history_live_test.go'
        target=destination/'attributed_history_live_test.go'
        target.write_text(live_overlay(live.read_text()))
        mapping[str(live)]=str(target)
        test_source=ROOT/'internal/planning/three_phase_test.go'
        test_target=destination/'three_phase_test.go'
        test_target.write_text(test_source.read_text()+(ROOT/'tools/benchmark182/testdata/unlimited_files_test.go.txt').read_text())
        mapping[str(test_source)]=str(test_target)
    overlay=destination/'overlay.json'
    overlay.write_text(json.dumps({'Replace':mapping},indent=2)+'\n')
    (destination/'build-contract.json').write_text(json.dumps({
        'selected_file_limit':None if unlimited else 16,
        'live_etl_file_limit':None if unlimited else 4,
        'production_source_modified':False,
        'context_ipc_and_cycle_limits_changed':False,
        'generation_calls':0},indent=2)+'\n')
    if unlimited:
        subprocess.run(['go','test','-overlay='+str(overlay),'./internal/planning',
                        '-run=^(TestIssue182UnlimitedFilesReachClient|TestThreePhaseScalesRawDiffContextWithinConfiguredCeiling)$','-count=1'],cwd=ROOT,check=True)
    subprocess.run(['go','test','-overlay='+str(overlay),'-c','-o',str(destination/'evaluation.test'),'./internal/cli'],cwd=ROOT,check=True)
    print(destination)

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('destination',type=pathlib.Path)
    parser.add_argument('--unlimited-files',action='store_true',help='remove count guards in the study planner and live ETL test; retain all other limits')
    args=parser.parse_args()
    build(args.destination,args.unlimited_files)
