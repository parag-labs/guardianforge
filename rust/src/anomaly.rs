//! Deterministic statistical/pattern anomaly detection, ported from Go `internal/anomaly`.
//!
//! It watches each agent's recent behaviour and flags loops (the same tool hammered over
//! and over), rate spikes, and cyclic A-B-A-B patterns. It is deterministic and cheap, and
//! runs on every event before any model is consulted.

use std::collections::HashMap;

use crate::models::{self, AgentEvent, Anomaly};

const WINDOW: usize = 12;
const LOOP_THRESHOLD: usize = 5;
const RATE_THRESHOLD: usize = 10;

fn clamp(f: f64) -> f64 {
    f.clamp(0.0, 1.0)
}

/// Reproduce Go's `float64(int(f*100+0.5)) / 100` (round half up for `f >= 0`).
fn round(f: f64) -> f64 {
    ((f * 100.0 + 0.5) as i64) as f64 / 100.0
}

fn fingerprint(ev: &AgentEvent) -> String {
    format!("{}:{}:{}", ev.r#type, ev.tool, ev.action)
}

/// Keeps a bounded per-agent history and scores anomalies.
#[derive(Debug, Default)]
pub struct Detector {
    history: HashMap<String, Vec<String>>,
}

impl Detector {
    /// Create an empty detector.
    pub fn new() -> Self {
        Detector {
            history: HashMap::new(),
        }
    }

    /// Record an event and return any anomalies detected for the agent.
    pub fn observe(&mut self, ev: &AgentEvent) -> Vec<Anomaly> {
        let entry = self.history.entry(ev.agent_id.clone()).or_default();
        entry.push(fingerprint(ev));
        if entry.len() > WINDOW {
            let start = entry.len() - WINDOW;
            entry.drain(0..start);
        }
        let h = entry.clone();

        let mut out = Vec::new();
        if let Some(a) = detect_loop(&ev.agent_id, &h) {
            out.push(a);
        }
        if let Some(a) = detect_rate(&ev.agent_id, &h) {
            out.push(a);
        }
        if let Some(a) = detect_cyclic(&ev.agent_id, &h) {
            out.push(a);
        }
        out
    }
}

fn detect_loop(agent: &str, h: &[String]) -> Option<Anomaly> {
    let last = h.last()?;
    let mut count = 0usize;
    for x in h.iter().rev() {
        if x == last {
            count += 1;
        } else {
            break;
        }
    }
    if count >= LOOP_THRESHOLD {
        let score = clamp(count as f64 / WINDOW as f64);
        return Some(Anomaly {
            agent_id: agent.to_string(),
            r#type: models::ANOMALY_LOOP.to_string(),
            score: round(score),
            detail: format!("{count} consecutive identical actions ({last})"),
        });
    }
    None
}

fn detect_rate(agent: &str, h: &[String]) -> Option<Anomaly> {
    if h.len() >= RATE_THRESHOLD {
        let score = clamp(h.len() as f64 / WINDOW as f64);
        return Some(Anomaly {
            agent_id: agent.to_string(),
            r#type: models::ANOMALY_RATE_SPIKE.to_string(),
            score: round(score),
            detail: format!("{} events within the observation window", h.len()),
        });
    }
    None
}

fn detect_cyclic(agent: &str, h: &[String]) -> Option<Anomaly> {
    if h.len() < 6 {
        return None;
    }
    let tail = &h[h.len() - 6..];
    let a = &tail[0];
    let b = &tail[1];
    if a == b {
        return None;
    }
    for (i, x) in tail.iter().enumerate() {
        let want = if i % 2 == 0 { a } else { b };
        if x != want {
            return None;
        }
    }
    Some(Anomaly {
        agent_id: agent.to_string(),
        r#type: models::ANOMALY_CYCLIC.to_string(),
        score: 0.7,
        detail: format!("cyclic pattern between \"{a}\" and \"{b}\""),
    })
}
