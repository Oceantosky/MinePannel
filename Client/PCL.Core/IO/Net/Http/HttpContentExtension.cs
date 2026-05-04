using CommunityToolkit.Mvvm.Messaging.Messages;
using System.Collections.Generic;
using System.Net.Http;
using System.Text;
using System.Text.Json;

namespace PCL.Core.IO.Net.Http.Client.Request;

public static class HttpContentExtension
{
    public static HttpRequestMessage WithContent(this HttpRequestMessage requestMessage, HttpContent content, string? contentType = null)
    {
        requestMessage.Content = content;
        if (contentType is not null) requestMessage.WithHeader("Content-Type", contentType);
        return requestMessage;
    }

    public static HttpRequestMessage WithContent(this HttpRequestMessage requestMessage, string content, string? contentType = null)
    {
        requestMessage.Content = contentType is null
            ? new StringContent(content, Encoding.UTF8)
            : new StringContent(content, Encoding.UTF8, contentType);
        return requestMessage;
    }

    public static HttpRequestMessage WithBinaryContent(this HttpRequestMessage requestMessage, byte[] content)
    {
        requestMessage.Content = new ByteArrayContent(content);
        return requestMessage;
    }

    public static HttpRequestMessage WithJsonContent<T>(this HttpRequestMessage requestMessage, T content, string? contentType = null)
    {
        requestMessage.Content = new StringContent(
            JsonSerializer.Serialize(content),
            Encoding.UTF8,
            contentType ?? "application/json");
        return requestMessage;
    }

    public static HttpRequestMessage WithFormContent(this HttpRequestMessage requestMessage, string form)
    {
        return requestMessage.WithContent(
            new ByteArrayContent(Encoding.UTF8.GetBytes(form)),
            "application/x-www-form-urlencoded");
    }

    public static HttpRequestMessage WithFormContent(this HttpRequestMessage requestMessage, IEnumerable<KeyValuePair<string, string>> pairs)
    {
        requestMessage.Content = new FormUrlEncodedContent(pairs);
        return requestMessage;
    }
}
