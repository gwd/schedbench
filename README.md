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

To use `schedbench`, first copy and modify the included
`sample.bench`, and modify as appropriate.  Then run the following
four commands on your Xen host in order:

- `schedbench [-t template] [-f filename ] plan`: Initialize "plan"
  for the benchmark in benchmark file (default: `test.bench`).  If
  `template` is given, then a new plan will be made in `filename`
  which is identical to the one found in `template`.

- `schedbench [-f filename ] run`: Run the runs in benchmark file
  which haven't been completed yet

- `schedbench [-f filename ] [-v N ] report`: Collate the data and
  give a text report to stdout with verbosity `N`

- `schedbench [-f filename ] htmlreport`: Collate the data into a
  self-contained html document to `stdout`

`schedbench` is compiled statically, so the report / plan side should
run even on a system that doesn't have libxl installed (such as,
perhaps, your dev box).

# Modifying `sample.bench`

This is sample.bench:

    {
        "Input": {
            "WorkerPresets": {
                "A": { "Args": [ "burnwait", "70", "200000" ] },
                "B": { "Args": [ "burnwait", "10", "300000",
                        "burnwait", "20", "300000",
                        "burnwait", "10", "300000",
                        "burnwait", "10", "300000",
                        "burnwait", "10", "300000",
                        "burnwait", "10", "300000",
                        "burnwait", "30", "300000" ] }
            },
            "SimpleMatrix": {
                "Schedulers": [
                    "credit",
                    "credit2"
                ],
                "Workers": [ "A", "B" ],
                "Count": [ 1, 2, 4, 8, 16 ]
            }
        },
        "WorkerType": 1,
        "RunConfig": {
            "Pool": "schedbench",
    	    "Cpus": [ 12, 13, 14, 15 ]
        }
    }

The `WorkerPresets` section defines two workers, `A` and `B`.  See
below for a description of the workers.

The `SimpleMatrix` section tells `schedbench` to make a plan that
includes `baseline` runs for both `A` and `B`, and then runs that
include `Count` numbers of both `A` and `B`.  In the default case, it
will run one `A` and one `B`, then two `A` workers and two `B`
workers, then four `A` workers and four `B` workers, and so on.

The `Schedulers` list tells SimpleMatrix to add the listed schedulers
into its matrix; i.e., run all the tests with `credit`, then run all the
tests with `credit2`.

`RunConfig` Contains global configuration inherited by each run if
none are given.  If you specify a `Pool` name, it will try to run all
the workers in that pool.  If no name is given, it defaults to
`Pool-0`.  You can also specify `Cpus`, which is a list of cpus that
should be in the target pool.

When `schedbench` runs each test, it will check to see if the
specified `RunConfig` configuration items match the pool to run the
VMs in.  If everything matches, then it runs the test.

If things don't match, and `schedbench` is able to create the pool,
then it creates the pool with the new parameters.  `schedbench` is
able to create a pool if the pool is not `Pool-0` and if the `Cpus` are
specified.

If the parameters don't match, and `schedbench` is not able to create
the pool, then it skips the test.

This allows you either to let `schedbench` manage all the cpupool
operations (by specifying a `Pool` other than the default pool, and
`Cpus`), or to use a pool but manage it yourself (by specifying `Pool`
but no cpupool), or to simply use the default cpupool (by not
specifying `Pool`).

Note that if you don't specify `Pool`, but you do specify `Cpus`,
`schedbench` will check to make sure that the default pool contains
the selected cpus and skip the tests if not.  Specifying `Cpus` when
using the default cpupool is recommended to make sure that you haven't
forgotten anything.

# Future work

This is definitely a work-in-progress.  My initial goal is just to get
a basic framework up to speed so that others can add to it.

For short-term work items, see [TODO.md]

# Understanding the results

## Workers

The workers take a queue of "work" and do it.  At the moment the only
type of work is called "burnwait", which takes two parameters:
kilo-ops and wait time time in nanoseconds.  Each burnwait, when run
will will:

 1. "Burn" cpu by doing the specified number of memory operations on a
 page of memory
 2. Queue up another iteration of the current worker for NOW()+
 specified wait time.

And you can specify several of these; work is done in a sequential
non-preemptible fashion.

The key thing about #2 from the scheduling perspective is that adding
itself to the queue happens *after* doing the specified amount of
work.  Which means that if the guest is preempted (or delayed) from
doing the work, it will take longer for the next bit of work to start.

Say that your burn time was 100us, and your sleep time was 100us.  You
start at t0us, burn for 100us (now at t100us), then set a timer for
100us and sleep; you wake up at t200us, do 100us of work (now at
t300us), then set a timer for 100us and sleep.

