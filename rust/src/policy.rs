//! The deterministic policy evaluator, ported from Go `internal/policy`.
//!
//! It checks structured rules against events - equality, contains, regexp, and windowed
//! counts - and emits violations, returning the strictest mode and severity among the
//! policies that fired. This runs first, before any model: the fast deterministic path
//! decides the clear cases.

use std::collections::HashMap;

use regex::Regex;

use crate::models::{self, AgentEvent, Policy, Rule, Violation};

const MAX_HISTORY: usize = 256;

/// Holds the policy set and per-agent sliding history for count-based rules.
#[derive(Debug, Default)]
pub struct Evaluator {
    policies: Vec<Policy>,
    history: HashMap<String, Vec<String>>,
    regexps: HashMap<String, Regex>,
}

impl Evaluator {
    /// Build an evaluator over the given policies, compiling regexp rules up front.
    pub fn new(policies: Vec<Policy>) -> Self {
        let mut e = Evaluator {
            policies,
            history: HashMap::new(),
            regexps: HashMap::new(),
        };
        e.compile();
        e
    }

    fn compile(&mut self) {
        self.regexps.clear();
        for p in &self.policies {
            for r in &p.rules {
                if r.op == models::OP_MATCHES {
                    // Go silently skips a rule whose pattern will not compile.
                    if let Ok(rx) = Regex::new(&r.value) {
                        self.regexps.insert(r.rule_id.clone(), rx);
                    }
                }
            }
        }
    }

    /// Return the current policy set.
    pub fn policies(&self) -> &[Policy] {
        &self.policies
    }

    /// Replace the policy set (used by the admin API) and recompile regexps.
    pub fn set_policies(&mut self, policies: Vec<Policy>) {
        self.policies = policies;
        self.compile();
    }

    /// Check an event against every policy; return violations, strictest mode, severity.
    pub fn evaluate(&mut self, ev: &AgentEvent) -> (Vec<Violation>, String, String) {
        let fp = format!("{}:{}:{}", ev.r#type, ev.tool, ev.action);
        let hist = self.history.entry(ev.agent_id.clone()).or_default();
        hist.push(fp);
        if hist.len() > MAX_HISTORY {
            let start = hist.len() - MAX_HISTORY;
            hist.drain(0..start);
        }
        let hist_snapshot = hist.clone();

        let mut violations = Vec::new();
        let mut mode = models::MODE_OBSERVE.to_string();
        let mut sev = models::SEVERITY_INFO.to_string();
        for p in &self.policies {
            for r in &p.rules {
                if matches(&self.regexps, ev, r, &hist_snapshot) {
                    violations.push(Violation {
                        policy_id: p.policy_id.clone(),
                        rule_id: r.rule_id.clone(),
                        agent_id: ev.agent_id.clone(),
                        severity: r.severity.clone(),
                        reason: format!("rule {} matched on {}", r.rule_id, r.field),
                    });
                    mode = models::stricter_mode(&mode, &p.mode);
                    sev = models::higher_severity(&sev, &r.severity);
                }
            }
        }
        (violations, mode, sev)
    }
}

fn matches(regexps: &HashMap<String, Regex>, ev: &AgentEvent, r: &Rule, hist: &[String]) -> bool {
    if r.op == models::OP_COUNT_OVER {
        let window = if r.window_size > 0 {
            r.window_size
        } else {
            hist.len()
        };
        let start = if hist.len() > window {
            hist.len() - window
        } else {
            0
        };
        let mut count: i64 = 0;
        for h in &hist[start..] {
            if h.contains(&r.value) {
                count += 1;
            }
        }
        return count > r.threshold;
    }

    let val = field_value(ev, &r.field);
    match r.op.as_str() {
        models::OP_EQUALS => val == r.value,
        models::OP_NOT_EQUALS => val != r.value,
        models::OP_CONTAINS => val.contains(&r.value),
        models::OP_MATCHES => match regexps.get(&r.rule_id) {
            Some(rx) => rx.is_match(&val),
            None => false,
        },
        _ => false,
    }
}

fn field_value(ev: &AgentEvent, field: &str) -> String {
    match field {
        "type" => ev.r#type.clone(),
        "tool" => ev.tool.clone(),
        "action" => ev.action.clone(),
        _ => {
            if let Some(key) = field.strip_prefix("metadata.") {
                ev.metadata.get(key).cloned().unwrap_or_default()
            } else {
                String::new()
            }
        }
    }
}
