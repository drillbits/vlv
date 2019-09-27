#!/usr/bin/env bash
set -euo pipefail
SCRIPTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -n "${DEBUG:-}" ]] && set -x
cd ${SCRIPTDIR} && cd ../

XC_ARCH=${XC_ARCH:-amd64}
XC_OS=${XC_OS:-darwin linux}

OUTDIR="$(pwd)/pkg"
mkdir -p ${OUTDIR}
rm -rf ${OUTDIR}/

cd cmd/vlv

gox \
    -os="${XC_OS}" \
    -arch="${XC_ARCH}" \
    -output="${OUTDIR}/{{.OS}}_{{.Arch}}/{{.Dir}}"
