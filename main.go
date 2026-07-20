package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/userpro/md4go"
	"github.com/userpro/md4go/ast"
	"github.com/xlab/treeprint"
)

var config = struct {
	program string

	// Flags
	help bool
	code bool
	one  bool
	tree bool
	// version  string

	// Options
	filePath string
	logFile  string
}{}

const (
	exitFailure = 1
	exitSuccess = 0
)

type CodeBlock struct {
	Detail *ast.CodeDetail
	Code   string
	Next   *CodeBlock
}

type MDNode struct {
	Text          string
	HeadingDetail *ast.HeadingDetail
	CodeBlock     *CodeBlock
	Next          *MDNode
	Parent        *MDNode
	Child         *MDNode
	Desc          string
}

type MDNodeParser struct {
	root    *MDNode
	last    *MDNode
	content string
}

var executors = map[string][]string{
	"sh":   {"sh", "-euc", "{CODE}", "--"},
	"bash": {"bash", "-euc", "{CODE}", "--"},
	"zsh":  {"zsh", "-euc", "{CODE}", "--"},
	"fish": {"fish", "-euc", "{CODE}", "--"},
	"dash": {"dash", "-euc", "{CODE}", "--"},
	"ksh":  {"ksh", "-euc", "{CODE}", "--"},
	"ash":  {"ash", "-euc", "{CODE}", "--"},
	// "shell":      {"sh", "-euc", "{CODE}", "--"},
	"awk":        {"awk", "{CODE}"},
	"js":         {"node", "-e", "{CODE}"},
	"javascript": {"node", "-e", "{CODE}"},
	"py":         {"python", "-c", "{CODE}"},
	"python":     {"python", "-c", "{CODE}"},
	"rb":         {"ruby", "-e", "{CODE}"},
	"ruby":       {"ruby", "-e", "{CODE}"},
	"php":        {"php", "-r", "{CODE}"},
	"cmd":        {"cmd.exe", "/c", "{CODE}"},
	"batch":      {"cmd.exe", "/c", "{CODE}"},
	"ps2":        {"powershell.exe", "-c", "{CODE}"},
	"powershell": {"powershell.exe", "-c", "{CODE}"},
}

var customExecutors = map[string][]string{}

func parseCustomExecutors() {
	// // Get and Set custom executor from enviroment
	// lang := string(newCodeBlock.Detail.Lang.Text)
	// if customExecutorEnv := os.Getenv("MD_" + strings.ToUpper(lang)); customExecutorEnv != "" {
	// 	if customExecutors[lang] == nil {
	// 		customExecutor := strings.Split(customExecutorEnv, ",")
	// 		log.Println(customExecutor)
	// 		customExecutors[lang] = customExecutor
	// 	}
	// }

	// Retrieve all environment variables
	envVars := os.Environ()

	// Declare a prefix to filter environment variables
	prefix := "MD_"

	// Iterate over the environment variables
	for _, envVar := range envVars {
		// Split the variable into key and value
		kv := strings.SplitN(envVar, "=", 2) // Split at the first '='
		if len(kv) == 2 && strings.HasPrefix(kv[0], prefix) {
			key := kv[0]
			val := kv[1]

			lang := strings.ToLower(key[3:])
			customExecutor := strings.Split(val, ",")
			customExecutors[lang] = customExecutor
		}
	}
}

func getExecutor(lang string) []string {
	log.Printf("customExecutors[lang]: %v\n", customExecutors[lang])
	log.Printf("executors[lang]: %v\n", executors[lang])
	if customExecutors[lang] != nil {
		return customExecutors[lang]
	} else {
		return executors[lang]
	}
}

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
			log.Printf("Matching doc with %s\n", path)
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

func (r *MDNodeParser) EnterBlock(t ast.BlockType, detail any) error {
	switch t {
	// case ast.BlockDoc:
	// case ast.BlockQuote:
	// case ast.BlockUL:
	// case ast.BlockOL:
	// case ast.BlockLI:
	// case ast.BlockHR:
	// case ast.BlockH:
	case ast.BlockCode:
		r.content = ""
	// case ast.BlockHTML:
	// case ast.BlockP:
	// case ast.BlockTable:
	// case ast.BlockTHead:
	// case ast.BlockTBody:
	// case ast.BlockTR:
	// case ast.BlockTH:
	// case ast.BlockTD:
	// case ast.BlockFootnoteDefSection:
	// case ast.BlockFootnoteDef:
	// case ast.BlockAdmonition:
	default:
	}
	return nil
}

