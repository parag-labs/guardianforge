from __future__ import annotations

import evaluation as ev


def test_detection_accuracy_zero_when_no_scenarios() -> None:
    assert ev.Report(scenarios=0).detection_accuracy() == 0.0


def test_detection_accuracy_fraction() -> None:
    assert ev.Report(scenarios=4, detect_correct=3).detection_accuracy() == 0.75
    assert ev.Report(scenarios=5, detect_correct=5).detection_accuracy() == 1.0


def test_report_defaults() -> None:
    r = ev.Report()
    assert r.audit_intact is True
    assert r.cases == []
    assert r.scenarios == 0


def test_case_defaults() -> None:
    c = ev.Case()
    assert c.intervened is False
    assert c.correct is False
    assert c.type == ""


def test_format_perfect_scorecard() -> None:
    r = ev.Report(
        scenarios=5, detect_correct=5, type_correct=5, false_positives=0, audit_intact=True
    )
    expected = (
        "GuardianForge Governance Evaluation\n\n"
        "Scenarios:                 5\n"
        "Detection accuracy:        100%\n"
        "Intervention correctness:  100%\n"
        "False positives:           0\n"
        "Audit chain intact:        yes\n"
    )
    assert r.format() == expected


def test_format_broken_audit_and_false_positives() -> None:
    r = ev.Report(
        scenarios=4, detect_correct=3, type_correct=3, false_positives=2, audit_intact=False
    )
    expected = (
        "GuardianForge Governance Evaluation\n\n"
        "Scenarios:                 4\n"
        "Detection accuracy:        75%\n"
        "Intervention correctness:  75%\n"
        "False positives:           2\n"
        "Audit chain intact:        NO\n"
    )
    assert r.format() == expected


def test_format_rounds_half_to_even() -> None:
    # 1/8 = 12.5% -> round half to even -> 12 (NOT 13). This is the Go %.0f contract.
    r = ev.Report(scenarios=8, detect_correct=1, type_correct=1)
    out = r.format()
    assert "Detection accuracy:        12%\n" in out
    assert "Intervention correctness:  12%\n" in out


def test_format_rounds_half_to_even_up_case() -> None:
    # 3/8 = 37.5% -> nearest even -> 38.
    r = ev.Report(scenarios=8, detect_correct=3, type_correct=3)
    out = r.format()
    assert "Detection accuracy:        38%\n" in out


def test_format_zero_scenarios() -> None:
    r = ev.Report(scenarios=0)
    out = r.format()
    assert "Scenarios:                 0\n" in out
    assert "Detection accuracy:        0%\n" in out
    assert "Intervention correctness:  0%\n" in out
