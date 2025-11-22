# Lunar Package Registry - Implementation TODO

> A comprehensive roadmap for building the Lunar Package Registry and Package Manager

## Overview

The Lunar Package Registry will be similar to npm/PyPI/crates.io, allowing developers to publish, discover, and install Lunar packages. This includes both the registry server and the `lunarpkg` CLI tool.

---

## Phase 1: Core Architecture & Design

### 1.1 Package Format Definition
- [ ] Define `lunar.json` package manifest specification
  - [ ] Package metadata (name, version, description, author, license)
  - [ ] Dependencies (runtime and dev dependencies)
  - [ ] Entry points and exports
  - [ ] Lunar version compatibility
  - [ ] Scripts (build, test, install hooks)
  - [ ] Repository and homepage URLs
  - [ ] Keywords and categories
- [ ] Define package naming conventions (scoped packages: `@org/package`)
- [ ] Version scheme (semantic versioning enforcement)
- [ ] Define directory structure for packages
  - [ ] `src/` - Source Lunar files
  - [ ] `lib/` - Compiled Lua files
  - [ ] `types/` - Type declaration files (.d.lunar)
  - [ ] `test/` - Test files
  - [ ] `docs/` - Package documentation
  - [ ] `README.md` - Package readme
  - [ ] `LICENSE` - License file
  - [ ] `CHANGELOG.md` - Version history

### 1.2 Package Archive Format
- [ ] Choose archive format (tar.gz, zip, or custom)
- [ ] Define compression strategy
- [ ] Implement package integrity verification (SHA-256 checksums)
- [ ] Add support for digital signatures for package verification
- [ ] Define excluded files/patterns (`.gitignore`-like `.lunarignore`)

### 1.3 Dependency Resolution
- [ ] Research and choose dependency resolution algorithm
  - [ ] Consider: Semantic versioning resolution
  - [ ] Handle version ranges (`^1.2.3`, `~1.2.3`, `>=1.0.0`)
  - [ ] Conflict resolution strategies
- [ ] Design dependency tree structure
- [ ] Handle circular dependencies detection
- [ ] Support for peer dependencies
- [ ] Support for optional dependencies
- [ ] Lock file mechanism (`lunar-lock.json`)
  - [ ] Deterministic installs
  - [ ] Version pinning
  - [ ] Integrity hashes

---

## Phase 2: Package Manager CLI (`lunarpkg`)

### 2.1 Core Commands
- [ ] `lunarpkg init` - Initialize new package
  - [ ] Interactive wizard for lunar.json creation
  - [ ] Template selection (library, application, tool)
  - [ ] Git integration
- [ ] `lunarpkg install [package]` - Install dependencies
  - [ ] Install all dependencies from lunar.json
  - [ ] Install specific package and add to dependencies
  - [ ] Support `--save-dev` flag for dev dependencies
  - [ ] Parallel download and installation
  - [ ] Progress indicators
- [ ] `lunarpkg uninstall [package]` - Remove packages
  - [ ] Remove from lunar_modules/
  - [ ] Update lunar.json
  - [ ] Update lock file
- [ ] `lunarpkg update [package]` - Update packages
  - [ ] Update all packages
  - [ ] Update specific package
  - [ ] Respect version constraints
- [ ] `lunarpkg list` - List installed packages
  - [ ] Tree view of dependencies
  - [ ] Flat list option
  - [ ] Show outdated packages
- [ ] `lunarpkg search [query]` - Search registry
  - [ ] Keyword search
  - [ ] Filter by category/tags
  - [ ] Show package stats (downloads, stars)
- [ ] `lunarpkg info [package]` - Show package details
  - [ ] Version history
  - [ ] Dependencies
  - [ ] Maintainers
  - [ ] Repository info

### 2.2 Publishing Commands
- [ ] `lunarpkg login` - Authenticate with registry
  - [ ] Support multiple auth methods (API key, OAuth)
  - [ ] Store credentials securely
- [ ] `lunarpkg logout` - Remove credentials
- [ ] `lunarpkg publish` - Publish package to registry
  - [ ] Pre-publish validation
  - [ ] Version bump checks
  - [ ] Build step execution
  - [ ] Upload to registry
  - [ ] Tag creation in git
- [ ] `lunarpkg unpublish` - Remove package version
  - [ ] Safety checks and confirmations
  - [ ] Version-specific or entire package
- [ ] `lunarpkg deprecate [package@version]` - Mark as deprecated
  - [ ] Add deprecation message
  - [ ] Show warnings on install

### 2.3 Utility Commands
- [ ] `lunarpkg run [script]` - Run package scripts
  - [ ] Execute scripts defined in lunar.json
  - [ ] Pre/post script hooks
- [ ] `lunarpkg test` - Run tests
  - [ ] Integrate with testing framework
