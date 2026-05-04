Imports System.IO.Compression
Imports System.Security.Cryptography
Imports System.Text.RegularExpressions
Imports System.Threading
Imports System.Threading.Tasks
Imports Newtonsoft.Json.Linq

''' <summary>
''' 云端同步引擎。
''' 负责全量包下载、增量清单对比、原子替换，以及白名单目录保护。
''' </summary>
Friend Module ModCloudSync

#Region "常量"

    ''' <summary>云端管控的同步目录</summary>
    Public ReadOnly SyncDirs As String() = {"mods", "config", "scripts"}

    ''' <summary>受保护的路径（玩家数据，严禁触碰）</summary>
    Public ReadOnly ProtectedPaths As String() = {
        "saves", "screenshots", "options.txt", "optionsof.txt",
        "journeymap", "voxelmap", "xaeroworldmap", "XaeroWaypoints",
        "resourcepacks", "shaderpacks", "servers.dat", "servers.dat_old"
    }

    ''' <summary>PCL.ini 文件名</summary>
    Private Const PclIniName As String = "PCL.ini"

    ''' <summary>增量同步最大并发下载数</summary>
    Private Const MaxConcurrentSyncDownloads As Integer = 8

    ''' <summary>临时同步目录后缀</summary>
    Private Const TempSyncSuffix As String = "_cloudsync_tmp"

#End Region

#Region "数据结构"

    ''' <summary>清单中的单个文件条目</summary>
    Private Class ManifestFile
        Public Property Hash As String
        Public Property Size As Long
    End Class

    ''' <summary>版本清单</summary>
    Private Class Manifest
        Public Property VersionID As String
        Public Property Timestamp As Long
        Public Property Files As Dictionary(Of String, ManifestFile)
    End Class

    ''' <summary>PCL.ini 解析结果</summary>
    Public Class PclIniInfo
        Public Property Version As String
        Public Property Name As String
        Public Property Info As String
        Public Property SyncUrl As String
        Public Property SyncPolicy As String
    End Class

#End Region

