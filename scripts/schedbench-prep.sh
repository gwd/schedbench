#!/bin/bash
#
# Prepare a system to run schedbench:
# - Unplug all dom0 vcpus to be used by the schedbench cpupool
# - Set up the schedbench cpupool
#
# Usage:
#  ./schedbench-prep.sh [scheduler]
#
# Where [scheduler] is `credit` or `credit2`
#
FIRST_CPU=1
LAST_CPU=15
extra="cpus=\"$FIRST_CPU-$LAST_CPU\""
#extra="cpus=\"8,10,12,14\""
if [[ -n "$1" ]] ; then
	extra="$extra sched=\"$1\""
fi
for cpu in $(seq $FIRST_CPU $LAST_CPU) ; do
	echo "0" > /sys/devices/system/cpu/cpu${cpu}/online || exit 1
done
xl cpupool-destroy schedbench
xl cpupool-cpu-remove Pool-0 ${FIRST_CPU}-${LAST_CPU}
echo xl cpupool-create cpupool.cfg $extra
xl cpupool-create cpupool.cfg $extra