- [ ] `lunarpkg build` - Build package
  - [ ] Compile .lunar to .lua
  - [ ] Generate type declarations
- [ ] `lunarpkg link` - Create symlink for local development
  - [ ] Link current package globally
  - [ ] Link global package to local project
- [ ] `lunarpkg outdated` - Check for outdated packages
  - [ ] Show current vs wanted vs latest
  - [ ] Color-coded output
- [ ] `lunarpkg audit` - Security vulnerability scanning
  - [ ] Check dependencies for known vulnerabilities
  - [ ] Suggest fixes
- [ ] `lunarpkg doctor` - Diagnose installation issues
  - [ ] Verify installation integrity
  - [ ] Check permissions
  - [ ] Validate lunar.json

### 2.4 Configuration
- [ ] Support `.lunarpkgrc` config file
  - [ ] Registry URL configuration
  - [ ] Default publish settings
  - [ ] Proxy settings
  - [ ] Install paths
- [ ] Support environment variables
- [ ] Per-project vs global configuration
- [ ] Support private registries

---

## Phase 3: Registry Server Backend

### 3.1 Technology Stack Decision
- [ ] Choose backend framework
  - [ ] Options: Go (current stack), Node.js, Rust, Python
  - [ ] Consider: Performance, ecosystem, team expertise
- [ ] Choose database
  - [ ] Metadata: PostgreSQL, MongoDB, or SQLite
  - [ ] Package storage: S3, MinIO, filesystem
  - [ ] Cache layer: Redis
- [ ] Choose authentication system
  - [ ] JWT tokens
  - [ ] OAuth integration
  - [ ] API keys
- [ ] Infrastructure decisions
  - [ ] Self-hosted vs cloud
  - [ ] CDN for package distribution
  - [ ] Load balancing strategy

### 3.2 API Endpoints
- [ ] Authentication Endpoints
  - [ ] `POST /api/auth/register` - User registration
  - [ ] `POST /api/auth/login` - User login
  - [ ] `POST /api/auth/logout` - User logout
  - [ ] `POST /api/auth/token/refresh` - Refresh token
  - [ ] `POST /api/auth/reset-password` - Password reset
- [ ] Package Endpoints
  - [ ] `GET /api/packages` - List/search packages
  - [ ] `GET /api/packages/:name` - Get package metadata
  - [ ] `GET /api/packages/:name/:version` - Get specific version
  - [ ] `POST /api/packages` - Publish new package
  - [ ] `PUT /api/packages/:name` - Update package metadata
  - [ ] `DELETE /api/packages/:name/:version` - Unpublish version
  - [ ] `GET /api/packages/:name/download/:version` - Download package
  - [ ] `POST /api/packages/:name/star` - Star package
  - [ ] `GET /api/packages/:name/stats` - Package statistics
- [ ] User Endpoints
  - [ ] `GET /api/users/:username` - Get user profile
  - [ ] `PUT /api/users/:username` - Update profile
  - [ ] `GET /api/users/:username/packages` - User's packages
- [ ] Organization Endpoints (for scoped packages)
  - [ ] `POST /api/orgs` - Create organization
  - [ ] `GET /api/orgs/:orgname` - Get organization
  - [ ] `PUT /api/orgs/:orgname` - Update organization
  - [ ] `POST /api/orgs/:orgname/members` - Add member
  - [ ] `DELETE /api/orgs/:orgname/members/:username` - Remove member

### 3.3 Database Schema
- [ ] Users table
  - [ ] id, username, email, password_hash, created_at, updated_at
  - [ ] API keys
  - [ ] Email verification status
- [ ] Packages table
  - [ ] id, name, description, owner_id, created_at, updated_at
  - [ ] Repository URL, homepage, license
  - [ ] Download count, star count
- [ ] Package_Versions table
  - [ ] id, package_id, version, tarball_url, checksum
  - [ ] Dependencies (JSON)
  - [ ] Published_at, deprecated, deprecation_message
- [ ] Organizations table
  - [ ] id, name, description, created_at
- [ ] Organization_Members table
  - [ ] org_id, user_id, role (owner, admin, member)
- [ ] Package_Stars table
  - [ ] user_id, package_id, starred_at
- [ ] Download_Stats table
  - [ ] package_id, version_id, date, count

### 3.4 Storage System
- [ ] Package tarball storage
  - [ ] Storage backend (S3-compatible, local filesystem)
  - [ ] URL generation and signing
  - [ ] Backup strategy
- [ ] CDN integration
  - [ ] CloudFlare, AWS CloudFront, or similar
  - [ ] Cache invalidation strategy
- [ ] Storage quota management
  - [ ] Per-user/org limits
  - [ ] Cleanup of old/unused packages

