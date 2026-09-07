using System.Security.Cryptography;
using System.Text;

namespace GuardianForge.Core;

/// <summary>
/// The tamper-evident, append-only audit log. Every governance decision and intervention
/// is chained by SHA-256, so any after-the-fact edit or reordering breaks the chain and is
/// caught by <see cref="Verify"/>.
/// </summary>
public sealed class AuditLog
{
    private readonly List<AuditEntry> _entries = new();
    private readonly object _lock = new();
    private int _seq;

    private static string Hash(string prev, string ts, string actor, string action, string details)
    {
        var bytes = SHA256.HashData(Encoding.UTF8.GetBytes($"{prev}|{ts}|{actor}|{action}|{details}"));
        return Convert.ToHexString(bytes).ToLowerInvariant();
    }

    /// <summary>Append a chained entry and return it.</summary>
    public AuditEntry Append(string actor, string action, string details, DateTimeOffset now)
    {
        lock (_lock)
        {
            var prev = _entries.Count > 0 ? _entries[^1].CurrentHash : "";
            _seq++;
            var ts = now.ToUniversalTime().ToString("o");
            var e = new AuditEntry
            {
                EntryId = $"audit-{_seq}",
                PreviousHash = prev,
                Timestamp = now.ToUniversalTime(),
                Actor = actor,
                Action = action,
                Details = details,
            };
            e.CurrentHash = Hash(prev, ts, actor, action, details);
            _entries.Add(e);
            return e;
        }
    }

    /// <summary>A copy of the log.</summary>
    public List<AuditEntry> Entries()
    {
        lock (_lock) return new List<AuditEntry>(_entries);
    }

    /// <summary>Number of entries.</summary>
    public int Count { get { lock (_lock) return _entries.Count; } }

    /// <summary>Recompute the chain and report whether it is intact.</summary>
    public bool Verify()
    {
        lock (_lock)
        {
            var prev = "";
            foreach (var e in _entries)
            {
                var ts = e.Timestamp.ToUniversalTime().ToString("o");
                var want = Hash(prev, ts, e.Actor, e.Action, e.Details);
                if (want != e.CurrentHash || e.PreviousHash != prev) return false;
                prev = e.CurrentHash;
            }
            return true;
        }
    }

    /// <summary>Backing list, for white-box tamper tests.</summary>
    internal List<AuditEntry> Backing => _entries;
}
