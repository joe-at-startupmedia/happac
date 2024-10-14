#!/usr/bin/env bash
HOST=$1
PORT=$2
/usr/bin/nc "$HOST" "$PORT" --recv-only | grep -q up
if [ $? -ne 0 ];
then
  exit 2
fi
