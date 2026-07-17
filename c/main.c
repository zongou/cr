#include "md4c/md4c.c"
#include "tree/tree.c"
#include "wcwidth/wcwidth.c"
#include <ctype.h>
#include <dirent.h>
#include <libgen.h>
#include <locale.h>
#include <stdarg.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <unistd.h>

struct {
    char *program;

    // Flags
    int help;
    int code;

    // Options
    char *file_path;
    char *log_file;
} config;

// Language configuration structure
struct Executor {
    const char  *name;
    const char **prefix_args;
    size_t       prefix_args_count;
};

// Language configuration argument arrays
static const char *sh_args[]         = {"$NAME", "-euc", "$CODE", "--"};
static const char *awk_args[]        = {"awk", "$CODE"};
static const char *node_args[]       = {"node", "-e", "$CODE"};
static const char *python_args[]     = {"python", "-c", "$CODE"};
static const char *ruby_args[]       = {"ruby", "-e", "$CODE"};
static const char *php_args[]        = {"php", "-r", "$CODE"};
static const char *cmd_args[]        = {"cmd.exe", "/c", "$CODE"};
static const char *powershell_args[] = {"powershell.exe", "-c", "$CODE"};

// Language configuration mappings
static const struct Executor executors[] = {
    {"sh", sh_args, 4},
    {"bash", sh_args, 4},
    {"zsh", sh_args, 4},
    {"fish", sh_args, 4},
    {"dash", sh_args, 4},
    {"ksh", sh_args, 4},
    {"ash", sh_args, 4},
    // {"shell", sh_args, 4},
    {"awk", awk_args, 2},
    {"js", node_args, 3},
    {"javascript", node_args, 3},
    {"py", python_args, 3},
    {"python", python_args, 3},
    {"rb", ruby_args, 3},
    {"ruby", ruby_args, 3},
    {"php", php_args, 3},
    {"cmd", cmd_args, 3},
    {"batch", cmd_args, 3},
    {"powershell", powershell_args, 3}};

const struct Executor *get_executor(const char *lang) {
    const struct Executor *config = NULL;
    // Find language configuration
    for (size_t i = 0; i < sizeof(executors) / sizeof(executors[0]);
         i++) {
        if (strcasecmp(executors[i].name, lang) == 0) {
            config = &executors[i];
            break;
        }
    }
    return config;
}

// Code block structure
typedef struct CODE_BLOCK CODE_BLOCK;
struct CODE_BLOCK {
    char       *info;
    char       *content;
    CODE_BLOCK *next;
};

// Markdown AST node structure
typedef struct MD_NODE MD_NODE;
struct MD_NODE {
    int         level;
    char       *text;
    char       *description;
    CODE_BLOCK *code_block;
    MD_NODE    *next;
    MD_NODE    *parent;
    MD_NODE    *child;
};

CODE_BLOCK *new_code_block(char *info) {
    CODE_BLOCK *block = malloc(sizeof(CODE_BLOCK));
    block->info       = info;
    block->content    = NULL;
    block->next       = NULL;
    return block;
}

MD_NODE *new_md_node() {
    MD_NODE *node     = malloc(sizeof(MD_NODE));
    node->level       = 0;
    node->text        = NULL;
    node->description = NULL;

    node->code_block = NULL;

    node->next   = NULL;
    node->child  = NULL;
    node->parent = NULL;

    return node;
}

// Callback structure to store state
typedef struct {
    int          depth;
    MD_BLOCKTYPE block_type;
    MD_SPANTYPE  span_type;
    char        *content;

    MD_NODE *root;
    MD_NODE *last;
} CallbackData;

char *substr(char *str, int start, int length) {
    if (!str || start < 0 || length < 0 || start + length > strlen(str)) {
        return NULL;
    }
    char *sub = (char *)malloc(length + 1);
    memcpy(sub, str + start, length);
    sub[length] = '\0';
    return sub;
}

char *strlower(char *str) {
    char *lower = strdup(str);
    for (int i = 0; lower[i]; i++) {
        lower[i] = tolower(lower[i]);
    }
    return lower;
}

void print_indention(int count) {
    for (int i = 0; i < count; i++) {
        printf("    ");
    }
}

void log_printf(const char *format, ...) {
    if (!config.log_file) {
        return;
    }

    FILE *log_fp = fopen(config.log_file, "a");
    if (!log_fp) {
        return;
    }

    va_list args;
    va_start(args, format);
    fprintf(log_fp, "%s:info: ", config.program);
    vfprintf(log_fp, format, args);
    va_end(args);
    fclose(log_fp);
}

