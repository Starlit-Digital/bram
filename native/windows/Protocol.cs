using System.Security.Cryptography;
using System.Text;
using System.Text.Json.Nodes;
using Org.BouncyCastle.Crypto.Modes;
using Org.BouncyCastle.Crypto.Parameters;

namespace Bram;

// Swift CryptoKit ChaChaPoly combined representation: nonce || ciphertext || tag.
static class Protocol
{
    public static JsonObject Seal(JsonObject plain, string secret, string kind, string sender)
    {
        var nonce = RandomNumberGenerator.GetBytes(12);
        var cipher = new Org.BouncyCastle.Crypto.Modes.ChaCha20Poly1305();
        cipher.Init(true, new AeadParameters(new KeyParameter(SHA256.HashData(Encoding.UTF8.GetBytes(secret))),
            128, nonce, Encoding.UTF8.GetBytes($"bram/1/{kind}/{sender}")));
        var input = Encoding.UTF8.GetBytes(plain.ToJsonString());
        var output = new byte[cipher.GetOutputSize(input.Length)];
        var count = cipher.ProcessBytes(input, 0, input.Length, output, 0);
        count += cipher.DoFinal(output, count);
        return new JsonObject { ["version"] = 1, ["kind"] = kind, ["sender"] = sender,
            ["box"] = Convert.ToBase64String(nonce.Concat(output.Take(count)).ToArray()) };
    }
    public static JsonObject Open(JsonObject envelope, string secret, string kind, string sender)
    {
        if (envelope["version"]?.GetValue<int>() != 1 || Text(envelope, "kind") != kind || Text(envelope, "sender") != sender)
            throw new InvalidDataException("Invalid envelope");
        var combined = Convert.FromBase64String(Text(envelope, "box"));
        if (combined.Length < 28) throw new InvalidDataException("Invalid ciphertext");
        var cipher = new Org.BouncyCastle.Crypto.Modes.ChaCha20Poly1305();
        cipher.Init(false, new AeadParameters(new KeyParameter(SHA256.HashData(Encoding.UTF8.GetBytes(secret))),
            128, combined[..12], Encoding.UTF8.GetBytes($"bram/1/{kind}/{sender}")));
        var input = combined[12..];
        var output = new byte[cipher.GetOutputSize(input.Length)];
        var count = cipher.ProcessBytes(input, 0, input.Length, output, 0);
        count += cipher.DoFinal(output, count);
        return JsonNode.Parse(output.AsSpan(0, count)) as JsonObject ?? throw new InvalidDataException("Invalid JSON");
    }
    public static string Text(JsonObject obj, string name) => obj[name]?.GetValue<string>() ?? throw new InvalidDataException("Missing " + name);
    public static string Stamp() => DateTime.UtcNow.ToString("yyyy-MM-dd'T'HH:mm:ss'Z'");
    public static bool IsCode(string value) => value.Length == 32 && value.All(c => char.IsAsciiHexDigit(c));
}