func (r *MDNodeParser) LeaveBlock(t ast.BlockType, detail any) error {
	switch t {
	// case ast.BlockDoc:
	// case ast.BlockQuote:
	// case ast.BlockUL:
	// case ast.BlockOL:
	// case ast.BlockLI:
	// case ast.BlockHR:
	case ast.BlockH:
		newNode := &MDNode{
			Text:          r.content,
			HeadingDetail: detail.(*ast.HeadingDetail),
		}

		if r.root == nil {
			r.root = newNode
		} else {
			if newNode.HeadingDetail.Level == r.last.HeadingDetail.Level {
				r.last.Next = newNode
				newNode.Parent = r.last.Parent
			} else if newNode.HeadingDetail.Level > r.last.HeadingDetail.Level {
				r.last.Child = newNode
				newNode.Parent = r.last
			} else {
				// Find the correct parent node based on heading levels
				for parent := r.last.Parent; parent != nil; parent = parent.Parent {
					if newNode.HeadingDetail.Level == parent.HeadingDetail.Level {
						parent.Next = newNode
						newNode.Parent = parent.Parent
						break
					}
				}
			}
		}

		r.last = newNode
		r.content = ""
	case ast.BlockCode:
		newCodeBlock := &CodeBlock{
			Detail: detail.(*ast.CodeDetail),
			Code:   r.content,
		}
		if r.last != nil {
			if r.last.CodeBlock == nil {
				r.last.CodeBlock = newCodeBlock
			} else {
				current := r.last.CodeBlock
				for current.Next != nil {
					current = current.Next
				}
				current.Next = newCodeBlock
			}
		}
		r.content = ""
	// case ast.BlockHTML:
	case ast.BlockP:
		if r.last != nil && r.last.CodeBlock == nil && r.content != "" {
			r.last.Desc = string(r.content)
		}
	// case ast.BlockTable:
	// case ast.BlockTHead:
	// case ast.BlockTBody:
	// case ast.BlockTR:
	// case ast.BlockTH:
	// case ast.BlockTD:
	// case ast.BlockFootnoteDefSection:
	// case ast.BlockFootnoteDef:
	// case ast.BlockAdmonition:
	default:
		r.content = ""
	}
	return nil
}

func (r *MDNodeParser) EnterSpan(t ast.SpanType, detail any) error {
	switch t {
	// case ast.SpanEm:
	// case ast.SpanStrong:
	// case ast.SpanLink:
	// case ast.SpanImg:
	// case ast.SpanCode:
	// case ast.SpanDel:
	// case ast.SpanLatexMath:
	// case ast.SpanLatexMathDisplay:
	// case ast.SpanWikilink:
	// case ast.SpanU:
	// case ast.SpanSpoiler:
	// case ast.SpanSuperscript:
	// case ast.SpanSubscript:
	// case ast.SpanFootnoteRef:
	// case ast.SpanMark:
	default:
	}
	return nil
}

func (r *MDNodeParser) LeaveSpan(t ast.SpanType, detail any) error {
	switch t {
	// case ast.SpanEm:
	// case ast.SpanStrong:
	// case ast.SpanLink:
	// case ast.SpanImg:
	// case ast.SpanCode:
	// case ast.SpanDel:
	// case ast.SpanLatexMath:
	// case ast.SpanLatexMathDisplay:
	// case ast.SpanWikilink:
	// case ast.SpanU:
	// case ast.SpanSpoiler:
	// case ast.SpanSuperscript:
	// case ast.SpanSubscript:
	// case ast.SpanFootnoteRef:
	// case ast.SpanMark:
	default:
	}
	return nil
}

func (r *MDNodeParser) Text(t ast.TextType, text []byte) error {
	switch t {
	case ast.TextNormal:
		r.content += string(text)
	// case ast.TextNullChar:
	// case ast.TextBR:
	// case ast.TextSoftBR:
	// case ast.TextEntity:
	case ast.TextCode:
		r.content += string(text)
	// case ast.TextHTML:
	// case ast.TextLatexMath:
	default:
	}
	return nil
}

func parseFile(filePath string) (*MDNode, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	parser := md4go.New()
	p := &MDNodeParser{}
	err = parser.Parse(src, p)
	if err != nil {
		return nil, err
	}
	return p.root, nil
}

func findNode(node *MDNode, heading string) *MDNode {
	if node == nil {
		return nil
	}

	if strings.EqualFold(node.Text, heading) {
		return node
	}

	// Search in child nodes
	found := findNode(node.Child, heading)
	if found != nil {
		return found
	}

	// Search in next sibling nodes
	return findNode(node.Next, heading)
}

