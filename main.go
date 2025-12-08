package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

var config = struct {
	program  string
	version  string
	key      string
	verbose  bool
	all      bool
	markdown bool
	code     bool
	help     bool
	filePath string
}{}

func findDoc() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return "", false
	}

	fileNameList := []string{"scripts.md", ".scripts.md", "README.md"}

	for {
		for _, nameItem := range fileNameList {
			path := filepath.Join(dir, nameItem)
			fmt.Printf("Looking for %s\n", path)
			info, err := os.Stat(path)
			if err == nil && !info.IsDir() {
				return path, true
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// reached root directory
			break
		}
		dir = parent
	}

	return "", false
}

func showHelp() {
	fmt.Fprintf(os.Stderr, `Usage: %s [OPTIONS] [HEADING] [ARGS...]
Options
  -h, --help              Print this help message
  -v, --verbose           Print debug information
  -m, --markdown          Print node markdown
  -c, --code              Print node code block
  -a, --all               Parse code blocks in all languages
  -f, --file [FILE]       Specify the file to parse
`, config.program)
}

func main() {
	config.program = path.Base(os.Args[0])
	argsCount := len(os.Args)

	argi := 1
ParseArg:
	for ; argi < len(os.Args); argi++ {
		currentArg := os.Args[argi]

		switch currentArg {
		case "--verbose", "-v":
			config.verbose = true
		case "--help", "-h":
			config.help = true
		case "--all", "-a":
			config.all = true
		case "--markdown", "-m":
			config.markdown = true
		case "--code", "-c":
			config.code = true
		case "--file", "-f":
			if argsCount > argi+1 && len(os.Args[argi+1]) > 0 {
				config.filePath = os.Args[argi+1]
				argi++
			} else {
				fmt.Printf("No file path specified after --file or -f\n")
				return
			}
		case "--key", "-k":
			if argsCount > argi+1 && len(os.Args[argi+1]) > 0 {
				config.key = os.Args[argi+1]
				argi++
			} else {
				fmt.Printf("No key specified after --key or -k\n")
				return
			}
		default:
			if len(currentArg) > 0 && currentArg[0] == '-' { // Is an option
				fileFlag := "--file="
				keyFlag := "--key="

				switch {
				case len(currentArg) > len(fileFlag)+1 && currentArg[0:len(fileFlag)] == fileFlag:
					config.filePath = currentArg[len(fileFlag):]
				case len(currentArg) > len(keyFlag)+1 && currentArg[0:len(keyFlag)] == keyFlag:
					config.key = currentArg[len(keyFlag):]
				default:
					fmt.Printf("Unknown option: %s\n", currentArg)
					return
				}
			} else { // Not an option
				break ParseArg
			}

		}

	}

	if config.verbose {
		fmt.Printf("flags: verbose=%t, help=%t, all=%t, markdown=%t, code=%t, file_path=%s, key=%s\n",
			config.verbose, config.help, config.all, config.markdown, config.code, config.filePath, config.key)
	}

	for ; argi < argsCount; argi++ {
		fmt.Printf("os.Args[argi]: %v\n", os.Args[argi])
	}

	if config.help {
		showHelp()
		return
	}

	if config.filePath == "" {
		filePath, t := findDoc()
		if t {
			config.filePath = filePath
			fmt.Printf("config.filePath: %v\n", config.filePath)
		} else {
			fmt.Printf("No Markdown file found\n")
			return
		}
	}

	os.Setenv("MD_FILE", config.filePath)
	os.Setenv("MD_EXE", os.Args[0])

	content, err := os.ReadFile(config.filePath)
	if err != nil {
		fmt.Printf("reading file: %v", err)
		return
	}

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(content)

	ast.Print(os.Stdout, doc)

	// Create the command `echo hello`
	cmd := exec.Command("sh", "-c", "echo MD_FILE=${MD_FILE} MD_EXE=${MD_EXE}")

	// Run the command and capture its output
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Print the output (it includes a newline at the end)
	fmt.Printf("Output: %s", output)
}
