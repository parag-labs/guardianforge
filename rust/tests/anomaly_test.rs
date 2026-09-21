use guardianforge::anomaly::Detector;
use guardianforge::models as m;

fn call(agent: &str, tool: &str) -> m::AgentEvent {
    m::AgentEvent {
        agent_id: agent.to_string(),
        r#type: m::EVENT_TOOL_CALL.to_string(),
        tool: tool.to_string(),
        ..Default::default()
    }
}

fn has_type(anomalies: &[m::Anomaly], t: &str) -> bool {
    anomalies.iter().any(|a| a.r#type == t)
}

#[test]
fn detects_loop() {
    let mut d = Detector::new();
    let mut last = Vec::new();
    for _ in 0..6 {
        last = d.observe(&call("a1", "search"));
    }
    let loops: Vec<&m::Anomaly> = last
        .iter()
        .filter(|a| a.r#type == m::ANOMALY_LOOP)
        .collect();
    assert_eq!(loops.len(), 1);
    assert_eq!(loops[0].score, 0.5);
    assert_eq!(
        loops[0].detail,
        "6 consecutive identical actions (TOOL_CALL:search:)"
    );
    assert_eq!(loops[0].agent_id, "a1");
}

#[test]
fn loop_threshold_exactly_five() {
    let mut d = Detector::new();
    let mut last = Vec::new();
    for _ in 0..4 {
        last = d.observe(&call("a1", "search"));
    }
    assert!(!has_type(&last, m::ANOMALY_LOOP));
    last = d.observe(&call("a1", "search"));
    assert!(has_type(&last, m::ANOMALY_LOOP));
}

#[test]
fn no_loop_for_varied_behaviour() {
    let mut d = Detector::new();
    let mut last = Vec::new();
    for tl in ["a", "b", "c", "d", "e"] {
        last = d.observe(&call("a1", tl));
    }
    assert!(!has_type(&last, m::ANOMALY_LOOP));
}

#[test]
fn detects_cyclic_pattern() {
    let mut d = Detector::new();
    let mut last = Vec::new();
    for i in 0..6 {
        last = d.observe(&call("a1", if i % 2 == 0 { "x" } else { "y" }));
    }
    let cyclic: Vec<&m::Anomaly> = last
        .iter()
        .filter(|a| a.r#type == m::ANOMALY_CYCLIC)
        .collect();
    assert_eq!(cyclic.len(), 1);
    assert_eq!(cyclic[0].score, 0.7);
    assert_eq!(
        cyclic[0].detail,
        "cyclic pattern between \"TOOL_CALL:x:\" and \"TOOL_CALL:y:\""
    );
}

#[test]
fn detects_rate_spike() {
    let mut d = Detector::new();
    let mut last = Vec::new();
    for i in 0..11 {
        let tool = format!("t{}", (b'a' + (i % 4) as u8) as char);
        last = d.observe(&call("a1", &tool));
    }
    let spikes: Vec<&m::Anomaly> = last
        .iter()
        .filter(|a| a.r#type == m::ANOMALY_RATE_SPIKE)
        .collect();
    assert_eq!(spikes.len(), 1);
    assert_eq!(spikes[0].score, 0.92);
    assert_eq!(spikes[0].detail, "11 events within the observation window");
}

#[test]
fn rate_spike_threshold_exactly_ten() {
    let mut d = Detector::new();
    let mut last = Vec::new();
    for i in 0..9 {
        let tool = format!("t{}", (b'a' + (i % 4) as u8) as char);
        last = d.observe(&call("a1", &tool));
    }
    assert!(!has_type(&last, m::ANOMALY_RATE_SPIKE));
    last = d.observe(&call("a1", "tj"));
    assert!(has_type(&last, m::ANOMALY_RATE_SPIKE));
}

#[test]
fn per_agent_isolation() {
    let mut d = Detector::new();
    for _ in 0..6 {
        d.observe(&call("noisy", "search"));
    }
    let got = d.observe(&call("quiet", "read"));
    assert!(got.is_empty());
}

#[test]
fn empty_history_no_crash() {
    let mut d = Detector::new();
    let out = d.observe(&call("a1", "only"));
    assert!(out.is_empty());
}
