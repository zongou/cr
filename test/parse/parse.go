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

	"github.com/userpro/md4go"
	"github.com/userpro/md4go/ast"
	"github.com/userpro/md4go/parser"
	"github.com/xlab/treeprint"
)

const (
	exitStatusError   = 1
	exitStatusSuccess = 0
)

type CodeBlock struct {
	Detail *ast.CodeDetail
	Code   string
	Next   *CodeBlock
}

type MDNode struct {
	Text          string
	HeadingDetail *ast.HeadingDetail
	// Tables       []Table
	// Lists        []List
	CodeBlock *CodeBlock
	// Paragraphs   []string
	// KeyValueMaps []map[string]string
	Next   *MDNode
	Parent *MDNode
	Child  *MDNode
	Desc   string
}

type MDNodeParser struct {
	root    *MDNode
	last    *MDNode
	content string
	depth   int
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

var (
	customExecutors = map[string][]string{}
	mergedExecutors = map[string][]string{}
)

var config = struct {
	program string

	// Flags
	help bool
	code bool
	// version  string

	// Options
	filePath string
	debugAST bool
	logFile  string
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

func formatBlockType(t ast.BlockType) string {
	switch t {
	case ast.BlockDoc:
		return "Document"
	case ast.BlockQuote:
		return "BlockQuote"
	case ast.BlockUL:
		return "UnorderedList"
	case ast.BlockOL:
		return "OrderedList"
	case ast.BlockLI:
		return "ListItem"
	case ast.BlockHR:
		return "HorizontalRule"
	case ast.BlockH:
		return "Heading"
	case ast.BlockCode:
		return "CodeBlock"
	case ast.BlockHTML:
		return "HTMLBlock"
	case ast.BlockP:
		return "Paragraph"
	case ast.BlockTable:
		return "Table"
	case ast.BlockTHead:
		return "TableHead"
	case ast.BlockTBody:
		return "TableBody"
	case ast.BlockTR:
		return "TableRow"
	case ast.BlockTH:
		return "TableHeaderCell"
	case ast.BlockTD:
		return "TableDataCell"
	case ast.BlockFootnoteDefSection:
		return "FootnoteDefinitionSection"
	case ast.BlockFootnoteDef:
		return "FootnoteDefinition"
	case ast.BlockAdmonition:
		return "Admonition"
	default:
		return fmt.Sprintf("BlockType(%d)", t)
	}
}

func formatSpanType(t ast.SpanType) string {
	switch t {
	case ast.SpanEm:
		return "Emphasis"
	case ast.SpanStrong:
		return "Strong"
	case ast.SpanLink:
		return "Link"
	case ast.SpanImg:
		return "Image"
	case ast.SpanCode:
		return "Code"
	case ast.SpanDel:
		return "Delete"
	case ast.SpanLatexMath:
		return "LatexMath"
	case ast.SpanLatexMathDisplay:
		return "LatexMathDisplay"
	case ast.SpanWikilink:
		return "Wikilink"
	case ast.SpanU:
		return "Underline"
	case ast.SpanSpoiler:
		return "Spoiler"
	case ast.SpanSuperscript:
		return "Superscript"
	case ast.SpanSubscript:
		return "Subscript"
	case ast.SpanFootnoteRef:
		return "FootnoteRef"
	case ast.SpanMark:
		return "Mark"
	default:
		return fmt.Sprintf("SpanType(%d)", t)
	}
}

func formatTextType(t ast.TextType) string {
	switch t {
	case ast.TextNormal:
		return "Normal"
	case ast.TextNullChar:
		return "NullChar"
	case ast.TextBR:
		return "BR"
	case ast.TextSoftBR:
		return "SoftBR"
	case ast.TextEntity:
		return "Entity"
	case ast.TextCode:
		return "Code"
	case ast.TextHTML:
		return "HTML"
	case ast.TextLatexMath:
		return "LatexMath"
	default:
		return fmt.Sprintf("TextType(%d)", t)
	}
}

func formatDetail(detail any) string {
	if detail == nil {
		return "<nil>"
	}

	switch d := detail.(type) {
	case *ast.HeadingDetail:
		return fmt.Sprintf("HeadingDetail{Level:%d}", d.Level)
	case *ast.CodeDetail:
		return fmt.Sprintf("CodeDetail{Info:%q Lang:%q Fence:%q}", d.Info.Text, d.Lang.Text, d.FenceChar)
	case *ast.ULDetail:
		return fmt.Sprintf("ULDetail{IsTight:%t Mark:%q}", d.IsTight, d.Mark)
	case *ast.OLDetail:
		return fmt.Sprintf("OLDetail{Start:%d IsTight:%t Mark:%q}", d.Start, d.IsTight, d.Mark)
	case *ast.LIDetail:
		return fmt.Sprintf("LIDetail{IsTask:%t TaskMark:%q TaskMarkOff:%d}", d.IsTask, d.TaskMark, d.TaskMarkOff)
	case *ast.TableDetail:
		return fmt.Sprintf("TableDetail{ColCount:%d HeadRowCount:%d BodyRowCount:%d}", d.ColCount, d.HeadRowCount, d.BodyRowCount)
	case *ast.TDDetail:
		return fmt.Sprintf("TDDetail{Align:%d}", d.Align)
	case *ast.LinkDetail:
		return fmt.Sprintf("LinkDetail{Href:%q Title:%q IsAutolink:%t}", d.Href.Text, d.Title.Text, d.IsAutolink)
	case *ast.ImgDetail:
		return fmt.Sprintf("ImgDetail{Src:%q Title:%q}", d.Src.Text, d.Title.Text)
	case *ast.AdmonitionDetail:
		return fmt.Sprintf("AdmonitionDetail{Type:%q}", d.Type.Text)
	case *ast.FootnoteRefDetail:
		return fmt.Sprintf("FootnoteRefDetail{ID:%d RefID:%d Label:%q}", d.ID, d.RefID, d.Label.Text)
	case *ast.FootnoteDefDetail:
		return fmt.Sprintf("FootnoteDefDetail{ID:%d RefCount:%d Label:%q}", d.ID, d.RefCount, d.Label.Text)
	case *ast.WikilinkDetail:
		return fmt.Sprintf("WikilinkDetail{Target:%q}", d.Target.Text)
	default:
		return fmt.Sprintf("%T(%v)", detail, detail)
	}
}

func printIndention(depth int) {
	for i := 0; i < depth; i++ {
		fmt.Printf("  ")
	}
}

func (r *MDNodeParser) EnterBlock(t ast.BlockType, detail any) error {
	printIndention(r.depth)
	r.depth++

	fmt.Printf("==>Block: %s", formatBlockType(t))
	if detail != nil {
		fmt.Printf(" detail=%s", formatDetail(detail))
	}
	fmt.Printf("\n")

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
	printIndention(r.depth - 1)
	if r.depth > 0 {
		r.depth--
	}

	fmt.Printf("<==Block: %s", formatBlockType(t))
	if detail != nil {
		fmt.Printf(" detail=%s", formatDetail(detail))
	}
	fmt.Printf("\n")

	switch t {
	// case ast.BlockDoc:
	// case ast.BlockQuote:
	// case ast.BlockUL:
	// case ast.BlockOL:
	// case ast.BlockLI:
	// case ast.BlockHR:
	case ast.BlockH:
		fmt.Printf("Heading: %v", r.content)

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

		// Get and Set custom executor from enviroment
		lang := string(newCodeBlock.Detail.Lang.Text)
		if customExecutorEnv := os.Getenv("MD_" + strings.ToUpper(lang)); customExecutorEnv != "" {
			if customExecutors[lang] == nil {
				customExecutor := strings.Split(customExecutorEnv, ",")
				log.Println(customExecutor)
				customExecutors[lang] = customExecutor
			}
		}

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
	printIndention(r.depth)
	r.depth++

	fmt.Printf("~~>Span: %s", formatSpanType(t))
	if detail != nil {
		fmt.Printf(" detail=%s", formatDetail(detail))
	}
	fmt.Printf("\n")

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
	printIndention(r.depth - 1)
	if r.depth > 0 {
		r.depth--
	}

	fmt.Printf("<~~Span: %s", formatSpanType(t))
	if detail != nil {
		fmt.Printf(" detail=%s", formatDetail(detail))
	}
	fmt.Printf("\n")

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
		fmt.Printf("Text: %s [%s]\n", text, formatTextType(t))
	}
	return nil
}

func Parse(src []byte) (*MDNode, error) {
	parser := md4go.New(md4go.WithFlags(parser.DialectGitHub))
	// parser := md4go.New()
	p := &MDNodeParser{}
	err := parser.Parse(src, p)
	if err != nil {
		return nil, err
	}
	return p.root, nil
}

func PrintNodeAst(node *MDNode, indent int) {
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += "    "
	}

	fmt.Printf("%sHeading: %s (Level %d)\n", indentStr, node.Text, node.HeadingDetail.Level)

	// // Print key-value maps
	// for i, kvMap := range node.KeyValueMaps {
	// 	fmt.Printf("%sKey-Value Map %d:\n", indentStr, i+1)
	// 	for key, value := range kvMap {
	// 		fmt.Printf("%s  %s = %s\n", indentStr, key, value)
	// 	}
	// }

	// // Print tables
	// for i, table := range node.Tables {
	// 	fmt.Printf("%sTable %d:\n", indentStr, i+1)
	// 	if len(table.Header) > 0 {
	// 		fmt.Printf("%s  Header: %v\n", indentStr, table.Header)
	// 	}
	// 	for j, row := range table.Rows {
	// 		fmt.Printf("%s  Row %d: %v\n", indentStr, j+1, row)
	// 	}
	// }

	// // Print paragraphs
	// for i, paragraph := range node.Paragraphs {
	// 	fmt.Printf("%sParagraph %d: %s\n", indentStr, i+1, paragraph)
	// }

	// // Print lists
	// for i, list := range node.Lists {
	// 	listType := "Unordered"
	// 	if list.IsOrdered {
	// 		listType = "Ordered"
	// 	}
	// 	fmt.Printf("%sList %d (%s):\n", indentStr, i+1, listType)
	// 	for j, item := range list.Items {
	// 		if item.Checked != nil {
	// 			// Print explicit true/false values for checked status
	// 			fmt.Printf("%s  %d. checked=%t %s\n", indentStr, j+1, *item.Checked, item.Text)
	// 		} else {
	// 			fmt.Printf("%s  %d. checked=false %s\n", indentStr, j+1, item.Text)
	// 		}
	// 	}
	// }

	// // Print code blocks
	// for i, codeBlock := range node.CodeBlocks {
	// 	fmt.Printf("%sCode Block %d (Language: %s):\n", indentStr, i+1, codeBlock.Language)
	// 	fmt.Printf("%s  Content: %s\n", indentStr, codeBlock.Content)
	// }

	currentCodeBlock := node.CodeBlock
	for currentCodeBlock != nil {
		fmt.Printf("%sCode Block (Language: %s):\n", indentStr, currentCodeBlock.Detail.Lang.Text)
		fmt.Printf("%s\n", currentCodeBlock.Code)
		currentCodeBlock = currentCodeBlock.Next
	}

	// Print children
	if node.Child != nil {
		PrintNodeAst(node.Child, indent+1)
	}

	// Print next sibling
	if node.Next != nil {
		PrintNodeAst(node.Next, indent)
	}
}

func FindNode(node *MDNode, heading string) *MDNode {
	if node == nil {
		return nil
	}

	if strings.EqualFold(node.Text, heading) {
		return node
	}

	// Search in child nodes
	found := FindNode(node.Child, heading)
	if found != nil {
		return found
	}

	// Search in next sibling nodes
	return FindNode(node.Next, heading)
}

func NodeToTree(node *MDNode) treeprint.Tree {
	var walk func(*MDNode, *treeprint.Tree)
	walk = func(node *MDNode, parent *treeprint.Tree) {
		for currentNode := node.Child; currentNode != nil; currentNode = currentNode.Next {
			if currentNode.CodeBlock != nil && mergedExecutors[string(currentNode.CodeBlock.Detail.Lang.Text)] != nil || currentNode.Child != nil {
				currentTree := (*parent).AddBranch(strings.ToLower(currentNode.Text))
				walk(currentNode, &currentTree)
			}
		}
	}

	root := treeprint.NewWithRoot(node.Text)
	walk(node, &root)
	return root
}

func NodeToTreeWithDesc(node *MDNode) treeprint.Tree {
	// Get max branch length
	maxLineRuneLen := 0
	const branchSymbolLen = 4

	var getMaxBrachRuneLen func(*MDNode)
	getMaxBrachRuneLen = func(node *MDNode) {
		for currentNode := node.Child; currentNode != nil; currentNode = currentNode.Next {
			if currentNode.CodeBlock != nil && mergedExecutors[string(currentNode.CodeBlock.Detail.Lang.Text)] != nil || currentNode.Child != nil {
				branchRuneLen := len([]rune(currentNode.Text)) + (currentNode.HeadingDetail.Level-1)*branchSymbolLen
				if branchRuneLen > maxLineRuneLen {
					maxLineRuneLen = branchRuneLen
				}
				getMaxBrachRuneLen(currentNode)
			}
		}
	}
	getMaxBrachRuneLen(node)

	var walk func(*MDNode, *treeprint.Tree)
	walk = func(node *MDNode, parent *treeprint.Tree) {
		for currentNode := node.Child; currentNode != nil; currentNode = currentNode.Next {
			if currentNode.CodeBlock != nil && mergedExecutors[string(currentNode.CodeBlock.Detail.Lang.Text)] != nil || currentNode.Child != nil {
				branchVal := strings.ToLower(currentNode.Text) + " " + strings.Repeat(" ", maxLineRuneLen-(currentNode.HeadingDetail.Level-1)*branchSymbolLen-len([]rune(currentNode.Text))) + " " + currentNode.Desc
				currentTree := (*parent).AddBranch(branchVal)
				walk(currentNode, &currentTree)
			}
		}
	}

	root := treeprint.NewWithRoot(node.Text)
	walk(node, &root)
	return root
}

func DocToTree(node *MDNode) treeprint.Tree {
	var walk func(*MDNode, *treeprint.Tree)
	walk = func(node *MDNode, parent *treeprint.Tree) {
		for currentNode := node; currentNode != nil; currentNode = currentNode.Next {
			currentTree := (*parent).AddBranch(currentNode.Text)
			currentTree.SetMetaValue(currentNode.HeadingDetail.Level)
			walk(currentNode.Child, &currentTree)
		}
	}

	root := treeprint.New()
	walk(node, &root)
	return root
}

func execNode(node *MDNode, originArgs []string) int {
	log.Printf("customExecutors: %v\n", customExecutors)
	log.Printf("mergedExecutors: %v\n", mergedExecutors)

	for currentCodeBlock := node.CodeBlock; currentCodeBlock != nil; currentCodeBlock = currentCodeBlock.Next {
		// Lookup language executor
		lang := string(currentCodeBlock.Detail.Lang.Text)

		executor, exists := mergedExecutors[string(lang)]
		if !exists {
			log.Printf("unsupported code block type: %s\n", lang)
			fmt.Fprintf(os.Stderr, "unsupported code block type: %s\n", lang)
			return exitStatusError
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
			return exitStatusError
		} else {
			log.Println("Command executed successfully!")
		}
	}

	return exitStatusSuccess
}

func showHint(docNode *MDNode) {
	for currentNode := docNode; currentNode != nil; currentNode = currentNode.Next {
		fmt.Print(NodeToTree(currentNode).String())
	}
}

func showHelp() {
	fmt.Fprintf(os.Stderr, `Usage: %s [OPTIONS] [HEADING] [ARGS...]
Options:
  -h, --help              Print this help message
  -c, --code              Print node code block
  -f, --file [FILE]       Path to MarkDown file
  -l, --log-file [FILE]   Path to log file for diagnostics
  --debug-ast             Print AST structure for debugging
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
		case "--debug-ast":
			config.debugAST = true
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
			os.Exit(exitStatusError)
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

	log.Printf("flags: help=%t, code=%t, file_path=%s, debug_ast=%t\n",
		config.help, config.code, config.filePath, config.debugAST)

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
			log.Println("No Markdown file found")
			fmt.Fprintf(os.Stderr, "No Markdown file found")
			os.Exit(exitStatusError)
		}
	}

	os.Setenv("MD_FILE", config.filePath)
	os.Setenv("MD_EXE", os.Args[0])

	content, err := os.ReadFile(config.filePath)
	if err != nil {
		log.Printf("reading file: %v\n", err)
		fmt.Fprintf(os.Stderr, "reading file: %v\n", err)
		os.Exit(exitStatusError)
	}

	// Parse MarkDown document
	docNode, err := Parse(content)
	if err != nil {
		log.Printf("parsing file: %v\n", err)
		fmt.Fprintf(os.Stderr, "parsing file: %v\n", err)
		os.Exit(exitStatusError)
	}

	// Merge executors map and customExecutors map
	merged := make(map[string][]string)
	for key, values := range executors {
		merged[key] = append([]string{}, values...) // create a copy to avoid mutating the original map
	}

	for key, values := range customExecutors {
		merged[key] = append([]string{}, values...) // create a copy to keep the original map safe
	}
	mergedExecutors = merged

	// for ; argi < argsCount; argi++ {
	// 	fmt.Printf("os.Args[argi]: %v\n", os.Args[argi])
	// }

	if argi < argsCount {
		cmd := os.Args[argi]
		args := os.Args[argi+1:]
		log.Printf("Matching node with '%s'\n", cmd)

		foundNode := FindNode(docNode, cmd)
		if foundNode != nil {
			log.Printf("Found node: %s (Level %d)\n", foundNode.Text, foundNode.HeadingDetail.Level)
			log.Printf("args: %v\n", args)

			if config.code {
				for cb := foundNode.CodeBlock; cb != nil; cb = cb.Next {
					fmt.Printf("%s", cb.Code)
				}
			} else {
				// Copy exit status
				exitStatus := execNode(foundNode, args)
				os.Exit(exitStatus)
			}
		} else {
			log.Printf("Node not found: %s\n", cmd)
			fmt.Fprintf(os.Stderr, "Node not found: %s\n", cmd)
			os.Exit(exitStatusError)
		}
	} else {
		showHint(docNode)
		fmt.Printf("%v", DocToTree(docNode))
		fmt.Printf("%v", NodeToTreeWithDesc(docNode))
	}
}
