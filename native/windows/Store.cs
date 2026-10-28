using System.Security.AccessControl;
using System.Security.Principal;
using System.Text.Json;
using System.Text.Json.Nodes;

namespace Bram;

sealed class Store
{
    public string Hub { get; } = Environment.GetEnvironmentVariable("BRAM_HUB") ?? Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.UserProfile), "Downloads", "grok-muse");
    public string Root => Path.Combine(Hub, "bram");
    public string Id { get; }
    public string Name { get; }
    public readonly object Gate = new();
    public Store()
    {
        Directory.CreateDirectory(Root);
        // Protect secrets and IPC requests before writing files. Permit the owner and SYSTEM.
        var acl = new DirectorySecurity(); acl.SetAccessRuleProtection(true, false);
        foreach (var sid in new[] { WindowsIdentity.GetCurrent().User!, new SecurityIdentifier(WellKnownSidType.LocalSystemSid, null) })
            acl.AddAccessRule(new FileSystemAccessRule(sid, FileSystemRights.FullControl,
                InheritanceFlags.ContainerInherit | InheritanceFlags.ObjectInherit, PropagationFlags.None, AccessControlType.Allow));
        new DirectoryInfo(Root).SetAccessControl(acl);
        foreach (var folder in new[] { "outbox", "delivered", "failed" }) Directory.CreateDirectory(Path.Combine(Root, folder));
        var identity = Path.Combine(Root, "machine.json");
        if (!File.Exists(identity)) Write(identity, new JsonObject { ["id"] = Environment.MachineName.ToLowerInvariant(), ["name"] = Environment.MachineName });
        var obj = Read(identity); Id = Protocol.Text(obj, "id"); Name = Protocol.Text(obj, "name");
        if (string.IsNullOrWhiteSpace(Id) || Id.Length >= 128) throw new InvalidDataException("Invalid machine identity");
    }
    public static JsonObject Read(string file) => JsonNode.Parse(File.ReadAllText(file)) as JsonObject ?? throw new InvalidDataException("Invalid file " + file);
    public static void Write(string file, JsonObject value)
    {
        var temp = file + "." + Guid.NewGuid() + ".tmp";
        try { File.WriteAllText(temp, value.ToJsonString(new JsonSerializerOptions { WriteIndented = true })); File.Move(temp, file, true); }
        finally { if (File.Exists(temp)) File.Delete(temp); }
    }
    public void Log(string text) { lock (Gate) File.AppendAllText(Path.Combine(Hub, "bram.log"), $"[{Protocol.Stamp()}] {text}\n"); }
    public JsonObject Peers() { lock (Gate) return File.Exists(Path.Combine(Root, "peers.json")) ? Read(Path.Combine(Root, "peers.json")) : new JsonObject(); }
    public void Trust(string id, string name, string secret)
    {
        lock (Gate)
        {
            var peers = Peers(); if (peers.ContainsKey(id)) throw new InvalidOperationException("Already paired");
            peers[id] = new JsonObject { ["id"] = id, ["name"] = name, ["secret"] = secret, ["paired"] = Protocol.Stamp() };
            Write(Path.Combine(Root, "peers.json"), peers);
        }
        Log("paired " + id);
    }
    public void Archive(string source, string folder, JsonObject message)
    {
        Write(Path.Combine(Root, folder, Path.GetFileName(source)), message); File.Delete(source);
    }
    public bool Seen(string id) => new[] { "outbox", "delivered", "failed" }.Any(folder => File.Exists(Path.Combine(Root, folder, id + ".json")));
}
