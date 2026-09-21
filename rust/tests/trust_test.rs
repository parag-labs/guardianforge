use guardianforge::models as m;
use guardianforge::trust::{Scorer, BASELINE};

fn violation(sev: &str) -> m::Violation {
    m::Violation {
        severity: sev.to_string(),
        ..Default::default()
    }
}

fn anomaly(t: &str, score: f64) -> m::Anomaly {
    m::Anomaly {
        r#type: t.to_string(),
        score,
        ..Default::default()
    }
}

#[test]
fn unseen_agent_starts_at_baseline() {
    let s = Scorer::new();
    assert_eq!(s.get("a1"), BASELINE);
    assert_eq!(s.get("a1"), 0.8);
}

#[test]
fn score_record_for_unseen_agent() {
    let s = Scorer::new();
    let sc = s.score("a1");
    assert_eq!(sc.agent_id, "a1");
    assert_eq!(sc.score, BASELINE);
    assert!(sc.factors.is_empty());
}

#[test]
fn violations_decay_trust() {
    let mut s = Scorer::new();
    let before = s.get("a1");
    let after = s.update("a1", &[violation(m::SEVERITY_CRITICAL)], &[], 0);
    assert!(after < before);
    assert_eq!(after, 0.5);
}

#[test]
fn severity_penalties_exact() {
    for (sev, expected) in [
        (m::SEVERITY_CRITICAL, 0.8 - 0.30),
        (m::SEVERITY_HIGH, 0.8 - 0.15),
        (m::SEVERITY_MEDIUM, 0.8 - 0.08),
        (m::SEVERITY_LOW, 0.8 - 0.03),
    ] {
        let mut s = Scorer::new();
        let after = s.update("a1", &[violation(sev)], &[], 0);
        assert!((after - expected).abs() < 1e-9);
    }
}

#[test]
fn clean_behaviour_recovers_trust() {
    let mut s = Scorer::new();
    s.update("a1", &[violation(m::SEVERITY_HIGH)], &[], 0);
    let low = s.get("a1");
    let after = s.update("a1", &[], &[], 0);
    assert!(after > low);
    assert!((after - 0.66).abs() < 1e-9);
}

#[test]
fn clean_behaviour_records_factor() {
    let mut s = Scorer::new();
    s.update("a1", &[], &[], 0);
    let sc = s.score("a1");
    assert_eq!(sc.factors.len(), 1);
    assert_eq!(sc.factors[0].name, "clean_behaviour");
    assert_eq!(sc.factors[0].delta, 0.01);
}

#[test]
fn anomaly_penalty_scaled_by_score() {
    let mut s = Scorer::new();
    let after = s.update("a1", &[], &[anomaly(m::ANOMALY_LOOP, 0.5)], 0);
    assert!((after - 0.775).abs() < 1e-9);
    let sc = s.score("a1");
    assert_eq!(sc.factors[0].name, "anomaly:loop");
}

#[test]
fn repeated_critical_triggers_isolation() {
    let mut s = Scorer::new();
    for _ in 0..3 {
        s.update("bad", &[violation(m::SEVERITY_CRITICAL)], &[], 0);
    }
    assert!(s.should_isolate("bad"));
    assert_eq!(s.get("bad"), 0.0);
    assert!(!s.should_isolate("good"));
}

#[test]
fn isolation_threshold_boundary() {
    let mut s = Scorer::new();
    s.update("a1", &[violation(m::SEVERITY_HIGH)], &[], 0);
    s.update("a1", &[violation(m::SEVERITY_HIGH)], &[], 0);
    assert!(!s.should_isolate("a1"));
    s.update("a1", &[violation(m::SEVERITY_HIGH)], &[], 0);
    assert!(!s.should_isolate("a1"));
    s.update("a1", &[violation(m::SEVERITY_HIGH)], &[], 0);
    assert!(s.should_isolate("a1"));
}

#[test]
fn trust_stays_in_range() {
    let mut s = Scorer::new();
    for _ in 0..20 {
        s.update("a1", &[violation(m::SEVERITY_CRITICAL)], &[], 0);
    }
    assert!(s.get("a1") >= 0.0);
    for _ in 0..200 {
        s.update("a1", &[], &[], 0);
    }
    assert!(s.get("a1") <= 1.0);
}

#[test]
fn multiple_violations_clamp_after_each() {
    let mut s = Scorer::new();
    let after = s.update(
        "a1",
        &[
            violation(m::SEVERITY_CRITICAL),
            violation(m::SEVERITY_CRITICAL),
        ],
        &[],
        0,
    );
    assert!((after - 0.2).abs() < 1e-9);
    assert_eq!(s.score("a1").factors.len(), 2);
}

#[test]
fn factors_replaced_each_update() {
    let mut s = Scorer::new();
    s.update("a1", &[violation(m::SEVERITY_LOW)], &[], 0);
    s.update("a1", &[], &[], 0);
    let sc = s.score("a1");
    assert_eq!(sc.factors.len(), 1);
    assert_eq!(sc.factors[0].name, "clean_behaviour");
}
