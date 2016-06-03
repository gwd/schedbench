
// Basic workloads:
//  - Do X amount of work, wait for future work to continue; multiple queues
//    - Most bulk transfer / kernel build benchmarks are of this nature
//  - Deadline workload: Work comes in at rate X, must be handled within Y time
//  - Latency workload:
//
// How work is generated
#include <stdlib.h>
#include <stdio.h>
#include <time.h>
#include <assert.h>
#include <stdint.h>
#include <sys/select.h>
#include <sys/mman.h>
#include <string.h>
#include <strings.h>

#define USEC 1000
#define MSEC 1000000
#define SEC  1000000000

#define PAGE_SIZE 4096

int64_t now(void) {
    int rc;
    struct timespec tp;

    rc = clock_gettime(CLOCK_MONOTONIC, &tp);

    assert(rc == 0);

    return tp.tv_sec * SEC + tp.tv_nsec;
}

void nsleep(int64_t wait_ns) {
    struct timespec tp;
    int rc;
    
    tp.tv_sec = wait_ns / SEC;
    tp.tv_nsec = wait_ns % SEC;

    //rc = pselect(1, NULL, NULL, NULL, &tp, NULL);
    nanosleep(&tp, NULL);

    assert(rc == 0);
}

// Work description:
// - Do mops million operations
struct work_desc {
    uint64_t mops;
    uint64_t wait_nsec;
    // uint64_t deadline_nsec;
};

void process_worker(struct work_desc wd);

struct queue_elem {
    int64_t start_ns;
    //uint64_t deadline_ns;
    struct work_desc wd;
    struct queue_elem *next;
};

struct {
    char * data;
    int size;
    unsigned counter;
    unsigned index;
    int64_t start_time;
    uint64_t mops_done;

    int64_t queue_max_delta;

    // Reporting
    int64_t report_interval_ms;
    int64_t last_report;
    uint64_t last_report_mops;
} work = { 0 };


struct queue_elem *eventqueue = NULL;

int eventqueue_insert(struct work_desc wd, uint64_t timer) {
    struct queue_elem *eq, **p;

    eq = malloc(sizeof(*eq));

    eq->wd = wd;
    eq->start_ns = now() + timer;
    
    for ( p = &eventqueue; *p && eq->start_ns > (*p)->start_ns; p = &((*p)->next) );

    eq->next = *p;
    *p = eq;
}

void report(void);

int eventqueue_loop(void) {
    while(eventqueue) {
        struct queue_elem *eq;
        
        report();
    
        int64_t delta_ns = eventqueue->start_ns - now();
        
        if (delta_ns > 0) {
            /* FIXME: Racy! If we get preempted here, we'll wait for the wrong amount of time */
            nsleep(delta_ns);
        }

        delta_ns = now() - eventqueue->start_ns;
        
        assert(delta_ns > 0);

        if ( delta_ns > work.queue_max_delta )
            work.queue_max_delta = delta_ns;

        eq = eventqueue;
        eventqueue = eventqueue->next;

        process_worker(eq->wd);

        free(eq);
    }
}

void report(void) {
    int64_t n = now();
    
    if ( (work.last_report == 0)
         || (n - work.last_report) > work.report_interval_ms * MSEC ) {

        printf("{ \"Now\":%lld, \"Mops\":%llu, \"MaxDelta\":%llu }\n",
               n, work.mops_done, work.queue_max_delta);
        fflush(stdout);

        work.queue_max_delta = 0;

        work.last_report = n;
    }
}

void worker_setup(void) {
    work.size = PAGE_SIZE;
    work.data = mmap(NULL, work.size, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0);
    
    assert(work.data != MAP_FAILED);
    
    printf("Mapped memory at %p\n", work.data);
    fflush(stdout);
    
    bzero(work.data, work.size);
    
    work.start_time = now();

}

void process_worker(struct work_desc wd) {
    int i;
    
    // Write sequentially to data for mops operations
    for ( i=0; i < wd.mops * 1000000 ; i++) {
        work.index++;
        if (work.index > work.size / sizeof(int))
            work.index -= work.size / sizeof(int);
        (*((volatile int *)work.data+work.index)) &= work.counter++;
    }
    work.mops_done += wd.mops;
    
    eventqueue_insert(wd, wd.wait_nsec);
}

/* report_interval [report_ms]
   burnwait [mops] [wait_nsec] */
int main(int argc, char *argv[]) {
    int i;

    printf("argc: %d\n", argc);
    
    work.report_interval_ms = 1000;
    
    worker_setup();
    
    for(i=1; i<argc; i++) {
        if(!strcmp(argv[i], "report_interval")) {
            i++;
            if(!(i<argc)) {
                fprintf(stderr, "Not enough aguments for report_ms");
                exit(1);
            }
            work.report_interval_ms=strtoul(argv[i], NULL, 0);
        } else if (!strcmp(argv[i], "burnwait")) {
            struct work_desc wd;
            
            i++;
            if(!(i<argc)) {
                fprintf(stderr, "Not enough aguments for burnwait");
                exit(1);
            }
            wd.mops=strtoul(argv[i], NULL, 0);

            i++;
            if(!(i<argc)) {
                fprintf(stderr, "Not enough aguments for burnwait");
                exit(1);
            }
            wd.wait_nsec=strtoul(argv[i], NULL, 0);

            eventqueue_insert(wd, 0);
        } else {
            fprintf(stderr, "Unknown toplevel command: %s\n", argv[i]);
            exit(1);
        }
    }

    fflush(stdout);
    printf("START JSON\n");
    fflush(stdout);

    eventqueue_loop();

}
