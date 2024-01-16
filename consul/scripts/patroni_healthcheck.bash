#!/usr/bin/env bash
HOST=$1
PORT=$2

PORTS=("5432" "6432" "$PORT")

for P in "${PORTS[@]}"
do
  pg_isready -h "$HOST" -p "$P"
  if [ $? -ne 0 ];
  then
    exit 2
  fi
done