void error(const char *format, ...) {
    va_list args;
    va_start(args, format);
    fprintf(stderr, "%s:error: ", config.program);
    vfprintf(stderr, format, args);
    va_end(args);
}

char *find_doc(char *program_basename) {
    char        file_pattern[3][PATH_MAX];
    char        current_dir[PATH_MAX];
    char        parent_dir[PATH_MAX];
    char        full_path[PATH_MAX];
    struct stat st;

    snprintf(file_pattern[0], sizeof(file_pattern[0]), "scripts.md");
    snprintf(file_pattern[1], sizeof(file_pattern[1]), ".scripts.md");
    snprintf(file_pattern[2], sizeof(file_pattern[2]), "README.md");

    if (getcwd(current_dir, sizeof(current_dir)) == NULL) {
        return NULL;
    }

    while (1) {
        char *dir_copy = strdup(current_dir);
        if (dir_copy == NULL) {
            return NULL;
        }
        strncpy(parent_dir, dirname(dir_copy), sizeof(parent_dir) - 1);
        parent_dir[sizeof(parent_dir) - 1] = '\0';
        free(dir_copy);

        int is_at_root = !strcmp(current_dir, parent_dir);

        for (int i = 0; i < sizeof(file_pattern) / sizeof(file_pattern[0]); i++) {
            char *file = file_pattern[i];
            snprintf(full_path, sizeof(full_path), "%s/%s", current_dir, file);
            log_printf("Looking for '%s' in dir '%s'\n", file, current_dir);
            if (stat(full_path, &st) == 0) {
                if (S_ISREG(st.st_mode) || S_ISLNK(st.st_mode)) {
                    return strdup(full_path);
                }
            }
        }

        strncpy(current_dir, parent_dir, sizeof(current_dir) - 1);
        current_dir[sizeof(current_dir) - 1] = '\0';
        if (is_at_root) {
            break;
        }
    }

    return NULL;
}

// Text callback - required by MD4C
static int text_callback(MD_TEXTTYPE type, const MD_CHAR *text, MD_SIZE size,
                         void *userdata) {
    CallbackData *data = (CallbackData *)userdata;

    if (data->content == NULL) {
        data->content = substr((char *)text, 0, size);
        if (data->content == NULL) {
            return -1;
        }
    } else {
        char *new_text = substr((char *)text, 0, size);
        if (new_text == NULL) {
            return -1;
        }

        size_t len1 = strlen(data->content);
        size_t len2 = size;
        char  *temp = (char *)realloc(data->content, len1 + len2 + 1);
        if (temp == NULL) {
            free(new_text);
            return -1;
        }
        data->content = temp;
        memcpy(data->content + len1, new_text, len2);
        data->content[len1 + len2] = '\0';
        free(new_text);
    }

    return 0;
}

// Block enter callback
static int enter_block_callback(MD_BLOCKTYPE type, void *detail,
                                void *userdata) {
    if (!userdata) {
        return 0;
    }
    CallbackData *data = (CallbackData *)userdata;
    data->block_type   = type;

    free(data->content);
    data->content = NULL;

    switch (type) {
        case MD_BLOCK_DOC:
            break;
        case MD_BLOCK_QUOTE:
            break;
        case MD_BLOCK_UL:
            if (detail) {
                MD_BLOCK_UL_DETAIL *d = (MD_BLOCK_UL_DETAIL *)detail;
            }
            break;
        case MD_BLOCK_OL:
            if (detail) {
                MD_BLOCK_OL_DETAIL *d = (MD_BLOCK_OL_DETAIL *)detail;
            }
            break;
        case MD_BLOCK_LI:
            if (detail) {
                MD_BLOCK_LI_DETAIL *d = (MD_BLOCK_LI_DETAIL *)detail;
            }
            break;
        case MD_BLOCK_HR:
            break;
        case MD_BLOCK_H:
            if (detail) {
                MD_BLOCK_H_DETAIL *d = (MD_BLOCK_H_DETAIL *)detail;
            }
            break;
        case MD_BLOCK_CODE:
            if (detail) {
                MD_BLOCK_CODE_DETAIL *d = (MD_BLOCK_CODE_DETAIL *)detail;
            }
            break;
        case MD_BLOCK_P:
            break;
        case MD_BLOCK_TABLE:
            if (detail) {
                MD_BLOCK_TABLE_DETAIL *d = (MD_BLOCK_TABLE_DETAIL *)detail;
            }
            break;
        case MD_BLOCK_TH:
            if (detail) {
                MD_BLOCK_TD_DETAIL *d = (MD_BLOCK_TD_DETAIL *)detail;
            }
            break;
        case MD_BLOCK_TD:
            if (detail) {
                MD_BLOCK_TD_DETAIL *d = (MD_BLOCK_TD_DETAIL *)detail;
            }
            break;
        case MD_BLOCK_THEAD:
            break;
        case MD_BLOCK_TBODY:
            break;
        case MD_BLOCK_TR:
            break;
        default:
            break;
    }

    data->depth++;
    return 0;
}

