# Areas for improvement

- Making a plan
 - Specify number of host CPUs, 'overload' metric
 - Way to tell how much cpu a specific
 - Exploration of worker configuration space, 'naming' of some useful worker configs, 'naming' some useful worker mixes

- Running
 - Specify running it in a specific cpupool (to change the scheduler / cpu topology without rebooting or changing hostts)
 - Option to re-run tests which have already been run
 - Allow credit1 / credit2 to co-exist in the same 'plan'?

- Reports
 - Plot-able output (cvs? gnuplot?) with various 'queries'
 - Better summary name

- Data
 - Figure out how to collect actual CPU time received

- Worker
 - Add a "missed deadline" worker (low cpu usage but with hard deadlines)
 - Explore multi-vcpu options

- Robustness / Cleanups
 - Improve error handling paths
 - Abstract VM manipulation a bit better
 
# Performance metrics

 - Aggregate throughput: Sum of all throughputs
 - Fairness (Objective): Amount of CPU time recieved corresponds to ideal
 - Fairness (Subjective): Degradation of performance corresponds to ideal
 - ?: Minimum latency tolerance such that the scheduler can meet 99.9% of deadlines (?)