Say now that after running for 50us, you get preempted for 100us.  Now
t0 you start burning; t50us you're interrupted; t150us you start
burning again; t200 you start burning and set a timer for 100us,
waking up at 300us.

Having several "threads" going on mitigates this: the particular
thread you interrupted is delayed, but the timers of the threads which
have already completed aren't.  So a worker configured to have a
single 50us burn cycle will be much more sensitive to scheduling
decisions than a worker configured with five 10us burn cycles.

## The test

Each benchmark does a range of 'runs'; each 'run' starts a fixed
number of workers, collects their throughput metrics, measures how
much cpu they're getting.  Workers report their total throughput about
every second; actual throughput is measured by

I'm running this on kodo2, an Intel with 2 sockets, 8 cores, and
hyperthreading enabled (so 16 logical cpus).  And I'm running the test
in a cpupool with 4 threads, with dom0 in a separate pool.

## The report

Data is collected and reported in averages:

 - Each worker reports cumulative time and cumulative operations once
   per second; controller collects cumulative cpu utilization once per
   second

 - This is used to calculate througput (ops / second) and utilization
   (%) for individual report "windows"

 - Maximum, minimum, and stddev of throughput per each window
 
 - Average throughput for the whole run is collected for each worker
 
 - For each type of worker, that average is then fed into stats
   acrross all workers -- an "average average", "max / min average"
   and "stdev average"

A good scheduler will be *consistent within a run* -- it will have a
small range between max and min hand have a small standard deviation.
It will also be *consistent between workers* -- workers of the same
type should have similar averages i.e., (max-min) and stdev of
averages between workers should be small.

Additionally, a scheduler should be "fair" between different kinds of
workloads.  Not all workloads will want to run 100% of the time; but a
workload should get either 1) the maximum fair share if time it's
entitled to, or 2) as much as it wants, whichever is less.  How much
it 'wants' we can tell by running it on an empty system.

So suppose a workload by itself wants 50%, and it's run on a 4-cpu
system with 5 other workloads.  Its "fair share" of cpu is 67% (4 / 6
= 0.67), so it should ideally get 50% of the cpu.  If instead it's run
on a system with 10 other workloads, its "fair share" is 40% (4 / 10 =
0.4), so it should ideally get 40%.

Each run will have a report that looks something like this:

    == RUN 4a+4b ==
    Set 0:  kHZ 2261062 burnwait 70 200000
    Set 1:  kHZ 2261062 burnwait 10 300000 burnwait 20 300000 burnwait 10 300000 burnwait 10 300000 burnwait 10 300000 burnwait 10 300000 burnwait 30 300000

     set   ttotal  tavgavg   tstdev  tavgmax  tavgmin  ttotmax  ttotmin   utotal  uavgavg   ustdev  uavgmax  uavgmin  utotmax  utotmin
       0 297265.05 74316.26   199.38 74631.67 74134.55 76383.90 73778.25     1.99     0.50     0.00     0.50     0.49     0.50     0.44
       1 301779.89 75444.97   852.14 76621.31 74213.47 84898.68 65949.73     1.96     0.49     0.01     0.50     0.48     0.52     0.39


The first is the title -- 4a+4b means there are 4 of worker type 'a',
and 4 of worker type 'b'; so in a 4-cpu pool, this is 2x
overcommitted; the "fair share" of each workload will be 50%.

The second is the configuration of each worker.  "Set 0" is worker
'a', which has a single "thread" tha twill burn for 70 kilo-ops, then
wait for 200us (200,000 nanoseconds).  "Set 1" is worker "b", which
has 7 workers, which burn for various amounts (10, 20, 10, 10, 10, 10,
and 30 kops respecpively) and each of which sleep for 300us.

Next we have a slew of summary data:

 - **set**: The set number
 
 - **ttotal**: Througput total for this set.  This is the *throughputs*
   (i.e., kops/sec) of all the workers of the set added together.

 - **tavgavg**: Average of the average throughputs of each individual
   worker in this set
 
 - **tstdev**, **tavgmax**, **tavgmin**: Standard devation, maximum, and minimum
   of average throughputs

 - **ttotmax**, **ttotmin**: Maximum and minimum throughput seen in any
   "window" of any worker of this set

 - **utotal**: Total average utilization of all workers in this set.  This
   is the individual utilizations of all workers added together

 - **uavgavg**, **ustdev**, **uavdmax**, **uavdmin**: Average, stdev, max, and min of
   the average utilization of all the workers in the set.

 - **utotmax**, **utotmin**: Maximum and minimum utilization seen in any
   "window" of any worker in this set.

Scheduler performance in this case is not bad overall: The system is
fully loaded (total utilization around 4.0); the aggregate throughput
of both types of workers is very close to 300Mops/sec; range of
averages pretty tight.

