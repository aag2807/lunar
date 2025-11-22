# Lunar Package Manager - Implementation Status

> This document tracks what's already implemented vs. what still needs to be built

Last Updated: 2025-11-22

---

## ✅ Already Implemented

### Package Manager CLI (`lunar` command)

**Core Commands:**
- ✅ `lunar init` - Create lunar.json manifest
  - ✅ Interactive prompts
  - ✅ Skip prompts with `-y` flag
  - ✅ Set project name with `--name`
  - ✅ Enable strict mode with `--strict`
- ✅ `lunar add <package>` - Add a package dependency
  - ✅ Add to regular dependencies
  - ✅ Add to devDependencies with `--dev`
  - ✅ Support GitHub repos (`user/repo`)
  - ✅ Support git URLs (`https://github.com/user/repo.git`)
  - ✅ Support version specs (`package@version`)
- ✅ `lunar remove <package>` - Remove a package
  - ✅ Remove from manifest
  - ✅ Remove from .lunar/ directory
- ✅ `lunar install` - Install all dependencies from lunar.json
  - ✅ Install regular dependencies
  - ✅ Install devDependencies
  - ✅ Create .lunar/ directory
- ✅ `lunar run <script>` - Run scripts from lunar.json
  - ✅ Execute defined scripts
  - ✅ Pass arguments to scripts

**Package Manifest (lunar.json):**
- ✅ Full manifest structure defined
  - ✅ name, version, description
  - ✅ main entry point
  - ✅ scripts object
  - ✅ dependencies and devDependencies
  - ✅ compilerOptions (target, strict, sourceMap, bundle, etc.)
  - ✅ keywords, author, license
  - ✅ repository info
- ✅ Default manifest generation
- ✅ Load/save functionality
- ✅ Find manifest in parent directories
- ✅ Add/remove dependencies programmatically

**Package Installation:**
- ✅ Git-based installation
  - ✅ Clone from GitHub
  - ✅ Support full git URLs
  - ✅ `--depth 1` shallow clones
  - ✅ Clean up .git directory after install
- ✅ LuaRocks fallback
  - ✅ Detect if luarocks is available
  - ✅ Install from LuaRocks as fallback
- ✅ .lunar/ modules directory
  - ✅ Packages installed to `.lunar/<package-name>`
- ✅ Package spec parsing
  - ✅ `name@version`
  - ✅ `user/repo` (GitHub shorthand)
  - ✅ Full git URLs
  - ✅ Plain names (defaults to latest)

**Default Source Resolution:**
- ✅ Defaults to `lunar-lang` GitHub organization
- ✅ Package name -> `github.com/lunar-lang/<name>`

**Project Scaffolding:**
- ✅ `lunar create <name>` command
- ✅ Template system implemented
  - ✅ basic template
  - ✅ cli template
  - ✅ web template
  - ✅ library template
- ✅ `lunar create list` - List available templates

---

## 🚧 Partially Implemented

### Package Manager
- ⚠️ **Version Resolution:** Currently only supports "latest" or git-based installs
  - ❌ No semantic versioning resolution
  - ❌ No version range support (`^1.2.3`, `~1.2.3`)
  - ❌ No lock file mechanism
  - ❌ No dependency tree resolution
  - ❌ No conflict detection

- ⚠️ **Package Source:**
  - ✅ Git URLs work
  - ✅ LuaRocks fallback works
  - ❌ No actual Lunar package registry yet
  - ❌ Hardcoded to `lunar-lang` org on GitHub

---

## ❌ Not Implemented (From TODO)

### Package Registry Server
- ❌ No registry server at all
- ❌ No package hosting
- ❌ No package search (beyond git)
- ❌ No package publishing API
- ❌ No user authentication
- ❌ No package metadata storage
- ❌ No download statistics
- ❌ No package discovery web interface

### Advanced CLI Features (From TODO)
- ❌ `lunarpkg` separate binary (currently just `lunar` subcommands)
- ❌ `lunar search` - Search registry
- ❌ `lunar info` - Show package details
- ❌ `lunar list` - List installed packages (tree view)
- ❌ `lunar update` - Update packages
- ❌ `lunar outdated` - Check for outdated packages
- ❌ `lunar audit` - Security vulnerability scanning
- ❌ `lunar doctor` - Diagnose issues
- ❌ `lunar link` - Local package development
- ❌ `lunar publish` - Publish to registry
- ❌ `lunar login/logout` - Authentication

### Missing Package Features
- ❌ Lock files (`lunar-lock.json`)
- ❌ Deterministic installs
- ❌ Version pinning
- ❌ Integrity hashes/checksums
- ❌ Dependency resolution algorithm
- ❌ Peer dependencies
- ❌ Optional dependencies
- ❌ Circular dependency detection
- ❌ Pre/post install hooks
- ❌ Binary packages
- ❌ Scoped packages (`@org/package`)
- ❌ Private packages
- ❌ Package exclusions (`.lunarignore`)

### Missing Registry Features (All)
Everything in `PACKAGE_REGISTRY_TODO.md` Phase 3-10

---

## 📊 Current Status Summary

