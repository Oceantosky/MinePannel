Imports System.Collections.ObjectModel
Imports System.Net.Http
Imports Newtonsoft.Json.Linq
Imports PCL.Core.App
Imports PCL.Core.Utils.Secret

''' <summary>
''' 云端预授权与设备绑定模块。
''' </summary>
Friend Module ModCloudAuth

#Region "属性"

    ''' <summary>服务端基地址（含端口），如 http://1.2.3.4:55000</summary>
    Public Property CloudServerUrl As String
        Get
            Return Config.Cloud.ServerUrl
        End Get
        Set(value As String)
            Config.Cloud.ServerUrl = value
        End Set
    End Property

    ''' <summary>设备绑定后存储的 binding_token（加密存储）</summary>
    Public Property CloudBindingToken As String
        Get
            Return Config.Cloud.BindingToken
        End Get
        Set(value As String)
            Config.Cloud.BindingToken = value
        End Set
    End Property

    ''' <summary>上次成功连接的域名</summary>
    Public Property CloudLastDomain As String
        Get
            Return Config.Cloud.LastDomain
        End Get
        Set(value As String)
            Config.Cloud.LastDomain = value
        End Set
    End Property

    ''' <summary>是否已绑定（本地有 binding_token）</summary>
    Public ReadOnly Property IsBound As Boolean
        Get
            Return Not String.IsNullOrEmpty(CloudBindingToken)
        End Get
    End Property

    ''' <summary>获取设备指纹（复用 Identify.LauncherId）</summary>
    Public ReadOnly Property DeviceId As String
        Get
            Return Identify.LauncherId
        End Get
    End Property

    ''' <summary>获取设备标签（OS + 主机名）</summary>
    Public ReadOnly Property DeviceLabel As String
        Get
            Return $"{Environment.OSVersion} - {Environment.MachineName}"
        End Get
    End Property

#End Region

#Region "API 调用"

    ''' <summary>
    ''' 查询服务端预授权是否开启。
    ''' </summary>
    ''' <param name="serverUrl">服务端基地址</param>
    ''' <returns>预授权是否开启，网络错误时返回 False</returns>
    Public Function CheckPreAuthStatus(Optional serverUrl As String = Nothing) As Boolean
        If String.IsNullOrEmpty(serverUrl) Then serverUrl = CloudServerUrl
        If String.IsNullOrEmpty(serverUrl) Then Return False
        Try
            Dim result As String = ModNet.NetGetCodeByRequestOnce(
                $"{serverUrl}/api/v1/preauth/status",
                IsJson:=True,
                Timeout:=10000
            )
            Dim json As JObject = JObject.Parse(result)
            Return json("data")("pre_auth_enabled").Value(Of Boolean)()
        Catch ex As Exception
            Log("[CloudAuth] 检查预授权状态失败：" & ex.Message)
            Return False
        End Try
    End Function

    ''' <summary>
    ''' 使用预授权密钥绑定当前设备。
    ''' </summary>
    ''' <param name="keyCode">管理员发送的预授权密钥</param>
    ''' <param name="errorMessage">失败时的错误信息</param>
    ''' <param name="serverUrl">服务端基地址（可选，默认使用已保存的地址）</param>
    ''' <returns>成功返回 binding_token，失败返回空字符串</returns>
    Public Function BindDevice(keyCode As String, ByRef errorMessage As String, Optional serverUrl As String = Nothing) As String
        If String.IsNullOrEmpty(serverUrl) Then serverUrl = CloudServerUrl
        If String.IsNullOrEmpty(serverUrl) Then
            errorMessage = "未配置服务器地址"
            Return ""
        End If
        If String.IsNullOrEmpty(keyCode) Then
            errorMessage = "请输入预授权密钥"
            Return ""
        End If

        ' 标准化密钥格式：去空白、去破折号、转大写
        Dim normalizedKey As String = keyCode.Replace(" ", "").Replace("-", "").ToUpperInvariant()
        If normalizedKey.StartsWith("CLOUD") Then normalizedKey = normalizedKey.Substring(5)
        normalizedKey = "CLOUD-" & String.Join("-", {
            normalizedKey.Substring(0, 4),
            normalizedKey.Substring(4, 4),
            normalizedKey.Substring(8, 4)
        })

        Dim localDeviceId As String = DeviceId
        Dim localDeviceLabel As String = DeviceLabel

        Log($"[CloudAuth] 开始绑定设备：{localDeviceLabel}（{localDeviceId}）")

        Try
            Dim payload As New JObject From {
                {"key_code", normalizedKey},
                {"device_id", localDeviceId},
                {"device_label", localDeviceLabel}
            }

            Dim result As String = ModNet.NetRequestOnce(
                $"{serverUrl}/api/v1/preauth/bind",
                "POST",
                payload.ToString(),
                "application/json",
                Timeout:=15000
            )

            Dim json As JObject = JObject.Parse(result)
            If json("success").Value(Of Boolean)() Then
                Dim token As String = json("data")("binding_token").Value(Of String)()
                Dim playerName As String = If(json("data")("player_name") IsNot Nothing, json("data")("player_name").Value(Of String)(), "")
                CloudBindingToken = token
                If Not String.IsNullOrEmpty(serverUrl) Then CloudServerUrl = serverUrl
                Log($"[CloudAuth] 设备绑定成功，玩家: {playerName}")
                Return token
            Else
                errorMessage = json("message").Value(Of String)()
                Log("[CloudAuth] 设备绑定失败：" & errorMessage)
                Return ""
            End If
        Catch ex As ModNet.HttpWebException
            errorMessage = $"服务器拒绝绑定 ({(CType(ex.StatusCode, Integer))})"
            Log("[CloudAuth] " & errorMessage)
            Return ""
        Catch ex As Exception
            errorMessage = "网络错误：" & ex.Message
            Log("[CloudAuth] 绑定请求失败：" & ex.Message)
            Return ""
        End Try
    End Function

    ''' <summary>
    ''' 向 HttpRequestMessage 注入 CloudAbroad 认证头。
    ''' 仅在已绑定时生效。
    ''' </summary>
    Public Sub SignCloudRequest(ByRef request As HttpRequestMessage)
        request.Headers.Add("X-Device-ID", DeviceId)
        If IsBound Then
            request.Headers.Add("X-Binding-Token", CloudBindingToken)
        End If
    End Sub

    ''' <summary>
    ''' 清除本地绑定状态。
    ''' </summary>
    Public Sub ClearBinding()
        CloudBindingToken = ""
        Log("[CloudAuth] 已清除本地绑定")
    End Sub

