using PCL.Core.App;
using PCL.Core.Logging;
using PCL.Core.Utils.Exts;
using System;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;

namespace PCL.Core.IO.Net.Http.Client.Request;

public static class HttpSenderExtension
{
    public static async Task<HttpResponseMessage> SendAsync(
        this HttpRequestMessage requestMessage,
        HttpClient? httpClient = null,
        bool addMetedata = true,
        bool enableLogging = true,
        HttpCompletionOption httpCompletionOption = HttpCompletionOption.ResponseContentRead,
        int retryTimes = 3,
        CancellationToken cancellationToken = default)
    {
        using var request = requestMessage;
        httpClient ??= NetworkService.GetClient();

        if(addMetedata)
        {
            request
                .WithHeader("User-Agent", $"PCL-Community/PCL2-CE/{Basics.VersionName} (pclc.cc)")
                .WithHeader("Referer", $"https://{Basics.VersionCode}.ce.open.pcl2.server/");
        }

        var requestId = Guid.NewGuid().ToString();
        if (enableLogging)
            LogWrapper.Info(
                "Request",
                $"Send request to {request.RequestUri} (method = {request.Method}, id = {requestId})");

        var resp = await NetworkService.GetRetryPolicy(retryTimes)
            .ExecuteAsync(
                async token =>
                {
                    if (enableLogging)
                        LogWrapper.Debug("Request", $"Try attempt (id = {requestId})");
                    try
                    {
                        var currentRequest = await request.CloneAsync().ConfigureAwait(false);
                        int redirectCount = 0;
                        while (true)
                        {
                            var response = await httpClient
                                .SendAsync(currentRequest, httpCompletionOption, token)
                                .ConfigureAwait(false);

                            if (response.StatusCode == System.Net.HttpStatusCode.MovedPermanently ||
                                response.StatusCode == System.Net.HttpStatusCode.Redirect ||
                                response.StatusCode == System.Net.HttpStatusCode.Found ||
                                response.StatusCode == System.Net.HttpStatusCode.SeeOther ||
                                response.StatusCode == System.Net.HttpStatusCode.TemporaryRedirect ||
                                (int)response.StatusCode == 308)
                            {
                                redirectCount++;
                                if (redirectCount > 5)
                                {
                                    response.Dispose();
                                    throw new Exception("Too many automatic redirections.");
                                }

                                var location = response.Headers.Location;
                                if (location != null)
                                {
                                    if (!location.IsAbsoluteUri)
                                        location = new Uri(currentRequest.RequestUri!, location);

                                    if (currentRequest.RequestUri != null && currentRequest.RequestUri.Scheme == "https" && location.Scheme == "http")
                                    {
                                        response.Dispose();
                                        throw new Exception($"[Security] Prevented automatic downgrade from HTTPS to HTTP via redirect to {location}");
                                    }

                                    currentRequest.Dispose();
                                    currentRequest = await request.CloneAsync().ConfigureAwait(false);
                                    currentRequest.RequestUri = location;
                                    response.Dispose();
                                    continue;
                                }
                            }
                            return response;
                        }
                    }
                    catch(Exception ex)
                    {
                        LogWrapper.Error(ex, "Request", $"Try attempt failed (id = {requestId})");
                        throw;
                    }
                },
                cancellationToken,
                false)
            .ConfigureAwait(false);

        if (enableLogging)
            LogWrapper.Info("Request", $"End request, got http status code {resp.StatusCode} (id = {requestId})");
        return resp;
    }
}
