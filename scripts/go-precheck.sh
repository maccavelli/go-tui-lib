#!/usr/bin/env bash
# The pre-add check is scripts/go-precheck.py. This file runs it, with the
# same arguments and exit status, because the machine-wide agent gate
# (~/.agents/hooks/lib/precommit-checks.sh) runs ./scripts/go-precheck.sh by
# that name (docs/decisions/0017-MADR-python-repository-scripts.md item 5).
exec "${PYTHON:-python3}" "$(dirname "$0")/go-precheck.py" "$@"