// Block leave callback
static int leave_block_callback(MD_BLOCKTYPE type, void *detail,
                                void *userdata) {
    if (!userdata) {
        return 0;
    }

    CallbackData *data = (CallbackData *)userdata;

    switch (type) {
        case MD_BLOCK_DOC:
        case MD_BLOCK_QUOTE:
        case MD_BLOCK_UL:
        case MD_BLOCK_OL:
            break;
        case MD_BLOCK_LI:
            break;
        case MD_BLOCK_HR:
            break;
        case MD_BLOCK_CODE:
            if (detail && data->last && data->content) {
                MD_BLOCK_CODE_DETAIL *c_detail = (MD_BLOCK_CODE_DETAIL *)detail;
                char                 *info     = substr((char *)c_detail->info.text, 0, c_detail->info.size);

                if (info) {
                    // printf("Node: %s, content: %s\n", data->last->text, data->content);
                    CODE_BLOCK *new_code = new_code_block(info);
                    new_code->info       = info;
                    new_code->content    = strdup(data->content);

                    CODE_BLOCK *last = data->last->code_block;
                    if (!last) {
                        data->last->code_block = new_code;
                    } else {
                        while (last->next) {
                            last = last->next;
                        }
                        last->next = new_code;
                    }
                }
            }
            break;
        case MD_BLOCK_HTML:
            break;
        case MD_BLOCK_TABLE:
            break;
        case MD_BLOCK_THEAD:
        case MD_BLOCK_TBODY:
            break;
        case MD_BLOCK_H: {
            MD_BLOCK_H_DETAIL *d        = (MD_BLOCK_H_DETAIL *)detail;
            MD_NODE           *new_node = new_md_node();
            new_node->level             = d->level;
            // Fix: Handle NULL content to prevent segfault on empty headings
            new_node->text = data->content ? strdup(data->content) : "";

            if (data->root == NULL) {
                data->root = new_node;
            } else {
                if (d->level == data->last->level) {
                    data->last->next = new_node;
                    new_node->parent = data->last->parent;
                } else if (d->level > data->last->level) {
                    data->last->child = new_node;
                    new_node->parent  = data->last;
                } else if (d->level < data->last->level) {
                    MD_NODE *parent = data->last->parent;
                    while (parent) {
                        if (parent->level == d->level) {
                            parent->next     = new_node;
                            new_node->parent = parent->parent;
                            break;
                        }
                        parent = parent->parent;
                    }
                }
            }
            data->last = new_node;
            break;
        }
        case MD_BLOCK_P:
            // Make sure we have a node to attach to
            if (data->last && !data->last->code_block) {
                data->last->description =
                    data->content == NULL ? NULL : strdup(data->content);
            }
            break;
        case MD_BLOCK_TR:
            break;
        case MD_BLOCK_TH:
            // printf("th: %d%d %s\n", data->row_index, data->cell_index,
            // data->content);
            break;
        case MD_BLOCK_TD:
            // printf("tb: %d%d %s\n", data->row_index, data->cell_index,
            // data->content);
            break;
    }

    free(data->content);
    data->content = NULL;

    if (data->depth > 0) {
        data->depth--;
    }
    return 0;
}

// Span enter callback
static int enter_span_callback(MD_SPANTYPE type, void *detail, void *userdata) {
    if (!userdata) {
        return 0;
    }
    CallbackData *data = (CallbackData *)userdata;
    data->span_type    = type;

    switch (type) {
        case MD_SPAN_CODE:
            break;
        case MD_SPAN_EM:
            break;
        case MD_SPAN_STRONG:
            break;
        case MD_SPAN_A:
            if (detail) {
                MD_SPAN_A_DETAIL *d = (MD_SPAN_A_DETAIL *)detail;
            }
            break;
        case MD_SPAN_IMG:
            if (detail) {
                MD_SPAN_IMG_DETAIL *d = (MD_SPAN_IMG_DETAIL *)detail;
            }
            break;
        case MD_SPAN_DEL:
            break;
        case MD_SPAN_LATEXMATH:
            break;
        case MD_SPAN_LATEXMATH_DISPLAY:
            break;
        case MD_SPAN_WIKILINK:
            if (detail) {
                MD_SPAN_WIKILINK_DETAIL *d = (MD_SPAN_WIKILINK_DETAIL *)detail;
            }
            break;
        case MD_SPAN_U:
            break;
    }

    data->depth++;
    return 0;
}

