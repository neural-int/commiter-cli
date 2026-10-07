"""Whole-file Git operation reconstruction, isolated synthetic repositories."""
import pathlib
import tempfile
import unittest
from evaluate import git

class WholeOperations(unittest.TestCase):
    def test_create_rename_mode_symlink_delete(self):
        with tempfile.TemporaryDirectory(prefix='benchmark151-ops-') as tmp:
            root=pathlib.Path(tmp)
            git(root,'init','-q')
            (root/'a').write_bytes(b'x\n')
            git(root,'add','a')
            self.assertEqual(git(root,'show',':a'),b'x\n')
            (root/'a').rename(root/'b')
            git(root,'add','--','a','b')
            self.assertEqual(git(root,'show',':b'),b'x\n')
            self.assertEqual(git(root,'ls-files','--','a'),b'')
            git(root,'update-index','--chmod=+x','b')
            self.assertTrue(git(root,'ls-files','-s','b').startswith(b'100755 '))
            (root/'link').symlink_to('b')
            git(root,'add','link')
            self.assertTrue(git(root,'ls-files','-s','link').startswith(b'120000 '))
            self.assertEqual(git(root,'show',':link'),b'b')
            (root/'b').unlink()
            git(root,'add','--','b')
            self.assertEqual(git(root,'ls-files','--','b'),b'')

if __name__ == '__main__': unittest.main()
