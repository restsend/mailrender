#!/bin/sh
ADDR=0.0.0.0:${PORT:=8000}
SIZELIMIT=${SIZELIMIT:=50}
AUTHOR=${AUTHOR:='https://github.com/restsend/mailrender'}
STORE=${STORE:='/tmp/mailrender'}
./mailrender -author ${AUTHOR} -http ${ADDR} -m ${SIZELIMIT} -store ${STORE}