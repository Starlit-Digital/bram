using System.Net;
using System.Net.Sockets;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json.Nodes;

namespace Bram;

sealed class Relay : IDisposable
{
    readonly Store store;
    readonly TcpListener listener;
    readonly CancellationTokenSource stop = new();
    readonly SemaphoreSlim inbound = new(16);
    readonly SemaphoreSlim pollGate = new(1);
    readonly object pairingGate = new();
    string? code;
    DateTime expiry;
    public Discovery Discovery { get; }
    public int Port { get; }
    public Relay(Store store, int port = 47831)
    {
        this.store = store;
        listener = new TcpListener(IPAddress.Any, port); listener.Start();
        Port = ((IPEndPoint)listener.LocalEndpoint).Port;
        Store.Write(Path.Combine(store.Root, "network.json"), new JsonObject { ["port"] = Port, ["machine"] = store.Id });
        Discovery = new Discovery(store, (ushort)Port);
        _ = Listen(); _ = Poll(); store.Log("Bram Windows started; network listening port " + Port);
    }
    public string BeginPairing()
    {
        lock (pairingGate) { code = Convert.ToHexString(RandomNumberGenerator.GetBytes(16)).ToLowerInvariant(); expiry = DateTime.UtcNow.AddSeconds(120); store.Log("pairing window opened for 120 seconds"); return code; }
    }
    async Task Listen()
    {
        while (!stop.IsCancellationRequested)
        {
            try
            {
                var client = await listener.AcceptTcpClientAsync(stop.Token);
                if (!await inbound.WaitAsync(0, stop.Token)) { client.Dispose(); continue; }
                _ = Serve(client);
            }
            catch (OperationCanceledException) { break; }
            catch (Exception ex) { if (!stop.IsCancellationRequested) store.Log("listener: " + ex.Message); }
        }
    }
    async Task Serve(TcpClient client)
    {
        using (client)
        using (var timeout = CancellationTokenSource.CreateLinkedTokenSource(stop.Token))
        {
            timeout.CancelAfter(TimeSpan.FromSeconds(10));
            try { var request = await ReadFrame(client.GetStream(), timeout.Token); var response = Handle(request); await WriteFrame(client.GetStream(), response, timeout.Token); }
            catch (Exception ex) { if (!stop.IsCancellationRequested) store.Log("connection rejected: " + ex.Message); }
            finally { inbound.Release(); }
        }
    }
    public static async Task<JsonObject> ReadFrame(NetworkStream stream, CancellationToken token)
    {
        using var data = new MemoryStream(); var buffer = new byte[4096];
        while (true)
        {
            int read = await stream.ReadAsync(buffer, token); if (read == 0) throw new EndOfStreamException();
            int newline = Array.IndexOf(buffer, (byte)10, 0, read); int count = newline < 0 ? read : newline;
            if (data.Length + count >= 65536) throw new InvalidDataException("Frame too large");
            data.Write(buffer, 0, count);
            if (newline >= 0) return JsonNode.Parse(data.ToArray()) as JsonObject ?? throw new InvalidDataException("Invalid envelope");
        }
    }
    public static async Task WriteFrame(NetworkStream stream, JsonObject obj, CancellationToken token)
    {
        var bytes = Encoding.UTF8.GetBytes(obj.ToJsonString() + "\n"); if (bytes.Length > 65536) throw new InvalidDataException("Frame too large");
        await stream.WriteAsync(bytes, token);
    }
    static async Task<JsonObject> Exchange(string host, int port, JsonObject obj)
    {
        using var timeout = new CancellationTokenSource(TimeSpan.FromSeconds(10));
        using var client = new TcpClient(); await client.ConnectAsync(host, port, timeout.Token);
        await WriteFrame(client.GetStream(), obj, timeout.Token); return await ReadFrame(client.GetStream(), timeout.Token);
    }
    JsonObject Reject(JsonObject obj, string reason)
    {
        Store.Write(Path.Combine(store.Root, "failed", Guid.NewGuid() + ".json"), new JsonObject { ["envelope"] = obj.DeepClone(), ["reason"] = reason, ["failed"] = Protocol.Stamp() });
        store.Log("network rejected: " + reason); return new JsonObject { ["error"] = "rejected" };
    }
    JsonObject Handle(JsonObject obj)
    {
        try
        {
            var sender = Protocol.Text(obj, "sender"); if (sender.Length == 0 || sender.Length >= 128 || sender == store.Id) return Reject(obj, "invalid sender");
            if (Protocol.Text(obj, "kind") == "pair")
            {
                lock (pairingGate)
                {
                    if (code == null || DateTime.UtcNow >= expiry) return Reject(obj, "pairing closed");
                    var plain = Protocol.Open(obj, code, "pair", sender);
                    var name = Protocol.Text(plain, "name"); var secret = Protocol.Text(plain, "secret"); var challenge = Protocol.Text(plain, "challenge");
                    if (Protocol.Text(plain, "id") != sender || challenge.Length >= 128 || Convert.FromBase64String(secret).Length != 32 || store.Peers().ContainsKey(sender)) return Reject(obj, "invalid or already paired");
                    var response = Protocol.Seal(new JsonObject { ["id"] = store.Id, ["name"] = store.Name, ["challenge"] = challenge }, code, "paired", store.Id);
                    store.Trust(sender, name, secret); code = null; return response;
                }
            }
            var peer = store.Peers()[sender] as JsonObject; if (peer == null) return Reject(obj, "unpaired sender");
            var shared = Protocol.Text(peer, "secret"); var message = Protocol.Open(obj, shared, "message", sender);
            var id = Protocol.Text(message, "id"); var created = Protocol.Text(message, "created");
            if (!Guid.TryParseExact(id, "D", out _) || Protocol.Text(message, "machine") != sender || Protocol.Text(message, "to") != "TestNoop" || string.IsNullOrEmpty(Protocol.Text(message, "from")) || Encoding.UTF8.GetByteCount(Protocol.Text(message, "text")) > 32768 || !created.EndsWith('Z') || !DateTimeOffset.TryParse(created, out _)) return Reject(obj, "invalid message or unsupported target");
            lock (store.Gate)
            {
                if (!store.Seen(id)) { Store.Write(Path.Combine(store.Root, "outbox", id + ".json"), message); store.Log($"network received {id} from {sender}"); }
            }
            return Protocol.Seal(new JsonObject { ["id"] = id, ["accepted"] = true }, shared, "receipt", store.Id);
        }
        catch { return Reject(obj, "authentication or storage failure"); }
    }
    public async Task Pair(string host, int port, string value, string? expected = null)
    {
        if (!Protocol.IsCode(value)) throw new InvalidDataException("Code must contain 32 hexadecimal characters");
        value = value.ToLowerInvariant(); var secret = Convert.ToBase64String(RandomNumberGenerator.GetBytes(32)); var challenge = Guid.NewGuid().ToString();
        var obj = Protocol.Seal(new JsonObject { ["id"] = store.Id, ["name"] = store.Name, ["secret"] = secret, ["challenge"] = challenge }, value, "pair", store.Id);
        var response = await Exchange(host, port, obj); var sender = Protocol.Text(response, "sender");
        if (sender == store.Id || (expected != null && expected != sender)) throw new InvalidDataException("Unexpected peer");
        var plain = Protocol.Open(response, value, "paired", sender);
        if (Protocol.Text(plain, "id") != sender || Protocol.Text(plain, "challenge") != challenge) throw new InvalidDataException("Invalid pairing response");
        store.Trust(sender, Protocol.Text(plain, "name"), secret);
    }
    public async Task Send(JsonObject message, string peerId, bool tamper = false)
    {
        var peer = store.Peers()[peerId] as JsonObject ?? throw new InvalidOperationException("Not paired");
        var nearby = Discovery.Nearby(); if (!nearby.TryGetValue(peerId, out var endpoint)) throw new InvalidOperationException("Peer not discovered");
        var secret = Protocol.Text(peer, "secret"); var obj = Protocol.Seal(message, secret, "message", store.Id);
        if (tamper) { var box = Convert.FromBase64String(Protocol.Text(obj, "box")); box[^1] ^= 1; obj["box"] = Convert.ToBase64String(box); }
        var response = await Exchange(endpoint.Address.ToString(), endpoint.Port, obj);
        if (tamper) { if (response["error"]?.GetValue<string>() != "rejected") throw new InvalidDataException("Tampering was not rejected"); store.Log("tampered message rejected by " + peerId); return; }
        var receipt = Protocol.Open(response, secret, "receipt", peerId);
        if (Protocol.Text(receipt, "id") != Protocol.Text(message, "id") || receipt["accepted"]?.GetValue<bool>() != true) throw new InvalidDataException("Invalid receipt");
    }
    public JsonObject Message(string to, string from, string text) => new() { ["id"] = Guid.NewGuid().ToString().ToUpperInvariant(), ["to"] = to, ["from"] = from, ["machine"] = store.Id, ["created"] = Protocol.Stamp(), ["text"] = text };
    public string Post(string to, string from, string text)
    {
        if (Encoding.UTF8.GetByteCount(text) > 32768) throw new InvalidDataException("Text too large");
        var msg = Message(to, from, text); var id = Protocol.Text(msg, "id"); Store.Write(Path.Combine(store.Root, "outbox", id + ".json"), msg); store.Log($"post {id} {from}->{to}"); return id;
    }
    async Task Poll()
    {
        while (!stop.IsCancellationRequested)
        {
            try { await PollOnce(); await Task.Delay(1000, stop.Token); }
            catch (OperationCanceledException) { break; }
            catch (Exception ex) { store.Log("outbox: " + ex.Message); await Task.Delay(1000); }
        }
    }
    async Task PollOnce()
    {
        if (!await pollGate.WaitAsync(0)) return;
        try
        {
            foreach (var file in Directory.GetFiles(Path.Combine(store.Root, "outbox"), "*.json"))
            {
                JsonObject msg; try { msg = Store.Read(file); } catch { continue; }
                var to = Protocol.Text(msg, "to"); var id = Protocol.Text(msg, "id");
                if (to.Contains('@'))
                {
                    var pieces = to.Split('@', 2);
                    if (!store.Peers().ContainsKey(pieces[1])) { msg["reason"] = "unknown remote peer"; msg["failed"] = Protocol.Stamp(); store.Archive(file, "failed", msg); continue; }
                    if (!Discovery.Nearby().ContainsKey(pieces[1])) continue;
                    var wire = (JsonObject)msg.DeepClone(); wire["to"] = pieces[0];
                    try { await Send(wire, pieces[1]); msg["relayed"] = Protocol.Stamp(); store.Archive(file, "delivered", msg); store.Log($"msg {id}: remote accepted"); }
                    catch (Exception ex) { store.Log($"msg {id}: remote retry: {ex.Message}"); }
                }
                else if (to == "TestNoop")
                {
                    msg["delivered"] = Protocol.Stamp(); store.Archive(file, "delivered", msg); store.Log($"msg {id}: delivered to TestNoop");
                }
                else { msg["reason"] = "unsupported target"; msg["failed"] = Protocol.Stamp(); store.Archive(file, "failed", msg); store.Log($"msg {id}: unsupported target -> failed"); }
            }
        }
        finally { pollGate.Release(); }
    }
    public void Dispose() { stop.Cancel(); listener.Stop(); Discovery.Dispose(); store.Log("Bram Windows stopped"); }
}
