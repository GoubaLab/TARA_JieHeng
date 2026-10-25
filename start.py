"""Compatibility launcher for the compiled Go service; no Python service starts."""
from pathlib import Path
import os
import sys

ROOT=Path(__file__).resolve().parent
if __name__=='__main__':
    binary=ROOT/'next'/'bin'/'jieheng'
    if not binary.is_file():
        raise SystemExit('请先运行 ./next/tools/build-go.sh 构建 Go 服务。')
    os.execv(str(binary),[str(binary),'--root',str(ROOT),*sys.argv[1:]])
