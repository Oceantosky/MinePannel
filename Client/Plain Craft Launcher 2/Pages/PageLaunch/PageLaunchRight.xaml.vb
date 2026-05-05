Imports PCL.Core.Utils.Exts
Imports PCL.Core.App
Public Class PageLaunchRight
    Implements IRefreshable

    Private Sub Init() Handles Me.Loaded
        PanBack.ScrollToHome()
        PanScroll = PanBack '不知道为啥不能在 XAML 设置
        PanLog.Visibility = If(ModeDebug, Visibility.Visible, Visibility.Collapsed)
        '社区版提示
        PanHint.Visibility = If(Setup.Get("UiLauncherCEHint"), Visibility.Visible, Visibility.Collapsed)
        LabHint1.Text = $"你正在使用 PCL 社区版！此版本为独立开发和维护，与官方版本维护路线不同，体验有所出入。{vbCrLf}{vbCrLf}如果你是意外下载到了社区版，我们十分建议您下载 PCL 官方版长期使用，此发行版本对新手用户体验可能不友好。{vbCrLf}此外，社区版的问题请向社区版的仓库提交 Issue，不要向官方仓库反馈社区版的问题哦！{vbCrLf}"
        LabHint2.Text = $"若要永久隐藏此提示，请输入正确的 PCL CE 开发组织名称。"
        ' 初始化 CloudAbroad
        CloudInit()
    End Sub

    '暂时关闭快照版提示
    Private Sub BtnHintClose_Click(sender As Object, e As EventArgs) Handles BtnHintClose.Click
        AniDispose(PanHint, True)
        States.Hint.CEMessage = False
    End Sub

