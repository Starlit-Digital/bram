using System.Net;
using System.Net.NetworkInformation;
using System.Net.Sockets;
using System.Runtime.InteropServices;
using System.Text.Json.Nodes;

namespace Bram;

sealed class Discovery : IDisposable
{
    [UnmanagedFunctionPointer(CallingConvention.Winapi)] delegate void Completion(uint status, IntPtr context, IntPtr result);
    [StructLayout(LayoutKind.Sequential)] struct Query
    {
        public uint Version, InterfaceIndex;
        public IntPtr Name, Callback, Context;
    }
    [StructLayout(LayoutKind.Sequential)] struct Header
    {
        public IntPtr Next, Name;
        public ushort Type, Length;
        public uint Flags, Ttl, Reserved;
    }
    [StructLayout(LayoutKind.Sequential)] struct Instance
    {
        public IntPtr Name, Host, IPv4, IPv6;
        public ushort Port, Priority, Weight;
        public uint PropertyCount;
        public IntPtr Keys, Values;
        public uint InterfaceIndex;
    }
    [DllImport("dnsapi.dll")] static extern uint DnsServiceBrowse(IntPtr request, IntPtr cancel);
    [DllImport("dnsapi.dll")] static extern uint DnsServiceBrowseCancel(IntPtr cancel);
    [DllImport("dnsapi.dll")] static extern uint DnsServiceResolve(IntPtr request, IntPtr cancel);
    [DllImport("dnsapi.dll")] static extern uint DnsServiceResolveCancel(IntPtr cancel);
    [DllImport("dnsapi.dll")] static extern void DnsServiceFreeInstance(IntPtr instance);
    [DllImport("dnsapi.dll")] static extern void DnsRecordListFree(IntPtr records, int freeType);
    static readonly List<Discovery> Roots = new();
    readonly Store store;
    readonly object gate = new();
    readonly NativeAdvertisement advertisement;
    readonly Completion browseCallback;
    readonly IntPtr browseQuery, browseName, browseCancel;
    readonly ManualResetEventSlim cancelled = new();
    readonly Dictionary<string, (IPAddress Address, int Port, DateTime Seen)> peers = new();
    readonly HashSet<string> names = new();
    readonly Dictionary<string, (string Target, int Port, DateTime Seen)> servers = new(StringComparer.OrdinalIgnoreCase);
    readonly Dictionary<string, IPAddress> addresses = new(StringComparer.OrdinalIgnoreCase);
    readonly Dictionary<string, Resolution> resolving = new();
    readonly System.Threading.Timer timer;
    readonly IPAddress localAddress, subnetMask;
    int probing;
    bool stopped;
    int debugRecords;
    const string Suffix = "._bram._tcp.local";
    public Discovery(Store store, ushort port)
    {
        this.store = store;
        var address = NetworkInterface.GetAllNetworkInterfaces().Where(n => n.OperationalStatus == OperationalStatus.Up && n.NetworkInterfaceType is NetworkInterfaceType.Ethernet or NetworkInterfaceType.Wireless80211)
            .SelectMany(n => n.GetIPProperties().UnicastAddresses).Select(a => a.Address)
            .First(a => a.AddressFamily == AddressFamily.InterNetwork && !IPAddress.IsLoopback(a) && a.GetAddressBytes()[0] != 169);
        localAddress = address;
        subnetMask = NetworkInterface.GetAllNetworkInterfaces().SelectMany(n => n.GetIPProperties().UnicastAddresses).First(a => a.Address.Equals(address)).IPv4Mask;
        browseCallback = OnBrowse;
        browseName = Marshal.StringToHGlobalUni("_bram._tcp.local");
        browseCancel = Marshal.AllocHGlobal(IntPtr.Size); Marshal.WriteIntPtr(browseCancel, IntPtr.Zero);
        var query = new Query { Version = 1, Name = browseName, Callback = Marshal.GetFunctionPointerForDelegate(browseCallback) };
        browseQuery = Marshal.AllocHGlobal(Marshal.SizeOf<Query>()); Marshal.StructureToPtr(query, browseQuery, false);
        lock (Roots) Roots.Add(this);
        var error = DnsServiceBrowse(browseQuery, browseCancel);
        if (error is not (0 or 9506)) throw new InvalidOperationException("Windows DNS-SD browse failed: " + error);
        advertisement = new NativeAdvertisement(store, port, address);
        timer = new System.Threading.Timer(_ => { Refresh(); Probe(); }, null, 1000, 15000);
        store.Log("Windows DNS-SD browse started");
    }
    void OnBrowse(uint status, IntPtr context, IntPtr records)
    {
        try
        {
            if (status is 995 or 1223) { cancelled.Set(); return; }
            if (status != 0) { store.Log("Windows DNS-SD browse status " + status); return; }
            for (var p = records; p != IntPtr.Zero;)
            {
                var header = Marshal.PtrToStructure<Header>(p);
                if (Interlocked.Increment(ref debugRecords) <= 25) store.Log($"DNS-SD record type {header.Type} ttl {header.Ttl} flags {header.Flags} name {Marshal.PtrToStringUni(header.Name)}");
                var recordName = Marshal.PtrToStringUni(header.Name)?.TrimEnd('.') ?? "";
                lock (gate)
                {
                    if (header.Type == 1) addresses[recordName] = new IPAddress(BitConverter.GetBytes(Marshal.ReadInt32(p, Marshal.SizeOf<Header>())));
                    if (header.Type == 33 && recordName.EndsWith(Suffix, StringComparison.OrdinalIgnoreCase))
                    {
                        var target = Marshal.PtrToStringUni(Marshal.ReadIntPtr(p, Marshal.SizeOf<Header>()))?.TrimEnd('.') ?? "";
                        var port = (ushort)Marshal.ReadInt16(p, Marshal.SizeOf<Header>() + 12);
                        servers[recordName] = (target, port, DateTime.UtcNow);
                    }
                }
                var name = header.Type == 12 ? Marshal.PtrToStringUni(Marshal.ReadIntPtr(p, Marshal.SizeOf<Header>())) : header.Type == 33 ? Marshal.PtrToStringUni(header.Name) : null;
                if (name != null)
                {
                    name = name.TrimEnd('.');
                    if (name.EndsWith(Suffix, StringComparison.OrdinalIgnoreCase) && name[..^Suffix.Length] != store.Id)
                    {
                        lock (gate)
                        {
                            if ((header.Flags & 4) != 0 || header.Ttl == 0) { names.Remove(name); peers.Remove(name[..^Suffix.Length]); }
                            else if (names.Count < 64) names.Add(name);
                        }
                    }
                }
                p = header.Next;
            }
            lock (gate)
            {
                foreach (var pair in servers)
                {
                    var id = pair.Key[..^Suffix.Length];
                    if (id == store.Id || !names.Contains(pair.Key) || !addresses.TryGetValue(pair.Value.Target, out var address)) continue;
                    var fresh = !peers.ContainsKey(id);
                    peers[id] = (address, pair.Value.Port, pair.Value.Seen);
                    if (fresh) store.Log($"discovered {id} at {address}:{pair.Value.Port} from DNS-SD records");
                }
            }
            Refresh();
        }
        catch (Exception ex) { store.Log("DNS-SD browse result: " + ex.Message); }
        finally { if (records != IntPtr.Zero) DnsRecordListFree(records, 1); }
    }
    void Refresh()
    {
        if (stopped) return;
        lock (gate)
        {
            foreach (var name in names.ToArray())
            {
                var id = name[..^Suffix.Length];
                if (peers.TryGetValue(id, out var peer) && (DateTime.UtcNow - peer.Seen).TotalSeconds < 60) continue;
                if (resolving.ContainsKey(name) || resolving.Count >= 16) continue;
                var operation = new Resolution(this, name); resolving[name] = operation; operation.Start();
            }
        }
        Publish();
    }
    bool OnSubnet(IPAddress address)
    {
        if (address.AddressFamily != AddressFamily.InterNetwork || address.Equals(localAddress) || IPAddress.IsLoopback(address)) return false;
        var bytes = address.GetAddressBytes(); var local = localAddress.GetAddressBytes(); var mask = subnetMask.GetAddressBytes();
        return bytes[0] < 224 && bytes.Where((b, i) => (b & mask[i]) != (local[i] & mask[i])).Any() == false;
    }
    void Probe()
    {
        if (stopped || Interlocked.Exchange(ref probing, 1) != 0) return;
        _ = Task.Run(() =>
        {
            try
            {
                DiscoveryProbe.Query(localAddress, subnetMask, OnSubnet, records => { if (stopped) DnsRecordListFree(records, 1); else OnBrowse(0, IntPtr.Zero, records); });
            }
            catch (Exception ex) { if (!stopped) store.Log("Bonjour recovery query: " + ex.Message); }
            finally { Volatile.Write(ref probing, 0); }
        });
    }
    sealed class Resolution
    {
        readonly Discovery owner;
        readonly string name;
        readonly Completion callback;
        readonly IntPtr query, text, cancel;
        System.Threading.Timer? timeout;
        bool released;
        public Resolution(Discovery owner, string name)
        {
            this.owner = owner; this.name = name; callback = Complete;
            text = Marshal.StringToHGlobalUni(name);
            cancel = Marshal.AllocHGlobal(IntPtr.Size); Marshal.WriteIntPtr(cancel, IntPtr.Zero);
            query = Marshal.AllocHGlobal(Marshal.SizeOf<Query>());
            Marshal.StructureToPtr(new Query { Version = 1, Name = text, Callback = Marshal.GetFunctionPointerForDelegate(callback) }, query, false);
        }
        public void Start()
        {
            var error = DnsServiceResolve(query, cancel);
            if (error is not (0 or 9506)) { Release(); return; }
            timeout = new System.Threading.Timer(_ => { lock (owner.gate) if (!released) DnsServiceResolveCancel(cancel); }, null, 8000, Timeout.Infinite);
        }
        void Complete(uint status, IntPtr context, IntPtr result)
        {
            try
            {
                owner.store.Log($"DNS-SD resolve status {status} for {name}");
                if (status == 0 && result != IntPtr.Zero && !owner.stopped)
                {
                    var value = Marshal.PtrToStructure<Instance>(result);
                    owner.store.Log($"DNS-SD resolved host {Marshal.PtrToStringUni(value.Host)} port {value.Port} has IPv4 {value.IPv4 != IntPtr.Zero}");
                    if (value.IPv4 != IntPtr.Zero && value.Port > 0)
                    {
                        var address = new IPAddress(BitConverter.GetBytes(Marshal.ReadInt32(value.IPv4)));
                        var id = name[..^Suffix.Length];
                        lock (owner.gate)
                        {
                            var fresh = !owner.peers.ContainsKey(id);
                            owner.peers[id] = (address, value.Port, DateTime.UtcNow);
                            if (fresh) owner.store.Log($"discovered {id} at {address}:{value.Port}");
                        }
                        owner.Publish();
                    }
                }
            }
            catch (Exception ex) { owner.store.Log("DNS-SD resolve: " + ex.Message); }
            finally { if (result != IntPtr.Zero) DnsServiceFreeInstance(result); Release(); }
        }
        void Release()
        {
            lock (owner.gate)
            {
                if (released) return; released = true; timeout?.Dispose();
                Marshal.FreeHGlobal(query); Marshal.FreeHGlobal(text); Marshal.FreeHGlobal(cancel); owner.resolving.Remove(name);
            }
        }
    }
    public Dictionary<string, (IPAddress Address, int Port)> Nearby()
    {
        lock (gate) return peers.Where(p => (DateTime.UtcNow - p.Value.Seen).TotalSeconds < 120).ToDictionary(p => p.Key, p => (p.Value.Address, p.Value.Port));
    }
    void Publish()
    {
        lock (gate)
        {
            try { var value = new JsonObject(); foreach (var pair in Nearby()) value[pair.Key] = new JsonObject { ["host"] = pair.Value.Address.ToString(), ["port"] = pair.Value.Port };
                Store.Write(Path.Combine(store.Root, "discovered.json"), value); }
            catch (Exception ex) { store.Log("discovery state: " + ex.Message); }
        }
    }
    public void Dispose()
    {
        stopped = true; timer.Dispose(); advertisement.Dispose();
        DnsServiceBrowseCancel(browseCancel);
        if (cancelled.Wait(TimeSpan.FromSeconds(2)))
        {
            Marshal.FreeHGlobal(browseQuery); Marshal.FreeHGlobal(browseName); Marshal.FreeHGlobal(browseCancel);
            lock (Roots) Roots.Remove(this);
        }
        // Outstanding resolves retain this object until their callbacks finish.
    }
}
