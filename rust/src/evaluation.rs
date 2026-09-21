//! The governance scorecard (pure scoring + formatting), ported from Go `internal/eval`.
//!
//! This is the pure part of the Go `eval` package: the [`Report`] and [`Case`] aggregates,
//! the detection-accuracy ratio, and the stable text [`Report::format`]. The scenario driver
//! (`Run`) is intentionally excluded - it wires the fleet, agents, LLM, and runtime, which
//! remain in Go.

/// Round to the nearest integer, ties to even - matching Go's `%.0f` (for `x >= 0`).
///
/// Note: the existing C# mirror formats with `{:0}`, which rounds half away from zero and
/// therefore diverges on exact-`.5` percentages (e.g. 12.5). Go is the stated reference, so
/// this port rounds half to even.
fn round_half_even(x: f64) -> i64 {
    let floor = x.floor();
    let diff = x - floor;
    let f = floor as i64;
    if diff < 0.5 {
        f
    } else if diff > 0.5 {
        f + 1
    } else if f % 2 == 0 {
        f
    } else {
        f + 1
    }
}

/// One scenario's outcome.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Case {
    /// The scenario identifier.
    pub id: String,
    /// Whether GuardianForge intervened.
    pub intervened: bool,
    /// The intervention type taken, if any.
    pub r#type: String,
    /// Whether the outcome matched the expectation.
    pub correct: bool,
}

/// The aggregate scorecard.
#[derive(Debug, Clone, PartialEq)]
pub struct Report {
    /// The number of scenarios run.
    pub scenarios: i64,
    /// The number where intervene/observe matched truth.
    pub detect_correct: i64,
    /// The number where the response type was correct.
    pub type_correct: i64,
    /// The number of healthy agents wrongly intervened on.
    pub false_positives: i64,
    /// Whether the audit chain stayed intact.
    pub audit_intact: bool,
    /// The per-scenario outcomes.
    pub cases: Vec<Case>,
}

impl Default for Report {
    fn default() -> Self {
        Report {
            scenarios: 0,
            detect_correct: 0,
            type_correct: 0,
            false_positives: 0,
            audit_intact: true,
            cases: Vec::new(),
        }
    }
}

impl Report {
    /// The fraction of scenarios where intervene/observe matched truth (0 if none).
    pub fn detection_accuracy(&self) -> f64 {
        if self.scenarios == 0 {
            return 0.0;
        }
        self.detect_correct as f64 / self.scenarios as f64
    }

    /// Render the scorecard in a stable, readable form.
    pub fn format(&self) -> String {
        let intact = if self.audit_intact { "yes" } else { "NO" };
        let type_acc = if self.scenarios > 0 {
            100.0 * self.type_correct as f64 / self.scenarios as f64
        } else {
            0.0
        };
        let mut s = String::from("GuardianForge Governance Evaluation\n\n");
        s.push_str(&format!("{:<27}{}\n", "Scenarios:", self.scenarios));
        s.push_str(&format!(
            "{:<27}{}%\n",
            "Detection accuracy:",
            round_half_even(100.0 * self.detection_accuracy())
        ));
        s.push_str(&format!(
            "{:<27}{}%\n",
            "Intervention correctness:",
            round_half_even(type_acc)
        ));
        s.push_str(&format!(
            "{:<27}{}\n",
            "False positives:", self.false_positives
        ));
        s.push_str(&format!("{:<27}{}\n", "Audit chain intact:", intact));
        s
    }
}
