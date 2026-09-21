//! Per-agent trust scoring, ported from Go `internal/trust`.
//!
//! Trust starts at a baseline and decays on violations and anomalies (weighted by
//! severity), recovering slowly on clean behaviour. Low trust is itself a governance
//! signal. The update is deterministic.

use std::collections::HashMap;

use crate::models::{self, Anomaly, TrustFactor, TrustScore, Violation};

/// The trust value an unseen agent starts at.
pub const BASELINE: f64 = 0.8;

fn clamp(f: f64) -> f64 {
    f.clamp(0.0, 1.0)
}

fn severity_penalty(sev: &str) -> f64 {
    match sev {
        models::SEVERITY_CRITICAL => 0.30,
        models::SEVERITY_HIGH => 0.15,
        models::SEVERITY_MEDIUM => 0.08,
        models::SEVERITY_LOW => 0.03,
        _ => 0.0,
    }
}

/// Maintains per-agent trust.
#[derive(Debug, Default)]
pub struct Scorer {
    scores: HashMap<String, TrustScore>,
}

impl Scorer {
    /// Create an empty scorer.
    pub fn new() -> Self {
        Scorer {
            scores: HashMap::new(),
        }
    }

    /// Return an agent's current trust (`BASELINE` if unseen).
    pub fn get(&self, agent_id: &str) -> f64 {
        self.scores
            .get(agent_id)
            .map(|s| s.score)
            .unwrap_or(BASELINE)
    }

    /// Return the full trust record for an agent.
    pub fn score(&self, agent_id: &str) -> TrustScore {
        if let Some(s) = self.scores.get(agent_id) {
            s.clone()
        } else {
            TrustScore {
                agent_id: agent_id.to_string(),
                score: BASELINE,
                ..Default::default()
            }
        }
    }

    /// Adjust trust from one event's violations/anomalies; clean events recover.
    pub fn update(
        &mut self,
        agent_id: &str,
        violations: &[Violation],
        anomalies: &[Anomaly],
        now: i64,
    ) -> f64 {
        let mut sc = self.scores.get(agent_id).cloned().unwrap_or(TrustScore {
            agent_id: agent_id.to_string(),
            score: BASELINE,
            ..Default::default()
        });

        let mut factors: Vec<TrustFactor> = Vec::new();
        if violations.is_empty() && anomalies.is_empty() {
            let delta = 0.01;
            sc.score = clamp(sc.score + delta);
            factors.push(TrustFactor {
                name: "clean_behaviour".to_string(),
                delta,
                reason: "no violation or anomaly".to_string(),
            });
        } else {
            for v in violations {
                let p = severity_penalty(&v.severity);
                sc.score = clamp(sc.score - p);
                factors.push(TrustFactor {
                    name: "violation".to_string(),
                    delta: -p,
                    reason: v.reason.clone(),
                });
            }
            for a in anomalies {
                let p = 0.05 * a.score;
                sc.score = clamp(sc.score - p);
                factors.push(TrustFactor {
                    name: format!("anomaly:{}", a.r#type),
                    delta: -p,
                    reason: a.detail.clone(),
                });
            }
        }

        sc.agent_id = agent_id.to_string();
        sc.factors = factors;
        sc.updated_at = now;
        let result = sc.score;
        self.scores.insert(agent_id.to_string(), sc);
        result
    }

    /// Report whether an agent's trust has decayed below the isolation floor.
    pub fn should_isolate(&self, agent_id: &str) -> bool {
        self.get(agent_id) < 0.3
    }
}
