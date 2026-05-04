using System;
using System.Net.Http;

namespace PCL.Core.IO.Net.Http.Client.Request;

public static class HttpBasicExtension
{
    public static HttpRequestMessage WithHttpVersionOption(this HttpRequestMessage requestMessage, Version httpVersion)
    {
        requestMessage.Version = httpVersion;
        return requestMessage;
    }
}
