use anyhow::{Context, Result};
use clap::Parser;
use pulldown_cmark::{Event, Options, Parser as MdParser, Tag, TagEnd};
use std::collections::HashMap;
use std::env;
use std::fs;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::process::Command;
use unicode_width::UnicodeWidthStr;

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Cli {
    /// Print code block
    #[arg(short = 'c', long = "code")]
    code: bool,

    /// List one command per line
    #[arg(short = '1')]
    one: bool,

    /// Print tree with description
    #[arg(short = 't', long = "tree")]
    tree: bool,

    /// Path to MarkDown file
    #[arg(short = 'f', long = "file", value_names=["PATH"])]
    file: Option<PathBuf>,

    /// Path to log file for diagnostics
    #[arg(short = 'l', long = "log-file", value_names=["PATH"])]
    log_file: Option<PathBuf>,

    /// Heading (as a command)
    #[arg(trailing_var_arg = true, value_names=["HEADING", "ARGS"])]
    command_and_args: Vec<String>,
}

#[derive(Debug, Clone)]
struct CodeBlock {
    lang: String,
    code: String,
}

#[derive(Debug, Default, Clone)]
struct MDNode {
    text: String,
    level: usize,
    description: String,
    code_blocks: Vec<CodeBlock>,
    children: Vec<MDNode>,
    used: bool,
}

struct App {
    executors: HashMap<String, Vec<String>>,
    custom_executors: HashMap<String, Vec<String>>,
    log_file: Option<PathBuf>,
}

impl App {
    fn new() -> Self {
        let mut executors = HashMap::with_capacity(20);
        macro_rules! e { ($k:expr => [$($v:expr),+ $(,)?]) => { executors.insert($k.to_string(), vec![$($v.to_string()),+]); } }
        e!("sh" => ["sh","-euc","{CODE}","--"]);
        e!("bash" => ["bash","-euc","{CODE}","--"]);
        e!("zsh" => ["zsh","-euc","{CODE}","--"]);
        e!("fish" => ["fish","-euc","{CODE}","--"]);
        e!("dash" => ["dash","-euc","{CODE}","--"]);
        e!("ksh" => ["ksh","-euc","{CODE}","--"]);
        e!("ash" => ["ash","-euc","{CODE}","--"]);
        e!("awk" => ["awk","{CODE}"]);
        e!("js" => ["node","-e","{CODE}"]);
        e!("javascript" => ["node","-e","{CODE}"]);
        e!("py" => ["python","-c","{CODE}"]);
        e!("python" => ["python","-c","{CODE}"]);
        e!("rb" => ["ruby","-e","{CODE}"]);
        e!("ruby" => ["ruby","-e","{CODE}"]);
        e!("php" => ["php","-r","{CODE}"]);
        e!("cmd" => ["cmd.exe","/c","{CODE}"]);
        e!("batch" => ["cmd.exe","/c","{CODE}"]);
        e!("ps2" => ["powershell.exe","-c","{CODE}"]);
        e!("powershell" => ["powershell.exe","-c","{CODE}"]);

        Self {
            executors,
            custom_executors: HashMap::new(),
            log_file: None,
        }
    }

    fn parse_custom_executors(&mut self) {
        for (k, v) in env::vars() {
            if let Some(rest) = k.strip_prefix("MD_") {
                let lang = rest.to_lowercase();
                let parts: Vec<String> = v.split(',').map(|s| s.to_string()).collect();
                self.custom_executors.insert(lang, parts);
            }
        }
    }

    fn get_executor(&self, lang: &str) -> Option<&[String]> {
        self.custom_executors
            .get(lang)
            .map(|v| v.as_slice())
            .or_else(|| self.executors.get(lang).map(|v| v.as_slice()))
    }

    fn find_doc(&self) -> Option<PathBuf> {
        let mut dir = env::current_dir().ok()?;
        let candidates = ["taskfile.md", ".taskfile.md", "README.md"];
        loop {
            for name in &candidates {
                let p = dir.join(name);
                if p.is_file() {
                    return Some(p);
                }
            }
            if !dir.pop() {
                break;
            }
        }
        None
    }

    fn attach_code(nodes: &mut [MDNode], heading: &str, lang: &str, code: &str) -> bool {
        for node in nodes.iter_mut() {
            if node.text.eq_ignore_ascii_case(heading) {
                node.code_blocks.push(CodeBlock {
                    lang: lang.to_string(),
                    code: code.to_string(),
                });
                return true;
            }
            if !node.children.is_empty()
                && Self::attach_code(&mut node.children, heading, lang, code)
            {
                return true;
            }
        }
        false
    }

    fn attach_description(nodes: &mut [MDNode], heading: &str, desc: &str) -> bool {
        for node in nodes.iter_mut() {
            if node.text.eq_ignore_ascii_case(heading) {
                if node.description.is_empty() && node.code_blocks.is_empty() {
                    node.description = desc.to_string();
                }
                return true;
            }
            if !node.children.is_empty()
                && Self::attach_description(&mut node.children, heading, desc)
            {
                return true;
            }
        }
        false
    }

