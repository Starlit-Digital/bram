using System.IO.Pipes;
using System.Runtime.InteropServices;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json.Nodes;

namespace Bram;

static class Program
{
    [DllImport("kernel32.dll")] static extern bool FreeConsole();
    public static string PipeName(Store store) => "BramWindows-" + Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes(store.Root.ToLowerInvariant())))[..20];
    [STAThread]
    static int Main(string[] args)
    {
        try
        {
            var store = new Store();
            if (args.Length > 0 && args[0] != "tray") return Command(store, args).GetAwaiter().GetResult();
            using var mutex = new Mutex(true, @"Local\" + PipeName(store), out var first);
            if (!first) return 0;
            FreeConsole(); ApplicationConfiguration.Initialize();
            using var relay = new Relay(store);
            using var tray = new Tray(store, relay);
            _ = ServePipe(store, relay, tray); Application.Run(tray); return 0;
        }
        catch (Exception ex) { Console.Error.WriteLine(ex.Message); return 1; }
    }
    static async Task<int> Command(Store store, string[] args)
    {
        if (args[0] is "--help" or "help")
        {
            Console.WriteLine("BramWindows [tray | status | peers | discover | pair-code | pair <peer> <code> | pair-host <host> <port> <code> [peer] | post <to> <from> <text> | tamper <peer> | quit]"); return 0;
        }
        using var pipe = new NamedPipeClientStream(".", PipeName(store), PipeDirection.InOut, PipeOptions.Asynchronous);
        await pipe.ConnectAsync(5000);
        // Elevated SSH clients have a different default token owner than the
        // interactive tray. Check the actual account SID instead.
        var owner = pipe.GetAccessControl().GetOwner(typeof(System.Security.Principal.SecurityIdentifier));
        if (!Equals(owner, System.Security.Principal.WindowsIdentity.GetCurrent().User))
            throw new UnauthorizedAccessException("The local pipe belongs to another user");
        using var writer = new StreamWriter(pipe, new UTF8Encoding(false), leaveOpen: true) { AutoFlush = true };
        using var reader = new StreamReader(pipe, Encoding.UTF8, leaveOpen: true);
        await writer.WriteLineAsync(new JsonObject { ["args"] = new JsonArray(args.Select(a => JsonValue.Create(a)).ToArray()) }.ToJsonString());
        var response = JsonNode.Parse(await reader.ReadLineAsync() ?? "{}")!.AsObject();
        Console.WriteLine(response.ToJsonString()); return response["ok"]?.GetValue<bool>() == true ? 0 : 1;
    }
    static async Task ServePipe(Store store, Relay relay, Tray tray)
    {
        while (!tray.Exiting)
        {
            try
            {
                using var pipe = new NamedPipeServerStream(PipeName(store), PipeDirection.InOut, 1, PipeTransmissionMode.Byte, PipeOptions.Asynchronous | PipeOptions.CurrentUserOnly);
                await pipe.WaitForConnectionAsync();
                using var timeout = new CancellationTokenSource(TimeSpan.FromSeconds(15));
                using var reader = new StreamReader(pipe, Encoding.UTF8, leaveOpen: true);
                using var writer = new StreamWriter(pipe, new UTF8Encoding(false), leaveOpen: true) { AutoFlush = true };
                JsonObject response;
                try
                {
                    var line = await reader.ReadLineAsync(timeout.Token); if (line == null || line.Length > 65536) throw new InvalidDataException("Invalid command");
                    var request = JsonNode.Parse(line)!.AsObject(); var args = request["args"]!.AsArray().Select(a => a!.GetValue<string>()).ToArray();
                    response = await Dispatch(store, relay, args);
                }
                catch (Exception ex) { response = new JsonObject { ["ok"] = false, ["error"] = ex.Message }; }
                await writer.WriteLineAsync(response.ToJsonString());
                if (response["quit"]?.GetValue<bool>() == true) tray.RequestExit();
            }
            catch (Exception ex) { if (!tray.Exiting) store.Log("local command: " + ex.Message); }
        }
    }
    static JsonObject SafePeers(Store store)
    {
        var peers = store.Peers();
        foreach (var pair in peers) pair.Value!.AsObject().Remove("secret");
        return peers;
    }
    static async Task<JsonObject> Dispatch(Store store, Relay relay, string[] args)
    {
        var result = new JsonObject { ["ok"] = true };
        switch (args[0])
        {
            case "status": result["machine"] = store.Id; result["name"] = store.Name; result["port"] = relay.Port; result["pid"] = Environment.ProcessId; result["peers"] = SafePeers(store); result["discovered"] = File.Exists(Path.Combine(store.Root, "discovered.json")) ? Store.Read(Path.Combine(store.Root, "discovered.json")) : new JsonObject(); break;
            case "peers": result["peers"] = SafePeers(store); break;
            case "discover": result["discovered"] = Store.Read(Path.Combine(store.Root, "discovered.json")); break;
            case "pair-code": result["code"] = relay.BeginPairing(); break;
            case "pair" when args.Length == 3:
                var nearby = relay.Discovery.Nearby(); if (!nearby.TryGetValue(args[1], out var endpoint)) throw new InvalidOperationException("Peer not discovered");
                await relay.Pair(endpoint.Address.ToString(), endpoint.Port, args[2], args[1]); break;
            case "pair-host" when args.Length is 4 or 5: await relay.Pair(args[1], int.Parse(args[2]), args[3], args.Length == 5 ? args[4] : null); break;
            case "post" when args.Length >= 4: result["id"] = relay.Post(args[1], args[2], string.Join(' ', args.Skip(3))); break;
            case "tamper" when args.Length == 2: await relay.Send(relay.Message("TestNoop", "TestNoop", "tamper verification"), args[1], true); break;
            case "quit": result["quit"] = true; break;
            default: throw new ArgumentException("Unknown or invalid command");
        }
        return result;
    }
}
