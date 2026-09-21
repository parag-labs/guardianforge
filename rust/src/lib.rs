//! Deterministic governance core of GuardianForge, ported from Go.
//!
//! This crate contains only the pure, dependency-free algorithmic core: the domain
//! types, the deterministic anomaly detector, the policy evaluator, the trust scorer,
//! and the evaluation scorecard. The LLM agents, HTTP API, observability sinks, fleet
//! scenario driver, and audit storage deliberately remain in the Go reference (and its
//! C# mirror) - they are not part of the portable decision core.
//!
//! The behaviour here is cross-checked against both the Go implementation and the
//! existing C# port; where they agree, that agreement is the contract.

pub mod anomaly;
pub mod evaluation;
pub mod models;
pub mod policy;
pub mod trust;