// Span leave callback
static int leave_span_callback(MD_SPANTYPE type, void *detail, void *userdata) {
    if (!userdata) {
        return 0;
    }

    CallbackData *data = (CallbackData *)userdata;

    if (data->depth > 0) {
        data->depth--;
    }
    return 0;
}

MD_NODE *parse_file(char *file_path) {
    FILE *fp = fopen(file_path, "rb");
    if (!fp) {
        error("Cannot open %s\n", file_path);
        return NULL;
    }

    // Get file size
    fseek(fp, 0, SEEK_END);
    long size = ftell(fp);
    fseek(fp, 0, SEEK_SET);

    if (size <= 0) {
        fclose(fp);
        error("Empty file: %s\n", file_path);
        return NULL;
    }

    // Allocate buffer and read file
    char *buffer =
        calloc(size + 1, 1); // Use calloc to ensure zero initialization
    if (!buffer) {
        fclose(fp);
        error("Memory allocation failed\n");
        return NULL;
    }

    size_t bytes_read  = fread(buffer, 1, size, fp);
    buffer[bytes_read] = '\0';
    fclose(fp);

    if (bytes_read == 0) {
        free(buffer);
        error("Failed to read file\n");
        return NULL;
    }

    // Initialize callback data
    CallbackData data = {.depth = 0, .root = NULL, .last = NULL};

    // Initialize parser with complete callback structure
    MD_PARSER parser   = {0}; // Zero initialize all fields
    parser.abi_version = 0;
    parser.flags       = MD_DIALECT_GITHUB;
    parser.enter_block = enter_block_callback;
    parser.leave_block = leave_block_callback;
    parser.enter_span  = enter_span_callback;
    parser.leave_span  = leave_span_callback;
    parser.text        = text_callback;

    int result = md_parse(buffer, bytes_read, &parser, &data);

    if (result != 0) {
        error("Markdown parsing failed with code %d\n", result);
        // } else {
        //     error( "Parsing completed successfully\n");
    }

    free(buffer);
    return data.root;
}

MD_NODE *find_node(MD_NODE *root, char *heading) {
    MD_NODE *current = root;
    while (current) {
        log_printf("current=%s\n", current->text);
        if (current->text && strcasecmp(current->text, heading) == 0) {
            return current;
        }
        if (current->child) {
            MD_NODE *result = find_node(current->child, heading);
            if (result) {
                return result;
            }
        }
        current = current->next;
    }
    return NULL;
}

int get_max_branch_width(MD_NODE *node) {
    int max = 0;

    for (MD_NODE *current = node->child; current; current = current->next) {
        if (current->code_block && get_executor(current->code_block->info) || current->child) {
            int w1 = (current->level - 1) * 4 + string_width(current->text);
            if (w1 > max) {
                max = w1;
            }
            int w2 = get_max_branch_width(current);
            if (w2 > max) {
                max = w2;
            }
        }
    }
    return max;
}

void node_to_tree_with_desc(MD_NODE *node, Tree *parent, int max_branch_width) {
    for (MD_NODE *current = node->child; current; current = current->next) {
        if (current->code_block && get_executor(current->code_block->info) || current->child) {
            // Repeated seperators
            int   seps_count = max_branch_width - (current->level - 1) * 4 - string_width(current->text);
            char *seperators = malloc(seps_count + 1);
            if (!seperators) {
                error("Memory allocation failed\n");
                exit(EXIT_FAILURE);
            }
            memset(seperators, ' ', seps_count);
            seperators[seps_count] = '\0';

            char *branch_val = malloc(1024);
            sprintf(branch_val, "%s %s %s", strlower(current->text), seperators, current->description ? current->description : "");
            Tree *current_tree = add_node(parent, branch_val);
            node_to_tree_with_desc(current, current_tree, max_branch_width);
        }
    }
}

