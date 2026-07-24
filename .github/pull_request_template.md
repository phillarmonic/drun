## Summary

<!-- What changes, and why? -->

## Compatibility classification

<!-- Required for user-visible language or runtime behavior. Check exactly one. -->

- [ ] Not user-visible
- [ ] Additive
- [ ] Compatible correction
- [ ] Deprecation
- [ ] Incompatible

Compatibility impact and evidence:

<!-- Explain why existing v2 scripts retain their documented behavior. Incompatible
changes require the approved exception described in the governance policy. -->

## Contract checklist

- [ ] Focused parser, engine, or domain tests are added or updated
- [ ] A permanent compatibility fixture is added for new or corrected language behavior
- [ ] `DRUN_V2_SPECIFICATION.md` or the canonical language reference is updated when behavior changes
- [ ] User documentation and examples are updated where applicable
- [ ] `./scripts/test-compatibility.sh` passes
- [ ] Any deprecation includes migration guidance and remains functional in v2
- [ ] Any incompatible v2 correction has two maintainer approvals and a public exception-log entry
