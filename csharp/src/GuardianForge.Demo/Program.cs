using GuardianForge.Agents;

// `dotnet run --project src/GuardianForge.Demo -- --eval` runs the governance scorecard and
// exits non-zero on any regression. Without --eval it prints a short walkthrough of the
// built-in scenarios.

if (args.Contains("--eval"))
{
    var report = await Evaluation.RunAsync();
    Console.Write(Evaluation.Format(report));
    if (report.DetectionAccuracy < 1.0 || report.FalsePositives > 0 || !report.AuditIntact)
    {
        Console.Error.WriteLine("FAIL: governance quality bar not met");
        Environment.Exit(1);
    }
    return;
}

var rep = await Evaluation.RunAsync();
Console.Write(Evaluation.Format(rep));
Console.WriteLine("\nPer-scenario outcomes:");
foreach (var (id, intervened, type, correct) in rep.Cases)
{
    var action = intervened ? type : "no intervention";
    Console.WriteLine($"  {(correct ? "OK " : "!! ")} {id,-22} -> {action}");
}
