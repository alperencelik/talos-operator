# Architecture Documentation

This directory contains comprehensive documentation about the talos-operator's architecture and internal structure.

## Available Guides

### [Quick Reference](quick-reference.md)
**Start here if you want a fast overview**

A concise guide providing:
- Visual project structure overview
- Quick reference tables for CRDs and controllers
- Common workflows and commands
- Data flow diagrams
- Quick navigation to key files

Perfect for: Developers who want to quickly understand the project layout, find specific components, or refresh their memory.

### [Module by Module Guide](module-by-module-guide.md)
**Read this for deep understanding**

A comprehensive, detailed explanation covering:
- Each module's purpose and responsibilities
- Detailed breakdowns of all 12 project modules
- Complete CRD reference with all fields
- Controller reconciliation logic
- Package utilities and their usage
- Data flow patterns
- Design principles and patterns

Perfect for: New contributors, those implementing features, or anyone wanting to deeply understand how the operator works.

## How to Use These Guides

1. **Quick learner?** Start with the [Quick Reference](quick-reference.md) to get oriented, then dive into specific sections of the [Module by Module Guide](module-by-module-guide.md) as needed.

2. **Deep diver?** Read the [Module by Module Guide](module-by-module-guide.md) from start to finish, using the [Quick Reference](quick-reference.md) as a bookmark.

3. **Working on a specific component?** Use the [Quick Reference](quick-reference.md) "Common Tasks → Files to Check" table to find what you need, then reference the corresponding section in the [Module by Module Guide](module-by-module-guide.md).

## Architecture Overview

```
User defines desired state → CRDs (api/)
                                ↓
Controllers watch and reconcile → Controllers (internal/controller/)
                                ↓
Controllers use tools → Packages (pkg/)
                                ↓
Interact with → Talos API & Kubernetes API
                                ↓
Result → Actual state matches desired state
```

## Key Concepts

### Declarative Management
Users declare what they want (e.g., "I want a 3-node control plane"), and the operator figures out how to make it happen.

### Controller Pattern
Each resource type has a controller that continuously works to make the actual state match the desired state.

### Mode-Based Deployment
The same CRDs work in different environments (container mode for development, metal mode for production) with mode-specific controller logic.

### Modularity
Components are separated by concern:
- API defines the interface
- Controllers implement the logic
- Packages provide reusable tools

## Contributing

When contributing to talos-operator:

1. **Understand the module structure** using these guides
2. **Follow the existing patterns** described in the documentation
3. **Update documentation** if you change architecture or add new modules
4. **Test your changes** against both container and metal modes when applicable

## Questions?

- Check the [Quick Reference](quick-reference.md) for fast answers
- Read the [Module by Module Guide](module-by-module-guide.md) for detailed explanations
- Review the main [Contributing Guide](../contributing.md) for contribution workflow
- Open an issue on GitHub if something is unclear

---

These architecture guides were created to help everyone understand talos-operator better. If you find them helpful, consider contributing improvements!
