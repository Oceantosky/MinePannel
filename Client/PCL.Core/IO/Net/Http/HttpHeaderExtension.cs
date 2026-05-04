using System;
using System.Collections.Generic;
using System.Net.Http;
using System.Net.Http.Headers;

namespace PCL.Core.IO.Net.Http.Client.Request;

public static class HttpHeaderHandler
{
    public static HttpRequestMessage WithHeader(this HttpRequestMessage requestMessage, string key, string value)
    {
        if (key.StartsWith("Content-", StringComparison.OrdinalIgnoreCase) && requestMessage.Content is not null)
            requestMessage.Content.Headers.TryAddWithoutValidation(key, value);
        else
            requestMessage.Headers.TryAddWithoutValidation(key, value);

        return requestMessage;
    }

    public static HttpRequestMessage WithHeaders(this HttpRequestMessage requestMessage, IDictionary<string, string> pairs)
    {
        ArgumentNullException.ThrowIfNull(pairs);

        foreach (var item in pairs)
        {
            requestMessage.WithHeader(item.Key, item.Value);
        }

        return requestMessage;
    }

    public static HttpRequestMessage WithHeader(this HttpRequestMessage requestMessage, KeyValuePair<string, string> pair) =>
        requestMessage.WithHeader(pair.Key, pair.Value);

    public static HttpRequestMessage WithAuthentication(this HttpRequestMessage requestMessage, string scheme, string token)
    {
        ArgumentException.ThrowIfNullOrEmpty(scheme);
        ArgumentException.ThrowIfNullOrEmpty(token);

        requestMessage.Headers.Authorization = new AuthenticationHeaderValue(scheme, token);
        return requestMessage;
    }

    public static HttpRequestMessage WithBearerToken(this HttpRequestMessage requestMessage, string token) =>
        requestMessage.WithAuthentication("Bearer", token);
}