#Region "主页"

    ''' <summary>
    ''' 刷新主页。
    ''' </summary>
    Private Sub Refresh() Handles Me.Loaded
        RunInNewThread(
        Sub()
            Try
                SyncLock RefreshLock
                    RefreshReal()
                End SyncLock
            Catch ex As Exception
                Log(ex, "加载 PCL 主页自定义信息失败", If(ModeDebug, LogLevel.Msgbox, LogLevel.Hint))
            End Try
        End Sub, $"刷新主页 #{GetUuid()}")
    End Sub
    Private Sub RefreshReal()
        Dim Content As String = ""
        Dim Url As String
        Select Case Setup.Get("UiCustomType")
            Case 1
                '加载本地文件
                Log("[Page] 主页自定义数据来源：本地文件")
                Content = ReadFile(ExePath & "PCL\Custom.xaml") 'ReadFile 会进行存在检测
            Case 2
                Url = Setup.Get("UiCustomNet")
Download:
                '加载联网文件
                If String.IsNullOrWhiteSpace(Url) Then Exit Select
                If Url = Setup.Get("CacheSavedPageUrl") AndAlso File.Exists(PathTemp & "Cache\Custom.xaml") Then
                    '缓存可用
                    Log("[Page] 主页自定义数据来源：联网缓存文件")
                    Content = ReadFile(PathTemp & "Cache\Custom.xaml")
                    '后台更新缓存
                    OnlineLoader.Start(Url)
                Else
                    '缓存不可用
                    Log("[Page] 主页自定义数据来源：联网全新下载")
                    Hint("正在加载主页……")
                    RunInUiWait(Sub() LoadContent("")) '在加载结束前清空页面
                    Setup.Set("CacheSavedPageVersion", "")
                    OnlineLoader.Start(Url) '下载完成后将会再次触发更新
                    Return
                End If
            Case 3
                Select Case Setup.Get("UiCustomPreset")
                    Case 0
                        Log("[Page] 主页预设：你知道吗")
                        Dim hintText As String = PageLaunchRight.GetRandomHint(False)
                        Content = $"
        <local:MyCard Title=""你知道吗？"" Margin=""0,0,0,15"">
            <TextBlock Margin=""25,38,23,15"" FontSize=""13.5"" IsHitTestVisible=""False"" Text=""{hintText}"" TextWrapping=""Wrap"" Foreground=""{{DynamicResource ColorBrush1}}"" />
            <local:MyIconButton Height=""22"" Width=""22"" Margin=""9"" VerticalAlignment=""Top"" HorizontalAlignment=""Right"" 
                EventType=""刷新主页"" EventData=""/""
                Logo=""M875.52 148.48C783.36 56.32 655.36 0 512 0 291.84 0 107.52 138.24 30.72 332.8l122.88 46.08C204.8 230.4 348.16 128 512 128c107.52 0 199.68 40.96 271.36 112.64L640 384h384V0L875.52 148.48zM512 896c-107.52 0-199.68-40.96-271.36-112.64L384 640H0v384l148.48-148.48C240.64 967.68 368.64 1024 512 1024c220.16 0 404.48-138.24 481.28-332.8L870.4 645.12C819.2 793.6 675.84 896 512 896z"" />
        </local:MyCard>"
                    Case 1
                        Log("[Page] 主页预设：预设 回声洞 是已被移除的主页预设")
                        MyMsgBox("回声洞 因为只有空壳因此已被移除，请前往设置选择其他预设主页", "提示")
                        Return
                    Case 2
                        Log("[Page] 主页预设：Minecraft 新闻")
                        Url = "https://pcl.mcnews.thestack.top"
                        GoTo Download
                    Case 3
                        Log("[Page] 主页预设：简单主页")
                        Url = "https://pclhomeplazaoss.lingyunawa.top:26994/d/Homepages/MFn233/Custom.xaml"
                        GoTo Download
                    Case 4
                        Log("[Page] 主页预设：每日整合包推荐")
                        Url = "https://pclsub.sodamc.com/"
                        GoTo Download
                    Case 5
                        Log("[Page] 主页预设：Minecraft 皮肤推荐")
                        Url = "https://forgepixel.com/pcl_sub_file"
                        GoTo Download
                    Case 6
                        Log("[Page] 主页预设：OpenBMCLAPI 仪表盘 Lite")
                        Url = "https://pcl-bmcl.milu.ink/"
                        GoTo Download
                    Case 7
                        Log("[Page] 主页预设：主页市场")
                        Url = "https://pclhomeplazaoss.lingyunawa.top:26994/d/Homepages/JingHai-Lingyun/Custom.xaml"
                        GoTo Download
                    Case 8
                        Log("[Page] 主页预设：更新日志")
                        Url = "https://pclhomeplazaoss.lingyunawa.top:26994/d/Homepages/Joker2184/UpdateHomepage.xaml"
                        GoTo Download
                    Case 9
                        Log("[Page] 主页预设：PCL 新功能说明书")
                        Url = "https://raw.gitcode.com/WForst-Breeze/WhatsNewPCL/raw/main/Custom.xaml"
                        GoTo Download
                    Case 10
                        Log("[Page] 主页预设：OpenMCIM Dashboard")
                        Url = "https://files.mcimirror.top/PCL"
                        GoTo Download
                    Case 11
                        Log("[Page] 主页预设：杂志主页")
                        Url = "https://pclhomeplazaoss.lingyunawa.top:26994/d/Homepages/Ext1nguisher/Custom.xaml"
                        GoTo Download
                    Case 12
                        Log("[Page] 主页预设：PCL GitHub 仪表盘")
                        Url = "https://ddf.pcl-community.org/Custom.xaml"
                        GoTo Download
                    Case 13
                        Log("[Page] 主页预设：Minecraft 更新摘要")
                        Url = "https://raw.gitcode.com/ENC_Euphony/PCL-AI-Summary-HomePage/raw/master/Custom.xaml"
                        GoTo Download
                    Case 14
                        Log("[Page] 主页预设：PCL CE 公告栏")
                        Url = "https://s3.pysio.online/pcl2-ce/apiv2/pages/announce.xaml"
                        GoTo Download
                    Case 15
                        Log("[Page] 主页预设：Minecraft 信息流")
                        RunInUiWait(
                            Sub()
                                If FrmHomepageNews Is Nothing Then FrmHomepageNews = New PageHomepageNewsView()
                                PanCustom.Children.Clear()
                                PanCustom.Children.Add(FrmHomepageNews)
                            End Sub)
                        Return
                End Select
        End Select
        RunInUi(Sub() LoadContent(Content))
    End Sub
    Private RefreshLock As New Object

    Public Shared Function GetRandomHint(Optional enableLengthLimit As Boolean = False, Optional raw As Boolean = False) As String
        Dim lines As String() = Nothing
        
        '外部文件
        Dim externalPath = ExePath & "PCL\hints.txt"
        If File.Exists(externalPath) Then
            Try
                lines = File.ReadAllLines(externalPath).Where(Function(l) Not String.IsNullOrWhiteSpace(l)).Select(Function(l) l.Trim()).ToArray()
            Catch
                Log($"[Page] 读取外部文件失败：{externalPath}", LogLevel.Hint)
            End Try
        End If
    
        '嵌入式资源
        If lines Is Nothing OrElse lines.Length = 0 Then
            Using reader As New StreamReader(Application.GetResourceStream(New Uri("pack://application:,,,/Plain Craft Launcher 2;component/Resources/hints.txt", UriKind.Absolute)).Stream)
                lines = reader.ReadToEnd().Split({vbCr, vbLf}, StringSplitOptions.RemoveEmptyEntries).Where(Function(l) Not String.IsNullOrWhiteSpace(l)).Select(Function(l) l.Trim()).ToArray()
            End Using
        End If
    
        '长度限制
        If enableLengthLimit Then
            Dim shortLines = lines.Where(Function(l) l.Length < 50).ToArray()
            If shortLines.Length > 0 Then lines = shortLines
        End If
    
        '随机返回
        Dim hint = lines(Random.Shared.Next(lines.Length))
        Return If(raw, hint, hint.Replace("&", "&amp;").Replace("<", "&lt;").Replace(">", "&gt;").Replace("""", "&quot;"))
    End Function

    '联网获取主页文件
    Private OnlineLoader As New LoaderTask(Of String, Integer)("下载主页", AddressOf OnlineLoaderSub) With {.ReloadTimeout = 10 * 60 * 1000}
    Private Sub OnlineLoaderSub(Task As LoaderTask(Of String, Integer))
        Dim Address As String = Task.Input '#3721 中连续触发两次导致内容变化
        Try
            '获取版本校验地址
            Dim VersionAddress As String
            If Address.Contains(".xaml") Then
                VersionAddress = Address.Replace(".xaml", ".xaml.ini")
            Else
                VersionAddress = Address.BeforeFirst("?")
                If Not VersionAddress.EndsWith("/") Then VersionAddress += "/"
                VersionAddress += "version"
                If Address.Contains("?") Then VersionAddress += "?" & Address.AfterFirst("?")
            End If
            '校验版本
            Dim Version As String = ""
            Dim NeedDownload As Boolean = True
            Try
                Version = NetGetCodeByRequestOnce(VersionAddress, Timeout:=10000)
                If Version.Length > 1000 Then Throw New Exception($"获取的主页版本过长（{Version.Length} 字符）")
                Dim CurrentVersion As String = Setup.Get("CacheSavedPageVersion")
                If Version <> "" AndAlso CurrentVersion <> "" AndAlso Version = CurrentVersion Then
                    Log($"[Page] 当前缓存的主页已为最新，当前版本：{Version}，检查源：{VersionAddress}")
                    NeedDownload = False
                Else
                    Log($"[Page] 需要下载联网主页，当前版本：{Version}，检查源：{VersionAddress}")
                End If
            Catch exx As Exception
                Log(exx, $"联网获取主页版本失败", LogLevel.Developer)
                Log($"[Page] 无法检查联网主页版本，将直接下载，检查源：{VersionAddress}")
            End Try
            '实际下载
            If NeedDownload Then
                Dim FileContent As String = NetGetCodeByRequestRetry(Address)
                Log($"[Page] 已联网下载主页，内容长度：{FileContent.Length}，来源：{Address}")
                Setup.Set("CacheSavedPageUrl", Address)
                Setup.Set("CacheSavedPageVersion", Version)
                WriteFile(PathTemp & "Cache\Custom.xaml", FileContent)
            End If
            '要求刷新
            RunInUi(AddressOf Refresh) '不直接调用 Refresh，以防止死循环（#6245）
        Catch ex As Exception
            Log(ex, $"下载主页失败（{Address}）", If(ModeDebug, LogLevel.Msgbox, LogLevel.Hint))
        End Try
    End Sub

    ''' <summary>
    ''' 立即强制刷新主页。
    ''' 必须在 UI 线程调用。
    ''' </summary>
    Public Sub ForceRefresh() Implements IRefreshable.Refresh
        Log("[Page] 要求强制刷新主页")
        ClearCache()
        '实际的刷新
        If FrmMain.PageCurrent.Page = FormMain.PageType.Launch Then
            PanBack.ScrollToHome()
            Refresh()
        Else
            FrmMain.PageChange(FormMain.PageType.Launch)
        End If
    End Sub

    ''' <summary>
    ''' 清空主页缓存信息。
    ''' </summary>
    Private Sub ClearCache()
        LoadedContentHash = -1
        OnlineLoader.Input = ""
        Setup.Set("CacheSavedPageUrl", "")
        Setup.Set("CacheSavedPageVersion", "")
        Log("[Page] 已清空主页缓存")
    End Sub

    ''' <summary>
    ''' 从文本内容中加载主页。
    ''' 必须在 UI 线程调用。
    ''' </summary>
    Private Sub LoadContent(Content As String)
        SyncLock LoadContentLock
            '如果加载目标内容一致则不加载
            Dim Hash = Content.GetHashCode()
            If Hash = LoadedContentHash Then Return
            LoadedContentHash = Hash
            '实际加载内容
            PanCustom.Children.Clear()
            If String.IsNullOrWhiteSpace(Content) Then
                Log($"[Page] 实例化：清空主页 UI，来源为空")
                Return
            End If
            Dim LoadStartTime As Date = Date.Now
            Try
                '修改时应同时修改 PageOtherHelpDetail.Init
                Content = ArgumentReplace(Content)
                Do While Content.Contains("xmlns")
                    Content = Content.RegexReplace("xmlns[^""']*(""|')[^""']*(""|')", "").Replace("xmlns", "")
                Loop
                Content = "<StackPanel xmlns=""http://schemas.microsoft.com/winfx/2006/xaml/presentation"" xmlns:sys=""clr-namespace:System;assembly=System.Runtime"" xmlns:x=""http://schemas.microsoft.com/winfx/2006/xaml"" xmlns:local=""clr-namespace:PCL;assembly=Plain Craft Launcher 2"">" & Content & "</StackPanel>"
                Log($"[Page] 实例化：加载主页 UI 开始，最终内容长度：{Content.Count}")
                PanCustom.Children.Add(GetObjectFromXML(Content))
            Catch ex As Exception
                If ModeDebug Then
                    Log(ex, "加载失败的主页内容：" & vbCrLf & Content)
                    If MyMsgBox(If(TypeOf ex Is UnauthorizedAccessException, ex.Message, $"主页内容编写有误，请根据下列错误信息进行检查：{vbCrLf}{ex.ToString()}"),
                                "加载主页界面失败", "重试", "取消") = 1 Then
                        GoTo Refresh '防止 SyncLock 死锁
                    End If
                Else
                    Log(ex, "加载主页界面失败", LogLevel.Hint)
                End If
                Return
            End Try
            Dim LoadCostTime = (Date.Now - LoadStartTime).Milliseconds
            Log($"[Page] 实例化：加载主页 UI 完成，耗时 {LoadCostTime}ms")
            If LoadCostTime > 3000 Then Hint($"主页加载过于缓慢（花费了 {Math.Round(LoadCostTime / 1000, 1)} 秒），请向主页作者反馈此问题，或暂时停止使用该主页")
        End SyncLock
        Return