#End Region

#Region "预授权对话框"

    Private _isPreAuthDialogShowing As Boolean = False
    Private _preAuthDialogLock As New Object()

    ''' <summary>
    ''' 显示预授权密钥输入对话框并执行绑定。
    ''' </summary>
    ''' <param name="serverUrl">服务端基地址（可选，默认使用已保存的地址）</param>
    ''' <returns>绑定成功返回 True，用户取消或失败返回 False</returns>
    Public Function ShowPreAuthDialog(Optional serverUrl As String = Nothing) As Boolean
        If String.IsNullOrEmpty(serverUrl) Then serverUrl = CloudServerUrl
        If String.IsNullOrEmpty(serverUrl) Then Return False

        SyncLock _preAuthDialogLock
            If _isPreAuthDialogShowing Then Return False
            If IsBound Then Return True ' 已经被其他线程绑定
            _isPreAuthDialogShowing = True
        End SyncLock

        Try
            Dim keyCode As String = ""
            RunInUiWait(Sub()
                keyCode = MyMsgBoxInput(
                    Title:="云端设备认证",
                    Text:="该服务器要求进行设备授权认证。请输入管理员发给您的预授权密钥：",
                    HintText:="CLOUD-XXXX-XXXX-XXXX",
                    Button1:="绑定",
                    Button2:="取消",
                    ValidateRules:=New ObjectModel.Collection(Of Validate) From {
                        New ValidateRegex(
                            "^(CLOUD-?)?[A-Fa-f0-9]{4}-?[A-Fa-f0-9]{4}-?[A-Fa-f0-9]{4}$",
                            "预授权密钥格式不正确，应为 CLOUD-XXXX-XXXX-XXXX"
                        )
                    }
                )
            End Sub)

            If String.IsNullOrEmpty(keyCode) Then
                Log("[CloudAuth] 用户取消了密钥输入")
                Return False
            End If

            Dim errorMsg As String = ""
            Dim token As String = BindDevice(keyCode, errorMsg, serverUrl)

            RunInUiWait(Sub()
                If Not String.IsNullOrEmpty(token) Then
                    MyMsgBox("设备认证成功！您现在可以获取云端实例了。", "认证成功")
                Else
                    MyMsgBox("认证失败：" & errorMsg, "认证失败", IsWarn:=True)
                End If
            End Sub)

            Return Not String.IsNullOrEmpty(token)
        Finally
            SyncLock _preAuthDialogLock
                _isPreAuthDialogShowing = False
            End SyncLock
        End Try
    End Function

    ''' <summary>
    ''' 在后台线程中执行完整的预授权检测与绑定流程。
    ''' 可在 FormMain_Loaded 中作为 Loader 任务调用。
    ''' </summary>
    Public Sub RunPreAuthFlow()
        If String.IsNullOrEmpty(CloudServerUrl) Then Return

        ' 已绑定的设备无需重新绑定
        If IsBound Then
            Log("[CloudAuth] 设备已绑定，跳过认证流程")
            Return
        End If

        ' 检查服务端是否开启了预授权
        If Not CheckPreAuthStatus() Then
            Log("[CloudAuth] 服务端未开启预授权，跳过认证流程")
            Return
        End If

        ' 在此直接调用，UI 切换由内部处理
        ShowPreAuthDialog()
    End Sub

#End Region

End Module
