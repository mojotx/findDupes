#define _XOPEN_SOURCE 500   /* for nftw */
#define _DARWIN_C_SOURCE     /* for strdup on macOS */

#include <errno.h>
#include <fcntl.h>
#include <stdarg.h>
#include <ftw.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <time.h>
#include <unistd.h>

#include <CommonCrypto/CommonDigest.h>

#define SHA256_LEN CC_SHA256_DIGEST_LENGTH /* 32 */
#define HT_BUCKETS 65521                  /* prime, for hash table */
#define MAX_FD     64                     /* max open fds for nftw */

/* ------------------------------------------------------------------ */
/* Hash-map: SHA-256 digest -> list of file paths + file size         */
/* ------------------------------------------------------------------ */

typedef struct path_node {
    char              *path;
    struct path_node  *next;
} path_node;

typedef struct entry {
    unsigned char  digest[SHA256_LEN];
    off_t          size;
    int            count;
    path_node     *paths;
    struct entry  *next;          /* chaining within bucket */
} entry;

static entry *buckets[HT_BUCKETS];
static int    verbose;

/* ------------------------------------------------------------------ */
/* Timestamped logging helper                                         */
/* ------------------------------------------------------------------ */

static void log_ts(const char *level, const char *fmt, ...)
    __attribute__((format(printf, 2, 3)));

static void log_ts(const char *level, const char *fmt, ...)
{
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    struct tm tm;
    localtime_r(&ts.tv_sec, &tm);

    char tbuf[64];
    strftime(tbuf, sizeof tbuf, "%Y-%m-%dT%H:%M:%S", &tm);
    fprintf(stderr, "%s.%03ld %s ", tbuf, ts.tv_nsec / 1000000, level);

    va_list ap;
    va_start(ap, fmt);
    vfprintf(stderr, fmt, ap);
    va_end(ap);

    fputc('\n', stderr);
}

/* FNV-1a over the 32-byte digest to pick a bucket */
static unsigned ht_index(const unsigned char *digest)
{
    uint32_t h = 2166136261u;
    for (int i = 0; i < SHA256_LEN; i++) {
        h ^= digest[i];
        h *= 16777619u;
    }
    return h % HT_BUCKETS;
}

static entry *ht_find(const unsigned char *digest)
{
    unsigned idx = ht_index(digest);
    for (entry *e = buckets[idx]; e; e = e->next)
        if (memcmp(e->digest, digest, SHA256_LEN) == 0)
            return e;
    return NULL;
}

static entry *ht_insert(const unsigned char *digest, const char *path, off_t size)
{
    entry *e = ht_find(digest);
    if (!e) {
        e = calloc(1, sizeof *e);
        if (!e) { perror("calloc"); exit(1); }
        memcpy(e->digest, digest, SHA256_LEN);
        e->size = size;
        unsigned idx = ht_index(digest);
        e->next = buckets[idx];
        buckets[idx] = e;
    }

    path_node *pn = malloc(sizeof *pn);
    if (!pn) { perror("malloc"); exit(1); }
    pn->path = strdup(path);
    if (!pn->path) { perror("strdup"); exit(1); }
    pn->next = e->paths;
    e->paths = pn;
    e->count++;
    return e;
}

/* ------------------------------------------------------------------ */
/* SHA-256 helper                                                     */
/* ------------------------------------------------------------------ */

static int sha256_file(const char *path, unsigned char out[SHA256_LEN])
{
    int fd = open(path, O_RDONLY);
    if (fd < 0) {
        log_ts("ERR", "error opening %s: %s", path, strerror(errno));
        return -1;
    }

    CC_SHA256_CTX ctx;
    CC_SHA256_Init(&ctx);

    unsigned char buf[65536];
    ssize_t n;
    while ((n = read(fd, buf, sizeof buf)) > 0)
        CC_SHA256_Update(&ctx, buf, (CC_LONG)n);

    if (n < 0) {
        log_ts("ERR", "error reading %s: %s", path, strerror(errno));
        close(fd);
        return -1;
    }

    CC_SHA256_Final(out, &ctx);
    close(fd);
    return 0;
}

