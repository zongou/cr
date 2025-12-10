package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	ext_ast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// Node represents a parsed markdown node containing various elements
type Node struct {
	Heading    string
	Level      int
	Tables     []Table
	Lists      []List
	CodeBlocks []CodeBlock
	Graphs     []Graph
	Children   []*Node
	Parent     *Node
}

// Table represents a markdown table
type Table struct {
	Header []string
	Rows   [][]string
}

// List represents a markdown list
type List struct {
	Items []string
	IsOrdered bool
}

// CodeBlock represents a markdown code block
type CodeBlock struct {
	Language string
	Content  string
}

// Graph represents a markdown graph (mermaid, plantuml, etc.)
type Graph struct {
	Type    string
	Content string
}

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

// parseMarkdown parses markdown content into a tree of nodes
func parseMarkdown(source []byte) *Node {
	// Create markdown parser with extensions
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table),
	)
	
	// Parse the document
	doc := md.Parser().Parse(text.NewReader(source))
	
	root := &Node{
		Heading: "root",
		Level:   0,
	}
	
	parseNode(doc, root, source)
	return root
}

// parseNode recursively parses AST nodes into our Node structure
func parseNode(astNode ast.Node, parentNode *Node, source []byte) {
	// Walk through children
	for child := astNode.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Heading:
			// Create a new node for this heading
			headingText := string(extractText(n, source))
			headingNode := &Node{
				Heading: headingText,
				Level:   n.Level,
				Parent:  parentNode,
			}
			
			// Add to parent's children
			parentNode.Children = append(parentNode.Children, headingNode)
			
			// Continue parsing within this heading node
			parseNode(child, headingNode, source)
			
		case *ext_ast.Table:
			// Parse table
			table := parseTable(n, source)
			if len(parentNode.Children) > 0 {
				// Add to the last heading node
				lastNode := parentNode.Children[len(parentNode.Children)-1]
				lastNode.Tables = append(lastNode.Tables, table)
			} else {
				// Add to current node
				parentNode.Tables = append(parentNode.Tables, table)
			}
			
		case *ast.List:
			// Parse list
			list := parseList(n, source)
			if len(parentNode.Children) > 0 {
				// Add to the last heading node
				lastNode := parentNode.Children[len(parentNode.Children)-1]
				lastNode.Lists = append(lastNode.Lists, list)
			} else {
				// Add to current node
				parentNode.Lists = append(parentNode.Lists, list)
			}
			
		case *ast.FencedCodeBlock:
			// Parse fenced code block
			codeBlock := CodeBlock{
				Language: string(n.Language(source)),
				Content:  string(n.Lines().Value(source)),
			}
			
			if len(parentNode.Children) > 0 {
				// Add to the last heading node
				lastNode := parentNode.Children[len(parentNode.Children)-1]
				lastNode.CodeBlocks = append(lastNode.CodeBlocks, codeBlock)
			} else {
				// Add to current node
				parentNode.CodeBlocks = append(parentNode.CodeBlocks, codeBlock)
			}
			
		case *ast.HTMLBlock:
			// Check if this HTML block contains a graph
			graph := parseGraph(n, source)
			if graph.Type != "" {
				if len(parentNode.Children) > 0 {
					// Add to the last heading node
					lastNode := parentNode.Children[len(parentNode.Children)-1]
					lastNode.Graphs = append(lastNode.Graphs, graph)
				} else {
					// Add to current node
					parentNode.Graphs = append(parentNode.Graphs, graph)
				}
			}
			
		default:
			// Continue parsing other node types
			parseNode(child, parentNode, source)
		}
	}
}

// extractText extracts plain text from an AST node
func extractText(node ast.Node, source []byte) []byte {
	var result []byte
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			result = append(result, n.Segment.Value(source)...)
		case *ast.String:
			result = append(result, n.Value...)
		default:
			result = append(result, extractText(child, source)...)
		}
	}
	return result
}