### 3.5 Security
- [ ] Rate limiting
  - [ ] Per endpoint limits
  - [ ] DDoS protection
- [ ] Input validation and sanitization
- [ ] Package scanning
  - [ ] Malware detection
  - [ ] Vulnerability scanning
  - [ ] License compliance checking
- [ ] Authentication & Authorization
  - [ ] Two-factor authentication (2FA)
  - [ ] API key rotation
  - [ ] Scope-based permissions
- [ ] HTTPS enforcement
- [ ] CORS configuration
- [ ] Content Security Policy

---

## Phase 4: Web Interface

### 4.1 Package Discovery Pages
- [ ] Homepage
  - [ ] Featured packages
  - [ ] Recently published
  - [ ] Trending packages
  - [ ] Statistics dashboard
- [ ] Search page
  - [ ] Advanced filtering
  - [ ] Sort options (relevance, downloads, stars, recent)
  - [ ] Pagination
- [ ] Browse by category/tags
- [ ] Package detail page
  - [ ] README rendering
  - [ ] Version selector
  - [ ] Dependency tree visualization
  - [ ] Download stats graph
  - [ ] GitHub integration (issues, stars)
  - [ ] Install instructions

### 4.2 User Pages
- [ ] User dashboard
  - [ ] Published packages
  - [ ] Starred packages
  - [ ] Activity feed
- [ ] Profile page
  - [ ] Bio, avatar, social links
  - [ ] Package list
  - [ ] Contribution stats
- [ ] Organization pages
  - [ ] Organization packages
  - [ ] Members list
  - [ ] Settings (for owners)

### 4.3 Publishing Interface
- [ ] Web-based package publishing (alternative to CLI)
- [ ] Version management
- [ ] Deprecation management
- [ ] Transfer ownership
- [ ] Access token management

---

## Phase 5: Advanced Features

### 5.1 Package Quality Metrics
- [ ] Package scoring system
  - [ ] Documentation quality
  - [ ] Test coverage
  - [ ] Dependencies health
  - [ ] Update frequency
  - [ ] Community engagement
- [ ] Badges for packages
  - [ ] Build status
  - [ ] Coverage
  - [ ] Version
  - [ ] License
  - [ ] Downloads

### 5.2 Developer Tools
- [ ] Package generator/scaffolding
  - [ ] Templates for common package types
  - [ ] Best practices baked in
- [ ] Type declaration generator
  - [ ] Auto-generate .d.lunar files
- [ ] Documentation generator
  - [ ] Extract docs from code comments
  - [ ] Generate API documentation
- [ ] Testing utilities
  - [ ] Test runner integration
  - [ ] Mocking libraries

### 5.3 Community Features
- [ ] Package reviews/ratings
- [ ] Comments on packages
- [ ] Report package issues
- [ ] Package recommendations
- [ ] "Awesome Lunar" curated list

### 5.4 Analytics & Monitoring
- [ ] Download analytics
  - [ ] Geographic distribution
  - [ ] Version distribution
  - [ ] Time-series data
- [ ] Server monitoring
  - [ ] Uptime tracking
  - [ ] Performance metrics
  - [ ] Error tracking
- [ ] User analytics
  - [ ] Package adoption trends
  - [ ] Popular dependencies

### 5.5 CI/CD Integration
- [ ] GitHub Actions integration
  - [ ] Automatic publishing on tag
  - [ ] Test running
- [ ] GitLab CI integration
- [ ] Pre-built workflows
- [ ] Status badges

---

## Phase 6: Documentation & Support

### 6.1 Developer Documentation
- [ ] Package manager usage guide
- [ ] Creating your first package tutorial
- [ ] Publishing guide
- [ ] Best practices
- [ ] API reference
- [ ] lunar.json specification
- [ ] CLI command reference

### 6.2 Registry Documentation
- [ ] API documentation
  - [ ] OpenAPI/Swagger spec
  - [ ] Authentication guide
  - [ ] Rate limits
- [ ] Self-hosting guide
- [ ] Security practices
- [ ] Privacy policy
- [ ] Terms of service

### 6.3 Support Channels
- [ ] FAQ section
- [ ] Troubleshooting guide
- [ ] Discord/Slack community
- [ ] GitHub discussions
- [ ] Support email

---

## Phase 7: Testing & Quality Assurance

### 7.1 CLI Testing
- [ ] Unit tests for all commands
- [ ] Integration tests
- [ ] End-to-end tests
- [ ] Cross-platform testing (Linux, macOS, Windows)
- [ ] Performance benchmarks
- [ ] Edge case handling

### 7.2 Server Testing
- [ ] API endpoint tests
- [ ] Database migration tests
- [ ] Load testing
- [ ] Security testing
- [ ] Backup/restore testing

