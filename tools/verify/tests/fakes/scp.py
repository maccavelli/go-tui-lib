"""A fake scp: logs its arguments, and copies SRC to HOST:DEST as
$VERIFY_FAKE_HOME/DEST."""

import json
import os
import shutil
import sys
from pathlib import Path

args = sys.argv[1:]
with open(os.environ["VERIFY_FAKE_LOG"], "a", encoding="utf-8") as log:
    log.write(json.dumps({"tool": "scp", "args": args}) + "\n")
src, dest = args[-2], args[-1]
_host, _, name = dest.partition(":")
shutil.copyfile(src, Path(os.environ["VERIFY_FAKE_HOME"]) / name)
