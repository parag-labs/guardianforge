"""The deterministic policy evaluator.

Ported from the Go ``internal/policy`` package. It checks structured rules against events
- equality, contains, regexp, and windowed counts - and emits violations, returning the
strictest mode and severity among the policies that fired. This runs first, before any
model: the fast deterministic path decides the clear cases.
"""

from __future__ import annotations

import re

import models
from models import AgentEvent, Policy, Rule, Violation

_MAX_HISTORY = 256


class Evaluator:
    """Holds the policy set and per-agent sliding history for count-based rules."""

    def __init__(self, policies: list[Policy]) -> None:
        self._policies = list(policies)
        self._history: dict[str, list[str]] = {}
        self._regexps: dict[str, re.Pattern[str]] = {}
        self._compile()

    def _compile(self) -> None:
        self._regexps = {}
        for p in self._policies:
            for r in p.rules:
                if r.op == models.OP_MATCHES:
                    try:
                        self._regexps[r.rule_id] = re.compile(r.value)
                    except re.error:
                        # Go silently skips a rule whose pattern will not compile.
                        pass

    def policies(self) -> list[Policy]:
        """Return a copy of the current policy set."""
        return list(self._policies)

    def set_policies(self, policies: list[Policy]) -> None:
        """Replace the policy set (used by the admin API)."""
        self._policies = list(policies)
        self._compile()

    def evaluate(self, ev: AgentEvent) -> tuple[list[Violation], str, str]:
        """Check an event against every policy; return violations, strictest mode, severity."""
        fp = f"{ev.type}:{ev.tool}:{ev.action}"
        hist = self._history.get(ev.agent_id, [])
        hist = [*hist, fp]
        if len(hist) > _MAX_HISTORY:
            hist = hist[len(hist) - _MAX_HISTORY :]
        self._history[ev.agent_id] = hist

        violations: list[Violation] = []
        mode = models.MODE_OBSERVE
        sev = models.SEVERITY_INFO
        for p in self._policies:
            for r in p.rules:
                if self._matches(ev, r, hist):
                    violations.append(
                        Violation(
                            policy_id=p.policy_id,
                            rule_id=r.rule_id,
                            agent_id=ev.agent_id,
                            severity=r.severity,
                            reason=f"rule {r.rule_id} matched on {r.field}",
                        )
                    )
                    mode = models.stricter_mode(mode, p.mode)
                    sev = models.higher_severity(sev, r.severity)
        return violations, mode, sev

    def _matches(self, ev: AgentEvent, r: Rule, hist: list[str]) -> bool:
        if r.op == models.OP_COUNT_OVER:
            window = r.window_size if r.window_size > 0 else len(hist)
            start = len(hist) - window if len(hist) > window else 0
            count = 0
            for h in hist[start:]:
                if r.value in h:
                    count += 1
            return count > r.threshold

        val = _field_value(ev, r.field)
        if r.op == models.OP_EQUALS:
            return val == r.value
        if r.op == models.OP_NOT_EQUALS:
            return val != r.value
        if r.op == models.OP_CONTAINS:
            return r.value in val
        if r.op == models.OP_MATCHES:
            rx = self._regexps.get(r.rule_id)
            return rx.search(val) is not None if rx is not None else False
        return False


def _field_value(ev: AgentEvent, field: str) -> str:
    if field == "type":
        return ev.type
    if field == "tool":
        return ev.tool
    if field == "action":
        return ev.action
    if field.startswith("metadata."):
        return ev.metadata.get(field[len("metadata.") :], "")
    return ""
