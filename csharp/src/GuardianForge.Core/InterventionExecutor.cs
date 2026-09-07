namespace GuardianForge.Core;

/// <summary>Applies a decided intervention to the monitored fleet. The executor at the end
/// of the governance loop: it changes fleet state or routes to a human.</summary>
public sealed class InterventionExecutor
{
    private readonly Fleet _fleet;

    public InterventionExecutor(Fleet fleet) => _fleet = fleet;

    /// <summary>Apply an intervention. Returns false for an unknown type.</summary>
    public bool Execute(Intervention i)
    {
        switch (i.Type)
        {
            case InterventionType.Pause:
                _fleet.Pause(i.TargetAgentId);
                return true;
            case InterventionType.RevokeTool:
                _fleet.RevokeTool(i.TargetAgentId, i.Parameters.GetValueOrDefault("tool", ""));
                return true;
            case InterventionType.InjectConstraint:
                var c = i.Parameters.GetValueOrDefault("constraint", "");
                _fleet.InjectConstraint(i.TargetAgentId, string.IsNullOrEmpty(c) ? i.Reason : c);
                return true;
            case InterventionType.Notify:
            case InterventionType.Escalate:
            case InterventionType.ForceReplan:
                // No fleet mutation - handled by the audit trail and (for escalate) the HITL queue.
                return true;
            default:
                return false;
        }
    }
}
