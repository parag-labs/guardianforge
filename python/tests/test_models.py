from __future__ import annotations

import models as m


def test_severity_rank_ordering() -> None:
    assert m.severity_rank(m.SEVERITY_INFO) == 0
    assert m.severity_rank(m.SEVERITY_LOW) == 1
    assert m.severity_rank(m.SEVERITY_MEDIUM) == 2
    assert m.severity_rank(m.SEVERITY_HIGH) == 3
    assert m.severity_rank(m.SEVERITY_CRITICAL) == 4


def test_severity_rank_unknown_is_zero() -> None:
    assert m.severity_rank("bogus") == 0
    assert m.severity_rank("") == 0


def test_higher_severity_picks_stronger() -> None:
    assert m.higher_severity(m.SEVERITY_INFO, m.SEVERITY_CRITICAL) == m.SEVERITY_CRITICAL
    assert m.higher_severity(m.SEVERITY_HIGH, m.SEVERITY_LOW) == m.SEVERITY_HIGH


def test_higher_severity_ties_keep_first() -> None:
    # b wins only if strictly higher; equal rank keeps a.
    assert m.higher_severity(m.SEVERITY_MEDIUM, m.SEVERITY_MEDIUM) == m.SEVERITY_MEDIUM


def test_mode_rank_ordering() -> None:
    assert m.mode_rank(m.MODE_OBSERVE) == 0
    assert m.mode_rank(m.MODE_SOFT) == 1
    assert m.mode_rank(m.MODE_HARD) == 2
    assert m.mode_rank(m.MODE_ESCALATE) == 3
    assert m.mode_rank("weird") == 0


def test_stricter_mode_picks_stricter() -> None:
    assert m.stricter_mode(m.MODE_OBSERVE, m.MODE_HARD) == m.MODE_HARD
    assert m.stricter_mode(m.MODE_ESCALATE, m.MODE_SOFT) == m.MODE_ESCALATE
    assert m.stricter_mode(m.MODE_SOFT, m.MODE_SOFT) == m.MODE_SOFT


def test_dataclass_defaults() -> None:
    ev = m.AgentEvent()
    assert ev.metadata == {}
    assert ev.type == ""
    rule = m.Rule()
    assert rule.severity == m.SEVERITY_INFO
    assert rule.threshold == 0
    policy = m.Policy()
    assert policy.mode == m.MODE_OBSERVE
    assert policy.scope == m.SCOPE_GLOBAL
    assert policy.rules == []
    score = m.TrustScore()
    assert score.factors == []
    report_defaults = m.Signal()
    assert report_defaults.max_mode == m.MODE_OBSERVE
    assert report_defaults.max_severity == m.SEVERITY_INFO


def test_decision_defaults() -> None:
    d = m.Decision()
    assert d.intervene is False
    assert d.escalate is False
    assert d.confidence == 0.0
