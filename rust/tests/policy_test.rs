use guardianforge::models as m;
use guardianforge::policy::Evaluator;

fn ev(agent: &str, typ: &str, tool: &str, action: &str) -> m::AgentEvent {
    m::AgentEvent {
        agent_id: agent.to_string(),
        r#type: typ.to_string(),
        tool: tool.to_string(),
        action: action.to_string(),
        ..Default::default()
    }
}

fn one(pid: &str, mode: &str, rule: m::Rule) -> m::Policy {
    m::Policy {
        policy_id: pid.to_string(),
        mode: mode.to_string(),
        rules: vec![rule],
        ..Default::default()
    }
}

fn rule(id: &str, field: &str, op: &str, value: &str, severity: &str) -> m::Rule {
    m::Rule {
        rule_id: id.to_string(),
        field: field.to_string(),
        op: op.to_string(),
        value: value.to_string(),
        severity: severity.to_string(),
        ..Default::default()
    }
}

#[test]
fn equality_rule_fires() {
    let p = one(
        "p1",
        m::MODE_HARD,
        rule(
            "r1",
            "tool",
            m::OP_EQUALS,
            "delete_database",
            m::SEVERITY_CRITICAL,
        ),
    );
    let mut e = Evaluator::new(vec![p]);
    let (v, mode, sev) = e.evaluate(&ev("a1", m::EVENT_TOOL_CALL, "delete_database", ""));
    assert_eq!(v.len(), 1);
    assert_eq!(mode, m::MODE_HARD);
    assert_eq!(sev, m::SEVERITY_CRITICAL);
    assert_eq!(v[0].reason, "rule r1 matched on tool");
    assert_eq!(v[0].agent_id, "a1");
}

#[test]
fn no_violation_for_clean_event() {
    let p = one(
        "p1",
        m::MODE_HARD,
        rule(
            "r1",
            "tool",
            m::OP_EQUALS,
            "delete_database",
            m::SEVERITY_CRITICAL,
        ),
    );
    let mut e = Evaluator::new(vec![p]);
    let (v, mode, sev) = e.evaluate(&ev("a1", m::EVENT_TOOL_CALL, "read_file", ""));
    assert!(v.is_empty());
    assert_eq!(mode, m::MODE_OBSERVE);
    assert_eq!(sev, m::SEVERITY_INFO);
}

#[test]
fn not_equals_rule() {
    let p = one(
        "p1",
        m::MODE_SOFT,
        rule("r1", "tool", m::OP_NOT_EQUALS, "safe", m::SEVERITY_LOW),
    );
    let mut e = Evaluator::new(vec![p]);
    assert_eq!(
        e.evaluate(&ev("a1", m::EVENT_TOOL_CALL, "danger", ""))
            .0
            .len(),
        1
    );
    assert!(e
        .evaluate(&ev("a1", m::EVENT_TOOL_CALL, "safe", ""))
        .0
        .is_empty());
}

#[test]
fn contains_rule() {
    let p = one(
        "p1",
        m::MODE_SOFT,
        rule("r1", "action", m::OP_CONTAINS, "drop", m::SEVERITY_MEDIUM),
    );
    let mut e = Evaluator::new(vec![p]);
    assert_eq!(
        e.evaluate(&ev("a1", m::EVENT_DECISION, "", "drop table"))
            .0
            .len(),
        1
    );
    assert!(e
        .evaluate(&ev("a1", m::EVENT_DECISION, "", "select rows"))
        .0
        .is_empty());
}

#[test]
fn regexp_rule() {
    let p = one(
        "p1",
        m::MODE_SOFT,
        rule(
            "r1",
            "action",
            m::OP_MATCHES,
            r"(?i)privilege|escalat",
            m::SEVERITY_HIGH,
        ),
    );
    let mut e = Evaluator::new(vec![p]);
    let (v, _, _) = e.evaluate(&ev(
        "a1",
        m::EVENT_DECISION,
        "",
        "requesting privilege escalation",
    ));
    assert_eq!(v.len(), 1);
}

#[test]
fn invalid_regexp_never_fires() {
    let p = one(
        "p1",
        m::MODE_SOFT,
        rule("r1", "action", m::OP_MATCHES, "(unclosed", m::SEVERITY_HIGH),
    );
    let mut e = Evaluator::new(vec![p]);
    assert!(e
        .evaluate(&ev("a1", m::EVENT_DECISION, "", "(unclosed"))
        .0
        .is_empty());
}

