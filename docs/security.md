# Security & guardrails

GuardianForge governs other agents, so its own trust model is strict: **the model
synthesizes, deterministic code decides the envelope and enforces, and everything is
recorded in a tamper-evident log.**

## Guardrails

- **The model cannot invent an action.** A supervisor decision naming an unknown
  intervention type is rejected; the closed set is `PAUSE`, `REVOKE_TOOL`,
  `INJECT_CONSTRAINT`, `FORCE_REPLAN`, `NOTIFY`, `ESCALATE`.
- **Observe means observe.** Under an `OBSERVE` policy the model can never produce an
  intervention, regardless of what it returns. This is enforced in the validator, not the
  prompt.
- **Fail safe.** Any validation failure — malformed output, unknown type, an intervene
  attempt under observe — falls back deterministically: to "no action" under observe, or to
  human escalation under a stricter mode. It never acts on output it couldn't trust.
- **Human-in-the-loop.** Escalations are queued for a human; the queue and its resolutions
  are part of the audit trail.
- **Tamper-evident audit.** Every decision and intervention is SHA-256 hash-chained. Editing
  a past entry or reordering the log makes a recomputed hash disagree, and `Verify` returns
  false — the evaluation asserts the chain stays intact throughout.
- **Least privilege by policy mode.** The strictest applicable policy mode caps the
  response; a soft policy can notify or constrain but not pause.

## What the evaluation asserts

The scorecard (`-eval`, and the CI gate) fails the build on:

- a **false positive** — any intervention against a healthy agent (must be 0);
- a **missed detection** — failing to act on an injected failure;
- a **wrong response type** — e.g. notifying when the policy demands escalation;
- a **broken audit chain**.

## Threat model note

Because the deterministic detectors produce the signal and the model's only output is a
decision that is independently validated, adversarial content in an event (a crafted tool
name or message) cannot directly cause an unsafe action: it can at most influence a
*proposal*, which still has to name a real intervention type allowed by the policy mode —
and a stricter-mode failure still escalates to a human.
