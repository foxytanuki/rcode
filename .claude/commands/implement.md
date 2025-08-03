# Implementation Command

## Purpose
Implement a specific task or phase from TASK.md, write the code, run tests, and update the task checklist upon completion.

## Usage
`/implement <task_number_or_phase> [task_file_path]`

Examples:
- `/implement 3 docs/_local/TASK.md` - Implement task #3 (Configuration Management)
- `/implement phase1 docs/_local/TASK.md` - Implement all remaining Phase 1 tasks
- `/implement next docs/_local/TASK.md` - Implement the next incomplete task

## Prompt

```
Implement the specified task(s) from {task_file_path} following these steps:

## Implementation Process

### 1. Task Analysis
First, identify the task(s) to implement:
- If a task number is specified, implement that specific major task
- If "phase1/phase2/phase3" is specified, implement all incomplete tasks in that phase
- If "next" is specified, find and implement the next incomplete task
- Read the task details including all subtasks
- Review any related specification or documentation

### 2. Pre-Implementation Setup
Before coding:
- Create necessary directories if they don't exist
- Set up the package structure
- Plan the implementation approach
- Identify dependencies and imports needed

### 3. Implementation Steps
For each task component:

#### a. Create Core Files
- Create the main implementation files as specified
- Add package declarations and imports
- Implement all required structs, interfaces, and types
- Add necessary constants and variables

#### b. Implement Functionality
- Write all required functions and methods
- Implement business logic according to specifications
- Add error handling and validation
- Include logging where appropriate

#### c. Add Tests
- Create corresponding test files (*_test.go)
- Write unit tests for all public functions
- Include edge cases and error scenarios
- Ensure minimum 80% code coverage

#### d. Documentation
- Add package-level documentation
- Document all exported types and functions
- Include usage examples where helpful
- Update README if needed

### 4. Quality Assurance
After implementation:
```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run tests
go test ./... -v

# Check compilation
go build ./...
```

### 5. Implementation Verification
After implementation, verify:
- [ ] All subtasks are implemented
- [ ] Code compiles without errors
- [ ] Tests pass successfully
- [ ] Code follows Go best practices
- [ ] Documentation is complete
- [ ] No TODO comments remain (unless intentional)

Note: Task completion status in TASK.md should be updated using the `/review` command after implementation is verified.

## Implementation Templates

### Package Structure Template
```go
// Package description
package packagename

import (
    "necessary/imports"
)

// Type definitions
type StructName struct {
    Field Type `json:"field" yaml:"field"`
}

// Interface definitions
type InterfaceName interface {
    Method() error
}

// Implementation
func (s *StructName) Method() error {
    // Implementation
    return nil
}
```

### Test File Template
```go
package packagename_test

import (
    "testing"
    "package/path"
)

func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   Type
        want    Type
        wantErr bool
    }{
        {
            name:  "valid case",
            input: value,
            want:  expected,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Function() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Function() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Task-Specific Implementation Guides

### Configuration Management (Task 3)
1. Create types.go with YAML tags for configuration
2. Implement loader with precedence: CLI > ENV > File > Default
3. Add validation for required fields
4. Support hot-reload capability

### HTTP Server (Task 5)
1. Set up router with middleware chain
2. Implement handlers with proper error responses
3. Add graceful shutdown on signals
4. Include health check endpoint

### CLI Client (Task 7)
1. Use flag package or cobra for argument parsing
2. Implement configuration file discovery
3. Add verbose/debug output options
4. Include version information

## Implementation Priorities

### Phase 1 (MVP) Priority Order
1. API types (foundation for communication)
2. Configuration (needed by all components)
3. Logger (used throughout the codebase)
4. HTTP Server (core functionality)
5. Editor Management (core business logic)
6. CLI Client (user interface)
7. Basic tests and documentation

### Best Practices Checklist
- [ ] Use meaningful variable names
- [ ] Keep functions small and focused
- [ ] Handle all errors explicitly
- [ ] Use interfaces for flexibility
- [ ] Write tests alongside implementation
- [ ] Document public APIs
- [ ] Use context for cancellation
- [ ] Implement timeouts where appropriate

## Error Handling Pattern
```go
if err != nil {
    return fmt.Errorf("context: %w", err)
}
```

## Logging Pattern
```go
logger.Info("operation started", 
    "component", "name",
    "action", "description")
```

## Post-Implementation Summary
After completing the implementation, provide:
1. List of files created/modified
2. Key implementation decisions made
3. Any deviations from original specification
4. Test results summary
5. Next recommended tasks

## Notes
- Focus on getting core functionality working first
- Tests can be basic initially but must exist
- Documentation can be minimal but must be accurate
- Follow the project's established patterns
- Ask for clarification if specifications are unclear
- Use `/review` command to update task completion status in TASK.md

Implement the requested task(s) systematically, ensuring each component is complete and tested before proceeding.
```

## Task Selection Logic
- **Specific number**: Implement that major task (e.g., "3" = Configuration Management)
- **Phase keyword**: Implement all remaining tasks in that phase
- **"next"**: Find the first incomplete task and implement it
- **No parameter**: Show available tasks and ask for selection

## Important
- Always run tests after implementation
- Create feature branches for significant changes
- Commit with descriptive messages
- Use `/review` command to verify and update task completion status