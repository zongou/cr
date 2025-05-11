package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/GoToUse/treeprint"
	"github.com/fatih/color"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

var config struct {
	program string

	// Flags
	help     bool
	verbose  bool
	all      bool
	markdown bool
	code     bool

	// Options
	filePath string
}

type MD_NODE struct {
	Heading     ast.Heading
	CodeBlocks  []ast.CodeBlock
	Children    []MD_NODE
	Env         map[string]string
	Parent      *MD_NODE
	Description string
}

// Define a struct for language configuration
type languageConfig struct {
	cmdName    string
	prefixArgs []string
}

// Create a map for language configurations
var languageConfigs = map[string]languageConfig{
	"awk":        {"awk", []string{"$CODE"}},
	"sh":         {"sh", []string{"-euc", "$CODE", "--"}},
	"bash":       {"bash", []string{"-euc", "$CODE", "--"}},
	"zsh":        {"zsh", []string{"-euc", "$CODE", "--"}},
	"fish":       {"fish", []string{"-euc", "$CODE", "--"}},
	"dash":       {"dash", []string{"-euc", "$CODE", "--"}},
	"ksh":        {"ksh", []string{"-euc", "$CODE", "--"}},
	"ash":        {"ash", []string{"-euc", "$CODE", "--"}},
	"shell":      {"sh", []string{"-euc", "$CODE", "--"}},
	"js":         {"node", []string{"-e", "$CODE"}},
	"javascript": {"node", []string{"-e", "$CODE"}},
	"py":         {"python", []string{"-c", "$CODE"}},
	"python":     {"python", []string{"-c", "$CODE"}},
	"rb":         {"ruby", []string{"-e", "$CODE"}},
	"ruby":       {"ruby", []string{"-e", "$CODE"}},
	"php":        {"php", []string{"-r", "$CODE"}},
	"cmd":        {"cmd.exe", []string{"/c", "$CODE"}},
	"batch":      {"cmd.exe", []string{"/c", "$CODE"}},
	"powershell": {"powershell.exe", []string{"-c", "$CODE"}},
}

func infoMsg(format string, a ...interface{}) {
	if config.verbose {
		fmt.Fprintf(os.Stderr, config.program+": "+format, a...)
	}
}

func findDoc(program string) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Check for documentation file in current directory
		patterns := []string{
			path.Join(currentDir, program+".md"),
			path.Join(currentDir, "."+program+".md"),
			path.Join(currentDir, "README.md"),
		}

		for _, pattern := range patterns {
			info, err := os.Stat(pattern)
			if err == nil {
				if info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
					return pattern, nil
				}
			}
		}

		// Move to parent directory
		parentDir := path.Dir(currentDir)
		if parentDir == currentDir {
			break // Reached the root directory
		}
		currentDir = parentDir
	}

	return "", fmt.Errorf("No markdown file found")
}

func getHeadingText(heading ast.Heading) string {
	if len(heading.Children) > 0 {
		if txt, ok := heading.Children[0].(*ast.Text); ok {
			return string(txt.Literal)
		}
	}
	return ""
}

