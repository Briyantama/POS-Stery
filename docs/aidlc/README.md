# AI-DLC for POS-Stery

Spec-driven delivery artifacts (AI-DLC: Inception, Construction, Operation). Principle: AI proposes, humans decide; the spec is approved before construction; every bolt leaves evidence.

## Rules

1. No code for a unit before its decision in `02-decisions.md` is *Decided* and Inception has been reviewed.
2. A value the documents do not define is stored as `tbd` and recorded as a decision gate, never invented.
3. Every decision and approval is appended to `audit.md`.
4. Bolt = Prepare, Plan, Build, Verify, Review.
5. Bounded autonomy: AI may write code, tests and docs on a branch; it asks before destructive migrations, production keys, merging or pushing.

## Active intents

- `intents/261108-stack-align-sela/`, status in `aidlc-state.md`.
