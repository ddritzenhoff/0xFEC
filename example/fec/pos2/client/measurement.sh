#!/bin/bash

# ./client-linux-x86_64 -file=output.txt -round="1" -host="22.22.22.22" -endpoint="1MB"

SCHEME="$(pos_get_variable scheme)"
ENDPOINT="$(pos_get_variable endpoint)"
ROUNDS="$(pos_get_variable rounds)"
for ((i = 0; i < ROUNDS; i += 1)); do
    ./client-linux-x86_64 -round="$i" -host="22.22.22.22" -endpoint="$ENDPOINT" -scheme="$SCHEME" >> condition.txt 2>&1
done