func parseDoc(doc ast.Node) []MD_NODE {
	var commands []MD_NODE
	var stack []*MD_NODE // Track current heading hierarchy

	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.GoToNext
		}

		switch v := node.(type) {
		case *ast.Heading:
			cmdNode := MD_NODE{Heading: *v}

			// Pop stack until we find appropriate parent level
			for len(stack) > 0 && stack[len(stack)-1].Heading.Level >= v.Level {
				stack = stack[:len(stack)-1]
			}

			if len(stack) == 0 {
				commands = append(commands, cmdNode)
				stack = append(stack, &commands[len(commands)-1])
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, cmdNode)
				current := &parent.Children[len(parent.Children)-1]
				current.Parent = parent
				stack = append(stack, current)
			}
		case *ast.Paragraph:
			if len(stack) > 0 {
				current := stack[len(stack)-1]
				if current.Description == "" && len(current.CodeBlocks) == 0 {
					var description strings.Builder
					ast.WalkFunc(node, func(child ast.Node, entering bool) ast.WalkStatus {
						if !entering {
							return ast.GoToNext
						}

						switch v := child.(type) {
						case *ast.Text:
							description.WriteString(strings.ReplaceAll(string(v.Literal), "\n", " "))
						case *ast.Hardbreak:
							description.WriteString("\n")
						}

						return ast.GoToNext
					})
					current.Description = description.String()
				}
			}

		case *ast.CodeBlock:
			if len(stack) > 0 {
				current := stack[len(stack)-1]
				if _, exists := languageConfigs[string(v.Info)]; config.all || exists {
					current.CodeBlocks = append(current.CodeBlocks, *v)
				}
			}

		case *ast.Table:
			if len(stack) > 0 {
				current := stack[len(stack)-1]
				if current.Env == nil {
					current.Env = make(map[string]string)
				}
				ast.WalkFunc(v, func(child ast.Node, entering bool) ast.WalkStatus {
					if !entering {
						return ast.GoToNext
					}

					switch v := child.(type) {
					case *ast.TableRow:
						if len(v.Children) >= 2 {
							keyNode, valNode := v.Children[0], v.Children[1]
							if keyText, ok := keyNode.GetChildren()[0].(*ast.Text); ok {
								if valText, ok := valNode.GetChildren()[0].(*ast.Text); ok {
									current.Env[string(keyText.Literal)] = string(valText.Literal)
								}
							}
						}
					}

					return ast.GoToNext
				})
			}
		}

		return ast.GoToNext
	})

	return commands
}

func executeNode(cmdNode MD_NODE, args []string) error {
	for _, codeBlock := range cmdNode.CodeBlocks {
		info := string(codeBlock.Info) // Convert []byte to string

		// Lookup language configuration
		config, exists := languageConfigs[info]
		if !exists {
			return fmt.Errorf("unsupported code block type: %s", info)
		}

		// Replace $CODE placeholder with the actual code block
		prefixArgs := make([]string, len(config.prefixArgs))
		for i, arg := range config.prefixArgs {
			prefixArgs[i] = strings.Replace(arg, "$CODE", string(codeBlock.Literal), 1)
		}

		cmdArgs := append(prefixArgs, args...)

		// Merge environment variables ensuring current node's variables take precedence
		envMap := make(map[string]string)
		for parent := cmdNode.Parent; parent != nil; parent = parent.Parent {
			for key, value := range parent.Env {
				if _, exists := envMap[key]; !exists {
					envMap[key] = value
				}
			}
		}
		for key, value := range cmdNode.Env {
			envMap[key] = value
		}

		// Convert map to slice of "key=value" strings
		var cmdEnv []string
		for key, value := range envMap {
			cmdEnv = append(cmdEnv, key+"="+value)
		}
		cmdEnv = append(os.Environ(), cmdEnv...)

		// Execute the command
		cmd := exec.Command(config.cmdName, cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = cmdEnv
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error executing command %s with args %v: %w", config.cmdName, cmdArgs, err)
		}
	}

	return nil
}

func findNode(nodes []MD_NODE, heading string) *MD_NODE {
	for _, node := range nodes {
		nodeHeading := getHeadingText(node.Heading)
		if strings.EqualFold(nodeHeading, heading) {
			return &node
		}
		if len(node.Children) > 0 {
			foundNode := findNode(node.Children, heading)
			if foundNode != nil {
				return foundNode
			}
		}
	}
	return nil
}

