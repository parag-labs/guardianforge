use guardianforge::models as m;

#[test]
fn severity_rank_ordering() {
    assert_eq!(m::severity_rank(m::SEVERITY_INFO), 0);
    assert_eq!(m::severity_rank(m::SEVERITY_LOW), 1);
    assert_eq!(m::severity_rank(m::SEVERITY_MEDIUM), 2);
    assert_eq!(m::severity_rank(m::SEVERITY_HIGH), 3);
    assert_eq!(m::severity_rank(m::SEVERITY_CRITICAL), 4);
}

#[test]
fn severity_rank_unknown_is_zero() {
    assert_eq!(m::severity_rank("bogus"), 0);
    assert_eq!(m::severity_rank(""), 0);
}

#[test]
fn higher_severity_picks_stronger() {
    assert_eq!(
        m::higher_severity(m::SEVERITY_INFO, m::SEVERITY_CRITICAL),
        m::SEVERITY_CRITICAL
    );
    assert_eq!(
        m::higher_severity(m::SEVERITY_HIGH, m::SEVERITY_LOW),
        m::SEVERITY_HIGH
    );
}

#[test]
fn higher_severity_ties_keep_first() {
    assert_eq!(
        m::higher_severity(m::SEVERITY_MEDIUM, m::SEVERITY_MEDIUM),
        m::SEVERITY_MEDIUM
    );
}

#[test]
fn mode_rank_ordering() {
    assert_eq!(m::mode_rank(m::MODE_OBSERVE), 0);
    assert_eq!(m::mode_rank(m::MODE_SOFT), 1);
    assert_eq!(m::mode_rank(m::MODE_HARD), 2);
    assert_eq!(m::mode_rank(m::MODE_ESCALATE), 3);
    assert_eq!(m::mode_rank("weird"), 0);
}

#[test]
fn stricter_mode_picks_stricter() {
    assert_eq!(
        m::stricter_mode(m::MODE_OBSERVE, m::MODE_HARD),
        m::MODE_HARD
    );
    assert_eq!(
        m::stricter_mode(m::MODE_ESCALATE, m::MODE_SOFT),
        m::MODE_ESCALATE
    );
    assert_eq!(m::stricter_mode(m::MODE_SOFT, m::MODE_SOFT), m::MODE_SOFT);
}

#[test]
fn struct_defaults() {
    let ev = m::AgentEvent::default();
    assert!(ev.metadata.is_empty());
    assert_eq!(ev.r#type, "");
    let rule = m::Rule::default();
    assert_eq!(rule.threshold, 0);
    let d = m::Decision::default();
    assert!(!d.intervene);
    assert!(!d.escalate);
}
