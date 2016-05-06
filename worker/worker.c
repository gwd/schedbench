
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
#include <strings.h>

#define USEC 1000
#define MSEC 1000000
#define SEC  1000000000

#define PAGE_SIZE 4096

int64_t now(void) {
    int rc;
    struct timespec tp;

    rc = clock_gettime(CLOCK_MONOTONIC_RAW, &tp);

    assert(rc == 0);

    return tp.tv_sec * SEC + tp.tv_nsec;
}

void nsleep(int64_t wait_ns) {
    struct timespec tp;
    int rc;
    
    tp.tv_sec = wait_ns / SEC;
    tp.tv_nsec = wait_ns % SEC;

    rc = pselect(1, NULL, NULL, NULL, &tp, NULL);

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

struct queue_elem *eventqueue = NULL;
int64_t queue_max_delta = 0;

int eventqueue_insert(struct work_desc wd, uint64_t timer) {
    struct queue_elem *eq, **p;

    eq = malloc(sizeof(*eq));

    eq->wd = wd;
    eq->start_ns = now() + timer;
    
    for ( p = &eventqueue; *p && eq->start_ns > (*p)->start_ns; p = &((*p)->next) );

    eq->next = *p;
    *p = eq;
}

int eventqueue_loop(void) {
    while(eventqueue) {
        struct queue_elem *eq;
        
        int64_t delta_ns = eventqueue->start_ns - now();
        
        if (delta_ns > 0) {
            /* FIXME: Racy! If we get preempted here, we'll wait for the wrong amount of time */
            nsleep(delta_ns);
        }

        delta_ns = now() - eventqueue->start_ns;
        
        assert(delta_ns > 0);

        if ( delta_ns > queue_max_delta )
            queue_max_delta = delta_ns;

        eq = eventqueue;
        eventqueue = eventqueue->next;

        process_worker(eq->wd);

        free(eq);
    }
}

struct {
    char * data;
    int size;
    unsigned counter;
    unsigned index;
    int64_t start_time;
    uint64_t mops_done;

    // Reporting
    int64_t last_report;
    uint64_t last_report_mops;
} work = { 0 };

void report(void) {
    int64_t n = now();

    //printf("last_report: %lld now: %lld delta %lld\n", last_report, n, n-last_report);

    if ( (work.last_report == 0) || (n - work.last_report) > SEC ) {
        int64_t mops_per_second_total = work.mops_done * SEC / (n - work.start_time);
        int64_t mops_per_second_period = (work.mops_done - work.last_report_mops) * SEC / (n - work.last_report);
        printf("Total: %15lld %15lld This: %15lld %15lld Max delta: %15lld \n", work.mops_done, mops_per_second_total, work.mops_done - work.last_report_mops, mops_per_second_period, queue_max_delta);
        
        work.last_report = n;
        work.last_report_mops = work.mops_done;
    }
}

void worker_setup(void) {
    work.size = PAGE_SIZE;
    work.data = mmap(NULL, work.size, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0);

    assert(work.data != MAP_FAILED);

    fprintf(stderr, "Mapped memory at %p\n", work.data);

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

    report();

    eventqueue_insert(wd, wd.wait_nsec);
}


int main(int argc, char * argv[]) {
    // Setup workers
    worker_setup();
    
    // Insert worker(s)
    struct work_desc wd;

    wd.mops = 2;
    wd.wait_nsec = 20 * MSEC;

    eventqueue_insert(wd, 0);
    eventqueue_insert(wd, 0);
    eventqueue_insert(wd, 0);
    eventqueue_insert(wd, 0);
    eventqueue_insert(wd, 0);

    
    eventqueue_loop();
}
