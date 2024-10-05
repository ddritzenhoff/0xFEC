# experiment3.sh

# stop program in case of error, treat unset variables as an error, and print out executed commands
set -eux

if test "$#" -ne 1; then
	echo "Usage: experiment.sh DUT"
	echo "e.g. experiment.sh tallinn"
	exit
fi

DUT=$1

echo "free hosts"
pos allocations free -k "$DUT"

echo "allocate hosts"
pos allocations allocate "$DUT"

echo "load experiment variables"
pos allocations set_variables "$DUT" "./dut/dut.yml"

echo "set images"
pos nodes image "$DUT" debian-bullseye

echo "reboot experiment hosts..."
pos nodes reset "$DUT"

# NOTE: this script expects the binaries to have already been built.
echo "copy over binaries"
scp "./dut/client/client-linux-x86_64" root@"$DUT":~
scp "./dut/server/server-linux-x86_64" root@"$DUT":~
scp "./dut/server/1kB" root@"$DUT":/tmp/
scp "./dut/server/16kB" root@"$DUT":/tmp/
scp "./dut/server/65kB" root@"$DUT":/tmp/
scp "./dut/server/1MB" root@"$DUT":/tmp/

echo "setup hosts"
# Queue up the commands. They will be executed once booting is done.
# Capture the returned command ID of one command to wait for it finish.
SETUP_CMD_ID=$(pos commands launch --infile "./dut/setup.sh" --queued --name setup)

echo "waiting for setup to finish"
pos commands await $SETUP_CMD_ID
