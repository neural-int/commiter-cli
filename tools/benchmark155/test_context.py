"""Checks context boundaries, not model semantic quality."""
import json
import pathlib
import subprocess
import unittest
from context import select

OBSERVER=pathlib.Path('/tmp/issue155-context-observer')

class ContextTest(unittest.TestCase):
    def observe(self, files):
        return json.loads(subprocess.run([str(OBSERVER)],input=json.dumps({'files':files}).encode(),capture_output=True,check=True).stdout)

    def test_one_step_calls_and_exact_source_refs(self):
        before='package sample\nfunc Target() int {return Adapter()}\nfunc Adapter() int {return Core()}\nfunc Core() int {return 1}\n'
        after=before.replace('return Adapter()', 'return Adapter()+1')
        files=[dict(id='F1',path='a.go',before=before,after=after)]
        out=select(files,self.observe(files))
        symbols={x['symbol'] for x in out['context']}
        self.assertIn('Target',symbols)
        self.assertIn('Adapter',symbols)
        self.assertNotIn('Core',symbols)
        adapters=[x for x in out['context'] if x['symbol']=='Adapter']
        self.assertEqual(len(adapters),1)
        self.assertEqual(len(adapters[0]['source_refs']),2)

    def test_ambiguous_call_not_silently_selected(self):
        files=[dict(id='F1',path='a.go',before='package sample\nfunc Target() int {return Core()}\n',after='package sample\nfunc Target() int {return Core()+1}\n'),dict(id='F2',path='b.go',before='package sample\nfunc Core() int {return 1}\n',after='package sample\nfunc Core() int {return 1}\n'),dict(id='F3',path='c.go',before='package sample\nfunc Core() int {return 2}\n',after='package sample\nfunc Core() int {return 2}\n')]
        out=select(files,self.observe(files))
        self.assertFalse(any(x['symbol']=='Core' for x in out['context']))
        self.assertTrue(all(x['status']=='ambiguous_declaration' for x in out['unresolved_calls']))
        facts=self.observe(files)
        facts['evidence'][0]['text']='forged'
        with self.assertRaisesRegex(ValueError,'source_mapping'):
            select(files,facts)

if __name__=='__main__':unittest.main()