### 7.3 QA Process
- [ ] Code review guidelines
- [ ] Automated testing in CI
- [ ] Beta testing program
- [ ] Bug bounty program

---

## Phase 8: Deployment & Operations

### 8.1 Infrastructure Setup
- [ ] Domain registration (lunarpkg.org or similar)
- [ ] SSL certificates
- [ ] Server provisioning
- [ ] Database setup
- [ ] CDN configuration
- [ ] Backup systems
- [ ] Monitoring setup (Prometheus, Grafana)

### 8.2 Deployment Pipeline
- [ ] Continuous deployment setup
- [ ] Staging environment
- [ ] Production environment
- [ ] Rollback procedures
- [ ] Database migration strategy

### 8.3 Operational Procedures
- [ ] Incident response plan
- [ ] Backup and recovery procedures
- [ ] Scaling strategy
- [ ] Cost management
- [ ] SLA definition

---

## Phase 9: Launch Preparation

### 9.1 Pre-Launch
- [ ] Seed registry with core packages
  - [ ] Standard library packages
  - [ ] Common utilities
  - [ ] Example packages
- [ ] Beta testing with community
- [ ] Security audit
- [ ] Performance optimization
- [ ] Documentation review

### 9.2 Launch
- [ ] Announcement blog post
- [ ] Social media campaign
- [ ] Submit to Hacker News, Reddit
- [ ] Reach out to Lua/gaming communities
- [ ] Create launch video/demo

### 9.3 Post-Launch
- [ ] Monitor server health
- [ ] Collect user feedback
- [ ] Fix critical bugs
- [ ] Regular status updates
- [ ] Community engagement

---

## Phase 10: Long-term Maintenance

### 10.1 Regular Updates
- [ ] Security patches
- [ ] Feature additions based on feedback
- [ ] Performance improvements
- [ ] Dependency updates

### 10.2 Community Growth
- [ ] Package showcase
- [ ] Monthly highlights
- [ ] Developer interviews
- [ ] Contribution recognition
- [ ] Meetups/conferences

### 10.3 Sustainability
- [ ] Funding model (sponsorships, donations, grants)
- [ ] Governance model
- [ ] Contributor guidelines
- [ ] Code of conduct
- [ ] Succession planning

---

## Technical Decisions Needed

### Priority Decisions
1. **Language for CLI**: Go (consistency with compiler) vs Rust (performance) vs Node.js (ecosystem)
2. **Backend framework**: Go net/http, Gin, Echo, or different language
3. **Database choice**: PostgreSQL (robust) vs MongoDB (flexible) vs SQLite (simple)
4. **Storage**: Self-hosted vs S3 vs Cloudflare R2
5. **Frontend**: Static site vs SPA (React/Vue) vs Server-side rendering (Next.js)

### Design Decisions
1. **Scoped packages**: Support from day 1 or add later?
2. **Monorepo support**: How to handle multi-package repositories?
3. **Private packages**: Support private registries/packages?
4. **Package size limits**: What's the max package size?
5. **Versioning**: Strict semver enforcement or flexible?

---

## Milestones

### Milestone 1: MVP (3-6 months)
- Basic CLI (init, install, publish)
- Simple registry API
- Basic web interface
- PostgreSQL + S3 storage

### Milestone 2: Beta (6-9 months)
- Full CLI feature set
- Complete API
- User accounts and auth
- Package search and discovery
- Documentation

### Milestone 3: Production (9-12 months)
- Security hardening
- Performance optimization
- CDN integration
- Full web interface
- Community features
- Official launch

### Milestone 4: Growth (12+ months)
- Advanced features
- Analytics
- CI/CD integrations
- Ecosystem growth
- Sustainability model

---

## Resources Needed

### Development Team
- [ ] 1-2 Backend developers
- [ ] 1 Frontend developer
- [ ] 1 DevOps engineer (part-time)
- [ ] UX/UI designer (contract)

### Infrastructure
- [ ] Web server (2-4 cores, 4-8GB RAM initially)
- [ ] Database server
- [ ] Storage (100GB-1TB initially)
- [ ] CDN service
- [ ] Monitoring tools

### Budget Estimate (Annual)
- [ ] Infrastructure: $500-2000/month
- [ ] Domain/SSL: $100/year
- [ ] Development time: Varies (open source vs funded)

---

## Success Metrics

- [ ] Number of published packages: 100+ in first 6 months
- [ ] Active users: 500+ in first year
- [ ] Daily downloads: 1000+ in first year
- [ ] Community size: 1000+ Discord/forum members
- [ ] Package quality: 80%+ with documentation and tests
- [ ] Uptime: 99.9%+

---

## Notes

- Start simple, iterate based on user feedback
- Study npm, crates.io, PyPI for inspiration
- Security and reliability are paramount
- Foster a welcoming community from day 1
- Consider sustainability from the beginning
