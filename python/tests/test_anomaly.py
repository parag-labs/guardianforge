from __future__ import annotations

import anomaly
import models as m


def call(agent: str, tool: str) -> m.AgentEvent:
    return m.AgentEvent(agent_id=agent, type=m.EVENT_TOOL_CALL, tool=tool)


def _types(anomalies: list[m.Anomaly]) -> set[str]:
    return {a.type for a in anomalies}


def test_detects_loop() -> None:
    d = anomaly.Detector()
    last: list[m.Anomaly] = []
    for _ in range(6):
        last = d.observe(call("a1", "search"))
    loops = [a for a in last if a.type == m.ANOMALY_LOOP]
    assert len(loops) == 1
    assert loops[0].score == 0.5
    assert loops[0].detail == "6 consecutive identical actions (TOOL_CALL:search:)"
    assert loops[0].agent_id == "a1"


def test_loop_threshold_exactly_five() -> None:
    d = anomaly.Detector()
    last: list[m.Anomaly] = []
    for _ in range(4):
        last = d.observe(call("a1", "search"))
    assert m.ANOMALY_LOOP not in _types(last)
    last = d.observe(call("a1", "search"))  # 5th identical
    assert m.ANOMALY_LOOP in _types(last)


def test_no_loop_for_varied_behaviour() -> None:
    d = anomaly.Detector()
    last: list[m.Anomaly] = []
    for tl in ["a", "b", "c", "d", "e"]:
        last = d.observe(call("a1", tl))
    assert m.ANOMALY_LOOP not in _types(last)


def test_detects_cyclic_pattern() -> None:
    d = anomaly.Detector()
    last: list[m.Anomaly] = []
    for i in range(6):
        last = d.observe(call("a1", "x" if i % 2 == 0 else "y"))
    cyclic = [a for a in last if a.type == m.ANOMALY_CYCLIC]
    assert len(cyclic) == 1
    assert cyclic[0].score == 0.7
    assert cyclic[0].detail == 'cyclic pattern between "TOOL_CALL:x:" and "TOOL_CALL:y:"'


def test_detects_rate_spike() -> None:
    d = anomaly.Detector()
    last: list[m.Anomaly] = []
    for i in range(11):
        tool = "t" + chr(ord("a") + i % 4)
        last = d.observe(call("a1", tool))
    spikes = [a for a in last if a.type == m.ANOMALY_RATE_SPIKE]
    assert len(spikes) == 1
    # round(11/12) = round(0.9166..) = 0.92
    assert spikes[0].score == 0.92
    assert spikes[0].detail == "11 events within the observation window"


def test_rate_spike_threshold_exactly_ten() -> None:
    d = anomaly.Detector()
    last: list[m.Anomaly] = []
    for i in range(9):
        last = d.observe(call("a1", "t" + chr(ord("a") + i % 4)))
    assert m.ANOMALY_RATE_SPIKE not in _types(last)
    last = d.observe(call("a1", "t" + chr(ord("a") + 9 % 4)))  # 10th
    assert m.ANOMALY_RATE_SPIKE in _types(last)


def test_per_agent_isolation() -> None:
    d = anomaly.Detector()
    for _ in range(6):
        d.observe(call("noisy", "search"))
    got = d.observe(call("quiet", "read"))
    assert got == []


def test_window_trims_history() -> None:
    d = anomaly.Detector()
    # 12 distinct then keep going; loop detector counts only trailing identical.
    for i in range(20):
        d.observe(call("a1", "tool" + str(i)))
    last = d.observe(call("a1", "steady"))
    assert m.ANOMALY_LOOP not in _types(last)


def test_empty_history_no_crash() -> None:
    d = anomaly.Detector()
    out = d.observe(call("a1", "only"))
    assert out == []