#[test]
fn count_over_detects_repetition() {
    let r = m::Rule {
        threshold: 4,
        window_size: 10,
        ..rule(
            "loop",
            "tool",
            m::OP_COUNT_OVER,
            "search",
            m::SEVERITY_MEDIUM,
        )
    };
    let p = one("p1", m::MODE_HARD, r);
    let mut e = Evaluator::new(vec![p]);
    let mut last = Vec::new();
    for _ in 0..6 {
        last = e.evaluate(&ev("a1", m::EVENT_TOOL_CALL, "search", "")).0;
    }
    assert!(!last.is_empty());
    let v = e.evaluate(&ev("a2", m::EVENT_TOOL_CALL, "search", "")).0;
    assert!(v.is_empty());
}

#[test]
fn count_over_threshold_is_strict() {
    let r = m::Rule {
        threshold: 4,
        window_size: 10,
        ..rule(
            "loop",
            "tool",
            m::OP_COUNT_OVER,
            "search",
            m::SEVERITY_MEDIUM,
        )
    };
    let p = one("p1", m::MODE_HARD, r);
    let mut e = Evaluator::new(vec![p]);
    let mut first_fire = 0;
    for i in 0..6 {
        let v = e.evaluate(&ev("a1", m::EVENT_TOOL_CALL, "search", "")).0;
        if !v.is_empty() && first_fire == 0 {
            first_fire = i + 1;
        }
    }
    assert_eq!(first_fire, 5);
}

#[test]
fn strictest_mode_and_highest_severity_win() {
    let observe = one(
        "obs",
        m::MODE_OBSERVE,
        rule("r1", "type", m::OP_EQUALS, "TOOL_CALL", m::SEVERITY_LOW),
    );
    let hard = one(
        "hard",
        m::MODE_HARD,
        rule("r2", "tool", m::OP_EQUALS, "rm", m::SEVERITY_HIGH),
    );
    let mut e = Evaluator::new(vec![observe, hard]);
    let (v, mode, sev) = e.evaluate(&ev("a1", m::EVENT_TOOL_CALL, "rm", ""));
    assert_eq!(v.len(), 2);
    assert_eq!(mode, m::MODE_HARD);
    assert_eq!(sev, m::SEVERITY_HIGH);
}

#[test]
fn metadata_field() {
    let p = one(
        "p1",
        m::MODE_SOFT,
        rule("r1", "metadata.pii", m::OP_EQUALS, "true", m::SEVERITY_HIGH),
    );
    let mut e = Evaluator::new(vec![p]);
    let mut event = ev("a1", m::EVENT_TOOL_RESULT, "read", "");
    event.metadata.insert("pii".to_string(), "true".to_string());
    assert_eq!(e.evaluate(&event).0.len(), 1);
}

#[test]
fn missing_metadata_field_is_empty() {
    let p = one(
        "p1",
        m::MODE_SOFT,
        rule("r1", "metadata.pii", m::OP_EQUALS, "", m::SEVERITY_HIGH),
    );
    let mut e = Evaluator::new(vec![p]);
    assert_eq!(
        e.evaluate(&ev("a1", m::EVENT_TOOL_RESULT, "read", ""))
            .0
            .len(),
        1
    );
}

#[test]
fn set_policies_replaces_and_recompiles() {
    let mut e = Evaluator::new(vec![]);
    assert!(e
        .evaluate(&ev("a1", m::EVENT_TOOL_CALL, "rm", ""))
        .0
        .is_empty());
    e.set_policies(vec![one(
        "p1",
        m::MODE_HARD,
        rule("r1", "tool", m::OP_EQUALS, "rm", m::SEVERITY_HIGH),
    )]);
    assert_eq!(
        e.evaluate(&ev("a1", m::EVENT_TOOL_CALL, "rm", "")).0.len(),
        1
    );
    assert_eq!(e.policies().len(), 1);
}

#[test]
fn count_over_default_window_uses_full_history() {
    let r = m::Rule {
        threshold: 2,
        window_size: 0,
        ..rule(
            "loop",
            "tool",
            m::OP_COUNT_OVER,
            "search",
            m::SEVERITY_MEDIUM,
        )
    };
    let p = one("p1", m::MODE_HARD, r);
    let mut e = Evaluator::new(vec![p]);
    let mut last = Vec::new();
    for _ in 0..3 {
        last = e.evaluate(&ev("a1", m::EVENT_TOOL_CALL, "search", "")).0;
    }
    assert!(!last.is_empty());
}