func showHints(cmdNodes []MD_NODE, verbose bool) {
	if cmdNodes != nil {
		var treeView func(cmdNode MD_NODE, level int, branch treeprint.Tree)
		treeView = func(cmdNode MD_NODE, level int, branch treeprint.Tree) {
			for _, child := range cmdNode.Children {
				if len(child.CodeBlocks) > 0 || len(child.Children) > 0 {
					branch := branch.AddBranch(getHeadingText(child.Heading))

					treeView(child, level+1, branch)
				}
			}
		}

		var treeViewWithDescription func(cmdNode MD_NODE, level int, branch treeprint.Tree, maxLineRuneLen int)
		treeViewWithDescription = func(cmdNode MD_NODE, level int, branch treeprint.Tree, maxLineRuneLen int) {
			for _, child := range cmdNode.Children {
				if len(child.CodeBlocks) > 0 || len(child.Children) > 0 {
					var sb strings.Builder

					heading := getHeadingText(child.Heading)
					headingLowerCased := strings.ToLower(heading)
					sb.WriteString(color.GreenString(headingLowerCased))

					discription := child.Description

					if verbose {
						for k, v := range child.Env {
							envPrettied := color.BlueString(k + "=" + v)
							if discription == "" {
								discription = envPrettied
							} else {
								discription = discription + "\n" + envPrettied
							}
						}
						for _, codeBlock := range child.CodeBlocks {
							codeBlockTrimmed := strings.TrimSuffix(string(codeBlock.Literal), "\n")
							codeBlockPrettied := "```" + string(codeBlock.Info) + "\n" + codeBlockTrimmed + "\n```"
							if discription == "" {
								discription = codeBlockPrettied
							} else {
								discription = discription + "\n" + codeBlockPrettied
							}
						}
					}

					linesOfDescription := strings.Split(discription, "\n")
					for i, line := range linesOfDescription {
						divider := "  "
						if i == 0 {
							sb.WriteString(divider)
							sb.WriteString(strings.Repeat(" ", maxLineRuneLen-(level+1)*4-len([]rune(heading))))
						} else {
							sb.WriteString("\n")
							sb.WriteString(divider)
							sb.WriteString(strings.Repeat(" ", maxLineRuneLen-(level+1)*4))
						}
						sb.WriteString(line)
					}

					branch := branch.AddBranch(sb.String())

					treeViewWithDescription(child, level+1, branch, maxLineRuneLen)
				}
			}
		}

		maxLineRuneLen := 0
		for _, cmdNode := range cmdNodes {
			tree := treeprint.New()
			treeView(cmdNode, 0, tree)
			lines := strings.Split(tree.String(), "\n")
			// Get maxLine length
			for _, line := range lines {
				runeLength := len([]rune(line)) // Returns the number of characters (runes)
				if runeLength > maxLineRuneLen {
					maxLineRuneLen = runeLength
				}
			}

			// fmt.Print(tree.String())
			// fmt.Printf("longestHeadingPath: %v\n", longestHeadingPath)

			treeWithDescription := treeprint.New()
			treeWithDescription.SetValue(getHeadingText(cmdNode.Heading))
			treeViewWithDescription(cmdNode, 0, treeWithDescription, maxLineRuneLen)
			fmt.Println(treeWithDescription.String())
		}
	}
}

func showHelp() {
	const indention = "    "
	var sb strings.Builder

	sb.WriteString("Run markdown codeblocks by its heading.\n\n")
	sb.WriteString(color.YellowString("USAGE:") + "\n")
	sb.WriteString(fmt.Sprintf("%s%s [--file FILE] <heading...> [-- <args...>]\n", indention, config.program))
	sb.WriteString("\n")

	sb.WriteString(color.YellowString("FLAGS:") + "\n")
	sb.WriteString(fmt.Sprintf("%s-h, --help        Show this help\n", indention))
	sb.WriteString(fmt.Sprintf("%s-v, --verbose     Print more information\n", indention))
	sb.WriteString("\n")

	sb.WriteString(color.YellowString("OPTIONS:") + "\n")
	sb.WriteString(fmt.Sprintf("%s-f, --file        MarkDown file to use\n", indention))
	sb.WriteString("\n")

	fmt.Fprint(os.Stderr, sb.String())
}