func nodeToTreeWithDesc(node *MDNode) treeprint.Tree {
	// Get max branch length
	maxBranchWidth := 0
	const branchSymbolWidth = 4

	var getMaxBranchWidth func(*MDNode)
	getMaxBranchWidth = func(node *MDNode) {
		for currentNode := node.Child; currentNode != nil; currentNode = currentNode.Next {
			if currentNode.CodeBlock != nil && getExecutor(string(currentNode.CodeBlock.Detail.Info.Text)) != nil || currentNode.Child != nil {
				branchWidth := (currentNode.HeadingDetail.Level-1)*branchSymbolWidth + runewidth.StringWidth(currentNode.Text)
				if branchWidth > maxBranchWidth {
					maxBranchWidth = branchWidth
				}
				getMaxBranchWidth(currentNode)
			}
		}
	}
	getMaxBranchWidth(node)

	var walk func(*MDNode, *treeprint.Tree)
	walk = func(node *MDNode, parent *treeprint.Tree) {
		for currentNode := node.Child; currentNode != nil; currentNode = currentNode.Next {
			if currentNode.CodeBlock != nil && getExecutor(string(currentNode.CodeBlock.Detail.Info.Text)) != nil || currentNode.Child != nil {
				branchVal := strings.ToLower(currentNode.Text) + " " + strings.Repeat(" ", maxBranchWidth-(currentNode.HeadingDetail.Level-1)*branchSymbolWidth-runewidth.StringWidth(currentNode.Text)) + " " + currentNode.Desc
				currentTree := (*parent).AddBranch(branchVal)
				walk(currentNode, &currentTree)
			}
		}
	}

	root := treeprint.NewWithRoot(node.Text)
	walk(node, &root)
	return root
}

func printOne(node *MDNode) {
	for currentNode := node.Child; currentNode != nil; currentNode = currentNode.Next {
		if currentNode.CodeBlock != nil && getExecutor(string(currentNode.CodeBlock.Detail.Info.Text)) != nil {
			fmt.Println(strings.ToLower(currentNode.Text))
			printOne(currentNode)
		} else if currentNode.Child != nil {
			printOne(currentNode)
		}
	}
}

func execNode(node *MDNode, originArgs []string) int {
	if node.CodeBlock == nil {
		log.Printf("no code blocks under this heading\n")
		fmt.Fprintf(os.Stderr, "no code blocks under this heading\n")
		return exitFailure
	}

	for currentCodeBlock := node.CodeBlock; currentCodeBlock != nil; currentCodeBlock = currentCodeBlock.Next {
		// Lookup language executor
		lang := string(currentCodeBlock.Detail.Lang.Text)

		// executor, exists := getExecutor(lang)
		// if !exists {
		// 	log.Printf("unsupported code block type: %s\n", lang)
		// 	fmt.Fprintf(os.Stderr, "unsupported code block type: %s\n", lang)
		// 	return exitFailure
		// }

		executor := getExecutor(lang)
		log.Printf("customExecutors: %v\n", customExecutors)
		log.Printf("executor: %v\n", executor)
		if executor == nil {
			log.Printf("unsupported code block type: %s\n", lang)
			fmt.Fprintf(os.Stderr, "unsupported code block type: %s\n", lang)
			return exitFailure
		}

		// Replace prefixed variable placeholders with actual values
		prefixArgs := make([]string, len(executor))
		for i, arg := range executor {
			if strings.Contains(arg, "{CODE}") {
				prefixArgs[i] = strings.ReplaceAll(arg, "{CODE}", currentCodeBlock.Code)
			} else {
				prefixArgs[i] = arg
			}
		}

		formattedArgs := append(prefixArgs, originArgs...)

		cmd := exec.Command(formattedArgs[0], formattedArgs[1:]...)
		// cmd.Env = append(cmd.Env, "A=Apple")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run the command and capture the output and error
		err := cmd.Run()

		// Get the exit status from the command
		if exitError, ok := err.(*exec.ExitError); ok {
			// If the command exited with an error, we can get the exit code
			if status := exitError.ExitCode(); status != 0 {
				log.Printf("Command exited with non-zero exit code: %d\n", status)
				return status
			}
		} else if err != nil {
			// Handle generic errors
			log.Printf("Error executing command: %v\n", err)
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return exitFailure
		} else {
			log.Println("Command executed successfully!")
		}
	}

	return exitSuccess
}

