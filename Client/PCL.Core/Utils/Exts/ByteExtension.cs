using System;
using System.Text;

namespace PCL.Core.Utils.Exts;

public static class ByteExtension
{
    public static string ToHexString(this ReadOnlySpan<byte> bytes) => Convert.ToHexString(bytes).ToLower();
    public static string FromByteToB64(this ReadOnlySpan<byte> bytes) => Convert.ToBase64String(bytes);
    public static string FromBytesToB64UrlSafe(this ReadOnlySpan<byte> bytes) => bytes.FromByteToB64().FromB64ToB64UrlSafe();
}
