from __future__ import annotations

import models as m
import policy


def ev(agent: str, typ: str, tool: str, action: str) -> m.AgentEvent:
    return m.AgentEvent(agent_id=agent, type=typ, tool=tool, action=action)


def one(pid: str, mode: str, **rule_kw: object) -> m.Policy:
    """Build a single-rule policy to keep the tests compact."""
    return m.Policy(policy_id=pid, mode=mode, rules=[m.Rule(**rule_kw)])  # type: ignore[arg-type]


def test_equality_rule_fires() -> None:
    p = one(
        "p1",
        m.MODE_HARD,
        rule_id="r1",
        field="tool",
        op=m.OP_EQUALS,
        value="delete_database",
        severity=m.SEVERITY_CRITICAL,
    )
    e = policy.Evaluator([p])
    v, mode, sev = e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "delete_database", ""))
    assert len(v) == 1
    assert mode == m.MODE_HARD
    assert sev == m.SEVERITY_CRITICAL
    assert v[0].reason == "rule r1 matched on tool"
    assert v[0].agent_id == "a1"


def test_no_violation_for_clean_event() -> None:
    p = one(
        "p1",
        m.MODE_HARD,
        rule_id="r1",
        field="tool",
        op=m.OP_EQUALS,
        value="delete_database",
        severity=m.SEVERITY_CRITICAL,
    )
    e = policy.Evaluator([p])
    v, mode, sev = e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "read_file", ""))
    assert v == []
    assert mode == m.MODE_OBSERVE
    assert sev == m.SEVERITY_INFO


def test_not_equals_rule() -> None:
    p = one(
        "p1", m.MODE_SOFT, rule_id="r1", field="tool", op=m.OP_NOT_EQUALS,
        value="safe", severity=m.SEVERITY_LOW,
    )
    e = policy.Evaluator([p])
    assert len(e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "danger", ""))[0]) == 1
    assert e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "safe", ""))[0] == []


def test_contains_rule() -> None:
    p = one(
        "p1", m.MODE_SOFT, rule_id="r1", field="action", op=m.OP_CONTAINS,
        value="drop", severity=m.SEVERITY_MEDIUM,
    )
    e = policy.Evaluator([p])
    assert len(e.evaluate(ev("a1", m.EVENT_DECISION, "", "drop table"))[0]) == 1
    assert e.evaluate(ev("a1", m.EVENT_DECISION, "", "select rows"))[0] == []


def test_regexp_rule() -> None:
    p = one(
        "p1", m.MODE_SOFT, rule_id="r1", field="action", op=m.OP_MATCHES,
        value=r"(?i)privilege|escalat", severity=m.SEVERITY_HIGH,
    )
    e = policy.Evaluator([p])
    v, _, _ = e.evaluate(ev("a1", m.EVENT_DECISION, "", "requesting privilege escalation"))
    assert len(v) == 1


def test_invalid_regexp_never_fires() -> None:
    p = one(
        "p1", m.MODE_SOFT, rule_id="r1", field="action", op=m.OP_MATCHES,
        value="(unclosed", severity=m.SEVERITY_HIGH,
    )
    e = policy.Evaluator([p])
    # Bad pattern is skipped at compile time; the rule simply never matches.
    assert e.evaluate(ev("a1", m.EVENT_DECISION, "", "(unclosed"))[0] == []


def test_count_over_detects_repetition() -> None:
    p = one(
        "p1", m.MODE_HARD, rule_id="loop", field="tool", op=m.OP_COUNT_OVER,
        value="search", threshold=4, window_size=10, severity=m.SEVERITY_MEDIUM,
    )
    e = policy.Evaluator([p])
    last: list[m.Violation] = []
    for _ in range(6):
        last, _, _ = e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "search", ""))
    assert len(last) > 0
    # A different agent with few calls should not fire (count is per-agent).
    v, _, _ = e.evaluate(ev("a2", m.EVENT_TOOL_CALL, "search", ""))
    assert v == []


def test_count_over_threshold_is_strict() -> None:
    p = one(
        "p1", m.MODE_HARD, rule_id="loop", field="tool", op=m.OP_COUNT_OVER,
        value="search", threshold=4, window_size=10, severity=m.SEVERITY_MEDIUM,
    )
    e = policy.Evaluator([p])
    fires_at = []
    for i in range(6):
        v, _, _ = e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "search", ""))
        if v:
            fires_at.append(i + 1)
    # count must be strictly greater than 4 -> first fire on the 5th event.
    assert fires_at[0] == 5


def test_strictest_mode_and_highest_severity_win() -> None:
    observe = one(
        "obs", m.MODE_OBSERVE, rule_id="r1", field="type", op=m.OP_EQUALS,
        value="TOOL_CALL", severity=m.SEVERITY_LOW,
    )
    hard = one(
        "hard", m.MODE_HARD, rule_id="r2", field="tool", op=m.OP_EQUALS,
        value="rm", severity=m.SEVERITY_HIGH,
    )
    e = policy.Evaluator([observe, hard])
    v, mode, sev = e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "rm", ""))
    assert len(v) == 2
    assert mode == m.MODE_HARD
    assert sev == m.SEVERITY_HIGH


def test_metadata_field() -> None:
    p = one(
        "p1", m.MODE_SOFT, rule_id="r1", field="metadata.pii", op=m.OP_EQUALS,
        value="true", severity=m.SEVERITY_HIGH,
    )
    e = policy.Evaluator([p])
    event = ev("a1", m.EVENT_TOOL_RESULT, "read", "")
    event.metadata = {"pii": "true"}
    v, _, _ = e.evaluate(event)
    assert len(v) == 1


def test_missing_metadata_field_is_empty() -> None:
    p = one(
        "p1", m.MODE_SOFT, rule_id="r1", field="metadata.pii", op=m.OP_EQUALS,
        value="", severity=m.SEVERITY_HIGH,
    )
    e = policy.Evaluator([p])
    # Absent metadata reads as "" so an eq "" rule fires.
    assert len(e.evaluate(ev("a1", m.EVENT_TOOL_RESULT, "read", ""))[0]) == 1


def test_unknown_field_is_empty() -> None:
    p = one(
        "p1", m.MODE_SOFT, rule_id="r1", field="nope", op=m.OP_EQUALS,
        value="x", severity=m.SEVERITY_HIGH,
    )
    e = policy.Evaluator([p])
    assert e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "t", "a"))[0] == []


def test_set_policies_replaces_and_recompiles() -> None:
    e = policy.Evaluator([])
    assert e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "rm", ""))[0] == []
    e.set_policies(
        [
            one(
                "p1", m.MODE_HARD, rule_id="r1", field="tool", op=m.OP_EQUALS,
                value="rm", severity=m.SEVERITY_HIGH,
            )
        ]
    )
    assert len(e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "rm", ""))[0]) == 1
    assert len(e.policies()) == 1


def test_count_over_default_window_uses_full_history() -> None:
    p = one(
        "p1", m.MODE_HARD, rule_id="loop", field="tool", op=m.OP_COUNT_OVER,
        value="search", threshold=2, window_size=0, severity=m.SEVERITY_MEDIUM,
    )
    e = policy.Evaluator([p])
    last: list[m.Violation] = []
    for _ in range(3):
        last, _, _ = e.evaluate(ev("a1", m.EVENT_TOOL_CALL, "search", ""))
    assert len(last) > 0
