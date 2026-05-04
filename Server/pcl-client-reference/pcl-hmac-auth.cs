// CloudAbroad PCL Client — HMAC Authentication Reference
// This file is a reference implementation for the PCL-CE C# desktop client.
// It signs HTTP requests using the pre-shared secret key from PCL.ini.

using System;
using System.Net.Http;
using System.Security.Cryptography;
using System.Text;

namespace MinePannel.Client
{
    public static class MinePannelAuth
    {
        /// <summary>
        /// Signs an HTTP request with HMAC-SHA256 using the pre-shared secret key.
        /// The signature covers: "timestamp:method:path:bodyHash"
        /// Where bodyHash is the lowercase hex SHA-256 hash of the request body (or empty byte array hash for GET/HEAD).
        /// </summary>
        /// <param name="secret">Pre-shared secret key from PCL.ini "Secret:" field</param>
        /// <param name="method">HTTP method string (GET, POST, etc.) — use HttpMethod.ToString()</param>
        /// <param name="urlPath">URL path + query string, e.g., "/sync/abc12345/instance.json"</param>
        /// <param name="body">Request body bytes (null for GET requests)</param>
        /// <returns>Tuple of (Authorization header value, X-Timestamp header value)</returns>
        public static (string AuthHeader, string TsHeader) SignRequest(
            string secret,
            string method,
            string urlPath,
            byte[] body)
        {
            var timestamp = DateTimeOffset.UtcNow.ToUnixTimeSeconds();
            var bodyHash = Hex(SHA256.HashData(body ?? Array.Empty<byte>()));
            var data = $"{timestamp}:{method}:{urlPath}:{bodyHash}";
            var hmac = Hex(HMACSHA256.HashData(
                Encoding.UTF8.GetBytes(secret),
                Encoding.UTF8.GetBytes(data)));
            return ($"Bearer {hmac}", timestamp.ToString());
        }

        /// <summary>
        /// Creates an HttpRequestMessage with HMAC authentication headers.
        /// </summary>
        public static HttpRequestMessage CreateSignedRequest(
            string secret,
            HttpMethod method,
            string url,
            byte[] body = null)
        {
            var uri = new Uri(url);
            var urlPath = uri.PathAndQuery; // e.g., "/sync/abc12345/instance.json"
            var (authHeader, tsHeader) = SignRequest(secret, method.ToString(), urlPath, body);

            var request = new HttpRequestMessage(method, url)
            {
                Content = body != null ? new ByteArrayContent(body) : null
            };
            request.Headers.Add("Authorization", authHeader);
            request.Headers.Add("X-Timestamp", tsHeader);
            return request;
        }

        /// <summary>
        /// Converts a byte array to lowercase hex string. Equivalent to Go's hex.EncodeToString.
        /// </summary>
        public static string Hex(byte[] bytes)
        {
            return Convert.ToHexString(bytes).ToLowerInvariant();
        }

        // ========== Usage Example ==========
        //
        // // Read secret from PCL.ini
        // string secret = ReadIniKey("PCL.ini", "Secret"); // e.g., "YOUR_SECRET_HERE"
        //
        // // Step 1: Get instance metadata
        // var client = new HttpClient();
        // var req1 = CloudAbroadAuth.CreateSignedRequest(secret, HttpMethod.Get,
        //     "http://192.168.100.101:55000/sync/ebc77384/instance.json");
        // var resp1 = await client.SendAsync(req1);
        // string instanceJson = await resp1.Content.ReadAsStringAsync();
        //
        // // Step 2: Parse ActiveVersion, get manifest
        // string version = ParseJson(instanceJson, "active_version");
        // var req2 = MinePannelAuth.CreateSignedRequest(secret, HttpMethod.Get,
        //     $"http://192.168.100.101:55000/sync/ebc77384/manifests/manifest_{version}.json");
        // var resp2 = await client.SendAsync(req2);
        //
        // // Step 3: Download files as needed...
        // // ...
        //
        // ========== PCL.ini Format ==========
        //
        // Version:CloudAbroad
        // Secret:YOUR_SECRET_HERE
        // Name:ServerName (abc12345)
        // Info:由 PCL 云端驱动的高速同步客户端
        // SyncUrl:http://192.168.100.101:55000/sync/abc12345/instance.json
        // SyncPolicy:Enforce
    }
}
