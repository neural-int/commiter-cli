"""Use #169's unchanged once-only runner and fail-closed validators."""
import sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
sys.path.insert(0,str(ROOT/'tools/benchmark169'))
import run as prior
prior.ROOT=ROOT
prior.DATA=ROOT/'docs/benchmarks/issue-170'
if __name__=='__main__':prior.main()
