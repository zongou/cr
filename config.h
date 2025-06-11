#ifndef CONFIG_H
#define CONFIG_H

extern struct config config;

struct config {
    char *program;

    // Flags
    int help;
    int verbose;
    int all;
    int markdown;
    int code;

    // Options
    char *file_path;
    char *key;  // New field for --key option
};

#endif