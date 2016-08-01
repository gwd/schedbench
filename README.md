# Summary

sched-sim is a "microbenchmark" for scheduling.  It is designed both
to be used as an actual benchmark (to measure the effects of
schedulers on artificial workloads in a repeatable manner) and to
provide "background competition" for testing the effect of schedulers
on real-work workloads or other benchmarks.

# Motivation

The basic problem with testing schedulers is that schedulers are only
really needed when there's not enough cpu to go around.  So to really
test the effectiveness of a scheduler, you need to see how workloads
compete with each other.

But normal workloads -- even benchmarks -- typically vary how much cpu
they use over time; which means that when you run several workloads
together, how they happen to align can have a dramatic difference on
how they end up performing -- much more so than the inherent
performance of the scheduler itself.

Moreover, different aspects of the workload can get lost in the noise,
making it difficult to see how changes in the scheduler affect a
single aspect of scheduling.

The basic idea of schedbench is to have artificial workloads whose cpu
utilization properties an be parametrized to isolate specific aspects
of workloads, and which are constant over time; and then to have a
controller which will start up a number of them, collect their
performance results, and then can report on the results.

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

The xen code also builds against libxl.  At the moment this is
hardcoded to the directory where I have libxl built on my dev box.  On
your system you can either edit `Makefile/controller`, putting your
built xen path in XENLIB_PATH, or remove both XENLIB_PATH and the two
runes which reference it (leaving the library names intact).

# Quick command reference

To use schedsim, run the following four commands on your Xen host in
order:

- `schedbench plan`: Initialize "plan" for the benchmark in test.bench

- `schedbench run`: Run the runs in test.bench which haven't been completed yet

- `schedbench report`: Collate the data and give a report

# Future work

This is definitely a work-in-progress.  My initial goal is just to get
a basic framework up to speed so that others can add to it.

For short-term work items, see [TODO.md]
