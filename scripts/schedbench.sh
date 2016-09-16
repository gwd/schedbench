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
    run="$1"
else
    run="test"
fi

file="${run}.bench"

if ! [[ -e $file ]] ; then
    echo "Can't find benchmark file $file"
    exit 1
fi

./schedbench -f ${file} plan || exit 1

for sched in credit2 credit; do
	prefix="${run}.${sched}"
	echo "$1 Testing $sched (prefix $prefix)"
	./schedbench-prep.sh $sched || break
	xl cpupool-list schedbench
	(./schedbench -f ${file} run | tee ${prefix}.log) || break
done
grep early ${run}*.log

./schedbench -f ${file} -v 1 report > ${run}.txt || break
