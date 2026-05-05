Imports System.IO
Imports System.Security.Cryptography
Imports System.Text
Imports Newtonsoft.Json
Imports Newtonsoft.Json.Linq

''' <summary>
''' 云端信息库引擎：本地信息生成、持久化、云端信息拉取。
''' 服务端与客户端共用 CloudInfo 数据结构，实现 Git 式差异对比。
''' </summary>
Friend Module ModCloudInfo

    ''' <summary>信息库文件名</summary>
    Public Const InfoFileName As String = "cloud_info.json"

    ''' <summary>仅同步这些目录下的文件</summary>
    Public ReadOnly SyncDirs As String() = {"mods", "config", "scripts", "resourcepacks", "shaderpacks"}

    ''' <summary>云端信息库中的单个文件条目</summary>
    Public Class CloudFileEntry
        <JsonProperty("hash")>
        Public Property Hash As String
        <JsonProperty("size")>
        Public Property Size As Long
    End Class

    ''' <summary>云端信息库（与 Go 端 Manifest 结构对齐）</summary>
    Public Class CloudInfo
        <JsonProperty("instance_id")>
        Public Property InstanceId As String
        <JsonProperty("cloud_name")>
        Public Property CloudName As String
        <JsonProperty("version_id")>
        Public Property VersionId As String
        <JsonProperty("mc_version")>
        Public Property McVersion As String
        <JsonProperty("modloader")>
        Public Property Modloader As String
        <JsonProperty("files")>
        Public Property Files As Dictionary(Of String, CloudFileEntry)
    End Class

    ''' <summary>本地整合包摘要</summary>
    Public Class LocalPackInfo
        Public Property InstanceDir As String
        Public Property InstanceName As String
        Public Property Info As CloudInfo
    End Class

    ''' <summary>
    ''' 扫描本地 instance 目录，生成 CloudInfo 信息库。
    ''' </summary>
    ''' <param name="instanceDir">整合包目录（如 versions/MyPack）</param>
    ''' <returns>生成的 CloudInfo，若 mods 目录为空则返回 Nothing</returns>
    Public Function GenerateLocalInfo(instanceDir As String) As CloudInfo
        Dim modsDir = Path.Combine(instanceDir, "mods")
        If Not Directory.Exists(modsDir) Then Return Nothing
        Dim modFiles = Directory.GetFiles(modsDir, "*.jar")
        If modFiles.Length = 0 Then Return Nothing

        Dim info As New CloudInfo With {
            .Files = New Dictionary(Of String, CloudFileEntry)
        }

        For Each syncDir In SyncDirs
            Dim dir = Path.Combine(instanceDir, syncDir)
            If Not Directory.Exists(dir) Then Continue For

            For Each file In Directory.GetFiles(dir, "*", SearchOption.AllDirectories)
                Dim relPath = Path.GetRelativePath(instanceDir, file).Replace("\"c, "/"c)
                Dim hash = ComputeFileSHA256(file)
                Dim size = New FileInfo(file).Length
                info.Files(relPath) = New CloudFileEntry With {.Hash = hash, .Size = size}
            Next
        Next

        Return info
    End Function

    ''' <summary>
    ''' 将 CloudInfo 保存到 PCL/cloud_info.json。
    ''' </summary>
    Public Sub SaveLocalInfo(instanceDir As String, info As CloudInfo)
        Dim pclDir = Path.Combine(instanceDir, "PCL")
        Directory.CreateDirectory(pclDir)
        Dim jsonStr = JsonConvert.SerializeObject(info, Formatting.Indented)
        File.WriteAllText(Path.Combine(pclDir, InfoFileName), jsonStr, Encoding.UTF8)
    End Sub

    ''' <summary>
    ''' 从 PCL/cloud_info.json 读取已保存的信息库。
    ''' </summary>
    Public Function LoadLocalInfo(instanceDir As String) As CloudInfo
        Dim path As String = System.IO.Path.Combine(instanceDir, "PCL", InfoFileName)
        If Not File.Exists(path) Then Return Nothing
        Dim jsonStr = File.ReadAllText(path, Encoding.UTF8)
        Return JsonConvert.DeserializeObject(Of CloudInfo)(jsonStr)
    End Function

    ''' <summary>
    ''' 从服务端拉取指定实例的信息库。
    ''' </summary>
    ''' <param name="serverUrl">服务端基地址</param>
    ''' <param name="instanceId">云端实例 ID</param>
    ''' <returns>云端 CloudInfo，失败返回 Nothing</returns>
    Public Function FetchRemoteInfo(serverUrl As String, instanceId As String) As CloudInfo
        Try
            Dim url = $"{serverUrl}/api/v1/sync/info?id={Uri.EscapeDataString(instanceId)}"
            Dim result As String = ModNet.NetGetCodeByRequestOnce(url, IsJson:=True, Timeout:=15000)
            Dim json As JObject = JObject.Parse(result)

            If json("success") IsNot Nothing AndAlso json("success").Value(Of Boolean)() Then
                Dim data = json("data")
                If data IsNot Nothing Then
                    Dim info As New CloudInfo With {
                        .InstanceId = If(data("instance_id") IsNot Nothing, data("instance_id").Value(Of String)(), ""),
                        .VersionId = If(data("version_id") IsNot Nothing, data("version_id").Value(Of String)(), ""),
                        .McVersion = If(data("mc_version") IsNot Nothing, data("mc_version").Value(Of String)(), ""),
                        .Modloader = If(data("modloader") IsNot Nothing, data("modloader").Value(Of String)(), ""),
                        .Files = New Dictionary(Of String, CloudFileEntry)
                    }
                    Dim files = data("files")
                    If files IsNot Nothing Then
                        For Each kv In CType(files, JObject).Properties()
                            Dim entry = kv.Value
                            info.Files(kv.Name) = New CloudFileEntry With {
                                .Hash = If(entry("hash") IsNot Nothing, entry("hash").Value(Of String)(), ""),
                                .Size = If(entry("size") IsNot Nothing, entry("size").Value(Of Long)(), 0)
                            }
                        Next
                    End If
                    Return info
                End If
            End If
        Catch ex As Exception
            Log("[CloudInfo] 拉取云端信息库失败: " & ex.Message)
        End Try
        Return Nothing
    End Function

    ''' <summary>
    ''' 计算文件的 SHA256 哈希值并返回十六进制字符串。失败返回空字符串。
    ''' </summary>
    Public Function ComputeFileSHA256(filePath As String) As String
        Try
            Using sha = SHA256.Create()
                Using fs = File.OpenRead(filePath)
                    Dim hash = sha.ComputeHash(fs)
                    Return BitConverter.ToString(hash).Replace("-", "").ToLowerInvariant()
                End Using
            End Using
        Catch
            Return ""
        End Try
    End Function

End Module
