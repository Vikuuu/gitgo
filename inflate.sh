#!/bin/bash
python3 -c "import sys, zlib; sys.stdout.buffer.write(zlib.decompress(sys.stdin.buffer.read()))"

