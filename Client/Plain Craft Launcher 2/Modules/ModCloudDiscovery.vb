Imports System.Net.Sockets
Imports Ae.Dns.Protocol
Imports Ae.Dns.Protocol.Enums
Imports Ae.Dns.Protocol.Records
Imports Newtonsoft.Json.Linq
Imports PCL.Core.IO.Net.Dns

''' <summary>
''' CloudAbroad SRV 域名解析与云端实例发现模块。
''' </summary>
Friend Module ModCloudDiscovery

    ''' <summary>SRV 记录中的服务名</summary>
    Private Const SrvServiceName As String = "_pclcesyn._tcp"

    ''' <summary>云端实例信息</summary>
    Public Class CloudInstance
        Public Property Id As String
        Public Property Name As String
        Public Property Version As String
        Public Property Description As String
        Public Property PlayerCount As Integer
        Public Property PackUrl As String
        Public Property IconUrl As String
        Public Property VersionId As String
        Public Property ModList As List(Of String)
    End Class

    ''' <summary>
    ''' 通过 SRV DNS 解析获取服务端地址。
    ''' 查询 _pclcesyn._tcp.{domain}，返回服务端基地址（含业务端口）。
    ''' </summary>
    ''' <param name="domain">用户输入的域名</param>
    ''' <param name="errorMessage">失败时的错误描述</param>
    ''' <returns>服务端基地址（如 http://1.2.3.4:55001，端口由 SRV 记录决定），失败返回空字符串</returns>
    Public Function ResolveCloudServer(domain As String, ByRef errorMessage As String) As String
        If String.IsNullOrEmpty(domain) Then
            errorMessage = "请输入服务器域名"
            Return ""
        End If

        domain = domain.Trim()
        ' 去掉可能误输入的 scheme
        If domain.StartsWith("http://", StringComparison.OrdinalIgnoreCase) Then
            domain = domain.Substring(7)
        ElseIf domain.StartsWith("https://", StringComparison.OrdinalIgnoreCase) Then
            domain = domain.Substring(8)
        End If
        ' 去掉尾部斜杠
        domain = domain.TrimEnd("/"c)

        Log($"[CloudDiscovery] 正在解析 SRV 记录：{SrvServiceName}.{domain}")

        Try
            Dim srvName As String = $"{SrvServiceName}.{domain}"
            Dim dnsTask As Task(Of DnsMessage) = DnsQuery.Instance.QueryAsync(srvName, DnsQueryType.SRV, CancellationToken.None)
            dnsTask.Wait(10000)

            If dnsTask.IsCompleted AndAlso dnsTask.Result IsNot Nothing AndAlso dnsTask.Result.Answers.Count > 0 Then
                Log($"[CloudDiscovery] 获得 {dnsTask.Result.Answers.Count} 条记录")
                ' 解析 SRV 记录获取第一个可用地址
                For Each answer In dnsTask.Result.Answers
                    Log($"[CloudDiscovery] 检查记录: Type={answer.Type}, Resource={If(answer.Resource IsNot Nothing, answer.Resource.GetType().Name, "null")}")
                    Dim unknownRes As DnsUnknownResource = TryCast(answer.Resource, DnsUnknownResource)
                    If unknownRes IsNot Nothing Then
                        Dim srvResource As New DnsSrvResource()
                        Dim offset As Integer = 0
                        Try
                            srvResource.ReadBytes(unknownRes.Raw, offset, unknownRes.Raw.Length)
                            Log($"[CloudDiscovery] SRV记录解析: Target={srvResource.Target}, Port={srvResource.Port}")
                            
                            If srvResource.Port > 0 AndAlso Not String.IsNullOrEmpty(srvResource.Target) AndAlso srvResource.Target <> "." Then
                                Dim target As String = srvResource.Target.TrimEnd("."c)
                                Dim controlUrl As String = NegotiateCloudServerUrl($"{target}:{srvResource.Port}")
                                If Not String.IsNullOrEmpty(controlUrl) Then
                                    Log($"[CloudDiscovery] SRV 解析成功：{srvResource.Target}:{srvResource.Port} -> {controlUrl}")
                                    Return controlUrl
                                End If
                            End If
                        Catch ex As Exception
                            Log($"[CloudDiscovery] 解析 SRV 记录失败: {ex.Message}")
                        End Try
                    End If
                Next

                errorMessage = "SRV 记录中未找到有效的服务端地址"
                Log("[CloudDiscovery] " & errorMessage)
                Return ""
            End If

            errorMessage = "无法解析服务器地址，请检查域名是否正确"
            Log("[CloudDiscovery] " & errorMessage)
            Return ""

        Catch ex As Exception
            errorMessage = "DNS 解析失败：" & ex.Message
            Log("[CloudDiscovery] " & errorMessage)
            Return ""
        End Try
    End Function

    ''' <summary>
    ''' 从服务端获取云端实例列表。
    ''' </summary>
    ''' <param name="serverUrl">服务端基地址</param>
    ''' <param name="errorMessage">失败时的错误描述</param>
    ''' <returns>实例列表，失败返回空列表</returns>
    Public Function FetchInstances(serverUrl As String, ByRef errorMessage As String) As List(Of CloudInstance)
        Dim instances As New List(Of CloudInstance)()

        If String.IsNullOrEmpty(serverUrl) Then
            errorMessage = "未配置服务器地址"
            Return instances
        End If

        Log($"[CloudDiscovery] 获取云端实例列表：{serverUrl}/api/v1/sync/instances")

        Try
            Dim json As JObject = ModNet.NetGetCodeByRequestOnce(
                $"{serverUrl}/api/v1/sync/instances",
                IsJson:=True,
                Timeout:=15000
            )

            ' 服务端返回格式：{success: true, data: {instance_list: [...]}}
            Dim items As JToken = Nothing
            If json("success") IsNot Nothing AndAlso json("success").Value(Of Boolean)() Then
                Dim data = json("data")
                If data IsNot Nothing AndAlso data.Type = JTokenType.Object Then
                    items = data("instance_list")
                ElseIf data IsNot Nothing AndAlso data.Type = JTokenType.Array Then
                    items = data
                End If
            End If

            If items IsNot Nothing AndAlso items.Type = JTokenType.Array Then
                For Each item As JObject In items.Children(Of JObject)()
                    Dim modList As New List(Of String)()
                    Dim modListNode As JToken = item("mod_list")
                    If modListNode IsNot Nothing AndAlso modListNode.Type = JTokenType.Array Then
                        For Each modName As JToken In modListNode
                            modList.Add(modName.Value(Of String)())
                        Next
                    End If
                    instances.Add(New CloudInstance With {
                        .Id = If(item("id") IsNot Nothing, item("id").Value(Of String)(), ""),
                        .Name = If(item("display_name") IsNot Nothing, item("display_name").Value(Of String)(), ""),
                        .Version = If(item("status") IsNot Nothing, item("status").Value(Of String)(), ""),
                        .Description = "",
                        .PlayerCount = 0,
                        .PackUrl = If(item("full_pack_url") IsNot Nothing, item("full_pack_url").Value(Of String)(), ""),
                        .IconUrl = "",
                        .VersionId = If(item("version_id") IsNot Nothing, item("version_id").Value(Of String)(), ""),
                        .ModList = modList
                    })
                Next
            End If

            Log($"[CloudDiscovery] 获取到 {instances.Count} 个云端实例")
            Return instances

        Catch ex As ModNet.HttpWebException
            errorMessage = $"服务器返回错误 ({(CType(ex.StatusCode, Integer))})"
            Log("[CloudDiscovery] " & errorMessage)
            Return instances
        Catch ex As Exception
            errorMessage = "获取实例列表失败：" & ex.Message
            Log("[CloudDiscovery] " & errorMessage)
            Return instances
        End Try
    End Function

    ''' <summary>
    ''' 完整的连接流程：SRV 解析 → 保存地址 → 预授权检测 → 获取实例列表。
    ''' 预授权对话框会通过 waitHandle 阻塞调用线程等待 UI 完成。
    ''' </summary>
    ''' <param name="domain">用户输入的域名</param>
    ''' <param name="instances">输出实例列表</param>
    ''' <param name="errorMessage">失败时的错误描述</param>
    ''' <returns>连接成功返回 True</returns>
    Public Function ConnectAndFetchInstances(domain As String, ByRef instances As List(Of CloudInstance), ByRef errorMessage As String) As Boolean
        instances = New List(Of CloudInstance)()

        ' 1. SRV 解析
        Dim serverUrl As String = ResolveCloudServer(domain, errorMessage)
        If String.IsNullOrEmpty(serverUrl) Then Return False

        ' 2. 保存服务器地址
        ModCloudAuth.CloudServerUrl = serverUrl
        ModCloudAuth.CloudLastDomain = domain

        ' 3. 检查预授权状态（纯 API 调用，可在后台线程执行）
        Dim needsPreAuth As Boolean = ModCloudAuth.CheckPreAuthStatus(serverUrl) AndAlso Not ModCloudAuth.IsBound
        If needsPreAuth Then
            ' 预授权对话框内部已处理 UI 线程切换，直接在当前线程调用即可
            Dim authResult As Boolean = ModCloudAuth.ShowPreAuthDialog(serverUrl)
            If Not authResult Then
                errorMessage = "设备认证未完成"
                Return False
            End If
        End If

        ' 4. 获取实例列表
        instances = FetchInstances(serverUrl, errorMessage)
        Return True
    End Function

    ''' <summary>
    ''' 协商 HTTP/HTTPS 协议并返回最终的服务器基地址
    ''' </summary>
    Private Function NegotiateCloudServerUrl(targetHostPort As String) As String
        Dim httpsUrl = $"https://{targetHostPort}"
        Dim httpUrl = $"http://{targetHostPort}"
        
        Try
            Log($"[CloudDiscovery] 尝试 HTTPS 连接：{httpsUrl}")
            ModNet.NetGetCodeByRequestOnce($"{httpsUrl}/health", Timeout:=3000)
            Log($"[CloudDiscovery] HTTPS 连接成功：{httpsUrl}")
            Return httpsUrl
        Catch ex As Exception
            Log($"[CloudDiscovery] HTTPS 连接失败，尝试协商 HTTP：{ex.Message}")
            
            Dim allowHttp As Boolean = False
            Dim waitHandle As New Threading.ManualResetEvent(False)
            RunInUi(Sub()
                Try
                    If MyMsgBox($"目标服务器 {targetHostPort} 未配置 HTTPS 或证书无效。{vbCrLf}是否允许降级使用不安全的 HTTP 传输？这可能会带来一定的安全风险。", "协议安全警告", "允许 HTTP", "取消连接", IsWarn:=True) = 1 Then
                        allowHttp = True
                    End If
                Finally
                    waitHandle.Set()
                End Try
            End Sub)
            waitHandle.WaitOne()
            
            If allowHttp Then
                Return httpUrl
            Else
                Return ""
            End If
        End Try
    End Function

    ''' <summary>
    ''' 扫描本地 Minecraft 目录，发现所有已安装的整合包。
    ''' </summary>
    ''' <param name="mcFolder">.minecraft 目录路径</param>
    ''' <returns>含有效 mods 目录的本地整合包信息列表</returns>
    Public Function ScanLocalInstances(mcFolder As String) As List(Of ModCloudInfo.LocalPackInfo)
        Dim results As New List(Of ModCloudInfo.LocalPackInfo)()

        If String.IsNullOrEmpty(mcFolder) OrElse Not Directory.Exists(mcFolder) Then
            Return results
        End If

        Dim versionsDir = System.IO.Path.Combine(mcFolder, "versions")
        If Not Directory.Exists(versionsDir) Then Return results

        For Each verDir In Directory.GetDirectories(versionsDir)
            Dim dirName = System.IO.Path.GetFileName(verDir)
            ' 跳过特殊目录和 Vanilla 版本（无 mods 目录的）
            If dirName = "cache" OrElse dirName = "BLClient" OrElse dirName = "PCL" Then Continue For
            Dim modsDir = System.IO.Path.Combine(verDir, "mods")
            If Not Directory.Exists(modsDir) Then Continue For
            Dim jarFiles = Directory.GetFiles(modsDir, "*.jar")
            If jarFiles.Length = 0 Then Continue For

            Dim info = ModCloudInfo.GenerateLocalInfo(verDir)
            If info IsNot Nothing Then
                results.Add(New ModCloudInfo.LocalPackInfo With {
                    .InstanceDir = verDir,
                    .InstanceName = dirName,
                    .Info = info
                })
                Dim fileCount As Integer = info.Files.Count
                Log($"[CloudDiscovery] 发现本地整合包: {dirName} ({fileCount} 个文件)")
            End If
        Next

        Log($"[CloudDiscovery] 本地扫描完成，发现 {results.Count} 个整合包")
        Return results
    End Function

