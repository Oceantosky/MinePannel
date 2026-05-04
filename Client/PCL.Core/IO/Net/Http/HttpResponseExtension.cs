using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Net.Http;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;

namespace PCL.Core.IO.Net.Http.Client.Request;

public static class HttpResponseExtension
{
    public static bool IsSuccess(this HttpResponseMessage responseMessage) => responseMessage.IsSuccessStatusCode;

    public static string AsString(this HttpResponseMessage responseMessage) { return responseMessage.AsStringAsync().GetAwaiter().GetResult(); }

    public static async Task<string> AsStringAsync(this HttpResponseMessage responseMessage, CancellationToken ct = default)
    {
        try
        {
            return await responseMessage.Content.ReadAsStringAsync(ct).ConfigureAwait(false);
        } catch(TaskCanceledException)
        {
            throw new TimeoutException("The request was canceled due to a timeout.");
        } catch(OperationCanceledException)
        {
            throw new TimeoutException("The operation was canceled.");
        }
    }

    public static async Task<Stream> AsStreamAsync(this HttpResponseMessage responseMessage, CancellationToken ct = default)
    {
        try
        {
            return await responseMessage.Content.ReadAsStreamAsync(ct).ConfigureAwait(false);
        } catch(TaskCanceledException)
        {
            throw new TimeoutException("The request was canceled due to a timeout.");
        } catch(OperationCanceledException)
        {
            throw new TimeoutException("The operation was canceled.");
        }
    }

    public static async Task<byte[]> AsByteArrayAsync(this HttpResponseMessage responseMessage, CancellationToken ct = default)
    {
        try
        {
            return await responseMessage.Content.ReadAsByteArrayAsync(ct).ConfigureAwait(false);
        } catch(TaskCanceledException)
        {
            throw new TimeoutException("The request was canceled due to a timeout.");
        } catch(OperationCanceledException)
        {
            throw new TimeoutException("The operation was canceled.");
        }
    }

    public static async Task<T?> AsJsonAsync<T>(
        this HttpResponseMessage responseMessage,
        JsonSerializerOptions? options = null,
        CancellationToken cancellationToken = default)
    {
        try
        {
            await using var stream = await responseMessage.AsStreamAsync(cancellationToken).ConfigureAwait(false);
            return await JsonSerializer.DeserializeAsync<T>(stream, options, cancellationToken)
                .ConfigureAwait(false);
        } catch(JsonException ex)
        {
            throw new InvalidDataException("Failed to deserialize JSON response.", ex);
        }
    }

    public static Dictionary<string, string[]> GetHeaders(this HttpResponseMessage responseMessage)
    {
        return responseMessage.Headers
            .Concat(responseMessage.Content.Headers)
            .ToDictionary(k => k.Key, v => v.Value.ToArray());
    }

    public static Dictionary<string, string[]> GetContentHeaders(this HttpResponseMessage responseMessage)
    { return responseMessage.Content.Headers.ToDictionary(k => k.Key, v => v.Value.ToArray()); }

    public static string[] GetHeader(this HttpResponseMessage responseMessage, string name)
    {
        if(responseMessage.Headers.TryGetValues(name, out var values) ||
            responseMessage.Content.Headers.TryGetValues(name, out values))
            return values.ToArray();

        return [];
    }

    public static bool TryGetHeader(this HttpResponseMessage responseMessage, string name, out string[] values)
    {
        if(responseMessage.Headers.TryGetValues(name, out var headerValues) ||
            responseMessage.Content.Headers.TryGetValues(name, out headerValues))
        {
            values = headerValues.ToArray();
            return true;
        }

        values = [];
        return false;
    }

    public static string? GetFirstHeaderValue(this HttpResponseMessage responseMessage, string name) { return responseMessage.GetHeader(name).FirstOrDefault(); }

    public static bool TryGetFirstHeaderValue(this HttpResponseMessage responseMessage, string name, out string value)
    {
        var values = responseMessage.GetHeader(name);
        if(values.Length == 0)
        {
            value = string.Empty;
            return false;
        }
        value = values.First();
        return true;
    }

    public static void EnsureSuccessStatusCode(this HttpResponseMessage responseMessage)
    {
        if(!responseMessage.IsSuccess())
            throw new HttpRequestException(
                $"HTTP request failed with status code {responseMessage.StatusCode}: {responseMessage.ReasonPhrase}");
    }

    public static async Task EnsureSuccessStatusCodeWithContentAsync(this HttpResponseMessage responseMessage, CancellationToken ct = default)
    {
        if(!responseMessage.IsSuccess())
        {
            var content = await responseMessage
                .AsStringAsync(ct)
                .ConfigureAwait(false);

            throw new HttpRequestException(
                $"HTTP request failed with status code {responseMessage.StatusCode}: {responseMessage.ReasonPhrase}. Response content: {content}");
        }
    }
}
