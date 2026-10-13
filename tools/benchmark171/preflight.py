"""Use unchanged #169 native tokenize-only preflight, without generation."""
import sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
sys.path.insert(0,str(ROOT/'tools/benchmark169'))
import preflight as prior
prior.ROOT=ROOT;prior.DATA=ROOT/'docs/benchmarks/issue-171'
if __name__=='__main__':prior.main()