func main() {
	config.program = path.Base(os.Args[0])

	// Parse options
	arg_index := 1
	for ; arg_index < len(os.Args); arg_index += 1 {
		// fmt.Printf("os.Args[%d]: %v\n", arg_index, os.Args[arg_index])
		current_arg := os.Args[arg_index]
		current_arg_len := len(current_arg)
		if current_arg_len > 1 && current_arg[0] == '-' { // Is option
			if current_arg[1] != '-' { // Is short options
				for short_opt_index := 1; short_opt_index < current_arg_len; short_opt_index++ {
					short_opt := current_arg[short_opt_index]
					switch short_opt {
					case 'v':
						config.verbose = true
						break
					case 'h':
						config.help = true
						break
					case 'a':
						config.all = true
						break
					case 'm':
						config.markdown = true
						break
					case 'c':
						config.code = true
						break
					case 'f':
						if short_opt_index < current_arg_len-1 {
							config.filePath = current_arg[:short_opt_index+1]
						} else {
							if (arg_index < len(os.Args)-1) && os.Args[arg_index+1][0] != '-' {
								config.filePath = os.Args[arg_index+1]
								arg_index += 1
							} else {
								fmt.Errorf("No file path specified after -f\n")
								os.Exit(1)
							}
						}
						short_opt_index = current_arg_len
						break
					default:
						fmt.Errorf("Unknown option: %s\n", current_arg[short_opt_index:])
						os.Exit(1)
					}
				}
			} else { // Is a long option
				switch current_arg {
				case "--verbose":
					config.verbose = true
					break
				case "--help":
					config.help = true
					break
				case "--all":
					config.all = true
					break
				case "--markdown":
					config.markdown = true
					break
				case "--code":
					config.code = true
					break
				default:
					if strings.HasPrefix(current_arg, "--file=") && current_arg_len > 7 {
						config.filePath = current_arg[7:]
					} else if current_arg == "--file" && arg_index < len(os.Args)-1 {
						config.filePath = os.Args[arg_index+1]
					} else {
						fmt.Errorf("Invalid argument: %s", current_arg)
						os.Exit(1)
					}
					break
				}
			}
		} else { // Not an option
			break
		}
	}

	if config.verbose {
		infoMsg("--verbose flag is set\n")
	}

	if config.help {
		infoMsg("--help flag is set\n")
		showHelp()
		os.Exit(0)
	}

	if config.all {
		infoMsg("--all flag is set\n")
	}

	if config.markdown {
		infoMsg("--markdown flag is set\n")
	}

	if config.code {
		infoMsg("--code flag is set\n")
	}

	if config.filePath == "" {
		filePath, err := findDoc(config.program)
		if err == nil {
			config.filePath = filePath
		} else {
			fmt.Errorf("Error: %v\n", err)
		}
	}

	content, err := os.ReadFile(config.filePath)
	if err != nil {
		fmt.Errorf("reading file: %v", err)
		return
	}

	os.Setenv("MD_EXE", os.Args[0])
	os.Setenv("MD_FILE", config.filePath)

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(content)

	mdNodes := parseDoc(doc)

	if arg_index < len(os.Args) {
		heading := os.Args[arg_index]
		sub_args := os.Args[arg_index+1:]
		infoMsg("heading: %s, arguments count: %d\n", heading, len(sub_args))
		nodeFound := findNode(mdNodes, heading)
		nodeHeading := getHeadingText(nodeFound.Heading)

		if nodeFound != nil {
			infoMsg("Found node: %s\n", nodeHeading)
			if config.markdown || config.code {
				if config.markdown {
				}
				if config.code {
					for _, codeBlock := range nodeFound.CodeBlocks {
						fmt.Printf("%s", string(codeBlock.Literal))
					}
				}
			} else {
				executeNode(*nodeFound, sub_args)
			}
		}
	} else {
		showHints(mdNodes, config.verbose)
	}
}
