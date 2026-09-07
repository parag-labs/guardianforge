using GuardianForge.Core;

namespace GuardianForge.Agents;

/// <summary>The governance scorecard: runs every failure scenario with the deterministic
/// mock supervisor and checks detection, response correctness, false positives, and audit
/// integrity.</summary>
public static class Evaluation
{
    public sealed class Report
    {
        public int Scenarios { get; set; }
        public int DetectCorrect { get; set; }
        public int TypeCorrect { get; set; }
        public int FalsePositives { get; set; }
        public bool AuditIntact { get; set; } = true;
        public List<(string Id, bool Intervened, string Type, bool Correct)> Cases { get; } = new();

        public double DetectionAccuracy => Scenarios == 0 ? 0 : (double)DetectCorrect / Scenarios;
    }

    public static async Task<Report> RunAsync()
    {
        var now = () => DateTimeOffset.FromUnixTimeSeconds(0);
        var rep = new Report();

        foreach (var sc in Scenarios.All())
        {
            rep.Scenarios++;
            var fleet = new Fleet();
            var sup = new Supervisor(new MockLlm());
            var eng = new Engine(Scenarios.DefaultPolicies(), fleet, sup, now);

            Intervention? last = null;
            foreach (var ev in sc.Events)
            {
                var outcome = await eng.ProcessAsync(ev);
                if (outcome.Intervention != null) last = outcome.Intervention;
            }

            var intervened = last != null;
            var detectOk = intervened == sc.ExpectIntervene;
            if (detectOk) rep.DetectCorrect++;
            if (intervened && !sc.ExpectIntervene) rep.FalsePositives++;

            var gotType = last?.Type ?? "";
            bool typeOk;
            if (sc.ExpectIntervene)
                typeOk = intervened && gotType == sc.ExpectType;
            else
                typeOk = !intervened;
            if (typeOk) rep.TypeCorrect++;

            if (!eng.Audit.Verify()) rep.AuditIntact = false;
            rep.Cases.Add((sc.Id, intervened, gotType, detectOk && typeOk));
        }
        return rep;
    }

    public static string Format(Report r)
    {
        var typeAcc = r.Scenarios == 0 ? 0 : 100.0 * r.TypeCorrect / r.Scenarios;
        return
            "GuardianForge Governance Evaluation\n\n" +
            $"Scenarios:                 {r.Scenarios}\n" +
            $"Detection accuracy:        {100.0 * r.DetectionAccuracy:0}%\n" +
            $"Intervention correctness:  {typeAcc:0}%\n" +
            $"False positives:           {r.FalsePositives}\n" +
            $"Audit chain intact:        {(r.AuditIntact ? "yes" : "NO")}\n";
    }
}