    fn parse_file(&self, path: &Path) -> Result<Vec<MDNode>> {
        let mut src = String::new();
        fs::File::open(path)?.read_to_string(&mut src)?;
        let options = Options::empty();
        let parser = MdParser::new_ext(&src, options);

        // Single pass: collect headings, code blocks, and descriptions
        let mut nodes: Vec<MDNode> = Vec::new();
        let mut stack: Vec<*mut MDNode> = Vec::new();
        let mut in_heading = false;
        let mut cur_level = 0usize;
        let mut cur_buf = String::new();

        let mut last_heading: Option<String> = None;
        let mut para_buf = String::new();
        let mut in_codeblock = false;
        let mut code_buf = String::new();
        let mut code_lang = String::new();

        for event in parser {
            match event {
                Event::Start(Tag::Heading { level, .. }) => {
                    in_heading = true;
                    cur_buf.clear();
                    cur_level = match level {
                        pulldown_cmark::HeadingLevel::H1 => 1,
                        pulldown_cmark::HeadingLevel::H2 => 2,
                        pulldown_cmark::HeadingLevel::H3 => 3,
                        pulldown_cmark::HeadingLevel::H4 => 4,
                        pulldown_cmark::HeadingLevel::H5 => 5,
                        pulldown_cmark::HeadingLevel::H6 => 6,
                    };
                }
                Event::End(TagEnd::Heading(_)) => {
                    in_heading = false;
                    let trimmed = cur_buf.trim();
                    if !trimmed.is_empty() {
                        last_heading = Some(trimmed.to_string());
                    }
                    let node = MDNode {
                        text: trimmed.to_string(),
                        level: cur_level,
                        ..Default::default()
                    };
                    unsafe {
                        while let Some(&ptr) = stack.last() {
                            if (*ptr).level < node.level {
                                break;
                            }
                            stack.pop();
                        }
                        if let Some(&parent_ptr) = stack.last() {
                            (*parent_ptr).children.push(node);
                            let child_ptr = (*parent_ptr).children.last_mut().unwrap() as *mut _;
                            stack.push(child_ptr);
                        } else {
                            nodes.push(node);
                            let root_ptr = nodes.last_mut().unwrap() as *mut _;
                            stack.push(root_ptr);
                        }
                    }
                }
                Event::Text(t) | Event::Code(t) => {
                    if in_heading {
                        cur_buf.push_str(&t);
                    } else if in_codeblock {
                        code_buf.push_str(&t);
                    } else {
                        para_buf.push_str(&t);
                    }
                }
                Event::Start(Tag::CodeBlock(kind)) => {
                    in_codeblock = true;
                    code_buf.clear();
                    code_lang = match kind {
                        pulldown_cmark::CodeBlockKind::Fenced(info) => {
                            info.split_whitespace().next().unwrap_or("").to_string()
                        }
                        pulldown_cmark::CodeBlockKind::Indented => String::new(),
                    };
                }
                Event::End(TagEnd::CodeBlock) => {
                    in_codeblock = false;
                    if let Some(ref h) = last_heading {
                        Self::attach_code(&mut nodes, h, &code_lang, &code_buf);
                    }
                    code_buf.clear();
                }
                Event::Start(Tag::Paragraph) => {
                    para_buf.clear();
                }
                Event::End(TagEnd::Paragraph) => {
                    if let Some(ref h) = last_heading {
                        Self::attach_description(&mut nodes, h, para_buf.trim());
                    }
                    para_buf.clear();
                }
                _ => {}
            }
        }

        Ok(nodes)
    }

    fn print_tree(&self, nodes: &mut [MDNode]) {
        fn mark_used(nodes: &mut [MDNode], app: &App) -> bool {
            let mut any_used = false;
            for node in nodes.iter_mut() {
                let mut used = false;
                for cb in &node.code_blocks {
                    if app.get_executor(&cb.lang).is_some() {
                        used = true;
                        break;
                    }
                }
                if !node.children.is_empty() && mark_used(&mut node.children, app) {
                    used = true;
                }
                node.used = used;
                if used {
                    any_used = true;
                }
            }
            any_used
        }

        fn compute_max_branch_width_for(node: &MDNode) -> usize {
            let mut max = 0usize;
            fn walk(n: &MDNode, max: &mut usize) {
                if n.used {
                    let branch_width = if n.level >= 1 { (n.level - 1) * 4 } else { 0 }
                        + UnicodeWidthStr::width(n.text.as_str());
                    if branch_width > *max {
                        *max = branch_width;
                    }
                    for c in &n.children {
                        walk(c, max);
                    }
                }
            }
            for c in &node.children {
                walk(c, &mut max);
            }
            max
        }

        fn print_subtree(node: &MDNode, _app: &App, prefix: &str, max_branch: usize) {
            let children: Vec<&MDNode> = node.children.iter().filter(|c| c.used).collect();
            for (i, child) in children.iter().enumerate() {
                let last = i + 1 == children.len();
                let branch = if last { "└── " } else { "├── " };
                let branch_width = if child.level >= 1 {
                    (child.level - 1) * 4
                } else {
                    0
                } + UnicodeWidthStr::width(child.text.as_str());
                let pad = max_branch.saturating_sub(branch_width);
                let name = child.text.to_lowercase();
                println!(
                    "{}{}{} {} {}",
                    prefix,
                    branch,
                    name,
                    " ".repeat(pad),
                    child.description
                );
                let next_prefix = if last {
                    format!("{}    ", prefix)
                } else {
                    format!("{}│   ", prefix)
                };
                print_subtree(child, _app, &next_prefix, max_branch);
            }
        }

        mark_used(nodes, self);
        for node in nodes.iter_mut() {
            if node.used {
                let max_branch = compute_max_branch_width_for(node);
                println!("{}", node.text);
                print_subtree(node, self, "", max_branch);
            }
        }
    }

