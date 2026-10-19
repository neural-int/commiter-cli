"""Build a study-only test binary; keep the production four-file limit intact."""
import argparse,json,pathlib,subprocess
ROOT=pathlib.Path(__file__).resolve().parents[2]
p=argparse.ArgumentParser();p.add_argument('destination',type=pathlib.Path);a=p.parse_args()
a.destination.mkdir(parents=True,exist_ok=False)
source=ROOT/'internal/planning/three_phase.go'
text=source.read_text();assert text.count('const CandidateMaxFiles = 4')==1
replacement=a.destination/'three_phase.go'
replacement.write_text(text.replace('const CandidateMaxFiles = 4','const CandidateMaxFiles = 16'))
overlay=a.destination/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(source):str(replacement.resolve())}},indent=2)+'\n')
subprocess.run(['go','test','-overlay='+str(overlay.resolve()),'-c','-o',str((a.destination/'evaluation.test').resolve()),'./internal/cli'],cwd=ROOT,check=True)
print(a.destination.resolve())
