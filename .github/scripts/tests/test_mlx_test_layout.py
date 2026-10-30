"""Verify Metal staging before tests without compiling Swift or running MLX."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "test-mlx-helper.sh"


class MetalTestLayout(unittest.TestCase):
    def test_stages_root_and_each_bundle_before_running_tests(self):
        with tempfile.TemporaryDirectory(prefix="mlx test layout ") as temporary:
            root = Path(temporary)
            scripts = root / ".github/scripts"
            scripts.mkdir(parents=True)
            shutil.copy2(SCRIPT, scripts / SCRIPT.name)
            (root / "dist").mkdir()
            library = b"fixture-metal-library"
            (root / "dist/mlx.metallib").write_bytes(library)
            products = root / "Products with spaces"
            products.mkdir()
            binaries = [products / f"{name}.xctest/Contents/MacOS/{name}"
                        for name in ("PackageTests", "Other Tests")]
            for binary in binaries:
                binary.parent.mkdir(parents=True)
                binary.write_text("test bundle fixture")
                binary.chmod(0o755)
            commands = root / "commands"
            commands.mkdir()
            swift = commands / "swift"
            swift.write_text('''#!/usr/bin/env python3
import os, pathlib, sys
root = pathlib.Path(os.environ["LAYOUT_ROOT"])
products = root / "Products with spaces"
if "--show-bin-path" in sys.argv:
    print(products)
elif sys.argv[1] == "test":
    assert "--skip-build" in sys.argv
    expected = (root / "dist/mlx.metallib").read_bytes()
    for directory in [products] + [p.parent for p in products.glob("*.xctest/Contents/MacOS/*") if p.name != "mlx.metallib"]:
        assert (directory / "mlx.metallib").read_bytes() == expected, directory
    (root / "swift-tests-ran").touch()
''')
            swift.chmod(0o755)
            go = commands / "go"
            go.write_text("#!/bin/sh\ntest -f \"$LAYOUT_ROOT/swift-tests-ran\"\n")
            go.chmod(0o755)
            environment = dict(os.environ, LAYOUT_ROOT=str(root),
                               PATH=str(commands) + os.pathsep + os.environ["PATH"])
            result = subprocess.run(["bash", str(scripts / SCRIPT.name)],
                                    env=environment, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertTrue((root / "swift-tests-ran").exists())
            self.assertEqual(result.stdout.count("MLX test library:"), 3)


if __name__ == "__main__":
    unittest.main()