#Region "存量实例标记"

    ''' <summary>可标记为云端同步的候选实例</summary>
    Public Class MarkableCandidate
        Public Property LocalPack As ModCloudInfo.LocalPackInfo
        Public Property CloudInfo As ModCloudInfo.CloudInfo
        Public Property Score As Double
        Public Property MatchedMods As Integer
    End Class

    ''' <summary>
    ''' 通过 Setup 持久化"已弹窗提示标记"的服务器标识。
    ''' 防止同一服务器重复弹出标记对话框。
    ''' </summary>
    Public Property MarkOfferedForServer As String
        Get
            Return Setup.Get("CloudMarkOfferedServer")
        End Get
        Set(value As String)
            Setup.Set("CloudMarkOfferedServer", value)
        End Set
    End Property

    ''' <summary>
    ''' 查找可标记为云端同步的本地实例。
    ''' 扫描本地 versions 目录，与云端实例列表进行匹配，返回匹配度最高的候选（最多 3 个）。
    ''' </summary>
    ''' <param name="mcFolder">.minecraft 目录路径</param>
    ''' <param name="cloudInfoList">云端实例的 CloudInfo 列表</param>
    ''' <returns>按匹配度降序排列的候选列表（最多 3 个）</returns>
    Public Function FindMarkableInstances(mcFolder As String, cloudInfoList As List(Of ModCloudInfo.CloudInfo)) As List(Of MarkableCandidate)
        Dim results As New List(Of MarkableCandidate)()

        If cloudInfoList Is Nothing OrElse cloudInfoList.Count = 0 Then Return results

        Dim localInstances = ScanLocalInstances(mcFolder)
        If localInstances.Count = 0 Then Return results

        For Each localPack In localInstances
            ' 跳过已有 cloud_info.json 的实例（已标记过）
            Dim infoPath = System.IO.Path.Combine(localPack.InstanceDir, "PCL", ModCloudInfo.InfoFileName)
            If File.Exists(infoPath) Then Continue For

            ' 查找最佳匹配
            Dim topMatches = ModCloudMatch.FindTopMatches(localPack.Info, cloudInfoList, topN:=1)
            If topMatches Is Nothing OrElse topMatches.Count = 0 Then Continue For

            Dim bestMatch = topMatches(0)
            If bestMatch.Score >= ModCloudMatch.MatchThreshold Then
                results.Add(New MarkableCandidate With {
                    .LocalPack = localPack,
                    .CloudInfo = bestMatch.CloudInfo,
                    .Score = bestMatch.Score,
                    .MatchedMods = bestMatch.MatchedMods
                })
            End If
        Next

        ' 按匹配度降序排序，取前 3
        results.Sort(Function(a, b) b.Score.CompareTo(a.Score))
        If results.Count > 3 Then results = results.Take(3).ToList()

        Log($"[CloudDiscovery] 找到 {results.Count} 个可标记的本地实例")
        Return results
    End Function

    ''' <summary>
    ''' 将本地整合包实例标记为云端可同步。
    ''' 写入 PCL.ini 和 cloud_info.json，之后即可使用增量同步。
    ''' </summary>
    ''' <param name="instanceDir">本地实例目录（如 versions/MyPack）</param>
    ''' <param name="cloudInfo">要绑定的云端实例 CloudInfo</param>
    ''' <returns>标记成功返回 True</returns>
    Public Function MarkAsCloudManaged(instanceDir As String, cloudInfo As ModCloudInfo.CloudInfo) As Boolean
        If String.IsNullOrEmpty(instanceDir) OrElse cloudInfo Is Nothing Then Return False
        If Not Directory.Exists(instanceDir) Then Return False

        Try
            Dim pclDir = System.IO.Path.Combine(instanceDir, "PCL")
            Directory.CreateDirectory(pclDir)

            ' 1. 写入 PCL.ini
            Dim serverUrl = ModCloudAuth.CloudServerUrl
            Dim pclIniPath = System.IO.Path.Combine(pclDir, "PCL.ini")
            Dim pclIniContent = $"Version:CloudAbroad{Environment.NewLine}" &
                               $"Name:{cloudInfo.InstanceId}{Environment.NewLine}" &
                               $"Info:由 CloudAbroad 驱动的高速同步客户端{Environment.NewLine}" &
                               $"SyncUrl:{serverUrl}/api/v1/sync/info?id={cloudInfo.InstanceId}{Environment.NewLine}" &
                               $"SyncPolicy:Enforce{Environment.NewLine}"
            File.WriteAllText(pclIniPath, pclIniContent, System.Text.Encoding.UTF8)
            Log($"[CloudDiscovery] 已写入 PCL.ini：{instanceDir}")

            ' 2. 生成并保存 cloud_info.json
            Dim localInfo = ModCloudInfo.GenerateLocalInfo(instanceDir)
            If localInfo IsNot Nothing Then
                localInfo.InstanceId = cloudInfo.InstanceId
                localInfo.VersionId = cloudInfo.VersionId
                localInfo.McVersion = cloudInfo.McVersion
                localInfo.Modloader = cloudInfo.Modloader
                ModCloudInfo.SaveLocalInfo(instanceDir, localInfo)
                Log($"[CloudDiscovery] 已保存 cloud_info.json：{instanceDir}")
            End If

            Return True
        Catch ex As Exception
            Log(ex, $"[CloudDiscovery] 标记云端同步失败：{instanceDir}")
            Return False
        End Try
    End Function

#End Region

End Module

