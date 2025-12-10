package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

// Node represents a parsed markdown node containing various elements
type Node struct {
	Heading    string
	Level      int
	Tables     []Table
	Lists      []List
	CodeBlocks []CodeBlock
	Paragraphs []string
	KeyValueMaps []map[string]string
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
	Items    []ListItem
	IsOrdered bool
}

// ListItem represents a list item that can be a regular item or a task list item
type ListItem struct {
	Text    string
	Checked *bool // nil for regular items, true/false for task list items
}

// CodeBlock represents a markdown code block
type CodeBlock struct {
	Language string
	Content  string
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
	debugAST bool
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
  --debug-ast             Print AST structure for debugging
`, config.program)
}

// parseMarkdown parses markdown content into a tree of nodes
func parseMarkdown(source []byte) *Node {
	// Create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.Tables
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(source)
	
	if config.debugAST {
		fmt.Println("=== AST Structure ===")
		printAST(doc, source, 0)
		fmt.Println("====================")
	}
	
	root := &Node{
		Heading: "root",
		Level:   0,
	}
	
	parseNode(doc, root, source)
	return root
}

// printAST prints the AST structure for debugging
func printAST(node ast.Node, source []byte, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	
	fmt.Printf("%s%s\n", indent, getNodeTypeName(node))
	
	// Print literal content if available
	if textNode, ok := node.(*ast.Text); ok && len(textNode.Literal) > 0 {
		fmt.Printf("%s  Literal: %q\n", indent, string(textNode.Literal))
	}
	
	// Print children
	for _, child := range node.GetChildren() {
		printAST(child, source, depth+1)
	}
}

// getNodeTypeName returns a string representation of the node type
func getNodeTypeName(node ast.Node) string {
	switch node.(type) {
	case *ast.Document:
		return "Document"
	case *ast.Heading:
		return "Heading"
	case *ast.Text:
		return "Text"
	case *ast.List:
		return "List"
	case *ast.ListItem:
		return "ListItem"
	case *ast.Paragraph:
		return "Paragraph"
	case *ast.CodeBlock:
		return "CodeBlock"
	case *ast.Table:
		return "Table"
	case *ast.TableHeader:
		return "TableHeader"
	case *ast.TableBody:
		return "TableBody"
	case *ast.TableRow:
		return "TableRow"
	case *ast.TableCell:
		return "TableCell"
	case *ast.Emph:
		return "Emph"
	case *ast.Strong:
		return "Strong"
	case *ast.Link:
		return "Link"
	case *ast.Image:
		return "Image"
	default:
		return fmt.Sprintf("Unknown(%T)", node)
	}
}

// parseNode recursively parses AST nodes into our Node structure
func parseNode(astNode ast.Node, parentNode *Node, source []byte) {
	// Keep track of the current node where content should be added
	currentNode := parentNode
	
	// Walk through children
	for _, child := range astNode.GetChildren() {
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
			
			// Update current node to this heading
			currentNode = headingNode
			
			// Continue parsing within this heading node
			parseNode(child, headingNode, source)
			
		case *ast.Table:
			// Parse table
			table := parseTable(n, source)
			
			// Check if this is a key-value table
			kvMap := tryCreateKeyValueMap(table)
			
			if kvMap != nil {
				currentNode.KeyValueMaps = append(currentNode.KeyValueMaps, kvMap)
			} else {
				currentNode.Tables = append(currentNode.Tables, table)
			}
			
		case *ast.List:
			// Parse list
			list := parseList(n, source)
			currentNode.Lists = append(currentNode.Lists, list)
			
			// If this is a task list, also add entries to the key-value map
			kvMap := make(map[string]string)
			hasTaskItems := false
			
			for _, item := range list.Items {
				if item.Checked != nil {
					// This is a task list item
					hasTaskItems = true
					if *item.Checked {
						kvMap[item.Text] = "1" // Checked task
					} else {
						kvMap[item.Text] = "0" // Unchecked task
					}
				}
			}
			
			// If we found task items, add the map to keyValueMaps
			if hasTaskItems {
				currentNode.KeyValueMaps = append(currentNode.KeyValueMaps, kvMap)
			}
			
		case *ast.Paragraph:
			// Parse paragraph
			paragraphText := string(extractText(n, source))
			if strings.TrimSpace(paragraphText) != "" {
				currentNode.Paragraphs = append(currentNode.Paragraphs, paragraphText)
			}
			
		case *ast.CodeBlock:
			// Parse code block
			codeBlock := CodeBlock{
				Language: string(n.Info),
				Content:  string(n.Literal),
			}
			
			currentNode.CodeBlocks = append(currentNode.CodeBlocks, codeBlock)
			
		default:
			// Continue parsing other node types
			parseNode(child, parentNode, source)
		}
	}
}

// tryCreateKeyValueMap checks if a table is a key-value table and creates a map if so
func tryCreateKeyValueMap(table Table) map[string]string {
	// Check if table has exactly two columns
	if len(table.Header) != 2 {
		return nil
	}
	
	// Check if headers are "key" and "value" (case insensitive)
	header1 := strings.ToLower(strings.TrimSpace(table.Header[0]))
	header2 := strings.ToLower(strings.TrimSpace(table.Header[1]))
	
	if !(header1 == "key" && header2 == "value") && 
	   !(header1 == "value" && header2 == "key") {
		return nil
	}
	
	// Create the key-value map
	kvMap := make(map[string]string)
	
	// Populate the map with rows
	for _, row := range table.Rows {
		if len(row) >= 2 {
			var key, value string
			
			// Handle column order (key could be in either column)
			if header1 == "key" {
				key = strings.TrimSpace(row[0])
				value = strings.TrimSpace(row[1])
			} else {
				key = strings.TrimSpace(row[1])
				value = strings.TrimSpace(row[0])
			}
			
			// Only add non-empty keys
			if key != "" {
				kvMap[key] = value
			}
		}
	}
	
	return kvMap
}

// extractText extracts plain text from an AST node
func extractText(node ast.Node, source []byte) []byte {
	var result []byte
	
	// Handle literal text directly
	if textNode, ok := node.(*ast.Text); ok {
		return textNode.Literal
	}
	
	// Handle other nodes recursively
	for _, child := range node.GetChildren() {
		switch n := child.(type) {
		case *ast.Text:
			result = append(result, n.Literal...)
		default:
			result = append(result, extractText(child, source)...)
		}
	}
	
	return result
}

// parseTable parses a markdown table into our Table struct
func parseTable(tableNode *ast.Table, source []byte) Table {
	table := Table{}
	
	// Parse header
	if len(tableNode.Children) > 0 {
		if header, ok := tableNode.Children[0].(*ast.TableHeader); ok {
			if len(header.Children) > 0 {
				if headerRow, ok := header.Children[0].(*ast.TableRow); ok {
					var headerCells []string
					for _, cell := range headerRow.Children {
						if tableCell, ok := cell.(*ast.TableCell); ok {
							cellText := string(extractText(tableCell, source))
							headerCells = append(headerCells, cellText)
						}
					}
					table.Header = headerCells
				}
			}
		}
	}
	
	// Parse rows
	for i, rowNode := range tableNode.Children {
		// Skip header row (index 0)
		if i == 0 {
			continue
		}
		
		if tableRow, ok := rowNode.(*ast.TableBody); ok {
			for _, row := range tableRow.Children {
				if tableRowNode, ok := row.(*ast.TableRow); ok {
					var cells []string
					for _, cell := range tableRowNode.Children {
						if tableCell, ok := cell.(*ast.TableCell); ok {
							cellText := string(extractText(tableCell, source))
							cells = append(cells, cellText)
						}
					}
					table.Rows = append(table.Rows, cells)
				}
			}
		}
	}
	
	return table
}

// parseList parses a markdown list into our List struct
func parseList(listNode *ast.List, source []byte) List {
	list := List{
		IsOrdered: listNode.ListFlags&ast.ListTypeOrdered != 0,
	}
	
	for _, item := range listNode.Children {
		if listItem, ok := item.(*ast.ListItem); ok {
			parsedItem := parseListItem(listItem, source)
			list.Items = append(list.Items, parsedItem)
		}
	}
	
	return list
}

// parseListItem parses a list item, detecting if it's a task list item
func parseListItem(itemNode *ast.ListItem, source []byte) ListItem {
	listItem := ListItem{}
	
	// Check if this is a task list item by examining children
	for _, child := range itemNode.GetChildren() {
		// Look for a paragraph containing the checkbox pattern
		if para, ok := child.(*ast.Paragraph); ok {
			// Extract text from paragraph
			text := string(extractText(para, source))
			
			// Check if it starts with a checkbox pattern
			if len(text) >= 3 {
				// Check for unchecked [ ]
				if text[0] == '[' && text[1] == ' ' && text[2] == ']' {
					checked := false
					listItem.Checked = &checked
					listItem.Text = text[3:] // Remove [ ] prefix
					if len(listItem.Text) > 0 && listItem.Text[0] == ' ' {
						listItem.Text = listItem.Text[1:] // Remove leading space
					}
					return listItem
				}
				
				// Check for checked [x] or [X]
				if text[0] == '[' && (text[1] == 'x' || text[1] == 'X') && text[2] == ']' {
					checked := true
					listItem.Checked = &checked
					listItem.Text = text[3:] // Remove [x] prefix
					if len(listItem.Text) > 0 && listItem.Text[0] == ' ' {
						listItem.Text = listItem.Text[1:] // Remove leading space
					}
					return listItem
				}
			}
			
			// Regular list item
			listItem.Text = text
			return listItem
		}
	}
	
	// Fallback: extract text directly
	listItem.Text = string(extractText(itemNode, source))
	return listItem
}

// printNode prints a node and its contents
func printNode(node *Node, indent int) {
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += "  "
	}
	
	fmt.Printf("%sHeading: %s (Level %d)\n", indentStr, node.Heading, node.Level)
	
	// Print key-value maps
	for i, kvMap := range node.KeyValueMaps {
		fmt.Printf("%sKey-Value Map %d:\n", indentStr, i+1)
		for key, value := range kvMap {
			fmt.Printf("%s  %s = %s\n", indentStr, key, value)
		}
	}
	
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
	
	// Print paragraphs
	for i, paragraph := range node.Paragraphs {
		fmt.Printf("%sParagraph %d: %s\n", indentStr, i+1, paragraph)
	}
	
	// Print lists
	for i, list := range node.Lists {
		listType := "Unordered"
		if list.IsOrdered {
			listType = "Ordered"
		}
		fmt.Printf("%sList %d (%s):\n", indentStr, i+1, listType)
		for j, item := range list.Items {
			if item.Checked != nil {
				// Print explicit true/false values for checked status
				fmt.Printf("%s  %d. checked=%t %s\n", indentStr, j+1, *item.Checked, item.Text)
			} else {
				fmt.Printf("%s  %d. checked=false %s\n", indentStr, j+1, item.Text)
			}
		}
	}
	
	// Print code blocks
	for i, codeBlock := range node.CodeBlocks {
		fmt.Printf("%sCode Block %d (Language: %s):\n", indentStr, i+1, codeBlock.Language)
		fmt.Printf("%s  Content: %s\n", indentStr, codeBlock.Content)
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
		case "--debug-ast":
			config.debugAST = true
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
		fmt.Printf("flags: verbose=%t, help=%t, all=%t, markdown=%t, code=%t, file_path=%s, key=%s, debug_ast=%t\n",
			config.verbose, config.help, config.all, config.markdown, config.code, config.filePath, config.key, config.debugAST)
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