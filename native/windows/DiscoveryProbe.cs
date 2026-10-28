using System.Buffers.Binary;
using System.Net;
using System.Net.Sockets;
using System.Runtime.InteropServices;
using System.Security.Cryptography;
using System.Text;

namespace Bram;

// RFC 6762 legacy queries use an ephemeral socket and receive unicast replies.
// The OS keeps ownership of multicast port 5353. A local subnet broadcast also
// reaches responders when WiFi drops multicast. Discovery never grants trust.
static class DiscoveryProbe
{
    [DllImport("dnsapi.dll", ExactSpelling = true)]
    static extern uint DnsExtractRecordsFromMessage_W(byte[] message, ushort length, out IntPtr records);
    [DllImport("dnsapi.dll")] static extern void DnsRecordListFree(IntPtr records, int freeType);

    public static void Query(IPAddress local, IPAddress mask, Func<IPAddress, bool> onSubnet, Action<IntPtr> receive)
    {
        using var socket = new UdpClient(new IPEndPoint(local, 0));
        socket.Client.ReceiveTimeout = 250;
        socket.Client.SetSocketOption(SocketOptionLevel.IP, SocketOptionName.MulticastInterface, local.GetAddressBytes());
        socket.Client.SetSocketOption(SocketOptionLevel.IP, SocketOptionName.MulticastTimeToLive, 255);
        var id = RandomNumberGenerator.GetBytes(2);
        var query = new List<byte> { id[0], id[1], 0, 0, 0, 1, 0, 0, 0, 0, 0, 0 };
        foreach (var label in "_bram._tcp.local".Split('.')) { query.Add((byte)label.Length); query.AddRange(Encoding.UTF8.GetBytes(label)); }
        query.AddRange(new byte[] { 0, 0, 12, 0, 1 });
        var bytes = query.ToArray();
        socket.Send(bytes, new IPEndPoint(IPAddress.Parse("224.0.0.251"), 5353));
        var addressBytes = local.GetAddressBytes(); var maskBytes = mask.GetAddressBytes();
        var broadcast = new IPAddress(addressBytes.Select((b, i) => (byte)(b | ~maskBytes[i])).ToArray());
        if (!broadcast.Equals(local) && !broadcast.Equals(IPAddress.Broadcast))
        {
            socket.EnableBroadcast = true;
            socket.Send(bytes, new IPEndPoint(broadcast, 5353));
        }
        var until = Environment.TickCount64 + 2000;
        for (var count = 0; count < 64 && Environment.TickCount64 < until;)
        {
            IPEndPoint source = new(IPAddress.Any, 0);
            byte[] response;
            try { response = socket.Receive(ref source); }
            catch (SocketException ex) when (ex.SocketErrorCode is SocketError.TimedOut or SocketError.WouldBlock or SocketError.ConnectionReset) { continue; }
            count++;
            if (source.Port != 5353 || !onSubnet(source.Address) || response.Length is < 12 or > 9000 ||
                response[0] != id[0] || response[1] != id[1] || (response[2] & 0x80) == 0 || (response[3] & 15) != 0) continue;
            var total = 0;
            // DnsExtractRecordsFromMessage expects host order header counts.
            for (var offset = 4; offset < 12; offset += 2)
            {
                var value = BinaryPrimitives.ReadUInt16BigEndian(response.AsSpan(offset)); total += value;
                BinaryPrimitives.WriteUInt16LittleEndian(response.AsSpan(offset), value);
            }
            if (total > 128) continue;
            var status = DnsExtractRecordsFromMessage_W(response, (ushort)response.Length, out var records);
            if (status == 0 && records != IntPtr.Zero) receive(records); // receiver owns and frees this list
            else if (records != IntPtr.Zero) DnsRecordListFree(records, 1);
        }
    }
}
