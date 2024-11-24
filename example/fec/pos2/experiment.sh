# stop program in case of error, treat unset variables as an error, and print out executed commands
set -eux

if test "$#" -ne 2; then
	echo "Usage: experiment.sh DUT"
	echo "e.g. experiment.sh tallinn tartu"
	exit
fi

DIR=$(dirname $(realpath $0))
CLIENT=$1
SERVER=$2

echo "free hosts"
pos allocations free -k "$CLIENT"
pos allocations free -k "$SERVER"

echo "allocate hosts"
pos allocations allocate "$CLIENT"
pos allocations allocate "$SERVER"

echo "load experiment variables"
pos allocations set_variables "$CLIENT" "$DIR/client/client.yml"
pos allocations set_variables "$SERVER" "$DIR/server/server.yml"

echo "set images"
pos nodes image "$CLIENT" debian-bullseye
pos nodes image "$SERVER" debian-bullseye

echo "reboot experiment hosts..."
CLIENT_REBOOT=$(pos nodes reset "$CLIENT" --non-blocking)
SERVER_REBOOT=$(pos nodes reset "$SERVER" --non-blocking)

pos commands await $CLIENT_REBOOT
pos commands await $SERVER_REBOOT

# NOTE: this script expects the binaries to have already been built.
echo "copy over files"
scp "$DIR/client/client-linux-x86_64" root@"$CLIENT":~
scp "$DIR/server/server-linux-x86_64" root@"$SERVER":~
scp "$DIR/server/1kB" root@"$SERVER":/tmp/
scp "$DIR/server/10kB" root@"$SERVER":/tmp/
scp "$DIR/server/50kB" root@"$SERVER":/tmp/
scp "$DIR/server/1MB" root@"$SERVER":/tmp/
scp "$DIR/server/10MB" root@"$SERVER":/tmp/
scp "$DIR/server/20MB" root@"$SERVER":/tmp/
scp "$DIR/server/30MB" root@"$SERVER":/tmp/

echo "setup hosts"
# Queue up the commands. They will be executed once booting is done.
# Capture the returned command ID of one command to wait for it finish.
pos commands launch --infile "$DIR/server/setup.sh" --queued --name server_setup "$SERVER"
# give the server a second to startup
sleep 1
CLIENT_SETUP=$(pos commands launch --infile "$DIR/client/setup.sh" --queued --name client_setup "$CLIENT")

pos commands await $CLIENT_SETUP
# give the interfaces time to come up
sleep 5
echo "execute measurements"
MEASUREMENT=$(pos commands launch --infile "$DIR/client/measurement.sh" --queued --name measurement "$CLIENT")

pos commands await $MEASUREMENT

echo "all done"