    fn print_one(&self, nodes: &[MDNode]) {
        fn walk(app: &App, nodes: &[MDNode]) {
            for n in nodes {
                if !n.code_blocks.is_empty() && app.get_executor(&n.code_blocks[0].lang).is_some() {
                    println!("{}", n.text.to_lowercase());
                }
                if !n.children.is_empty() {
                    walk(app, &n.children);
                }
            }
        }
        walk(self, nodes);
    }

    fn exec_node(&self, node: &MDNode, origin_args: &[String], file_dir: &Path) -> i32 {
        if node.code_blocks.is_empty() {
            eprintln!("no code blocks under this heading");
            return 1;
        }
        for cb in &node.code_blocks {
            let lang = cb.lang.as_str();
            let Some(exe) = self.get_executor(lang) else {
                eprintln!("unsupported code block type: {}", lang);
                return 1;
            };
            let mut formatted: Vec<String> = Vec::with_capacity(exe.len() + origin_args.len());
            for a in exe {
                if a.contains("{LANG}") || a.contains("{CODE}") {
                    formatted.push(a.replace("{LANG}", lang).replace("{CODE}", &cb.code));
                } else {
                    formatted.push(a.clone());
                }
            }
            formatted.extend_from_slice(origin_args);
            if formatted.is_empty() {
                eprintln!("no executor for language");
                return 1;
            }
            let mut cmd = Command::new(&formatted[0]);
            if formatted.len() > 1 {
                cmd.args(&formatted[1..]);
            }
            cmd.current_dir(file_dir);
            cmd.stdin(std::process::Stdio::inherit());
            cmd.stdout(std::process::Stdio::inherit());
            cmd.stderr(std::process::Stdio::inherit());
            match cmd.status() {
                Ok(s) => {
                    if !s.success() {
                        return s.code().unwrap_or(1);
                    }
                }
                Err(e) => {
                    eprintln!("Error executing command: {}", e);
                    return 1;
                }
            }
        }
        0
    }
}

fn main() -> Result<()> {
    let cli = Cli::parse();
    let mut app = App::new();
    if let Some(p) = cli.log_file.as_ref() {
        app.log_file = Some(p.clone());
    }
    app.parse_custom_executors();

    let file_path = if let Some(p) = cli.file.as_ref() {
        p.clone()
    } else if let Some(found) = app.find_doc() {
        found
    } else {
        eprintln!("No markdown file found");
        std::process::exit(1);
    };

    let file_dir = fs::canonicalize(&file_path)?
        .parent()
        .map(|p| p.to_path_buf())
        .unwrap_or_else(|| PathBuf::from("."));

    env::set_var("CR_FILE", &file_path);
    env::set_var(
        "CR",
        env::current_exe()
            .unwrap_or_else(|_| PathBuf::from("cr"))
            .to_string_lossy()
            .as_ref(),
    );

    let mut nodes = app.parse_file(&file_path).context("parsing markdown")?;

    if let Some(heading) = cli.command_and_args.first() {
        let subcommand_args: Vec<String> = cli.command_and_args.iter().skip(1).cloned().collect();
        fn find<'a>(nodes: &'a [MDNode], h: &str) -> Option<&'a MDNode> {
            for node in nodes {
                if node.text.eq_ignore_ascii_case(h) {
                    return Some(node);
                }
                if !node.children.is_empty() {
                    if let Some(found) = find(&node.children, h) {
                        return Some(found);
                    }
                }
            }
            None
        }
        if let Some(found) = find(&nodes, heading) {
            if cli.code {
                for cb in &found.code_blocks {
                    print!("{}", cb.code);
                }
            } else if cli.one {
                app.print_one(std::slice::from_ref(found));
            } else if cli.tree {
                app.print_tree(&mut [found.clone()]);
            } else {
                let status = app.exec_node(found, &subcommand_args, &file_dir);
                std::process::exit(status);
            }
        } else {
            eprintln!("Node not found: {}", heading);
            std::process::exit(1);
        }
    } else {
        if cli.one {
            app.print_one(&mut nodes);
        } else {
            app.print_tree(&mut nodes);
        }
    }
    Ok(())
}
