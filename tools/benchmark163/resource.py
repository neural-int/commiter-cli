"""macOS process/resource observations, kept separate from MLX allocation."""
import ctypes
import json
import os
import pathlib
import subprocess
import threading
import time

FIELDS = '''user_time system_time pkg_idle_wkups interrupt_wkups pageins wired_size resident_size phys_footprint proc_start_abstime proc_exit_abstime child_user_time child_system_time child_pkg_idle_wkups child_interrupt_wkups child_pageins child_elapsed_abstime diskio_bytesread diskio_byteswritten cpu_time_qos_default cpu_time_qos_maintenance cpu_time_qos_background cpu_time_qos_utility cpu_time_qos_legacy cpu_time_qos_user_initiated cpu_time_qos_user_interactive billed_system_time serviced_system_time logical_writes lifetime_max_phys_footprint instructions cycles billed_energy serviced_energy interval_max_phys_footprint runnable_time'''.split()


class UsageV4(ctypes.Structure):
    _fields_ = [('uuid', ctypes.c_uint8 * 16)] + [('ri_' + name, ctypes.c_uint64) for name in FIELDS]


def usage(pid):
    library = ctypes.CDLL('/usr/lib/libproc.dylib', use_errno=True)
    function = library.proc_pid_rusage
    function.argtypes = [ctypes.c_int, ctypes.c_int, ctypes.c_void_p]
    function.restype = ctypes.c_int
    data = UsageV4()
    if function(pid, 4, ctypes.byref(data)) != 0:
        return {'available': False, 'errno': ctypes.get_errno()}
    return {'available': True, 'footprint_bytes': data.ri_phys_footprint,
            'rss_bytes': data.ri_resident_size,
            'lifetime_peak_footprint_bytes': data.ri_lifetime_max_phys_footprint}


def system():
    result = {}
    for name, args in [('vm_stat', ['vm_stat']), ('swap', ['sysctl', 'vm.swapusage']),
                       ('pressure', ['memory_pressure', '-Q'])]:
        try:
            p = subprocess.run(args, capture_output=True, timeout=10)
            result[name] = {'exit': p.returncode, 'stdout': p.stdout.decode(errors='replace'),
                            'stderr': p.stderr.decode(errors='replace')}
        except subprocess.TimeoutExpired:
            result[name] = {'error': 'timeout'}
    return result


def observed_call(command, payload, timeout, interval=0.1):
    before = system()
    started = time.monotonic()
    proc = subprocess.Popen(command, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            env=dict(os.environ, HF_HUB_OFFLINE='1', TRANSFORMERS_OFFLINE='1'))
    samples = []
    stop = threading.Event()

    def sample():
        while not stop.is_set():
            samples.append(dict(usage(proc.pid), elapsed_seconds=time.monotonic() - started))
            stop.wait(interval)

    worker = threading.Thread(target=sample)
    worker.start()
    timed_out = False
    try:
        stdout, stderr = proc.communicate(payload, timeout=timeout)
    except subprocess.TimeoutExpired:
        timed_out = True
        proc.kill()
        stdout, stderr = proc.communicate()
    finally:
        elapsed = time.monotonic() - started
        stop.set(); worker.join()
    after = system()
    valid = [row for row in samples if row['available']]
    return {'exit': proc.returncode, 'timed_out': timed_out, 'stdout': stdout.decode(errors='replace'),
            'stderr': stderr.decode(errors='replace'), 'total_seconds': elapsed,
            'sample_interval_seconds': interval, 'samples': samples,
            'sampled_peak_footprint_bytes': max((r['footprint_bytes'] for r in valid), default=None),
            'observed_lifetime_peak_footprint_bytes': max((r['lifetime_peak_footprint_bytes'] for r in valid), default=None),
            'sampled_peak_rss_bytes': max((r['rss_bytes'] for r in valid), default=None),
            'system_before': before, 'system_after': after,
            'mlx_active_bytes': None, 'mlx_peak_bytes': None, 'mlx_cache_bytes': None,
            'metal_allocation_bytes': None, 'ttft_seconds': None,
            'resource_qualification': 'not_assessed',
            'limitation': 'API lifetime peak and sampled observations may miss the final exit interval; MLX/Metal/TTFT need helper instrumentation'}


if __name__ == '__main__':
    import argparse
    import sys
    parser = argparse.ArgumentParser()
    parser.add_argument('--preflight-output', required=True)
    args = parser.parse_args()
    self_usage = usage(os.getpid())
    child = observed_call([sys.executable, '-c', 'import time; print("resource preflight"); time.sleep(0.3)'], b'', 5)
    # This only tests measurement and lifecycle; it is neither a model call nor a resource GO.
    result = {'model_calls': 0, 'self': self_usage, 'child': child,
              'pass': self_usage['available'] and child['exit'] == 0
                      and len(child['samples']) >= 2 and child['sampled_peak_footprint_bytes'] is not None,
              'semantic_or_model_resource_qualification': 'not_evaluated'}
    with pathlib.Path(args.preflight_output).open('x') as out:
        json.dump(result, out, ensure_ascii=False, indent=2); out.write('\n')
    print(json.dumps({'pass': result['pass'], 'child_samples': len(child['samples']), 'model_calls': 0}))
    raise SystemExit(0 if result['pass'] else 1)