// parseTable parses a markdown table into our Table struct
func parseTable(tableNode *ext_ast.Table, source []byte) Table {
	table := Table{}
	
	// Parse header - first child should be the header row
	if header := tableNode.FirstChild(); header != nil {
		if tableHeader, ok := header.(*ext_ast.TableHeader); ok {
			// Header row is the first child of TableHeader
			if headerRow := tableHeader.FirstChild(); headerRow != nil {
				if headerRowNode, ok := headerRow.(*ext_ast.TableRow); ok {
					var headerCells []string
					for cell := headerRowNode.FirstChild(); cell != nil; cell = cell.NextSibling() {
						if tableCell, ok := cell.(*ext_ast.TableCell); ok {
							cellText := string(extractText(tableCell, source))
							headerCells = append(headerCells, cellText)
						}
					}
					table.Header = headerCells
				}
			}
		}
	}
	
	// Parse rows - skip the header row and delimiter row
	bodyStarted := false
	for row := tableNode.FirstChild(); row != nil; row = row.NextSibling() {
		// Skip header and delimiter rows
		if !bodyStarted {
			// Check if this is the delimiter row (second row)
			if _, ok := row.(*ext_ast.TableHeader); ok {
				continue // Skip header
			}
			// Next row after header is delimiter, so skip it too
			bodyStarted = true
			continue
		}
		
		// Process actual data rows
		if tableRow, ok := row.(*ext_ast.TableRow); ok {
			var cells []string
			for cell := tableRow.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if tableCell, ok := cell.(*ext_ast.TableCell); ok {
					cellText := string(extractText(tableCell, source))
					cells = append(cells, cellText)
				}
			}
			table.Rows = append(table.Rows, cells)
		}
	}
	
	return table
}

// parseList parses a markdown list into our List struct
func parseList(listNode *ast.List, source []byte) List {
	list := List{
		IsOrdered: listNode.Marker != '*', // Simplified detection
	}
	
	for item := listNode.FirstChild(); item != nil; item = item.NextSibling() {
		if listItem, ok := item.(*ast.ListItem); ok {
			text := string(extractText(listItem, source))
			list.Items = append(list.Items, text)
		}
	}
	
	return list
}

// parseGraph parses HTML blocks to detect graphs (like mermaid)
func parseGraph(htmlNode *ast.HTMLBlock, source []byte) Graph {
	content := string(htmlNode.Lines().Value(source))
	
	// Simple detection for mermaid diagrams
	if len(content) > 15 && content[:10] == "```mermaid" {
		return Graph{
			Type:    "mermaid",
			Content: content,
		}
	}

	// Example for detecting other graph types like PlantUML:
	// if len(content) > 15 && content[:8] == "```plantuml" {
	//     return Graph{
	//         Type:    "plantuml",
	//         Content: content,
	//     }
	// }
	
	// Could add more graph types here
	return Graph{} // Empty graph if not detected
}

// printNode prints a node and its contents
func printNode(node *Node, indent int) {
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += "  "
	}
	
	fmt.Printf("%sHeading: %s (Level %d)\n", indentStr, node.Heading, node.Level)
	
	// Print tables
	for i, table := range node.Tables {
		fmt.Printf("%sTable %d:\n", indentStr, i+1)
		if len(table.Header) > 0 {
			fmt.Printf("%s  Header: %v\n", indentStr, table.Header)
		}
		for j, row := range table.Rows {
			fmt.Printf("%s  Row %d: %v\n", indentStr, j+1, row)
		}
	}
	
	// Print lists
	for i, list := range node.Lists {
		listType := "Unordered"
		if list.IsOrdered {
			listType = "Ordered"
		}
		fmt.Printf("%sList %d (%s):\n", indentStr, i+1, listType)
		for j, item := range list.Items {
			fmt.Printf("%s  %d. %s\n", indentStr, j+1, item)
		}
	}
	
	// Print code blocks
	for i, codeBlock := range node.CodeBlocks {
		fmt.Printf("%sCode Block %d (Language: %s):\n", indentStr, i+1, codeBlock.Language)
		fmt.Printf("%s  Content: %s\n", indentStr, codeBlock.Content)
	}
	
	// Print graphs
	for i, graph := range node.Graphs {
		fmt.Printf("%sGraph %d (Type: %s)\n", indentStr, i+1, graph.Type)
		// Content might be large, so we'll just show size
		fmt.Printf("%s  Size: %d characters\n", indentStr, len(graph.Content))
	}
	
	// Print children
	for _, child := range node.Children {
		printNode(child, indent+1)
	}
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

	// Parse markdown into nodes
	rootNode := parseMarkdown(content)
	
	// Print the parsed nodes
	printNode(rootNode, 0)

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