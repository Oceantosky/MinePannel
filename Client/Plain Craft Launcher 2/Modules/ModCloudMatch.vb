Imports System.Text.RegularExpressions

''' <summary>
''' 云端拟合度匹配引擎。
''' 通过 mod 文件名匹配（忽略版本号后缀）计算本地整合包与云端实例的相似度。
''' </summary>
Friend Module ModCloudMatch

    ''' <summary>匹配结果</summary>
    Public Class MatchResult
        Public Property CloudInfo As ModCloudInfo.CloudInfo
        Public Property Score As Double
        Public Property MatchedMods As Integer
        Public Property TotalLocalMods As Integer
        Public Property TotalCloudMods As Integer
    End Class

    ''' <summary>拟合度阈值：低于此分数不推荐匹配</summary>
    Public Const MatchThreshold As Double = 0.3

    ''' <summary>
    ''' 从服务端获取所有云端实例的信息库摘要列表。
    ''' </summary>
    ''' <param name="serverUrl">服务端基地址</param>
    ''' <returns>云端实例的 CloudInfo 列表（仅含元数据，不含完整文件列表）</returns>
    Public Function FetchCloudInfoList(serverUrl As String) As List(Of ModCloudInfo.CloudInfo)
        Dim result As New List(Of ModCloudInfo.CloudInfo)()

        Try
            Dim url = $"{serverUrl}/api/v1/sync/instances"
            Dim json As JObject = ModNet.NetGetCodeByRequestOnce(url, IsJson:=True, Timeout:=15000)

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
                For Each instance In items.Children(Of JObject)()
                    Dim files As New Dictionary(Of String, ModCloudInfo.CloudFileEntry)
                    Dim modListNode As JToken = instance("mod_list")
                    If modListNode IsNot Nothing AndAlso modListNode.Type = JTokenType.Array Then
                        For Each modName As JToken In modListNode
                            Dim fileName As String = modName.Value(Of String)()
                            files("mods/" & fileName) = New ModCloudInfo.CloudFileEntry With {
                                .Hash = "",
                                .Size = 0
                            }
                        Next
                    End If
                    result.Add(New ModCloudInfo.CloudInfo With {
                        .InstanceId = If(instance("id") IsNot Nothing, instance("id").Value(Of String)(), ""),
                        .CloudName = If(instance("display_name") IsNot Nothing, instance("display_name").Value(Of String)(), ""),
                        .VersionId = If(instance("version_id") IsNot Nothing, instance("version_id").Value(Of String)(), ""),
                        .McVersion = If(instance("mc_version") IsNot Nothing, instance("mc_version").Value(Of String)(), ""),
                        .Modloader = If(instance("modloader") IsNot Nothing, instance("modloader").Value(Of String)(), ""),
                        .Files = files
                    })
                Next
            End If
        Catch ex As Exception
            Log("[CloudMatch] 拉取云端实例列表失败: " & ex.Message)
        End Try

        Return result
    End Function

    ''' <summary>
    ''' 从 mod 文件名中提取匹配键（移除版本号后缀）。
    ''' 例：jei-1.20.1-15.2.0.jar → jei
    ''' </summary>
    Private Function ExtractModKey(fileName As String) As String
        ' 移除 .jar 扩展名
        Dim name = fileName
        If name.EndsWith(".jar", StringComparison.OrdinalIgnoreCase) Then
            name = name.Substring(0, name.Length - 4)
        End If
        ' 移除末尾的版本号模式：-数字.数字... 或 -数字
        ' 匹配从末尾开始的版本号段，如 -1.20.1-15.2.0 或 -1.20.1
        name = Regex.Replace(name, "-\d+(?:\.\d+)*(-[a-zA-Z]+[\d.]+)?$", "")
        ' 转小写以进行不区分大小写的匹配
        Return name.ToLowerInvariant().Trim()
    End Function

    ''' <summary>
    ''' 从 CloudInfo 的文件列表中提取所有 mod 文件的匹配键集合。
    ''' </summary>
    Private Function GetModKeys(info As ModCloudInfo.CloudInfo) As HashSet(Of String)
        Dim keys As New HashSet(Of String)
        If info Is Nothing OrElse info.Files Is Nothing Then Return keys
        For Each kv In info.Files
            Dim path = kv.Key
            If path.StartsWith("mods/", StringComparison.OrdinalIgnoreCase) AndAlso
               path.EndsWith(".jar", StringComparison.OrdinalIgnoreCase) Then
                Dim fileName = System.IO.Path.GetFileName(path)
                Dim key = ExtractModKey(fileName)
                If Not String.IsNullOrEmpty(key) Then
                    keys.Add(key)
                End If
            End If
        Next
        Return keys
    End Function

    ''' <summary>
    ''' 计算 Jaccard 相似度分数：|交集| / |并集|。
    ''' </summary>
    ''' <param name="localInfo">本地信息库</param>
    ''' <param name="cloudInfo">云端信息库</param>
    ''' <returns>0.0 到 1.0 之间的相似度分数</returns>
    Public Function ComputeMatchScore(localInfo As ModCloudInfo.CloudInfo, cloudInfo As ModCloudInfo.CloudInfo) As Double
        If localInfo Is Nothing OrElse cloudInfo Is Nothing Then Return 0.0

        Dim localKeys = GetModKeys(localInfo)
        Dim cloudKeys = GetModKeys(cloudInfo)

        If localKeys.Count = 0 OrElse cloudKeys.Count = 0 Then Return 0.0

        ' 计算交集大小
        Dim intersection = localKeys.Intersect(cloudKeys).Count()
        ' 计算并集大小
        Dim union = localKeys.Union(cloudKeys).Count()

        If union = 0 Then Return 0.0
        Return CDbl(intersection) / CDbl(union)
    End Function

    ''' <summary>
    ''' 为本地整合包查找拟合度最高的 N 个云端实例。
    ''' </summary>
    ''' <param name="localInfo">本地信息库</param>
    ''' <param name="cloudInfoList">所有云端实例的信息库列表</param>
    ''' <param name="topN">返回前 N 个匹配</param>
    ''' <returns>按分数降序排列的匹配结果列表</returns>
    Public Function FindTopMatches(localInfo As ModCloudInfo.CloudInfo,
                                   cloudInfoList As List(Of ModCloudInfo.CloudInfo),
                                   Optional topN As Integer = 3) As List(Of MatchResult)
        Dim results As New List(Of MatchResult)()

        If localInfo Is Nothing OrElse cloudInfoList Is Nothing Then Return results

        Dim localKeys = GetModKeys(localInfo)

        For Each cloudInfo In cloudInfoList
            Dim cloudKeys = GetModKeys(cloudInfo)
            Dim intersection = localKeys.Intersect(cloudKeys).Count()
            Dim union = localKeys.Union(cloudKeys).Count()
            Dim score As Double = If(union > 0, CDbl(intersection) / CDbl(union), 0.0)

            results.Add(New MatchResult With {
                .CloudInfo = cloudInfo,
                .Score = score,
                .MatchedMods = intersection,
                .TotalLocalMods = localKeys.Count,
                .TotalCloudMods = cloudKeys.Count
            })
        Next

        ' 按分数降序排列，取前 N 个
        results.Sort(Function(a, b) b.Score.CompareTo(a.Score))
        If results.Count > topN Then
            results = results.GetRange(0, topN)
        End If

        Return results
    End Function

End Module
