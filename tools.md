# AI Assistant Tools Documentation

This document describes all the tools available to the AI assistant and their functionalities.

## Tool List

| Tool | Basic Functionality |
|------|---------------------|
<!-- | `read` | Read files or directories from the local filesystem | -->
<!-- | `write` | Write or overwrite files on the local filesystem | -->
<!-- | `edit` | Perform exact string replacements in existing files | -->
<!-- | `glob` | Fast file pattern matching (e.g., `**/*.js`) | -->
| `grep` | Fast content search using regular expressions |
<!-- | `bash` | Execute bash commands in a persistent shell session | -->
<!-- | `webfetch` | Fetch and convert web content from URLs | -->
<!-- | `websearch` | Search the web using Exa AI | -->
| `question` | Ask the user questions during execution |
| `todowrite` | Create and manage structured task lists |
| `skill` | Load specialized domain-specific skills |

---

## In-Depth Documentation

### read

Reads a file or directory from the local filesystem.

**Parameters:**
- `filePath` (string, required): Absolute path to the file or directory
- `limit` (number, optional): Maximum number of lines to read (default: 2000)
- `offset` (number, optional): Line number to start reading from (1-indexed)

**Usage Notes:**
- Returns content with each line prefixed by line numbers
- Directories show entries one per line with trailing `/` for subdirectories
- Files longer than 2000 characters have lines truncated
- Can read image files and PDFs and return them as file attachments

---

### write

Writes content to a file on the local filesystem, creating it or overwriting it.

**Parameters:**
- `filePath` (string, required): Absolute path to the file
- `content` (string, required): Content to write

**Usage Notes:**
- **MUST** use the Read tool first before writing to an existing file
- Can create new files without prior reading
- Use this for creating documentation, source code, configuration files, etc.

---

### edit

Performs exact string replacements in existing files.

**Parameters:**
- `filePath` (string, required): Absolute path to the file to modify
- `oldString` (string, required): Text to find and replace
- `newString` (string, required): Replacement text
- `replaceAll` (boolean, optional): Replace all occurrences (default: false)

**Usage Notes:**
- **MUST** use the Read tool first before editing
- Match must be exact including whitespace and indentation
- If `oldString` appears multiple times, provide more context to identify the correct location
- The new string must be different from oldString

---

### glob

Fast file pattern matching tool that works with any codebase size.

**Parameters:**
- `pattern` (string, required): Glob pattern (e.g., `**/*.js`, `src/**/*.ts`)
- `path` (string, optional): Directory to search in (defaults to current working directory)

**Usage Notes:**
- Supports glob syntax like `**` for recursive matching
- Returns matching file paths sorted by modification time
- Ideal for finding files by name patterns across large codebases

---

### grep

Fast content search tool that searches file contents using regular expressions.

**Parameters:**
- `pattern` (string, required): Regex pattern to search for
- `include` (string, optional): File pattern to include (e.g., `*.js`, `*.{ts,tsx}`)
- `path` (string, optional): Directory to search in (defaults to current working directory)

**Usage Notes:**
- Returns file paths and line numbers with at least one match
- Results sorted by modification time
- Supports full regex syntax (e.g., `log.*Error`, `function\s+\w+`)
- Do not use for counting matches - use bash with `rg` instead

---

### bash

Executes bash commands in a persistent shell session.

**Parameters:**
- `command` (string, required): The command to execute
- `description` (string, optional): Clear, concise description of what the command does
- `timeout` (number, optional): Timeout in milliseconds (default: 120000ms)
- `workdir` (string, optional): Working directory for the command

**Usage Notes:**
- Commands run in the workspace root by default
- Use `workdir` parameter instead of `cd` commands
- Avoid using for file operations (read, write, edit) or searching (find, grep) - use specialized tools
- Output truncated at 2000 lines or 51200 bytes

---

### webfetch

Fetches content from a specified URL and converts it to a requested format.

