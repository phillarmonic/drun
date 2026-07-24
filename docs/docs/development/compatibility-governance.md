# Compatibility and language governance

drun develops features quickly while treating existing automation conservatively.
Within a major version, compatibility is a release requirement rather than a
best-effort goal.

## The v2 compatibility commitment

A script accepted by an earlier stable v2 release must remain accepted by later
v2.x.x releases with the same documented language behavior. This cumulative
contract protects:

- parsing and validation;
- task discovery and task selection;
- parameter constraints and defaults;
- interpolation and control flow;
- dry-run semantics; and
- documented runtime behavior.

The contract does not freeze diagnostic wording, log formatting, timing, or
other undocumented output. It also cannot guarantee the behavior or
availability of operating systems, external tools, network services, or other
dependencies outside drun.

## Classifying changes

Every pull request that changes user-visible language behavior declares one of
these classifications:

| Classification | Meaning | Governance |
| --- | --- | --- |
| **Additive** | Introduces syntax or behavior without changing existing scripts. | Normal review, focused tests, specification and documentation updates, and a new compatibility fixture. |
| **Compatible correction** | Corrects behavior while preserving the protected contract. | Requires evidence that the compatibility corpus still passes and a fixture for the corrected behavior. |
| **Deprecation** | Retains behavior but warns that it will not continue into the next major version. | Requires migration guidance and compatibility coverage. Removal waits for v3. |
| **Incompatible** | Changes or removes protected v2 behavior. | Deferred to v3 unless approved through the exception process below. |

This keeps the common path fast: additive work receives ordinary review.
Elevated review is reserved for changes that alter an established contract.

## Compatibility corpus and release gate

The versioned fixtures in
`internal/compatibility/testdata/` are the executable compatibility contract.
The initial `v2.0` corpus covers representative parsing, validation, task
selection, defaults, interpolation, control flow, and dry-run behavior.

- Add a permanent fixture and semantic assertion for every new or deliberately
  corrected language behavior.
- Do not rewrite or delete an existing fixture to make a change pass.
- Assert semantic outcomes, not incidental colors, spacing, or diagnostic text.
- Use `scripts/test-compatibility.sh` locally. Pull requests and releases run
  the same suite as a named, required gate.

The examples regression suite remains valuable broad smoke coverage. Examples
may evolve to teach current features; compatibility fixtures are immutable
records of previously supported behavior.

## Deprecations

Deprecations in v2 are warning-first and non-breaking. The existing behavior
and its compatibility fixture remain for the rest of v2, while documentation
provides a supported migration path. Removal may occur in v3.

## Exceptional incompatible corrections

A v2 incompatibility may be considered only for a severe correctness,
security, or data-loss concern. It requires:

1. Explicit approval from two maintainers;
2. A compatibility note in the exception log below;
3. Migration guidance and, where feasible, a deprecation period;
4. Updates to the language specification, compatibility fixtures, and release
   notes; and
5. Focused tests demonstrating both the problem and the intended correction.

Maintainer approval is judgment, not an expansion of the exception categories.
Convenience, cleanup, and implementation simplicity are not sufficient reasons.

## Compatibility exception log

There are currently no approved v2 compatibility exceptions.

When an exception is approved, append an entry containing the release, affected
behavior, rationale, approving maintainers, and migration guidance. Do not
rewrite earlier entries.