// Execute code blocks for a given node
int exec_node(MD_NODE *node, char **args, int num_args) {
    int exit_code;
    log_printf("Executing node: %s\n", node->text);

    log_printf("Setting up environment variables\n");
    // First collect all nodes from root to target in a stack
    log_printf("Env stack size: %d\n", node->level);
    MD_NODE *stack[node->level];
    int      stack_size = 0;
    MD_NODE *current    = node;
    while (current) {
        stack[stack_size++] = current;
        current             = current->parent;
    }

    CODE_BLOCK *block = node->code_block;
    while (block) {
        if (block->info && block->content) {
            const char            *lang     = block->info;
            const struct Executor *executor = get_executor(lang);

            if (executor) {
                log_printf("Executing code block: \n```%s\n%s```\n", block->info,
                           block->content);
                log_printf("Using language profile: %s\n", executor->name);

                // Fork and execute
                pid_t pid = fork();
                if (pid == -1) {
                    perror("fork failed");
                    return 1;
                }

                if (pid == 0) {
                    // Child process
                    // Calculate number of arguments needed
                    int total_args = executor->prefix_args_count; // Prefix arguments
                    if (num_args > 0)
                        total_args += num_args; // User arguments

                    // Allocate argument array
                    char **exec_args = calloc(total_args + 1, sizeof(char *));
                    if (!exec_args) {
                        _exit(1);
                    }

                    // Fill argument array with prefix args first
                    int arg_idx = 0;
                    for (size_t i = 0; i < executor->prefix_args_count; i++) {
                        if (strcmp(executor->prefix_args[i], "$CODE") == 0) {
                            exec_args[arg_idx++] = block->content;
                        } else if (strcmp(executor->prefix_args[i], "$NAME") == 0) {
                            exec_args[arg_idx++] = (char *)executor->name;
                        } else {
                            exec_args[arg_idx++] = (char *)executor->prefix_args[i];
                        }
                    }

                    // Add user arguments
                    for (int i = 0; i < num_args; i++) {
                        exec_args[arg_idx++] = args[i];
                    }

                    exec_args[arg_idx] = NULL;

                    execvp(exec_args[0], exec_args);
                    perror("execvp failed");
                    free(exec_args);
                    _exit(1);
                } else {
                    // Parent process
                    int status;
                    waitpid(pid, &status, 0);

                    exit_code = WEXITSTATUS(status);
                    // if (!WIFEXITED(status) || exit_code != 0) {
                    //     info("Command failed with status %d\n", exit_code);
                    // } else {
                    //     info("Command completed successfully %d\n", exit_code);
                    // }
                    log_printf("Command exit code: %d\n", exit_code);
                }
            } else {
                error("%s: Unsupported language: %s\n", config.program, lang);
                return 1;
            }
        }
        if (exit_code) {
            break;
        }
        block = block->next;
    }
    return exit_code;
}

void show_hint(MD_NODE *doc_node) {
    for (MD_NODE *current = doc_node; current; current = current->next) {
        int max_branch_width = get_max_branch_width(current);

        Tree *docTree = new_tree(doc_node->text);
        node_to_tree_with_desc(doc_node, docTree, max_branch_width);
        char *tree_str = print_tree(docTree);
        printf("%s", tree_str);
    }
}

void show_help() {
    printf("Usage: %s [OPTIONS] [HEADING] [ARGS...]\n"
           "Options\n"
           "  -h, --help              Print this help message\n"
           "  -c, --code              Print node code block\n"
           "  -f, --file [FILE]       Specify the file to parse\n"
           "  -l, --log-file [FILE]   Path to log file for diagnostics\n",
           config.program);
}

