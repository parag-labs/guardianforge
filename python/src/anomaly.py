"""Deterministic statistical/pattern anomaly detection.

Ported from the Go ``internal/anomaly`` package. It watches each agent's recent behaviour
and flags loops (the same tool hammered over and over), rate spikes, and cyclic A-B-A-B
patterns. It is deterministic and cheap, and runs on every event before any model is
consulted.
"""

from __future__ import annotations

import models
from models import AgentEvent, Anomaly

_WINDOW = 12
_LOOP_THRESHOLD = 5
_RATE_THRESHOLD = 10


def _clamp(f: float) -> float:
    if f > 1:
        return 1.0
    if f < 0:
        return 0.0
    return f


def _round(f: float) -> float:
    """Reproduce Go's ``float64(int(f*100+0.5)) / 100`` (round half up for f >= 0)."""
    return float(int(f * 100 + 0.5)) / 100


def _fp(ev: AgentEvent) -> str:
    """Fingerprint of an event for pattern matching."""
    return f"{ev.type}:{ev.tool}:{ev.action}"


class Detector:
    """Keeps a bounded per-agent history and scores anomalies."""

    def __init__(self) -> None:
        self._history: dict[str, list[str]] = {}
        self._window = _WINDOW
        self._loop_threshold = _LOOP_THRESHOLD
        self._rate_threshold = _RATE_THRESHOLD

    def observe(self, ev: AgentEvent) -> list[Anomaly]:
        """Record an event and return any anomalies detected for the agent."""
        h = self._history.get(ev.agent_id, [])
        h = [*h, _fp(ev)]
        if len(h) > self._window:
            h = h[len(h) - self._window :]
        self._history[ev.agent_id] = h

        out: list[Anomaly] = []
        loop = self._detect_loop(ev.agent_id, h)
        if loop is not None:
            out.append(loop)
        rate = self._detect_rate(ev.agent_id, h)
        if rate is not None:
            out.append(rate)
        cyclic = self._detect_cyclic(ev.agent_id, h)
        if cyclic is not None:
            out.append(cyclic)
        return out

    def _detect_loop(self, agent: str, h: list[str]) -> Anomaly | None:
        if not h:
            return None
        last = h[-1]
        count = 0
        i = len(h) - 1
        while i >= 0 and h[i] == last:
            count += 1
            i -= 1
        if count >= self._loop_threshold:
            score = _clamp(count / self._window)
            return Anomaly(
                agent_id=agent,
                type=models.ANOMALY_LOOP,
                score=_round(score),
                detail=f"{count} consecutive identical actions ({last})",
            )
        return None

    def _detect_rate(self, agent: str, h: list[str]) -> Anomaly | None:
        if len(h) >= self._rate_threshold:
            score = _clamp(len(h) / self._window)
            return Anomaly(
                agent_id=agent,
                type=models.ANOMALY_RATE_SPIKE,
                score=_round(score),
                detail=f"{len(h)} events within the observation window",
            )
        return None

    def _detect_cyclic(self, agent: str, h: list[str]) -> Anomaly | None:
        if len(h) < 6:
            return None
        tail = h[len(h) - 6 :]
        a, b = tail[0], tail[1]
        if a == b:
            return None
        for i, x in enumerate(tail):
            want = a if i % 2 == 0 else b
            if x != want:
                return None
        return Anomaly(
            agent_id=agent,
            type=models.ANOMALY_CYCLIC,
            score=0.7,
            detail=f'cyclic pattern between "{a}" and "{b}"',
        )
