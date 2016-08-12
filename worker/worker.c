/*
 * Copyright (C) 2016 George W. Dunlap, Citrix Systems UK Ltd
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation; either version 2 of the
 * License only.
 *
 * This program is distributed in the hope that it will be useful, but
 * WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * General Public License for more details.
 * 
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA
 * 02110-1301, USA.
 */
    
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

static inline uint64_t rdtsc(void)
{
    uint32_t low, high;

    __asm__ __volatile__("rdtsc" : "=a" (low), "=d" (high));

    return ((uint64_t)high << 32) | low;
}

uint64_t start_tsc, kHZ=0;

void init_clock(void) {
    start_tsc = rdtsc();
}

void set_kHZ(uint64_t val) {
    kHZ = val;
}

int64_t now(void) {
    int rc;
    struct timespec tp;

    if (kHZ == 0) {
        fprintf(stderr, "now(): kHZ uninitialized!\n");
        while(1);
    }

    return (rdtsc()-start_tsc) * MSEC / kHZ;
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
// - Do kops thousand operations
struct work_desc {
    uint64_t kops;
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
    uint64_t kops_done;

    int64_t queue_max_delta;

    // Reporting
    int64_t report_interval_ms;
    int64_t next_report;
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

void report(int64_t n);

int eventqueue_loop(void) {
    while(eventqueue) {
        struct queue_elem *eq;
        int64_t n = now();
        
        report(n);
    
        int64_t delta_ns = eventqueue->start_ns - n;
        
        while (delta_ns > 0) {
            /* FIXME: Racy! If we get preempted here, we'll wait for the wrong amount of time */
            nsleep(delta_ns);
            n = now();
            // Deal gracefully with time jitter due to moving across sockets
            delta_ns = eventqueue->start_ns - n;
        }

        delta_ns = n - eventqueue->start_ns;

        assert(delta_ns >= 0);

        if ( delta_ns > work.queue_max_delta )
            work.queue_max_delta = delta_ns;

        eq = eventqueue;
        eventqueue = eventqueue->next;

        process_worker(eq->wd);

        free(eq);
    }
}

void report(int64_t n) {
    if ( (work.next_report == 0)
         || n > work.next_report ) {

        printf("{ \"Now\":%lld, \"Kops\":%llu, \"MaxDelta\":%llu }\n",
               n, work.kops_done, work.queue_max_delta);
        fflush(stdout);

        work.queue_max_delta = 0;

        if (!work.next_report) 
            work.next_report = n;
        
        work.next_report += work.report_interval_ms * MSEC;
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
    for ( i=0; i < wd.kops * 1000 ; i++) {
        work.index++;
        if (work.index > work.size / sizeof(int))
            work.index -= work.size / sizeof(int);
        (*((volatile int *)work.data+work.index)) &= work.counter++;
    }
    work.kops_done += wd.kops;
    
    eventqueue_insert(wd, wd.wait_nsec);
}

/* report_interval [report_ms]
   burnwait [kops] [wait_nsec] */
int main(int argc, char *argv[]) {

    init_clock();
    
    int i;

    printf("argc: %d\n", argc);
    
    work.report_interval_ms = 1000;
    
    for(i=1; i<argc; i++) {
        if(!strcmp(argv[i], "kHZ")) {
            uint64_t setkHZ;
            i++;
            if(!(i<argc)) {
                fprintf(stderr, "Not enough aguments for kHZ");
                exit(1);
            }
            setkHZ = strtoull(argv[i], NULL, 0);
            printf("Setting kHZ to %llu\n", setkHZ);
            set_kHZ(setkHZ);
        } else if(!strcmp(argv[i], "report_interval")) {
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
            wd.kops=strtoul(argv[i], NULL, 0);

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

    if(kHZ == 0) {
        fprintf(stderr, "kHZ not set!\n");
        while(1);
    }
    
    worker_setup();
    
    fflush(stdout);
    printf("START JSON\n");
    fflush(stdout);

    eventqueue_loop();

}
