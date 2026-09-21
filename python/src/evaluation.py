"""The governance scorecard (pure scoring + formatting).

Ported from the pure parts of the Go ``internal/eval`` package: the ``Report`` and ``Case``
aggregates, the detection-accuracy ratio, and the stable text ``format``. The scenario
driver (``Run``) is intentionally excluded here - it wires the fleet, agents, LLM, and
runtime, which remain in Go.
"""

from __future__ import annotations

import math
from dataclasses import dataclass, field


def _round_half_even(x: float) -> int:
    """Round to the nearest integer, ties to even - matching Go's ``%.0f`` (x >= 0)."""
    floor = math.floor(x)
    diff = x - floor
    if diff < 0.5:
        return floor
    if diff > 0.5:
        return floor + 1
    return floor if floor % 2 == 0 else floor + 1


def _percent(value: float) -> str:
    return f"{_round_half_even(value)}%"


@dataclass
class Case:
    """One scenario's outcome."""

    id: str = ""
    intervened: bool = False
    type: str = ""
    correct: bool = False


@dataclass
class Report:
    """The aggregate scorecard."""

    scenarios: int = 0
    detect_correct: int = 0
    type_correct: int = 0
    false_positives: int = 0
    audit_intact: bool = True
    cases: list[Case] = field(default_factory=list)

    def detection_accuracy(self) -> float:
        """Fraction of scenarios where intervene/observe matched truth (0 if none)."""
        if self.scenarios == 0:
            return 0.0
        return self.detect_correct / self.scenarios

    def format(self) -> str:
        """Render the scorecard in a stable, readable form."""
        intact = "yes" if self.audit_intact else "NO"
        type_acc = 0.0
        if self.scenarios > 0:
            type_acc = 100.0 * self.type_correct / self.scenarios
        return (
            "GuardianForge Governance Evaluation\n\n"
            + f"{'Scenarios:':<27}{self.scenarios}\n"
            + f"{'Detection accuracy:':<27}{_percent(100.0 * self.detection_accuracy())}\n"
            + f"{'Intervention correctness:':<27}{_percent(type_acc)}\n"
            + f"{'False positives:':<27}{self.false_positives}\n"
            + f"{'Audit chain intact:':<27}{intact}\n"
        )
