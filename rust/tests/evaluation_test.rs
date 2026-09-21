use guardianforge::evaluation::{Case, Report};

#[test]
fn detection_accuracy_zero_when_no_scenarios() {
    let r = Report {
        scenarios: 0,
        ..Default::default()
    };
    assert_eq!(r.detection_accuracy(), 0.0);
}

#[test]
fn detection_accuracy_fraction() {
    let r = Report {
        scenarios: 4,
        detect_correct: 3,
        ..Default::default()
    };
    assert_eq!(r.detection_accuracy(), 0.75);
    let r2 = Report {
        scenarios: 5,
        detect_correct: 5,
        ..Default::default()
    };
    assert_eq!(r2.detection_accuracy(), 1.0);
}

#[test]
fn report_defaults() {
    let r = Report::default();
    assert!(r.audit_intact);
    assert!(r.cases.is_empty());
    assert_eq!(r.scenarios, 0);
}

#[test]
fn case_defaults() {
    let c = Case::default();
    assert!(!c.intervened);
    assert!(!c.correct);
    assert_eq!(c.r#type, "");
}

#[test]
fn format_perfect_scorecard() {
    let r = Report {
        scenarios: 5,
        detect_correct: 5,
        type_correct: 5,
        false_positives: 0,
        audit_intact: true,
        ..Default::default()
    };
    let expected = "GuardianForge Governance Evaluation\n\n\
        Scenarios:                 5\n\
        Detection accuracy:        100%\n\
        Intervention correctness:  100%\n\
        False positives:           0\n\
        Audit chain intact:        yes\n";
    assert_eq!(r.format(), expected);
}

#[test]
fn format_broken_audit_and_false_positives() {
    let r = Report {
        scenarios: 4,
        detect_correct: 3,
        type_correct: 3,
        false_positives: 2,
        audit_intact: false,
        ..Default::default()
    };
    let expected = "GuardianForge Governance Evaluation\n\n\
        Scenarios:                 4\n\
        Detection accuracy:        75%\n\
        Intervention correctness:  75%\n\
        False positives:           2\n\
        Audit chain intact:        NO\n";
    assert_eq!(r.format(), expected);
}

#[test]
fn format_rounds_half_to_even() {
    // 1/8 = 12.5% -> round half to even -> 12 (NOT 13). This is the Go %.0f contract.
    let r = Report {
        scenarios: 8,
        detect_correct: 1,
        type_correct: 1,
        ..Default::default()
    };
    let out = r.format();
    assert!(out.contains("Detection accuracy:        12%\n"));
    assert!(out.contains("Intervention correctness:  12%\n"));
}

#[test]
fn format_rounds_half_to_even_up_case() {
    // 3/8 = 37.5% -> nearest even -> 38.
    let r = Report {
        scenarios: 8,
        detect_correct: 3,
        type_correct: 3,
        ..Default::default()
    };
    assert!(r.format().contains("Detection accuracy:        38%\n"));
}

#[test]
fn format_zero_scenarios() {
    let r = Report::default();
    let out = r.format();
    assert!(out.contains("Scenarios:                 0\n"));
    assert!(out.contains("Detection accuracy:        0%\n"));
    assert!(out.contains("Intervention correctness:  0%\n"));
}
