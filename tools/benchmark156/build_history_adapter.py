"""Compile unchanged H23/production benchmark sources at their immutable revision."""
import io,pathlib,subprocess,tarfile
ROOT=pathlib.Path(__file__).resolve().parents[2]
REV='e88f61bb06d8e609d5ad7fbcde1a40281f997e89'
DEST=pathlib.Path('/private/tmp/benchmark156-history-source')
DEST.mkdir(exist_ok=True)
raw=subprocess.run(['git','archive',REV,'go.mod','go.sum','internal','tools/benchmark149'],cwd=ROOT,capture_output=True,check=True).stdout
with tarfile.open(fileobj=io.BytesIO(raw)) as archive:archive.extractall(DEST,filter='data')
main=DEST/'tools/benchmark149/main.go'
main.write_text(main.read_text().replace('func main() {','func originalBenchmarkMain() {',1))
(DEST/'tools/benchmark149/history_adapter.go').write_bytes((ROOT/'tools/benchmark156/history_adapter.go.txt').read_bytes())
subprocess.run(['go','build','-o','/private/tmp/benchmark156-history','./tools/benchmark149'],cwd=DEST,check=True)
print(REV)
