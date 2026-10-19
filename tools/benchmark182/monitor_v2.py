"""H180-compatible OS sampler using Mach CPU counters, never top."""
import ctypes,datetime,importlib.util,json,pathlib,threading,time
_spec=importlib.util.spec_from_file_location('legacy_sampler',pathlib.Path(__file__).with_name('run.py'))
_base=importlib.util.module_from_spec(_spec);_spec.loader.exec_module(_base)
_original=_base.command
_lib=ctypes.CDLL('/usr/lib/libSystem.B.dylib');_lib.mach_host_self.restype=ctypes.c_uint
_lib.host_statistics.argtypes=[ctypes.c_uint,ctypes.c_int,ctypes.POINTER(ctypes.c_int),ctypes.POINTER(ctypes.c_uint)]
_last=None
def command(argv):
 global _last
 if argv[0]!='top':return _original(argv)
 ticks=(ctypes.c_int*4)();count=ctypes.c_uint(4)
 code=_lib.host_statistics(_lib.mach_host_self(),3,ticks,ctypes.byref(count))
 if code:return {'exit':code,'stdout':'','stderr':'host_statistics failed'}
 current=[v & 0xffffffff for v in ticks];delta=[(a-b)&0xffffffff for a,b in zip(current,_last)] if _last else None;_last=current
 return {'exit':0,'stdout':json.dumps({'ticks':current,'busy_percent':100*(1-delta[2]/sum(delta)) if delta and sum(delta) else None}),'stderr':''}
_base.command=command
sample=_base.sample
class Monitor:
 def __init__(self,out):self.out=out;self.done=threading.Event();self.failed=False;self.last_valid=time.monotonic();self.active=None
 def loop(self):
  while not self.done.is_set():
   r=sample();valid='gpu' in r and all(r[k].get('exit')==0 for k in ['cpu','vm','pressure','swap','power','thermal']);r['valid']=valid
   if valid:self.last_valid=time.monotonic()
   if time.monotonic()-self.last_valid>15 or any(p['rss_kib']>=12*1024*1024 for p in r.get('helper_processes',[])):
    self.failed=True
    if self.active:self.active.terminate()
   with self.out.open('a') as f:f.write(json.dumps(r)+'\n')
   self.done.wait(max(0,5-r['collection_seconds']))
 def start(self):self.thread=threading.Thread(target=self.loop);self.thread.start()
 def stop(self):self.done.set();self.thread.join()
