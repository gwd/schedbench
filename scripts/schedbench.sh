#!/bin/bash
#
# Do full runs (plan + run) for both credit and credit2 schedulers
#
# Usage:
#  ./schedbench.sh [n]
#
# Where [n] is the run number
#
if [[ -n "$1" ]] ; then
	run="$1."
fi

for sched in credit2 credit; do
	prefix="${run}${sched}."
	file="${run}${sched}.bench"
	echo "$1 Testing $sched (prefix $prefix)"
	./schedbench-prep.sh $sched || break
	xl cpupool-list schedbench
	./schedbench -f ${file} plan || break
	(./schedbench -f ${file} run | tee ${prefix}log) || break
	#./schedbench -f ${file} -v 1 report > ${prefix}out || break
done
grep early ${run}*.log