/* ------------------------------------------------------------------ */
/* nftw callback                                                      */
/* ------------------------------------------------------------------ */

static int walk_cb(const char *fpath, const struct stat *sb,
                   int typeflag, struct FTW *ftwbuf)
{
    (void)ftwbuf;

    if (typeflag != FTW_F)
        return 0;
    if (!S_ISREG(sb->st_mode))
        return 0;

    if (verbose)
        log_ts("DBG", "scanning %s (%lld bytes)", fpath, (long long)sb->st_size);

    unsigned char digest[SHA256_LEN];
    if (sha256_file(fpath, digest) < 0)
        return 0;

    if (verbose) {
        char hexbuf[SHA256_LEN * 2 + 1];
        for (int i = 0; i < SHA256_LEN; i++)
            snprintf(hexbuf + i * 2, 3, "%02x", digest[i]);
        log_ts("DBG", "%s: %s", fpath, hexbuf);
    }

    ht_insert(digest, fpath, sb->st_size);
    return 0;
}

/* ------------------------------------------------------------------ */
/* Print duplicates                                                   */
/* ------------------------------------------------------------------ */

static void print_duplicates(void)
{
    for (unsigned i = 0; i < HT_BUCKETS; i++) {
        for (entry *e = buckets[i]; e; e = e->next) {
            if (e->count < 2)
                continue;

            /* dim hash line (ANSI bright black / grey) */
            printf("\033[90m");
            for (int j = 0; j < SHA256_LEN; j++)
                printf("%02x", e->digest[j]);
            printf(": %lld (%d)\033[0m\n", (long long)e->size, e->count);

            for (path_node *pn = e->paths; pn; pn = pn->next)
                printf("\"%s\"\n", pn->path);
            printf("\n");
        }
    }
}

/* ------------------------------------------------------------------ */
/* Cleanup                                                            */
/* ------------------------------------------------------------------ */

static void ht_free(void)
{
    for (unsigned i = 0; i < HT_BUCKETS; i++) {
        entry *e = buckets[i];
        while (e) {
            entry *next_e = e->next;
            path_node *pn = e->paths;
            while (pn) {
                path_node *next_pn = pn->next;
                free(pn->path);
                free(pn);
                pn = next_pn;
            }
            free(e);
            e = next_e;
        }
    }
}

/* ------------------------------------------------------------------ */
/* main                                                               */
/* ------------------------------------------------------------------ */

int main(int argc, char *argv[])
{
    int opt;
    while ((opt = getopt(argc, argv, "vh")) != -1) {
        switch (opt) {
        case 'v': verbose = 1; break;
        case 'h': /* fall through */
        default:
            fprintf(stderr, "Usage: %s [-v] <directory> [directory...]\n", argv[0]);
            return (opt == 'h') ? 0 : 1;
        }
    }

    if (optind >= argc) {
        fprintf(stderr, "Usage: %s [-v] <directory> [directory...]\n", argv[0]);
        return 1;
    }

    struct timespec t0, t1;
    clock_gettime(CLOCK_MONOTONIC, &t0);

    for (int i = optind; i < argc; i++) {
        if (nftw(argv[i], walk_cb, MAX_FD, FTW_PHYS) != 0)
            log_ts("ERR", "error walking %s: %s", argv[i], strerror(errno));
    }

    printf("\n");
    print_duplicates();

    clock_gettime(CLOCK_MONOTONIC, &t1);
    double elapsed = (t1.tv_sec - t0.tv_sec) + (t1.tv_nsec - t0.tv_nsec) / 1e9;
    log_ts("INF", "Elapsed time: %.3fs", elapsed);

    ht_free();
    return 0;
}