func showHelp() {
	fmt.Printf(`Usage: %s [OPTIONS] [HEADING] [ARGS...]
Options:
  -h, --help              Print this help message
  -c, --code [HEADING]    Print node code block
  -1 [HEADING]            List one command per line
  -t, --tree [HEADING]    Print tree with description
  -f, --file [FILE]       Path to MarkDown file
  -l, --log-file [FILE]   Path to log file for diagnostics
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
		case "--help", "-h":
			config.help = true
		case "--code", "-c":
			config.code = true
		case "-1":
			config.one = true
		case "--tree", "-t":
			config.tree = true
		case "--file", "-f":
			if argsCount > argi+1 && len(os.Args[argi+1]) > 0 {
				config.filePath = os.Args[argi+1]
				argi++
			} else {
				fmt.Fprintf(os.Stderr, "No file path specified after --file or -f\n")
				return
			}
		case "--log-file", "-l":
			if argsCount > argi+1 && len(os.Args[argi+1]) > 0 {
				config.logFile = os.Args[argi+1]
				argi++
			} else {
				fmt.Fprintf(os.Stderr, "No filePath specified after --log-file or -l\n")
				return
			}
		default:
			if len(currentArg) > 0 && currentArg[0] == '-' { // Is an option
				fileFlag := "--file="
				logFileFlag := "--log-file="

				switch {
				case len(currentArg) > len(fileFlag)+1 && currentArg[0:len(fileFlag)] == fileFlag:
					config.filePath = currentArg[len(fileFlag):]
				case len(currentArg) > len(logFileFlag)+1 && currentArg[0:len(logFileFlag)] == logFileFlag:
					config.logFile = currentArg[len(logFileFlag):]
				default:
					fmt.Fprintf(os.Stderr, "Unknown option: %s\n", currentArg)
					return
				}
			} else { // Not an option
				break ParseArg
			}

		}

	}

	// By default, keep logging quiet. If a log file is provided, write logs there.
	log.SetOutput(io.Discard)
	if config.logFile != "" {
		file, err := os.OpenFile(config.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening log file: %v\n", err)
			os.Exit(exitFailure)
		}
		defer file.Close()
		log.SetOutput(file)
	}

	// Set the log output to the file

	// Logging messages
	// log.Println("This is an info log message.")
	// log.Println("This is another log message.")
	// log.Println("Logging an error: This is an error message.")
	// log.Printf("Logging with formatting: %s %d\n", "Count", 1)

	// // You can set the log flags if needed (optional)
	// log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// log.Println("This log entry includes date, time, and source file info.")

	log.Printf("flags: help=%t, code=%t, file_path=%s\n",
		config.help, config.code, config.filePath)

	if config.help {
		showHelp()
		return
	}

	if config.filePath == "" {
		filePath, t := findDoc()
		if t {
			config.filePath = filePath
			log.Printf("Found doc: %v\n", config.filePath)
		} else {
			log.Println("No markdown file found")
			fmt.Fprintf(os.Stderr, "No markdown file found")
			os.Exit(exitFailure)
		}
	}

	os.Setenv("CR", os.Args[0])
	os.Setenv("CR_FILE", config.filePath)

	parseCustomExecutors()

	// Parse MarkDown document
	docNode, err := parseFile(config.filePath)
	if err != nil {
		log.Printf("parsing file: %v\n", err)
		fmt.Fprintf(os.Stderr, "parsing file: %v\n", err)
		os.Exit(exitFailure)
	}

	// for ; argi < argsCount; argi++ {
	// 	fmt.Printf("os.Args[argi]: %v\n", os.Args[argi])
	// }

	if argi < argsCount {
		cmd := os.Args[argi]
		args := os.Args[argi+1:]
		log.Printf("Matching node with '%s'\n", cmd)

		foundNode := findNode(docNode, cmd)
		if foundNode != nil {
			log.Printf("Found node: %s (Level %d)\n", foundNode.Text, foundNode.HeadingDetail.Level)
			log.Printf("args: %v\n", args)

			if config.code {
				for cb := foundNode.CodeBlock; cb != nil; cb = cb.Next {
					fmt.Printf("%s", cb.Code)
				}
			} else if config.one {
				printOne(foundNode)
			} else if config.tree {
				fmt.Print(nodeToTreeWithDesc(foundNode).String())
			} else {
				// Copy exit status
				exitStatus := execNode(foundNode, args)
				os.Exit(exitStatus)
			}
		} else {
			log.Printf("Node not found: %s\n", cmd)
			fmt.Fprintf(os.Stderr, "Node not found: %s\n", cmd)
			os.Exit(exitFailure)
		}
	} else {
		if config.one {
			printOne(docNode)
		} else {
			for currentNode := docNode; currentNode != nil; currentNode = currentNode.Next {
				fmt.Print(nodeToTreeWithDesc(currentNode).String())
			}
		}
	}
}
