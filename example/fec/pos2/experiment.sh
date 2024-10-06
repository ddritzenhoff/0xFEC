# experiment3.sh

# stop program in case of error, treat unset variables as an error, and print out executed commands
set -eux

if test "$#" -ne 2; then
	echo "Usage: experiment.sh DUT"
	echo "e.g. experiment.sh tallinn tartu"
	exit
fi

CLIENT=$1
SERVER=$2

echo "free hosts"
pos allocations free -k "$CLIENT"
pos allocations free -k "$SERVER"

echo "allocate hosts"
pos allocations allocate "$CLIENT"
pos allocations allocate "$SERVER"

echo "load experiment variables"
pos allocations set_variables "$CLIENT" "./client/client.yml"

echo "set images"
pos nodes image "$CLIENT" debian-bullseye
pos nodes image "$SERVER" debian-bullseye

echo "reboot experiment hosts..."
pos nodes reset "$CLIENT" --non-blocking
pos nodes reset "$SERVER"

# this is to make up for the potential race condition in which the server finishes booting before the client.
sleep 5

# NOTE: this script expects the binaries to have already been built.
echo "copy over binaries"
scp "./client/client-linux-x86_64" root@"$CLIENT":~
scp "./server/server-linux-x86_64" root@"$SERVER":~
scp "./server/1kB" root@"$SERVER":/tmp/
scp "./server/16kB" root@"$SERVER":/tmp/
scp "./server/65kB" root@"$SERVER":/tmp/
scp "./server/1MB" root@"$SERVER":/tmp/

echo "setup hosts"
# Queue up the commands. They will be executed once booting is done.
# Capture the returned command ID of one command to wait for it finish.
CLIENT_SETUP_CMD_ID=$(pos commands launch --infile "./client/setup.sh" --queued --name client_setup)
pos commands launch --infile "./server/setup.sh" --queued --name server_setup

echo "waiting for setup to finish"
pos commands await $CLIENT_SETUP_CMD_ID

echo "all done"