int main(int argc, char **argv) {
    // Set the locale to the user's default environment.
    setlocale(LC_ALL, "");


    config.program = basename(argv[0]);

    // Parse options
    int argi = 1;
    while (argi < argc) {
        char *current_arg     = argv[argi];
        int   current_arg_len = strlen(current_arg);
        // printf("%s\n", current_arg);

        if (current_arg_len > 1 && current_arg[0] == '-') { // Is option
            if (current_arg[1] != '-') {                    // Is short options
                for (int short_opt_index = 1; short_opt_index < current_arg_len; short_opt_index++) {
                    char short_opt = current_arg[short_opt_index];
                    switch (short_opt) {
                        case 'h':
                            config.help = 1;
                            break;
                        case 'c':
                            config.code = 1;
                            break;
                        case 'f':                                        // Pattern: -f**, -f **
                            if (short_opt_index < current_arg_len - 1) { // Not the last char
                                config.file_path = current_arg + short_opt_index + 1;
                            } else {
                                // Current argument is not the last argument,
                                // and next argument is not an option.
                                if (argi < argc - 1 && argv[argi + 1] && argv[argi + 1][0] != '-') {
                                    config.file_path = argv[argi + 1];
                                    argi++;
                                } else {
                                    error("No file path specified after -f\n");
                                    return 1;
                                }
                            }
                            short_opt_index = current_arg_len; // Go to parse next argument
                            break;
                        case 'l':                                        // Pattern: -k**, -k **
                            if (short_opt_index < current_arg_len - 1) { // Not the last char
                                config.log_file = current_arg + short_opt_index + 1;
                            } else {
                                // Current argument is not the last argument,
                                // and next argument is not an option.
                                if (argi < argc - 1 && argv[argi + 1] && argv[argi + 1][0] != '-') {
                                    config.log_file = argv[argi + 1];
                                    argi++;
                                } else {
                                    error("No key specified after -k\n");
                                    return 1;
                                }
                            }
                            short_opt_index = current_arg_len; // Go to parse next argument
                            break;
                        default:
                            error("Unknown option: %c\n", short_opt);
                            return 1;
                    }
                }
            } else { // Is a long option
                if (strcmp(current_arg, "--help") == 0) {
                    config.help = 1;
                } else if (strcmp(current_arg, "--code") == 0) {
                    config.code = 1;
                } else if (strncmp(current_arg, "--file=", 7) == 0 && current_arg_len > 7) { // Pattern: --file=**
                    config.file_path = current_arg + 7;
                } else if (strcmp(current_arg, "--file") == 0 && argi < argc - 1) { // Pattern: --file **
                    argi++;
                    if (argv[argi]) {
                        config.file_path = argv[argi];
                    }
                } else if (strncmp(current_arg, "--log-file=", 11) == 0 && current_arg_len > 11) { // Pattern: --file=**
                    config.log_file = current_arg + 7;
                } else if (strcmp(current_arg, "--log-file") == 0 && argi < argc - 1) { // Pattern: --file **
                    argi++;
                    if (argv[argi]) {
                        config.log_file = argv[argi];
                    }
                } else {
                    error("Unknown option: %s\n", current_arg);
                    return 1;
                }
            }
        } else { // Not an option
            break;
        }

        argi++;
    }

    log_printf("flags: help=%d, code=%d, file_path=%s\n",
               config.help, config.code, config.file_path);

    if (config.help) {
        show_help();
        return EXIT_SUCCESS;
    }

    // Find and read markdown file
    if (!config.file_path) {
        config.file_path = find_doc(config.program);
        fflush(stdout);
    }

    if (!config.file_path) {
        log_printf("No markdown file found\n");
        fprintf(stderr, "No markdown file found\n");
        return EXIT_FAILURE;
    }
    setenv("MD_FILE", config.file_path, 1);
    log_printf("Using doc: %s\n", config.file_path);
    setenv("MD_EXE", argv[0], 1);

    MD_NODE *doc_node = parse_file(config.file_path);

    // Check if parsing was successful
    if (!doc_node) {
        log_printf("Failed to parse file: %s\n", config.file_path);
        fprintf(stderr, "Failed to parse file: %s\n", config.file_path);
        return EXIT_FAILURE;
    }

    if (argi < argc) {
        // First non-option argument is the node path, everything after that are
        // arguments to the code
        char  *cmd      = argv[argi];
        char **cmd_args = argv + argi + 1;
        int    num_args = argc - argi - 1;

        log_printf("Looking for node_path '%s'\n", cmd);

        // Find the node using the path (which may contain slashes)
        // Start search from the first level of children, not the document root
        MD_NODE *foundNode = NULL;
        if (doc_node) {
            foundNode = find_node(doc_node, cmd);
        }

        if (foundNode) {
            log_printf("Found node: %s\n", foundNode->text);
            foundNode->next  = NULL; // Do not print next node
            foundNode->child = NULL; // Do not print child node

            if (config.code) {
                if (config.code) {
                    CODE_BLOCK *code_block = foundNode->code_block;
                    while (code_block) {
                        printf("%s", code_block->content);
                        code_block = code_block->next;
                    }
                }
            } else {
                return exec_node(foundNode, cmd_args, num_args);
            }
        } else {
            error("Cannot find node: %s\n", cmd);
            return 1;
        }
    } else {
        log_printf("No command specified, printing hints.\n");
        show_hint(doc_node);
    }

    return 0;
}
