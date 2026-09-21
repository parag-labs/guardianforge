from __future__ import annotations

from datetime import datetime, timezone

import models as m
import trust


def fixed() -> datetime:
    return datetime.fromtimestamp(0, tz=timezone.utc)


def test_unseen_agent_starts_at_baseline() -> None:
    s = trust.Scorer()
    assert s.get("a1") == trust.BASELINE
    assert s.get("a1") == 0.8


def test_score_record_for_unseen_agent() -> None:
    s = trust.Scorer()
    sc = s.score("a1")
    assert sc.agent_id == "a1"
    assert sc.score == trust.BASELINE
    assert sc.factors == []


def test_violations_decay_trust() -> None:
    s = trust.Scorer()
    before = s.get("a1")
    after = s.update("a1", [m.Violation(severity=m.SEVERITY_CRITICAL)], None, fixed())
    assert after < before
    # 0.8 - 0.30 = 0.5 exactly.
    assert after == 0.5


def test_severity_penalties_exact() -> None:
    for sev, expected in [
        (m.SEVERITY_CRITICAL, 0.8 - 0.30),
        (m.SEVERITY_HIGH, 0.8 - 0.15),
        (m.SEVERITY_MEDIUM, 0.8 - 0.08),
        (m.SEVERITY_LOW, 0.8 - 0.03),
    ]:
        s = trust.Scorer()
        after = s.update("a1", [m.Violation(severity=sev)], None, fixed())
        assert abs(after - expected) < 1e-9


def test_clean_behaviour_recovers_trust() -> None:
    s = trust.Scorer()
    s.update("a1", [m.Violation(severity=m.SEVERITY_HIGH)], None, fixed())
    low = s.get("a1")
    after = s.update("a1", None, None, fixed())
    assert after > low
    # 0.65 + 0.01 = 0.66.
    assert abs(after - 0.66) < 1e-9


def test_clean_behaviour_records_factor() -> None:
    s = trust.Scorer()
    s.update("a1", None, None, fixed())
    sc = s.score("a1")
    assert len(sc.factors) == 1
    assert sc.factors[0].name == "clean_behaviour"
    assert sc.factors[0].delta == 0.01
    assert sc.updated_at == fixed()


def test_anomaly_penalty_scaled_by_score() -> None:
    s = trust.Scorer()
    after = s.update("a1", None, [m.Anomaly(type=m.ANOMALY_LOOP, score=0.5)], fixed())
    # 0.8 - 0.05*0.5 = 0.775.
    assert abs(after - 0.775) < 1e-9
    sc = s.score("a1")
    assert sc.factors[0].name == "anomaly:loop"


def test_repeated_critical_triggers_isolation() -> None:
    s = trust.Scorer()
    for _ in range(3):
        s.update("bad", [m.Violation(severity=m.SEVERITY_CRITICAL)], None, fixed())
    assert s.should_isolate("bad")
    # 0.8 -> 0.5 -> 0.2 -> clamp 0.0.
    assert s.get("bad") == 0.0
    assert not s.should_isolate("good")


def test_isolation_threshold_boundary() -> None:
    s = trust.Scorer()
    # Two highs: 0.8 -> 0.65 -> 0.5 (>=0.3, not isolated).
    s.update("a1", [m.Violation(severity=m.SEVERITY_HIGH)], None, fixed())
    s.update("a1", [m.Violation(severity=m.SEVERITY_HIGH)], None, fixed())
    assert not s.should_isolate("a1")
    # One more high: 0.5 -> 0.35, still not isolated (0.35 >= 0.3).
    s.update("a1", [m.Violation(severity=m.SEVERITY_HIGH)], None, fixed())
    assert not s.should_isolate("a1")
    # One more: 0.35 -> 0.20 < 0.3.
    s.update("a1", [m.Violation(severity=m.SEVERITY_HIGH)], None, fixed())
    assert s.should_isolate("a1")


def test_trust_stays_in_range() -> None:
    s = trust.Scorer()
    for _ in range(20):
        s.update("a1", [m.Violation(severity=m.SEVERITY_CRITICAL)], None, fixed())
    assert s.get("a1") >= 0.0
    for _ in range(200):
        s.update("a1", None, None, fixed())
    assert s.get("a1") <= 1.0


def test_multiple_violations_clamp_after_each() -> None:
    s = trust.Scorer()
    after = s.update(
        "a1",
        [m.Violation(severity=m.SEVERITY_CRITICAL), m.Violation(severity=m.SEVERITY_CRITICAL)],
        None,
        fixed(),
    )
    # 0.8 -> 0.5 -> 0.2.
    assert abs(after - 0.2) < 1e-9
    assert len(s.score("a1").factors) == 2


def test_factors_replaced_each_update() -> None:
    s = trust.Scorer()
    s.update("a1", [m.Violation(severity=m.SEVERITY_LOW)], None, fixed())
    s.update("a1", None, None, fixed())
    sc = s.score("a1")
    # Only the latest update's single clean factor remains.
    assert len(sc.factors) == 1
    assert sc.factors[0].name == "clean_behaviour"
