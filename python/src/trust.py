"""Per-agent trust scoring.

Ported from the Go ``internal/trust`` package. Trust starts at a baseline and decays on
violations and anomalies (weighted by severity), recovering slowly on clean behaviour. Low
trust is itself a governance signal. The update is deterministic.
"""

from __future__ import annotations

from datetime import datetime

import models
from models import Anomaly, TrustFactor, TrustScore, Violation

BASELINE = 0.8


def _clamp(f: float) -> float:
    if f > 1:
        return 1.0
    if f < 0:
        return 0.0
    return f


def _severity_penalty(sev: str) -> float:
    if sev == models.SEVERITY_CRITICAL:
        return 0.30
    if sev == models.SEVERITY_HIGH:
        return 0.15
    if sev == models.SEVERITY_MEDIUM:
        return 0.08
    if sev == models.SEVERITY_LOW:
        return 0.03
    return 0.0


class Scorer:
    """Maintains per-agent trust."""

    def __init__(self) -> None:
        self._scores: dict[str, TrustScore] = {}

    def get(self, agent_id: str) -> float:
        """Return an agent's current trust (``BASELINE`` if unseen)."""
        sc = self._scores.get(agent_id)
        return sc.score if sc is not None else BASELINE

    def score(self, agent_id: str) -> TrustScore:
        """Return the full trust record for an agent."""
        sc = self._scores.get(agent_id)
        if sc is not None:
            return sc
        return TrustScore(agent_id=agent_id, score=BASELINE)

    def update(
        self,
        agent_id: str,
        violations: list[Violation] | None,
        anomalies: list[Anomaly] | None,
        now: datetime,
    ) -> float:
        """Adjust trust from one event's violations/anomalies; clean events recover."""
        violations = violations or []
        anomalies = anomalies or []
        sc = self._scores.get(agent_id)
        if sc is None:
            sc = TrustScore(agent_id=agent_id, score=BASELINE)

        factors: list[TrustFactor] = []
        if not violations and not anomalies:
            delta = 0.01
            sc.score = _clamp(sc.score + delta)
            factors.append(
                TrustFactor(name="clean_behaviour", delta=delta, reason="no violation or anomaly")
            )
        else:
            for v in violations:
                p = _severity_penalty(v.severity)
                sc.score = _clamp(sc.score - p)
                factors.append(TrustFactor(name="violation", delta=-p, reason=v.reason))
            for a in anomalies:
                p = 0.05 * a.score
                sc.score = _clamp(sc.score - p)
                factors.append(
                    TrustFactor(name=f"anomaly:{a.type}", delta=-p, reason=a.detail)
                )

        sc.agent_id = agent_id
        sc.factors = factors
        sc.updated_at = now
        self._scores[agent_id] = sc
        return sc.score

    def should_isolate(self, agent_id: str) -> bool:
        """Report whether an agent's trust has decayed below the isolation floor."""
        return self.get(agent_id) < 0.3