#Region "PCL.ini 解析"

    ''' <summary>
    ''' 解析 PCL.ini 文件内容。
    ''' </summary>
    Public Function ParsePclIni(content As String) As PclIniInfo
        Dim info As New PclIniInfo()
        If String.IsNullOrEmpty(content) Then Return info

        For Each line In content.Split({vbCrLf, vbLf, vbCr}, StringSplitOptions.RemoveEmptyEntries)
            Dim parts As String() = line.Split(":"c, 2)
            If parts.Length < 2 Then Continue For
            Dim key As String = parts(0).Trim().ToLowerInvariant()
            Dim value As String = parts(1).Trim()

            Select Case key
                Case "version" : info.Version = value
                Case "name" : info.Name = value
                Case "info" : info.Info = value
                Case "syncurl" : info.SyncUrl = value
                Case "syncpolicy" : info.SyncPolicy = value
            End Select
        Next
        Return info
    End Function

#End Region

#Region "全量包同步"

    ''' <summary>
    ''' 下载云端实例的压缩包，并返回临时文件路径。
    ''' </summary>
    ''' <param name="packUrl">云端整合包地址</param>
    ''' <returns>成功返回临时压缩包路径，失败返回空字符串</returns>
    Public Function DownloadCloudPackZip(packUrl As String) As String
        If String.IsNullOrEmpty(packUrl) Then Return ""

        packUrl = NegotiateCloudSyncUrl(packUrl)
        If String.IsNullOrEmpty(packUrl) Then Return ""

        Log("[CloudSync] 开始下载云端整合包，来源：" & packUrl)

        Dim tempZip As String = PathTemp & "cloud_pack_" & GetUuid() & ".zip"

        Try
            Directory.CreateDirectory(PathTemp)
            If File.Exists(tempZip) Then File.Delete(tempZip)
            ModNet.NetDownloadByLoader(packUrl, tempZip)

            If Not File.Exists(tempZip) Then
                Log("[CloudSync] 云端整合包下载失败")
                Return ""
            End If

            Log("[CloudSync] 云端整合包下载完成：" & tempZip)
            Return tempZip
        Catch ex As Exception
            Log(ex, "[CloudSync] 云端整合包下载发生异常", LogLevel.Feedback)
            Try
                If File.Exists(tempZip) Then File.Delete(tempZip)
            Catch : End Try
            Return ""
        End Try
    End Function

    ''' <summary>
    ''' 下载全量包并执行原子同步。
    ''' </summary>
    ''' <param name="packUrl">全量包下载地址</param>
    ''' <param name="instanceDir">目标 Minecraft 实例目录（{McFolder} 根）</param>
    ''' <returns>同步成功返回 True</returns>
    Public Function DownloadAndSyncFullPack(packUrl As String, instanceDir As String) As Boolean
        If String.IsNullOrEmpty(packUrl) OrElse String.IsNullOrEmpty(instanceDir) Then Return False
        
        packUrl = NegotiateCloudSyncUrl(packUrl)
        If String.IsNullOrEmpty(packUrl) Then Return False
        
        If Not Directory.Exists(instanceDir) Then
            Log("[CloudSync] 目标实例目录不存在：" & instanceDir)
            Return False
        End If

        Log("[CloudSync] 开始全量同步，来源：" & packUrl)

        Dim tempZip As String = PathTemp & "cloud_fullpack_" & GetUuid() & ".zip"
        Dim tempExtract As String = instanceDir & TempSyncSuffix

        Try
            ' 1. 下载全量包
            Directory.CreateDirectory(PathTemp)
            If File.Exists(tempZip) Then File.Delete(tempZip)
            ModNet.NetDownloadByLoader(packUrl, tempZip)
            If Not File.Exists(tempZip) Then
                Log("[CloudSync] 全量包下载失败")
                Return False
            End If

            ' 2. 解压到临时目录
            If Directory.Exists(tempExtract) Then Directory.Delete(tempExtract, True)
            Using archive As System.IO.Compression.ZipArchive = System.IO.Compression.ZipFile.OpenRead(tempZip)
                For Each entry As System.IO.Compression.ZipArchiveEntry In archive.Entries
                    Dim fullDestPath As String = System.IO.Path.GetFullPath(System.IO.Path.Combine(tempExtract, entry.FullName))
                    Dim fullExtractPath As String = System.IO.Path.GetFullPath(tempExtract & System.IO.Path.DirectorySeparatorChar)
                    If Not fullDestPath.StartsWith(fullExtractPath, StringComparison.OrdinalIgnoreCase) Then
                        Throw New System.IO.IOException("Zip entry trying to traverse outside destination directory: " & entry.FullName)
                    End If
                    If Not entry.FullName.EndsWith("/") AndAlso Not entry.FullName.EndsWith("\") Then
                        System.IO.Directory.CreateDirectory(System.IO.Path.GetDirectoryName(fullDestPath))
                        entry.ExtractToFile(fullDestPath, True)
                    End If
                Next
            End Using
            File.Delete(tempZip)

            ' 3. 解析 PCL.ini
            Dim pclIniPath As String = System.IO.Path.Combine(tempExtract, PclIniName)
            Dim pclInfo As PclIniInfo = Nothing
            If File.Exists(pclIniPath) Then
                pclInfo = ParsePclIni(ReadFile(pclIniPath))
                Log($"[CloudSync] 云端版本：{pclInfo.Name}（{pclInfo.Version}）")
                File.Delete(pclIniPath)
            End If

            ' 4. 原子替换同步目录
            Dim success As Boolean = AtomicReplaceSyncDirs(tempExtract, instanceDir)

            ' 5. 保存版本信息
            If success AndAlso pclInfo IsNot Nothing Then
                CloudLastSyncedVersion = pclInfo.Version
                If Not String.IsNullOrEmpty(pclInfo.SyncUrl) Then
                    CloudManifestUrl = pclInfo.SyncUrl
                End If
            End If

            ' 6. 清理临时目录
            If Directory.Exists(tempExtract) Then Directory.Delete(tempExtract, True)

            Log("[CloudSync] 全量同步完成")
            Return success

        Catch ex As Exception
            Log(ex, "[CloudSync] 全量同步失败", LogLevel.Feedback)
            ' 清理残留
            Try
                If File.Exists(tempZip) Then File.Delete(tempZip)
                If Directory.Exists(tempExtract) Then Directory.Delete(tempExtract, True)
            Catch : End Try
            Return False
        End Try
    End Function

#End Region

#Region "增量同步"

    ''' <summary>
    ''' 根据服务端 Manifest 执行增量同步。
    ''' </summary>
    ''' <param name="manifestUrl">Manifest API 地址（来自 PCL.ini 的 SyncUrl）</param>
    ''' <param name="instanceDir">目标 Minecraft 实例目录</param>
    ''' <returns>同步成功返回 True</returns>
    Public Function IncrementalSync(manifestUrl As String, instanceDir As String) As Boolean
        If String.IsNullOrEmpty(manifestUrl) OrElse String.IsNullOrEmpty(instanceDir) Then Return False

        manifestUrl = NegotiateCloudSyncUrl(manifestUrl)
        If String.IsNullOrEmpty(manifestUrl) Then Return False

        Log("[CloudSync] 开始增量同步...")

        Dim tempDir As String = instanceDir & TempSyncSuffix

        Try
            ' 1. 获取 Manifest
            Dim manifest As Manifest = FetchManifest(manifestUrl)
            If manifest Is Nothing OrElse manifest.Files Is Nothing OrElse manifest.Files.Count = 0 Then
                Log("[CloudSync] Manifest 为空，跳过增量同步")
                Return False
            End If

            Log($"[CloudSync] Manifest 版本：{manifest.VersionID}，共 {manifest.Files.Count} 个文件")

            ' 2. 创建临时目录
            If Directory.Exists(tempDir) Then Directory.Delete(tempDir, True)
            Directory.CreateDirectory(tempDir)

            ' 3. 对比 hash，仅下载变更文件（并行）
            Dim baseUrl As String = GetBaseUrlFromManifest(manifestUrl)
            Dim changedCount As Integer = 0

            Dim parallelOpts As New ParallelOptions With {
                .MaxDegreeOfParallelism = MaxConcurrentSyncDownloads
            }
            Parallel.ForEach(manifest.Files, parallelOpts,
                Sub(kvp)
                    Dim relPath As String = kvp.Key
                    Dim mf As ManifestFile = kvp.Value
                    Dim localPath As String = System.IO.Path.Combine(instanceDir, relPath)

                    ' 计算本地文件 SHA256
                    Dim localHash As String = Nothing
                    If File.Exists(localPath) Then
                        localHash = ComputeFileSHA256(localPath)
                    End If

                    ' Hash 一致则跳过
                    If String.Equals(localHash, mf.Hash, StringComparison.OrdinalIgnoreCase) Then
                        Return
                    End If

                    ' 下载差异文件
                    Dim tempPath As String = System.IO.Path.Combine(tempDir, relPath)
                    Directory.CreateDirectory(System.IO.Path.GetDirectoryName(tempPath))

                    Dim fileUrl As String = $"{baseUrl}/{relPath}"
                    Try
                        ModNet.NetDownloadByLoader(fileUrl, tempPath)
                        Interlocked.Increment(changedCount)
                    Catch ex As Exception
                        Log("[CloudSync] 下载文件失败：" & relPath & "：" & ex.Message)
                    End Try
                End Sub)

            Log($"[CloudSync] 共 {changedCount} 个文件需要更新")

            ' 4. 清理本地冗余文件（在受控目录中，但不在 Manifest 中的文件）
            For Each syncDir In SyncDirs
                Dim localSyncPath As String = System.IO.Path.Combine(instanceDir, syncDir)
                If Not Directory.Exists(localSyncPath) Then Continue For

                Dim localFiles As String() = Directory.GetFiles(localSyncPath, "*.*", SearchOption.AllDirectories)
                For Each localFile In localFiles
                    ' 计算相对路径
                    Dim relPath As String = localFile.Substring(instanceDir.Length).TrimStart("\"c, "/"c).Replace("\"c, "/"c)
                    If IsProtected(relPath) Then Continue For

                    If Not manifest.Files.ContainsKey(relPath) Then
                        Try
                            File.Delete(localFile)
                            Log($"[CloudSync] 删除云端已移除的冗余文件：{relPath}")
                        Catch ex As Exception
                        End Try
                    End If
                Next
            Next

            ' 5. 将新下载的文件移动到正确位置
            If Directory.Exists(tempDir) Then
                Dim tempFiles As String() = Directory.GetFiles(tempDir, "*.*", SearchOption.AllDirectories)
                For Each tempFile In tempFiles
                    Dim relPath As String = tempFile.Substring(tempDir.Length).TrimStart("\"c, "/"c).Replace("\"c, "/"c)
                    Dim dstFile As String = System.IO.Path.Combine(instanceDir, relPath)
                    
                    Directory.CreateDirectory(System.IO.Path.GetDirectoryName(dstFile))
                    If File.Exists(dstFile) Then File.Delete(dstFile)
                    File.Move(tempFile, dstFile)
                Next
            End If

            ' 6. 保存版本
            CloudLastSyncedVersion = manifest.VersionID

            ' 7. 清理
            If Directory.Exists(tempDir) Then Directory.Delete(tempDir, True)

            Return True

        Catch ex As Exception
            Log(ex, "[CloudSync] 增量同步失败", LogLevel.Feedback)
            Try
                If Directory.Exists(tempDir) Then Directory.Delete(tempDir, True)
            Catch : End Try
            Return False
        End Try
    End Function

    ''' <summary>
    ''' 获取远程 Manifest JSON。
    ''' </summary>
    Private Function FetchManifest(manifestUrl As String, Optional depth As Integer = 0) As Manifest
        Try
            Dim json As JObject = ModNet.NetGetCodeByRequestOnce(manifestUrl, IsJson:=True, Timeout:=30000)

            If json("success") IsNot Nothing Then
                json = json("data")
            End If

            ' 容错：旧版 PCL.ini 的 SyncUrl 错误指向 instance.json（元数据），
            ' 元数据不含 files 字段但包含 active_version，检测到此情况则回退到正确的 Manifest API
            If json("files") Is Nothing AndAlso json("active_version") IsNot Nothing Then
                If depth > 0 Then Throw New Exception("Manifest URL recursion limit exceeded")
                Log("[CloudSync] 检测到元数据端点，尝试回退到 Manifest API...")
                Dim uri As New Uri(manifestUrl)
                Dim pathParts As String() = uri.AbsolutePath.Trim("/"c).Split("/"c)
                If pathParts.Length >= 2 AndAlso pathParts(0) = "sync" Then
                    Dim instanceId As String = pathParts(1)
                    Dim correctedUrl As String = $"{uri.Scheme}://{uri.Host}:{uri.Port}/api/v1/sync/info?id={instanceId}"
                    Log("[CloudSync] 回退 URL：" & correctedUrl)
                    Return FetchManifest(correctedUrl, depth + 1)
                End If
            End If

            Dim manifest As New Manifest With {
                .VersionID = If(json("version_id") IsNot Nothing, json("version_id").Value(Of String)(), ""),
                .Timestamp = If(json("timestamp") IsNot Nothing, json("timestamp").Value(Of Long)(), 0),
                .Files = New Dictionary(Of String, ManifestFile)()
            }

            Dim filesNode As JToken = json("files")
            If filesNode IsNot Nothing Then
                For Each fileProp As JProperty In filesNode.Children(Of JProperty)()
                    Dim fileObj As JObject = fileProp.Value
                    manifest.Files(fileProp.Name) = New ManifestFile With {
                        .Hash = If(fileObj("hash") IsNot Nothing, fileObj("hash").Value(Of String)(), ""),
                        .Size = If(fileObj("size") IsNot Nothing, fileObj("size").Value(Of Long)(), 0)
                    }
                Next
            End If

            Return manifest
        Catch ex As Exception
            Log("[CloudSync] 获取 Manifest 失败：" & ex.Message)
            Return Nothing
        End Try
    End Function

    ''' <summary>
    ''' 从 Manifest URL 提取 base URL（去掉文件名部分）。
    ''' 例如 http://ip:55001/sync/{id}/files → 用于拼接文件下载 URL
    ''' </summary>
    Private Function GetBaseUrlFromManifest(manifestUrl As String) As String
        ' Manifest URL 格式: http://ip:55000/api/v1/sync/manifest?id=xxx
        ' 文件下载 URL:    http://ip:55001/sync/{id}/files/{path}
        ' 从 manifestUrl 中提取 id，构造 files 路径

        Dim uri As New Uri(manifestUrl)
        Dim query As String = uri.Query
        Dim idMatch As Match = System.Text.RegularExpressions.Regex.Match(query, "id=([^&]+)")
        If idMatch.Success Then
            Dim instanceId As String = idMatch.Groups(1).Value
            Return $"{uri.Scheme}://{uri.Host}:55001/sync/{instanceId}/files"
        End If

        ' Fallback: 去掉路径末尾的文件名
        Return manifestUrl.Substring(0, manifestUrl.LastIndexOf("/"c))
    End Function

#End Region

#Region "原子替换"

    ''' <summary>
    ''' 将临时目录中的同步管控文件原子替换到目标目录。
    ''' 仅替换 SyncDirs 中的目录，ProtectedPaths 中的路径不会被触碰。
    ''' </summary>
    Private Function AtomicReplaceSyncDirs(tempDir As String, targetDir As String) As Boolean
        Try
            For Each syncDir In SyncDirs
                Dim srcPath As String = System.IO.Path.Combine(tempDir, syncDir)
                Dim dstPath As String = System.IO.Path.Combine(targetDir, syncDir)

                If Directory.Exists(srcPath) Then
                    ' 删除旧目录，以新目录替换
                    If Directory.Exists(dstPath) Then
                        Directory.Delete(dstPath, True)
                    End If
                    Directory.Move(srcPath, dstPath)
                    Log($"[CloudSync] 已更新目录：{syncDir}")
                End If
            Next

            ' 同步根目录下的单个文件（如 PCL.ini 已被删除，此处处理其他配置文件）
            For Each filePath In Directory.GetFiles(tempDir, "*.*", SearchOption.TopDirectoryOnly)
                Dim fileName As String = System.IO.Path.GetFileName(filePath)
                Dim dstFile As String = System.IO.Path.Combine(targetDir, fileName)

                ' 跳过受保护的文件
                If IsProtected(fileName) Then Continue For

                ' SHA256 校验后替换
                If File.Exists(dstFile) Then
                    Dim srcHash As String = ComputeFileSHA256(filePath)
                    Dim dstHash As String = ComputeFileSHA256(dstFile)
                    If String.Equals(srcHash, dstHash, StringComparison.OrdinalIgnoreCase) Then
                        File.Delete(filePath)
                        Continue For
                    End If
                    File.Delete(dstFile)
                End If
                File.Move(filePath, dstFile)
            Next

            Return True
        Catch ex As Exception
            Log(ex, "[CloudSync] 原子替换失败", LogLevel.Feedback)
            Return False
        End Try
    End Function

    ''' <summary>
    ''' 判断路径是否在受保护列表中。
    ''' </summary>
    Private Function IsProtected(relativePath As String) As Boolean
        Dim normalized As String = relativePath.Replace("/"c, "\"c).TrimStart("\"c).ToLowerInvariant()
        For Each protectedPath In ProtectedPaths
            If normalized.StartsWith(protectedPath.ToLowerInvariant()) Then
                Return True
            End If
        Next
        Return False
    End Function

#End Region

#Region "工具方法"

    ''' <summary>上次同步的版本号</summary>
    Public CloudLastSyncedVersion As String = ""

    ''' <summary>Manifest 增量同步地址</summary>
    Public CloudManifestUrl As String = ""

    ''' <summary>
    ''' 计算文件的 SHA256 哈希值（hex 字符串）。
    ''' </summary>
    Public Function ComputeFileSHA256(filePath As String) As String
        Try
            Using sha As SHA256 = SHA256.Create()
                Using stream As FileStream = File.OpenRead(filePath)
                    Dim hashBytes As Byte() = sha.ComputeHash(stream)
                    Return BitConverter.ToString(hashBytes).Replace("-", "").ToLowerInvariant()
                End Using
            End Using
        Catch
            Return Nothing
        End Try
    End Function

#End Region

    ''' <summary>
    ''' 协商 HTTP/HTTPS 协议及 CDN 回退，返回最终同步基地址
    ''' </summary>
    Public Function NegotiateCloudSyncUrl(targetUrl As String) As String
        If targetUrl.StartsWith("http://", StringComparison.OrdinalIgnoreCase) OrElse targetUrl.StartsWith("https://", StringComparison.OrdinalIgnoreCase) Then Return targetUrl
        
        Dim httpsUrl = $"https://{targetUrl}"
        Dim httpUrl = $"http://{targetUrl}"
        Try
            Log($"[CloudSync] 尝试 HTTPS 连接：{httpsUrl}")
            ' 仅发起握手，若返回 HTTP 错误（如 401 授权失败）也算握手成功
            ModNet.NetGetCodeByRequestOnce(httpsUrl, Timeout:=3000)
            Return httpsUrl
        Catch ex As ModNet.HttpWebException
            ' HTTP 状态码错误说明 HTTPS TLS 握手成功
            Return httpsUrl
        Catch ex As Exception
            Log($"[CloudSync] HTTPS 握手失败，准备降级 HTTP：{ex.Message}")
            Dim allowHttp As Boolean = False
            Dim waitHandle As New Threading.ManualResetEvent(False)
            RunInUi(Sub()
                Try
                    If MyMsgBox($"同步服务器 {targetUrl} 未配置 HTTPS 或证书无效。{vbCrLf}是否允许降级使用不安全的 HTTP 传输进行同步？", "协议安全警告", "允许 HTTP", "取消同步", IsWarn:=True) = 1 Then
                        allowHttp = True
                    End If
                Finally
                    waitHandle.Set()
                End Try
            End Sub)
            waitHandle.WaitOne()
            
            If allowHttp Then Return httpUrl
            Return ""
        End Try
    End Function

End Module
