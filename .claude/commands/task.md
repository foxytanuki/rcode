# Task Generation Command

## Purpose
Generate a comprehensive, trackable implementation task list (TASK.md) from a specification document (SPEC.md).

## Usage
`/task <spec_file_path>`

Example: `/task docs/_local/SPEC.md`

## Prompt

```
Given the specification document at {spec_file_path}, create a comprehensive TASK.md file that:

1. **Analyzes the specification** to identify all implementation requirements
2. **Organizes tasks into logical phases**:
   - Phase 1: MVP (Minimum Viable Product) - Core functionality only
   - Phase 2: Complete Version - Full feature set
   - Phase 3: Extended Version - Advanced features and polish

3. **Breaks down each phase** into major task groups with:
   - Clear, actionable task names
   - Detailed subtasks with checkboxes for tracking
   - Logical dependencies between tasks
   - Appropriate granularity (not too broad, not too detailed)

4. **Includes supplementary sections**:
   - Testing Checklist (unit, integration, end-to-end)
   - Performance Targets with measurable goals
   - Security Checklist for vulnerability prevention
   - Documentation Requirements
   - Maintenance Tasks for long-term sustainability

5. **Adds progress tracking**:
   - Progress indicators for each phase
   - Overall completion percentage
   - Notes section for development guidelines

## Task Structure Template

Each major task should follow this pattern:
```markdown
### [Number]. [Task Group Name]
- [ ] Main implementation file
  - [ ] Specific component or function
  - [ ] Another component
  - [ ] Tests for this component
- [ ] Supporting file
  - [ ] Configuration
  - [ ] Validation
- [ ] Documentation
```

## Key Principles

1. **Actionable**: Each task should be a concrete action that can be completed
2. **Measurable**: Tasks should have clear completion criteria
3. **Ordered**: Tasks within a phase should follow logical dependencies
4. **Testable**: Include test tasks alongside implementation tasks
5. **Balanced**: Break large tasks into 3-5 subtasks, combine tiny tasks

## Output Requirements

- Create a TASK.md file in the same directory as the specification
- Use markdown checkbox format for all tasks: `- [ ]`
- Number all major task groups sequentially
- Include progress tracking sections
- Ensure all features from the spec are covered
- Add helpful notes about development workflow

## Example Task Breakdown

For a typical Go project component:
```markdown
### 3. Configuration Management (internal/config)
- [ ] Create `internal/config/types.go` with config structures
  - [ ] Define main Config struct
  - [ ] Define sub-configuration types
  - [ ] Add YAML/JSON tags
- [ ] Implement `internal/config/loader.go`
  - [ ] Load from file
  - [ ] Apply environment overrides
  - [ ] Apply CLI overrides
  - [ ] Set defaults
- [ ] Create `internal/config/validator.go`
  - [ ] Validate required fields
  - [ ] Check value ranges
  - [ ] Verify dependencies
- [ ] Write unit tests for config management
```

Generate the TASK.md file based on the provided specification, ensuring comprehensive coverage of all requirements while maintaining practical, implementable task sizes.
```

## Notes
- The command reads the specification and generates a detailed task list
- Tasks are organized by implementation phases for better project management
- Each task includes checkboxes for progress tracking
- The generated file can be used with project management tools that support markdown checklists