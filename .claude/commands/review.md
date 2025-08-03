# Code Review and Task Update Command

## Purpose
Review current project implementation, perform code review, and update the TASK.md checklist to reflect completed work.

## Usage
`/review [task_file_path]`

Example: `/review docs/_local/TASK.md`

## Prompt

```
Perform a comprehensive code review and update the task tracking file at {task_file_path}:

## Review Process

### 1. Analyze Current Implementation
- Check git status and diff to understand recent changes (if git repo exists)
- Examine project directory structure
- Review all source files created or modified
- Identify which components have been implemented

### 2. Code Review Checklist
For each implemented component, verify:
- **Correctness**: Does the code match the specification requirements?
- **Completeness**: Are all required functions/methods implemented?
- **Structure**: Does it follow the planned architecture?
- **Dependencies**: Are all necessary imports and packages included?
- **Documentation**: Are there appropriate comments and docs?
- **Testing**: Are unit tests present where required?
- **Best Practices**: Does it follow Go idioms and conventions?

### 3. Update Task Checklist
Based on the review, update the TASK.md file:
- Mark completed tasks with `[x]` 
- Keep incomplete tasks as `[ ]`
- Update progress percentages for each phase
- Update overall progress tracking

### 4. Verification Criteria
A task should only be marked complete if:
- The implementation exists in the codebase
- The code compiles without errors
- Core functionality is implemented (even if tests are pending)
- Required files and structures are in place

### 5. Review Output Format
Provide a summary including:
```markdown
## Review Summary

### Completed Tasks
- ✅ Task name: Brief description of what was implemented
- ✅ Another task: Implementation details

### In Progress
- 🔄 Task name: What's done and what remains

### Not Started
- ⏳ Next priority tasks to tackle

### Progress Update
- Phase 1: X/Y tasks (Z%)
- Phase 2: X/Y tasks (Z%)
- Phase 3: X/Y tasks (Z%)
- Overall: X/Y major tasks (Z%)
```

## Review Patterns

### Project Structure Review
```bash
# Check directory structure
find . -type d -name "*.go" | head -20
ls -la cmd/ internal/ pkg/

# Check for Go files
find . -name "*.go" -type f | wc -l

# Review Makefile targets
make help 2>/dev/null || cat Makefile

# Check go.mod dependencies
cat go.mod
```

### Git Repository Review (if applicable)
```bash
# Check git status
git status

# Review recent commits
git log --oneline -10

# Check changed files
git diff --name-only

# Review unstaged changes
git diff
```

### Code Quality Checks
```bash
# Run go fmt check
go fmt ./...

# Run go vet
go vet ./...

# Check for compilation
go build ./...

# Run tests if available
go test ./...
```

## Task Marking Rules

### Mark as Complete [x]
- File/directory exists as specified
- Basic implementation is present
- Code compiles (if applicable)
- Core functionality works

### Keep as Incomplete [ ]
- File/directory doesn't exist
- Only stub/placeholder code
- Significant functionality missing
- Compilation errors

### Subtask Completion
- Parent task is only complete when ALL subtasks are done
- Partial subtask completion keeps parent incomplete
- Exception: Documentation tasks can be marked separately

## Progress Calculation

### Phase Progress
```
Phase Progress = (Completed Major Tasks / Total Major Tasks) * 100
```

### Overall Progress  
```
Overall Progress = (Total Completed Tasks / Total Tasks) * 100
```

### Update Format
Always update these sections in TASK.md:
1. Individual task checkboxes
2. Phase progress percentages
3. Overall progress percentage
4. Add timestamp comment if significant milestone reached

## Common Review Scenarios

### New Project Setup
- Check for go.mod with correct module name
- Verify directory structure matches specification
- Confirm Makefile has required targets
- Ensure documentation files exist

### Component Implementation
- Verify package structure is correct
- Check all required types/structs defined
- Confirm interfaces match specification
- Validate error handling present

### Configuration Module
- Check YAML struct tags present
- Verify default values implemented
- Confirm environment variable override logic
- Validate config file loading

### HTTP Server/Client
- Check endpoint handlers implemented
- Verify request/response types match API spec
- Confirm error responses handled
- Validate middleware chain setup

Perform the review systematically and update the task file accurately to reflect the true state of the project implementation.
```

## Notes
- This command performs both code review and task tracking updates
- It checks actual implementation against the task list
- Only marks tasks complete when code actually exists
- Provides detailed progress reporting
- Can be run periodically to keep task tracking accurate