Refresh:
        ForceRefresh()
    End Sub
    Private LoadedContentHash As Integer = -1
    Private LoadContentLock As New Object

#End Region

#Region "云端连接"

    Private _CurrentCloudInstances As List(Of ModCloudDiscovery.CloudInstance)

    Private Sub CloudInit()
        PanCloud.Visibility = Visibility.Visible
        If Not String.IsNullOrEmpty(ModCloudAuth.CloudServerUrl) Then
            ShowCloudConnected()
            ' 后台自动拉取实例列表以备使用
            RunInNewThread(
            Sub()
                Dim errorMsg As String = ""
                Dim instances = ModCloudDiscovery.FetchInstances(ModCloudAuth.CloudServerUrl, errorMsg)
                _CurrentCloudInstances = instances
            End Sub, "Fetch Cloud Instances")
        Else
            ShowCloudDisconnected()
        End If
    End Sub

    Private Sub ShowCloudDisconnected()
        PanCloudDisconnected.Visibility = Visibility.Visible
        PanCloudHeader.Visibility = Visibility.Collapsed
        PanCloudCards.Visibility = Visibility.Collapsed
    End Sub

    Private Sub ShowCloudConnected()
        PanCloudDisconnected.Visibility = Visibility.Collapsed
        PanCloudHeader.Visibility = Visibility.Visible
        PanCloudCards.Visibility = Visibility.Visible
        LabCloudDomain.Text = If(String.IsNullOrEmpty(ModCloudAuth.CloudLastDomain),
            New Uri(ModCloudAuth.CloudServerUrl).Host, ModCloudAuth.CloudLastDomain)
            
        RefreshCloudCards()
    End Sub

    Private Sub RefreshCloudCards()
        CloudCardsPanel.Children.Clear()

        ' 获取本地已同步的游戏
        Dim localSynced = GetSyncedLocalInstances()
        For Each gameName In localSynced
            Dim instanceDir = System.IO.Path.Combine(McFolderSelected, "versions", gameName)
            Dim displayName As String = gameName
            Dim cloudInfoLoad = ModCloudInfo.LoadLocalInfo(instanceDir)
            If cloudInfoLoad IsNot Nothing AndAlso Not String.IsNullOrEmpty(cloudInfoLoad.CloudName) Then
                displayName = cloudInfoLoad.CloudName
            Else
                Dim pclIniPath = System.IO.Path.Combine(instanceDir, ModMinePannel.PclIniName)
                If IO.File.Exists(pclIniPath) Then
                    Try
                        Dim info = ModMinePannel.ParsePclIni(ReadFile(pclIniPath))
                        If Not String.IsNullOrEmpty(info.Name) Then
                            Dim cleanName = System.Text.RegularExpressions.Regex.Replace(info.Name, "\s*\([^)]*\)$", "").Trim()
                            If Not String.IsNullOrEmpty(cleanName) Then displayName = cleanName
                        End If
                    Catch
                    End Try
                End If
            End If
            
            Dim card = CreateDashedCard(displayName, "已同步")
            Dim capturedGameName = gameName
            Dim capturedInstanceDir = instanceDir
            AddHandler card.MouseLeftButtonDown,
                Sub()
                    Dim cloudInfo = ModCloudInfo.LoadLocalInfo(capturedInstanceDir)
                    If cloudInfo IsNot Nothing Then
                        Hint("开始检查增量更新...", HintType.Info)
                        RunInNewThread(
                        Sub()
                            Try
                                Dim manifestUrl As String = $"{ModCloudAuth.CloudServerUrl}/api/v1/sync/info?id={cloudInfo.InstanceId}"
                                Dim success = ModMinePannel.IncrementalSync(manifestUrl, capturedInstanceDir)
                                RunInUi(
                                Sub()
                                    If success Then
                                        Hint("增量同步完成！", HintType.Finish)
                                    Else
                                        Hint("已是最新版本，或同步跳过。", HintType.Finish)
                                    End If
                                End Sub)
                            Catch ex As Exception
                                Log(ex, "手动触发增量同步失败")
                                RunInUi(Sub() Hint("增量同步失败，请检查日志", HintType.Critical))
                            End Try
                        End Sub, "Manual Cloud Sync")
                    Else
                        Hint("读取云端配置失败", HintType.Critical)
                    End If
                End Sub
            CloudCardsPanel.Children.Add(card)
        Next

        ' 添加 + 按钮
        Dim addCard = CreateDashedCard("从云端添加游戏", "＋", True)
        AddHandler addCard.MouseLeftButtonDown, 
            Sub() 
                If _CurrentCloudInstances IsNot Nothing AndAlso _CurrentCloudInstances.Count > 0 Then
                    ShowInstanceDialog(_CurrentCloudInstances)
                Else
                    Hint("正在获取云端实例列表，请稍后再试或云端无实例", HintType.Info)
                End If
            End Sub
        CloudCardsPanel.Children.Add(addCard)
    End Sub

    Private Function GetSyncedLocalInstances() As List(Of String)
        Dim list As New List(Of String)
        Try
            Dim versionsDir = System.IO.Path.Combine(McFolderSelected, "versions")
            If System.IO.Directory.Exists(versionsDir) Then
                For Each folder In System.IO.Directory.GetDirectories(versionsDir)
                    If System.IO.File.Exists(System.IO.Path.Combine(folder, "PCL", ModCloudInfo.InfoFileName)) Then
                        list.Add(System.IO.Path.GetFileName(folder))
                    End If
                Next
            End If
        Catch ex As Exception
        End Try
        Return list
    End Function

    Private Function CreateDashedCard(title As String, subtitle As String, Optional isAdd As Boolean = False) As Grid
        Dim grid As New Grid With {
            .Width = 140,
            .Height = 140,
            .Margin = New Thickness(0, 0, 15, 15),
            .Cursor = Cursors.Hand,
            .Background = Brushes.Transparent
        }

        Dim rect As New Shapes.Rectangle With {
            .StrokeDashArray = New DoubleCollection({4, 4}),
            .StrokeThickness = 2,
            .RadiusX = 8,
            .RadiusY = 8,
            .Fill = Brushes.Transparent
        }
        rect.SetResourceReference(Shapes.Shape.StrokeProperty, "ColorBrush3")
        grid.Children.Add(rect)

        Dim stack As New StackPanel With {
            .VerticalAlignment = VerticalAlignment.Center,
            .HorizontalAlignment = HorizontalAlignment.Center
        }

        If isAdd Then
            Dim tbIcon = New TextBlock With {
                .Text = subtitle,
                .FontSize = 36,
                .HorizontalAlignment = HorizontalAlignment.Center,
                .Margin = New Thickness(0, 0, 0, 8),
                .FontWeight = FontWeights.Bold
            }
            tbIcon.SetResourceReference(TextBlock.ForegroundProperty, "ColorBrush4")
            stack.Children.Add(tbIcon)
            
            Dim tbTitle = New TextBlock With {
                .Text = title,
                .FontSize = 13,
                .HorizontalAlignment = HorizontalAlignment.Center
            }
            tbTitle.SetResourceReference(TextBlock.ForegroundProperty, "ColorBrush2")
            stack.Children.Add(tbTitle)
        Else
            Dim tbTitle = New TextBlock With {
                .Text = title,
                .FontSize = 15,
                .HorizontalAlignment = HorizontalAlignment.Center,
                .TextWrapping = TextWrapping.Wrap,
                .TextAlignment = TextAlignment.Center,
                .MaxWidth = 120,
                .Margin = New Thickness(0, 0, 0, 8)
            }
            tbTitle.SetResourceReference(TextBlock.ForegroundProperty, "ColorBrush1")
            stack.Children.Add(tbTitle)
            
            Dim tbSub = New TextBlock With {
                .Text = subtitle,
                .FontSize = 12,
                .HorizontalAlignment = HorizontalAlignment.Center
            }
            tbSub.SetResourceReference(TextBlock.ForegroundProperty, "ColorBrushGray2")
            stack.Children.Add(tbSub)
        End If

        grid.Children.Add(stack)
        Return grid
    End Function

    ''' <summary>点击"添加新的云同步"卡片，弹出输入框</summary>
    Private Sub CardCloudAdd_MouseLeftButtonDown(sender As Object, e As MouseButtonEventArgs) Handles CardCloudAdd.MouseLeftButtonDown
        Dim domain As String = MyMsgBoxInput("连接到 CloudAbroad", "请输入你要连接的服务器域名或 IP 地址：", 
            HintText:="例如: mc.example.com", Button1:="连接", Button2:="取消")

        If String.IsNullOrEmpty(domain) Then Return
        domain = domain.Trim()
        If String.IsNullOrEmpty(domain) Then Return

        Hint("正在连接...", HintType.Info)

        RunInNewThread(
        Sub()
            Dim errorMsg As String = ""
            Dim instances As List(Of ModCloudDiscovery.CloudInstance) = Nothing
            Dim success As Boolean = ModCloudDiscovery.ConnectAndFetchInstances(domain, instances, errorMsg)

            RunInUi(
            Sub()
                If success Then
                    _CurrentCloudInstances = instances
                    ShowCloudConnected()
                    RunLocalMatchAndSuggest()
                    RunLocalInstanceMarkCheck(instances)
                Else
                    Hint("连接失败：" & errorMsg, HintType.Critical)
                End If
            End Sub)
        End Sub, "Cloud Connect")
    End Sub



    Private Sub ShowInstanceDialog(instances As List(Of ModCloudDiscovery.CloudInstance))
        Dim selections As New List(Of IMyRadio)()
        For Each inst In instances
            Dim title As String = inst.Name
            If Not String.IsNullOrEmpty(inst.Version) Then
                title &= "（" & inst.Version & "）"
            End If
            If inst.PlayerCount > 0 Then
                title &= "  [" & inst.PlayerCount & " 人在线]"
            End If
            Dim item As New MyListItem With {
                .Title = title,
                .MinHeight = 36
            }
            If Not String.IsNullOrEmpty(inst.Description) Then
                item.ToolTip = inst.Description
            End If
            selections.Add(item)
        Next

        Dim selectedIndex As Integer? = MyMsgBoxSelect(selections, "选择云端实例", "同步", "取消")
        If selectedIndex.HasValue Then
            Dim inst As ModCloudDiscovery.CloudInstance = instances(selectedIndex.Value)
            StartMinePannel(inst)
        End If
    End Sub

    Private Sub StartMinePannel(inst As ModCloudDiscovery.CloudInstance)
        Log($"[MinePannel] 用户选择云端实例：{inst.Name}（{inst.Version}）")

        If Not String.IsNullOrEmpty(inst.PackUrl) Then
            Dim packUrl As String = ModMinePannel.NegotiateMinePannelUrl(inst.PackUrl)
            If String.IsNullOrEmpty(packUrl) Then Return
            
            Dim tempZip As String = PathTemp & "cloud_pack_" & GetUuid() & ".zip"
            Directory.CreateDirectory(PathTemp)
            
            Dim dlTask As New LoaderDownload("下载云端包：" & inst.Name, New List(Of NetFile) From {New NetFile({packUrl}, tempZip)}) With {.ProgressWeight = 10, .Block = True}
            
            Dim deployTask As New LoaderTask(Of Integer, Integer)("准备部署云端实例",
            Sub()
                If Not File.Exists(tempZip) Then Throw New Exception("云端同步包下载失败")
                Try
                    ' ModpackInstall 将会自动创建新任务并加入任务栏
                    Dim installLoader = ModModpack.ModpackInstall(tempZip, inst.Name, isOnlineInstall:=True)
                    AddHandler installLoader.OnStateChangedThread,
                        Sub(Loader As LoaderBase, NewState As LoadState, OldState As LoadState)
                            If NewState = LoadState.Finished Then
                                Try
                                    Dim instanceDir = System.IO.Path.Combine(McFolderSelected, "versions", inst.Name)
                                    ModCloudInfo.SaveLocalInfo(instanceDir, New ModCloudInfo.CloudInfo With {
                                        .InstanceId = inst.ID
                                    })
                                    RunInUi(AddressOf RefreshCloudCards)
                                Catch ex2 As Exception
                                    Log(ex2, "保存云端同步信息失败")
                                End Try
                            End If
                        End Sub
                    installLoader.HasOnStateChangedThread = True
                Catch ex As CancelledException
                    ' 用户主动取消或校验失败，静默忽略
                Catch ex As Exception
                    Log(ex, "云端实例部署失败", LogLevel.Msgbox)
                End Try
            End Sub) With {.ProgressWeight = 0.1, .Block = True}
            
            Dim loaders As New List(Of LoaderBase) From {dlTask, deployTask}
            Dim loaderCombo As New LoaderCombo(Of String)("云端实例部署：" & inst.Name, loaders)
            
            loaderCombo.Start()
            LoaderTaskbarAdd(loaderCombo)
            Hint("云端实例下载与部署已启动！请在左下角查看进度。", HintType.Info)
        Else
            Hint("该实例暂无可用的同步包", HintType.Info)
        End If
    End Sub

    Private Sub RunLocalMatchAndSuggest()
        RunInNewThread(
        Sub()
            Try
                If McInstanceSelected Is Nothing Then Return
                Dim localPack = ModCloudInfo.GenerateLocalInfo(McInstanceSelected.PathInstance)
                If localPack Is Nothing Then Return

                Dim cloudInfoList = ModCloudMatch.FetchCloudInfoList(ModCloudAuth.CloudServerUrl)
                If cloudInfoList Is Nothing OrElse cloudInfoList.Count = 0 Then Return

                For Each m In ModCloudMatch.FindTopMatches(localPack, cloudInfoList)
                    If m.Score >= ModCloudMatch.MatchThreshold Then
                        Dim waitHandle As New Threading.ManualResetEvent(False)
                        RunInUi(
                        Sub()
                            Try
                                Dim msg = $"发现云端实例与本地整合包非常匹配！{vbCrLf}{vbCrLf}" &
                                          $"云端实例：{m.CloudInfo.InstanceId}{vbCrLf}" &
                                          $"匹配度：{m.Score:P1}（{m.MatchedMods} 个 mod 匹配）{vbCrLf}{vbCrLf}" &
                                          $"是否直接绑定此云端实例并开始增量同步？"
                                If MyMsgBox(msg, "发现可同步的云端实例", "绑定并同步", "忽略") = 1 Then
                                    ' 写入绑定信息
                                    Setup.Set("MinePannelInstanceId", m.CloudInfo.InstanceId)
                                    Setup.Set("MinePannelInstanceName", m.CloudInfo.InstanceId)
                                    ' 执行增量同步
                                    Dim manifestUrl As String = $"{ModCloudAuth.CloudServerUrl}/api/v1/sync/info?id={m.CloudInfo.InstanceId}"
                                    Dim instanceDir As String = McInstanceSelected.PathInstance
                                    ModMinePannel.IncrementalSync(manifestUrl, instanceDir)
                                    Hint("云端实例已成功绑定，并且增量同步已启动！", HintType.Finish)
                                End If
                            Finally
                                waitHandle.Set()
                            End Try
                        End Sub)
                        waitHandle.WaitOne()
                    End If
                Next
            Catch ex As Exception
                Log(ex, "[CloudUI] 本地匹配扫描异常", LogLevel.Developer)
            End Try
        End Sub, "Cloud Local Match")
    End Sub

    ''' <summary>首次 SRV 连接后，检查是否有可标记为云同步的本地实例</summary>
    Private Sub RunLocalInstanceMarkCheck(cloudInstances As List(Of ModCloudDiscovery.CloudInstance))
        ' 同一服务器已提示过则跳过
        If ModCloudDiscovery.MarkOfferedForServer = ModCloudAuth.CloudServerUrl Then Return
        If String.IsNullOrEmpty(McFolderSelected) Then Return

        RunInNewThread(
        Sub()
            Try
                Dim cloudInfoList = ModCloudMatch.FetchCloudInfoList(ModCloudAuth.CloudServerUrl)
                If cloudInfoList Is Nothing OrElse cloudInfoList.Count = 0 Then Return

                Dim candidates = ModCloudDiscovery.FindMarkableInstances(McFolderSelected, cloudInfoList)
                If candidates Is Nothing OrElse candidates.Count = 0 Then Return

                Dim waitHandle As New Threading.ManualResetEvent(False)
                RunInUi(
                Sub()
                    Try
                        ShowMarkDialog(candidates, cloudInstances)
                        ' 标记已提示（无论用户如何选择，避免重复弹窗）
                        ModCloudDiscovery.MarkOfferedForServer = ModCloudAuth.CloudServerUrl
                    Finally
                        waitHandle.Set()
                    End Try
                End Sub)
                waitHandle.WaitOne()
            Catch ex As Exception
                Log(ex, "[CloudUI] 存量实例扫描异常", LogLevel.Developer)
            End Try
        End Sub, "Cloud Mark Check")
    End Sub

    ''' <summary>显示标记候选对话框，让用户选择要标记的本地实例或安装全新实例</summary>
    Private Sub ShowMarkDialog(candidates As List(Of ModCloudDiscovery.MarkableCandidate), cloudInstances As List(Of ModCloudDiscovery.CloudInstance))
        Dim selections As New List(Of IMyRadio)()

        For Each candidate In candidates
            Dim title As String = $"{candidate.LocalPack.InstanceName} → 云端：{candidate.CloudInfo.InstanceId}"
            Dim subtitle As String = $"匹配度：{candidate.Score:P0} | 重叠 Mod：{candidate.MatchedMods} 个"
            Dim item As New MyListItem With {
                .Title = title,
                .Info = subtitle,
                .MinHeight = 36
            }
            selections.Add(item)
        Next

        ' 最后一项：安装全新云端实例
        selections.Add(New MyListItem With {
            .Title = "[安装全新的云端实例]",
            .Info = "从云端下载完整整合包并安装",
            .MinHeight = 36
        })

        Dim selectedIndex As Integer? = MyMsgBoxSelect(selections, "发现可同步的本地整合包",
            "标记并同步", "取消")

        If Not selectedIndex.HasValue Then Return

        If selectedIndex.Value < candidates.Count Then
            ' 标记选中的本地实例
            Dim candidate = candidates(selectedIndex.Value)
            RunInNewThread(
            Sub()
                Dim success = ModCloudDiscovery.MarkAsCloudManaged(candidate.LocalPack.InstanceDir, candidate.CloudInfo)
                RunInUi(
                Sub()
                    If success Then
                        Hint($"已将「{candidate.LocalPack.InstanceName}」标记为云端同步！", HintType.Finish)
                        RefreshCloudCards()
                    Else
                        Hint("标记失败，请检查日志", HintType.Critical)
                    End If
                End Sub)
            End Sub, "Cloud Mark Instance")
        Else
            ' 安装全新实例
            ShowInstanceDialog(cloudInstances)
        End If
    End Sub

    Private Sub BtnCloudRefresh_Click(sender As Object, e As EventArgs) Handles BtnCloudRefresh.Click
        If String.IsNullOrEmpty(ModCloudAuth.CloudServerUrl) Then Return
        
        Hint("正在同步云端状态...", HintType.Info)
        RunInNewThread(
        Sub()
            ' 1. 获取最新的云端实例列表
            Dim errorMsg As String = ""
            Dim instances = ModCloudDiscovery.FetchInstances(ModCloudAuth.CloudServerUrl, errorMsg)
            
            ' 2. 检查本地所有已同步的实例的增量更新
            Dim localSynced = GetSyncedLocalInstances()
            Dim updatedCount = 0
            Dim nameUpdated As Boolean = False
            For Each gameName In localSynced
                Dim instanceDir = System.IO.Path.Combine(McFolderSelected, "versions", gameName)
                Dim cloudInfo = ModCloudInfo.LoadLocalInfo(instanceDir)
                If cloudInfo IsNot Nothing Then
                    Try
                        If instances IsNot Nothing Then
                            Dim matchedInstance = instances.FirstOrDefault(Function(i) i.Id = cloudInfo.InstanceId)
                            If matchedInstance IsNot Nothing Then
                                If String.IsNullOrEmpty(cloudInfo.CloudName) OrElse cloudInfo.CloudName <> matchedInstance.Name Then
                                    cloudInfo.CloudName = matchedInstance.Name
                                    ModCloudInfo.SaveLocalInfo(instanceDir, cloudInfo)
                                    nameUpdated = True
                                End If
                            End If
                        End If

                        Dim manifestUrl As String = $"{ModCloudAuth.CloudServerUrl}/api/v1/sync/info?id={cloudInfo.InstanceId}"
                        If ModMinePannel.IncrementalSync(manifestUrl, instanceDir) Then
                            updatedCount += 1
                        End If
                    Catch ex As Exception
                        Log(ex, $"刷新时更新本地实例 {gameName} 失败", LogLevel.Developer)
                    End Try
                End If
            Next

            If nameUpdated Then
                McInstanceListForceRefresh = True
                LoaderFolderRun(McInstanceListLoader, McFolderSelected, LoaderFolderRunType.ForceRun, MaxDepth:=1, ExtraPath:="versions\")
            End If

            ' 3. 更新 UI
            RunInUi(
            Sub()
                If instances IsNot Nothing AndAlso instances.Count > 0 Then
                    _CurrentCloudInstances = instances
                    RefreshCloudCards()
                    If updatedCount > 0 Then
                        Hint($"云端状态对齐完成！已拉取 {updatedCount} 个本地实例的更新。", HintType.Finish)
                    Else
                        Hint("云端状态对齐完成！目前都是最新版本。", HintType.Finish)
                    End If
                ElseIf String.IsNullOrEmpty(errorMsg) Then
                    _CurrentCloudInstances = instances
                    RefreshCloudCards()
                    Hint("刷新完成，云端服务器暂无可用实例。", HintType.Info)
                Else
                    Hint("刷新失败：" & errorMsg, HintType.Critical)
                End If
            End Sub)
        End Sub, "Fetch Cloud Instances")
    End Sub

    Private Sub BtnCloudDisconnect_Click(sender As Object, e As EventArgs) Handles BtnCloudDisconnect.Click
        ModCloudAuth.CloudServerUrl = ""
        ModCloudAuth.CloudLastDomain = ""
        ShowCloudDisconnected()
        Log("[MinePannel] 已断开云端连接")
    End Sub

#End Region

End Class
