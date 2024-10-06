# ./client-linux-x86_64 -file=output.txt -round="1" -host="22.22.22.22" -endpoint="1MB"

ENDPOINT="$(pos_get_variable endpoint)"
ROUNDS="$(pos_get_variable rounds)"
for ROUND in $ROUNDS; do
    ./client-linux-x86_64 -file=output.txt -round="$ROUND" -host="22.22.22.22" -endpoint="$ENDPOINT"
done