| Feature Category | Implementation | Notes |
|-----------------|----------------|-------|
| **CLI Basic Commands** | ✅ 80% | Core commands done, missing advanced features |
| **Manifest (lunar.json)** | ✅ 100% | Fully specified and working |
| **Git Installation** | ✅ 100% | Works well |
| **LuaRocks Integration** | ✅ 80% | Basic fallback works, no type generation |
| **Version Resolution** | ❌ 10% | Only supports "latest" |
| **Lock Files** | ❌ 0% | Not implemented |
| **Registry Server** | ❌ 0% | Doesn't exist |
| **Web Interface** | ❌ 0% | Doesn't exist |
| **Publishing** | ❌ 0% | No way to publish |
| **Package Search** | ❌ 0% | No registry to search |

**Overall Package Manager Completion: ~30%**
- ✅ Basic local package management works
- ✅ Can install from Git/GitHub
- ❌ No real registry infrastructure
- ❌ No publishing workflow
- ❌ No ecosystem yet

---

## 🎯 What Works Today

You can already:

```bash
# Create a new project
lunar init
lunar init -y --name my-app

# Add dependencies from GitHub
lunar add user/some-lunar-package
lunar add https://github.com/user/repo.git
lunar add some-package --dev

# Install all dependencies
lunar install

# Remove packages
lunar remove some-package

# Run scripts
lunar run build
lunar run dev
lunar run test

# Create new projects from templates
lunar create my-cli --template cli
lunar create my-lib --template library
```

**Working manifest example:**
```json
{
  "name": "my-project",
  "version": "1.0.0",
  "main": "main.lunar",
  "scripts": {
    "build": "lunar --bundle main.lunar -o dist/bundle.lua",
    "dev": "lunar --watch --run main.lunar",
    "test": "lunar --test tests/"
  },
  "dependencies": {
    "some-lib": "https://github.com/user/some-lib.git"
  },
  "devDependencies": {
    "test-framework": "https://github.com/lunar-lang/test.git"
  },
  "compilerOptions": {
    "target": "lua53",
    "strict": true,
    "bundle": false
  }
}
```

---

## 🚀 Next Steps (Priority Order)

Based on what's already implemented, here's what should be built next:

### Phase 1: Enhance Existing CLI (1-2 weeks)
1. **Lock file implementation**
   - Create `lunar-lock.json` format
   - Generate on install
   - Use for reproducible builds

2. **Better version support**
   - Parse semantic versions
   - Support version ranges
   - Implement basic resolution

3. **Add missing CLI commands**
   - `lunar list` - Show installed packages
   - `lunar outdated` - Check for updates
   - `lunar clean` - Remove .lunar/

4. **Improve error handling**
   - Better error messages
   - Validation of lunar.json
   - Network error retries

### Phase 2: Simple Registry (2-3 months)
1. **Minimal registry server**
   - Simple API for package metadata
   - File-based storage (later move to DB)
   - Package tarball hosting

2. **Publishing workflow**
   - `lunar publish` command
   - Authentication (API keys)
   - Version validation

3. **Basic web interface**
   - List packages
   - Search
   - Package detail pages

### Phase 3: Full Registry (3-6 months)
Follow `PACKAGE_REGISTRY_TODO.md` phases

---

## 💡 Key Insights

### What's Good
1. **Solid Foundation:** The lunar.json format and CLI structure are well-designed
2. **Git-First Approach:** Using GitHub as default source is smart for early days
3. **Integration:** Package manager is integrated into main `lunar` CLI (not separate binary)

### What Needs Work
1. **No Registry:** Biggest gap - can't publish or discover packages easily
2. **Version Hell:** Without proper version resolution, dependency conflicts will hurt
3. **No Lock Files:** Can't guarantee reproducible builds
4. **Documentation:** These features aren't documented anywhere for users

### Recommendations
1. **Short term:** Focus on lock files and better version handling
2. **Medium term:** Build a minimal registry (even just a static site + git)
3. **Long term:** Full registry infrastructure as outlined in TODO

---

## 📝 Documentation Needed

Users have no way to know these features exist! Need:

1. **User Guide**
   - How to use `lunar init`
   - How to add/remove packages
   - How to run scripts
   - Package specifier formats

2. **Package Author Guide**
   - How to structure a Lunar package
   - lunar.json best practices
   - How to publish (once available)

3. **Reference**
   - lunar.json schema
   - CLI command reference
   - Module resolution algorithm

**Action:** Add package manager section to documentation site TODO

---

## 🔄 Updated Timeline

Given what's already implemented:

| Milestone | Timeline | Description |
|-----------|----------|-------------|
| **Lock Files** | 1-2 weeks | Add lunar-lock.json |
| **CLI Polish** | 2-4 weeks | Add missing commands, better UX |
| **Minimal Registry** | 2-3 months | Basic server + publishing |
| **Beta Registry** | 4-6 months | Full features, web interface |
| **Production Registry** | 6-9 months | Scaling, security, CDN |

This is **significantly faster** than the original 9-12 month estimate because the foundation is already built!

---

## 📚 Related Documents

- [PACKAGE_REGISTRY_TODO.md](./PACKAGE_REGISTRY_TODO.md) - Full registry roadmap
- [DOCUMENTATION_SITE_TODO.md](./DOCUMENTATION_SITE_TODO.md) - Doc site needs package manager section
- [README.md](./README.md) - Planning overview

---

**Status Legend:**
- ✅ Implemented and working
- ⚠️ Partially implemented
- ❌ Not implemented
- 🚧 In progress
