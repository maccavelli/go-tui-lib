"""A fake ssh: logs its arguments, and runs the remote command's program,
read from stdin, in this Python, with $VERIFY_FAKE_HOME as its home and
$VERIFY_FAKE_TMP as its temporary directory."""

import json
import os
import subprocess
import sys

args = sys.argv[1:]
with open(os.environ["VERIFY_FAKE_LOG"], "a", encoding="utf-8") as log:
    log.write(json.dumps({"tool": "ssh", "args": args}) + "\n")
home, tmp = os.environ["VERIFY_FAKE_HOME"], os.environ["VERIFY_FAKE_TMP"]
env = dict(os.environ, HOME=home, USERPROFILE=home, TMPDIR=tmp, TEMP=tmp, TMP=tmp)
sys.exit(subprocess.run([sys.executable, "-"], stdin=sys.stdin, env=env, check=False).returncode)
