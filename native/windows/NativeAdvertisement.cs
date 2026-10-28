using System.Net;
using System.Net.NetworkInformation;
using System.Runtime.InteropServices;

namespace Bram;

// Windows' DNS-SD publisher shares the OS mDNS responder instead of competing
// for per-address UDP 5353 sockets. Registration is scoped to the LAN adapter.
sealed class NativeAdvertisement : IDisposable
{
    [UnmanagedFunctionPointer(CallingConvention.Winapi)] delegate void Completion(uint status, IntPtr context, IntPtr instance);
    [StructLayout(LayoutKind.Sequential)] struct Request
    {
        public uint Version, InterfaceIndex;
        public IntPtr Instance, Callback, Context, Credentials;
        public int UnicastEnabled;
    }
    [DllImport("dnsapi.dll", CharSet = CharSet.Unicode)] static extern IntPtr DnsServiceConstructInstance(string service, string host, ref uint ipv4, IntPtr ipv6, ushort port, ushort priority, ushort weight, uint count, IntPtr keys, IntPtr values);
    [DllImport("dnsapi.dll")] static extern uint DnsServiceRegister(IntPtr request, IntPtr cancel);
    [DllImport("dnsapi.dll")] static extern uint DnsServiceDeRegister(IntPtr request, IntPtr cancel);
    [DllImport("dnsapi.dll")] static extern void DnsServiceFreeInstance(IntPtr instance);
    // Keep callbacks rooted until deregistration completes, including late callbacks.
    static readonly List<NativeAdvertisement> Roots = new();
    readonly Store store;
    readonly Completion callback;
    readonly ManualResetEventSlim deregistered = new();
    IntPtr instance, request;
    bool stopping;
    public NativeAdvertisement(Store store, ushort port, IPAddress address)
    {
        this.store = store;
        var adapter = NetworkInterface.GetAllNetworkInterfaces().First(n => n.GetIPProperties().UnicastAddresses.Any(a => a.Address.Equals(address)));
        uint ipv4 = BitConverter.ToUInt32(address.GetAddressBytes());
        instance = DnsServiceConstructInstance(store.Id + "._bram._tcp.local", store.Id + ".local", ref ipv4, IntPtr.Zero, port, 0, 0, 0, IntPtr.Zero, IntPtr.Zero);
        if (instance == IntPtr.Zero) throw new InvalidOperationException("Windows DNS-SD instance construction failed");
        callback = (status, _, result) =>
        {
            try { if (!stopping) store.Log("Windows DNS-SD registration status " + status + " for " + (result == IntPtr.Zero ? "none" : Marshal.PtrToStringUni(Marshal.ReadIntPtr(result)))); }
            finally { if (result != IntPtr.Zero) DnsServiceFreeInstance(result); if (stopping) deregistered.Set(); }
        };
        var data = new Request { Version = 1, InterfaceIndex = (uint)adapter.GetIPProperties().GetIPv4Properties().Index, Instance = instance, Callback = Marshal.GetFunctionPointerForDelegate(callback), UnicastEnabled = 0 };
        request = Marshal.AllocHGlobal(Marshal.SizeOf<Request>()); Marshal.StructureToPtr(data, request, false);
        lock (Roots) Roots.Add(this);
        var error = DnsServiceRegister(request, IntPtr.Zero);
        if (error != 9506 && error != 0) { Release(); throw new InvalidOperationException("Windows DNS-SD register failed: " + error); }
    }
    void Release()
    {
        if (instance != IntPtr.Zero) { DnsServiceFreeInstance(instance); instance = IntPtr.Zero; }
        if (request != IntPtr.Zero) { Marshal.FreeHGlobal(request); request = IntPtr.Zero; }
        lock (Roots) Roots.Remove(this);
    }
    public void Dispose()
    {
        if (stopping) return; stopping = true;
        var error = DnsServiceDeRegister(request, IntPtr.Zero);
        if (error is not (0 or 9506) || deregistered.Wait(TimeSpan.FromSeconds(2))) Release();
        // Otherwise retain the request and delegate until process exit. The OS
        // removes process registrations even after a forced app termination.
    }
}