**Parameters:**
- `url` (string, required): Fully-formed valid URL
- `format` (string, optional): Output format - "text", "markdown", or "html" (default: "markdown")
- `timeout` (number, optional): Timeout in seconds (max: 120)

**Usage Notes:**
- HTTP URLs automatically upgraded to HTTPS
- Returns content in the specified format
- Results may be summarized if content is very large
- Use this for retrieving and analyzing web content

---

### websearch

Searches the web using Exa AI and can scrape content from specific URLs.

**Parameters:**
- `query` (string, required): Search query
- `numResults` (number, optional): Number of results to return (default: 8)
- `type` (string, optional): Search type - "auto", "fast", or "deep" (default: "auto")
- `livecrawl` (string, optional): "fallback" or "preferred" for live crawling (default: "fallback")
- `contextMaxCharacters` (number, optional): Maximum characters for context (default: 10000)

**Usage Notes:**
- Ideal for current events and recent data
- Returns content from the most relevant websites
- Supports live crawling modes for up-to-date information
- Current year is 2026 - use this when searching for recent information

---

### codesearch

Searches and gets relevant context for programming tasks using Exa Code API.

**Parameters:**
- `query` (string, required): Search query for APIs, libraries, or frameworks
- `tokensNum` (number, optional): Number of tokens to return (default: 5000, range: 1000-50000)

**Usage Notes:**
- Returns comprehensive code examples, documentation, and API references
- Optimized for finding specific programming patterns and solutions
- Use higher values for comprehensive documentation, lower for focused queries
- Ideal for understanding how to use specific libraries or APIs

---

### question

Allows the AI to ask the user questions during execution.

**Parameters:**
- `questions` (array, required): Array of question objects
  - `question` (string): Complete question
  - `header` (string): Very short label (max 30 chars)
  - `options` (array): Available choices with `label` and `description`
  - `multiple` (boolean): Allow selecting multiple choices

**Usage Notes:**
- Automatically adds a "Type your own answer" option when custom is enabled
- Use for gathering user preferences, clarifying instructions, or offering choices
- Recommended options should be listed first with "(Recommended)" appended

---

### task

Launches a new agent to handle complex, multistep tasks autonomously.

**Parameters:**
- `description` (string, required): Short description of the task (3-5 words)
- `prompt` (string, required): Detailed task description for the agent
- `subagent_type` (string, required): Type of specialized agent to use
- `command` (string, optional): The command that triggered this task
- `task_id` (string, optional): Resume a previous subagent session

**Usage Notes:**
- Available subagent types: "general" (general-purpose), "explore" (codebase exploration)
- Use for complex tasks requiring multiple steps or extensive searching
- Agents start with fresh context unless `task_id` is provided
- Can launch multiple agents concurrently for parallel execution

---

### todowrite

Creates and manages a structured task list for tracking progress.

**Parameters:**
- `todos` (array, required): Array of task objects
  - `content` (string): Brief description of the task
  - `priority` (string): Priority level - "high", "medium", or "low"
  - `status` (string): Current status - "pending", "in_progress", "completed", or "cancelled"

**Usage Notes:**
- Use for complex multistep tasks (3+ distinct steps)
- Use for non-trivial tasks requiring careful planning
- Update status in real-time as work progresses
- Only one task should be "in_progress" at a time

---

### skill

Loads specialized skills that provide domain-specific instructions and workflows.

**Parameters:**
- `name` (string, required): Name of the skill from available_skills

**Usage Notes:**
- Skills provide specialized instructions for specific tasks
- Currently no skills are available in the default configuration
- May be extended based on user needs or system configuration

---

## Best Practices

1. **Use the right tool for the job**: Each tool is optimized for specific tasks
2. **Read before write/edit**: Always read existing files before modifying them
3. **Prefer specialized search tools**: Use glob/grep over bash find/grep
4. **Use task lists for complex work**: Track progress on multi-step tasks
5. **Ask questions when needed**: Use the question tool to clarify requirements
6. **Leverage sub-agents**: Use the task tool for complex exploratory work