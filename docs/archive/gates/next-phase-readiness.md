# Next Phase Readiness

**Date**: 2026-03-16
**Next Phase**: Marketmonkey absorption

## Preconditions Met

- [x] Quality-service identity fully removed from code, configs, compose, and docs
- [x] Validator, consumer, emulator services deleted
- [x] Kafka adapter and all Kafka infrastructure removed
- [x] Docker Compose simplified to foundation stack (nats + configctl + server)
- [x] HTTP API cleaned of validator/results endpoints
- [x] Server readiness checker simplified to configctl-only dependency
- [x] Go workspace pruned to only active modules
- [x] Makefile cleaned of removed service targets
- [x] Documentation rewritten for market-foundry identity
- [x] Architecture audit and decision records created
- [x] Prohibited carryovers documented

## What the Next Phase Should Do

### 1. Absorb Marketmonkey Patterns
- Study marketmonkey's domain model, architecture patterns, and conventions
- Identify which patterns align with the preserved foundation
- Plan domain-by-domain integration

### 2. Implement New Domains
- Start with observation and evidence as foundational domains
- Follow the domain readiness guidelines in `domain-readiness.md`
- Each domain: define domain model → application use cases → adapters → actors → interfaces

### 3. Evolve the CLI
- Adapt raccoon-cli commands to market-foundry's domain
- Remove or redesign quality-specific commands (runtime-smoke, scenario-smoke, results-inspect, trace-pack)
- Add new commands for market-foundry-specific validation

### 4. Evolve Infrastructure
- Evaluate whether NATS remains sufficient or additional infrastructure is needed
- Extend Docker Compose as new services are added
- Add new configuration files for new services

## Boundaries to Respect

1. **Layer discipline**: domain → application → adapters → actors → interfaces → cmd
2. **Module isolation**: Each Go module has its own go.mod; no circular dependencies
3. **Actor model**: Services are orchestrated through Hollywood actors, not raw goroutines
4. **Messaging contracts**: All inter-service communication goes through defined contracts
5. **Config lifecycle**: Reuse the configctl foundation for configuration management

## Risks to Monitor

| Risk | Mitigation |
|------|------------|
| Scope creep during absorption | Absorb incrementally, domain by domain |
| Quality-service patterns returning | Enforce prohibited carryovers list |
| CLI falling out of sync with new structure | Update CLI checks as structure evolves |
| Over-engineering foundation before domains are clear | Start minimal, evolve based on real needs |

## Definition of Done for Next Phase

- [ ] At least one new domain (observation or evidence) is implemented end-to-end
- [ ] CLI commands reflect the new domain structure
- [ ] Docker Compose supports the new service topology
- [ ] All tests pass across the workspace
- [ ] Quality-gate passes at all profiles (fast, ci, deep)
- [ ] Documentation reflects the new system state
