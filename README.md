# Overview

sched-sim is a "microbenchmark" for scheduling.  It is designed both
to be used as an actual benchmark (to measure the effects of
schedulers on artificial workloads in a repeatable manner) and to
provide "background competition" for testing the effect of schedulers
on real-work workloads or other benchmarks.

There are three levels:

 - ''Workers'' These are individual scheduling units (either VMs or
   processes) that perform some artificial work and report their
   results.

 - ''Benchmark Run'' This is a daemon that takes parameters for a
   benchmark run, starts up the appropriate number of workers in the
   specified configuration(s), and collects the results.

 - ''Benchmark'' In the same daemon, this will execute a series of
   benchmark runs in order to collect specific information about a
   scheduler's performance

 - ''Analysis'' Analyze the results to give information about the benchmark

# Quick command reference

- `schedsim plan`: Initialize "plan" for the benchmark in test.bench

- `schedsim run`: Run the runs in test.bench which haven't been completed yet

- `schedsim report`: Collate the data and give a report

# Build notes

The Xen version is compiled against rumpkernels.  This works fairly
well in general, but the NetBSD kernel's nanosleep system call
quantizes sleep times to the system timer tick rate, which defaults to
100; meaning the shortest amount of time you can sleep is 10ms -- far
too long for the kinds of testing we want to do.  The rumpkernel guys
were kind enough to give me a hack to override the NetBSD system call
with one which would actually sleep for the exact time requested
without quantization.

So to build, you need to download my branch:

`https://github.com/gwd/rumprun out/nanosleep-fix/v1`

Then see the [Rumprun Build
Tutorial](https://github.com/rumpkernel/wiki/wiki/Tutorial:-Building-Rumprun-Unikernels)
tutorial for building rumprun.

Then after building, add the rumprun binary directory to your path, thus:

`export PATH="$PATH:/path/to/rumprun.git/rumprun/bin/"`

NB also that the rumprun build system seems to build things in
different sub-directories based on the branch name you're on (if the
branch is not `master`); so you may want to add a symbolic link from
`rumprun.git/rumprun` to whatever directory it ends up making.

# Future work

This is definitely a work-in-progress.  My initial goal is just to get
a basic framework up to speed so that others can add to it.
