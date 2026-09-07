using System.Text.Json;
using GuardianForge.Core;

namespace GuardianForge.Agents;

/// <summary>How a supervisor decision was rejected in validation.</summary>
public enum RejectKind { None, Malformed, UnknownType, ObserveViolated, MissingType }

/// <summary>The result of validating a raw supervisor decision.</summary>
public sealed class ValidationResult
{
    public Decision? Decision { get; init; }
    public RejectKind Reject { get; init; }
    public string Message { get; init; } = "";
    public bool Ok => Reject == RejectKind.None;
}

/// <summary>
/// The supervisor synthesizes decisions via a model, then validates them. The model
/// proposes; this class makes sure nothing it proposes acts without passing the policy-mode
/// guardrails. A rejected decision fails safe.
/// </summary>
public sealed class Supervisor
{
    private readonly ILlmClient _model;

    public Supervisor(ILlmClient model) => _model = model;

    /// <summary>Ask the model for a decision and validate it, failing safe on rejection.</summary>
    public async Task<(Decision Decision, string? Rejected)> DecideAsync(Signal signal, CancellationToken ct = default)
    {
        string raw;
        try
        {
            raw = await _model.SynthesizeAsync(signal, ct);
        }
        catch (Exception ex)
        {
            return (FailSafe(signal), ex.Message);
        }

        var result = Validate(raw, signal);
        if (!result.Ok)
            return (FailSafe(signal), result.Message);
        return (result.Decision!, null);
    }

    /// <summary>Parse and hard-check a raw decision against the signal's policy mode.</summary>
    public static ValidationResult Validate(string raw, Signal signal)
    {
        Decision? d;
        try
        {
            d = Json.Deserialize<Decision>(raw);
        }
        catch (JsonException ex)
        {
            return new ValidationResult { Reject = RejectKind.Malformed, Message = $"malformed supervisor output: {ex.Message}" };
        }
        if (d is null)
            return new ValidationResult { Reject = RejectKind.Malformed, Message = "malformed supervisor output: null" };

        if (d.Intervene)
        {
            if (string.IsNullOrEmpty(d.Type))
                return new ValidationResult { Reject = RejectKind.MissingType, Message = "model chose to intervene without a type" };
            if (!InterventionType.All.Contains(d.Type))
                return new ValidationResult { Reject = RejectKind.UnknownType, Message = $"unknown intervention type \"{d.Type}\"" };
            // The core guardrail: an OBSERVE policy can never produce an intervention.
            if (signal.MaxMode == PolicyMode.Observe)
                return new ValidationResult { Reject = RejectKind.ObserveViolated, Message = "model tried to intervene under an OBSERVE policy" };
        }
        return new ValidationResult { Decision = d };
    }

    private static Decision FailSafe(Signal signal)
    {
        if (signal.MaxMode == PolicyMode.Observe)
            return new Decision { Intervene = false, Reason = "fail-safe: observe-only", Confidence = 0 };
        return new Decision { Intervene = true, Type = InterventionType.Escalate, Escalate = true, Reason = "fail-safe: model output rejected; escalating to a human", Confidence = 0 };
    }
}
