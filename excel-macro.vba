Sub GenerateQuickPhrases()
    ' Excel VBA 宏 - 生成快捷短语
    ' 使用方法：
    ' 1. 在 A1 单元格粘贴输入文本，如：
    '    h9729701196 http://card-manager-production-a1b5.up.railway.app/query?card=9729701196_PBZdfC 9729756452 http://card-manager-production-a1b5.up.railway.app/query?card=9729756452_nPVkgs
    ' 2. 运行此宏
    ' 3. 结果生成在 B 列开始
    
    Dim inputText As String
    Dim phones() As String
    Dim urls() As String
    Dim i As Integer, j As Integer
    Dim phone As String, url As String
    Dim ws As Worksheet
    
    Set ws = ActiveSheet
    inputText = ws.Range("A1").Value
    
    If inputText = "" Then
        MsgBox "请在 A1 单元格粘贴输入文本", vbExclamation
        Exit Sub
    End If
    
    ' 清理输入文本
    inputText = Replace(inputText, "h", " ") ' 去掉 h 前缀
    inputText = Trim(inputText)
    
    ' 分割成数组
    Dim items() As String
    items = Split(inputText, " ")
    
    ' 提取手机号和URL
    Dim phoneList As Object
    Dim urlList As Object
    Set phoneList = CreateObject("System.Collections.ArrayList")
    Set urlList = CreateObject("System.Collections.ArrayList")
    
    For i = LBound(items) To UBound(items)
        Dim item As String
        item = Trim(items(i))
        If item <> "" Then
            ' 判断是手机号还是URL
            If Left(item, 4) = "http" Then
                urlList.Add item
            ElseIf IsNumeric(item) And Len(item) = 10 Then
                phoneList.Add item
            End If
        End If
    Next i
    
    ' 检查数据
    If phoneList.Count = 0 Then
        MsgBox "未找到有效的手机号（10位数字）", vbExclamation
        Exit Sub
    End If
    
    ' 生成快捷短语
    Dim row As Integer
    row = 2 ' 从第2行开始输出
    
    ' 清空旧数据
    ws.Range("B:E").ClearContents
    
    ' 设置表头
    ws.Range("B1").Value = "快捷短语编码"
    ws.Range("C1").Value = "快捷短语内容"
    ws.Range("D1").Value = "快捷短语分组"
    
    For i = 0 To phoneList.Count - 1
        phone = phoneList(i)
        
        ' 查找对应的URL
        url = ""
        For j = 0 To urlList.Count - 1
            If InStr(urlList(j), phone) > 0 Then
                url = urlList(j)
                Exit For
            End If
        Next j
        
        ' 如果没有找到对应URL，生成默认URL
        If url = "" Then
            url = "http://card-manager-production-a1b5.up.railway.app/query?card=" & phone
        End If
        
        ' 生成内容
        Dim content As String
        content = (i + 1) & "组手机号 " & phone & vbCrLf & _
                  "查看验证码 " & url & vbCrLf & _
                  "登陆流程:腾讯视频登录界面，把这个区号+86改为+1(重点），输入手机号，获取验证码登陆"
        
        ' 写入Excel
        ws.Cells(row, 2).Value = i + 1                    ' B列：编码
        ws.Cells(row, 3).Value = content                  ' C列：内容
        ws.Cells(row, 4).Value = "手机号"                  ' D列：分组
        
        row = row + 1
    Next i
    
    ' 自动调整列宽
    ws.Columns("B").AutoFit
    ws.Columns("C").ColumnWidth = 60
    ws.Columns("D").AutoFit
    
    MsgBox "生成完成！共 " & phoneList.Count & " 条记录", vbInformation
    
End Sub

Sub GenerateWithPrefix()
    ' 带区号选择的版本
    ' 在 A1 粘贴数据，A2 输入区号（+1 或 +852），然后运行
    
    Dim inputText As String
    Dim prefix As String
    Dim ws As Worksheet
    
    Set ws = ActiveSheet
    inputText = ws.Range("A1").Value
    prefix = ws.Range("A2").Value
    
    If inputText = "" Then
        MsgBox "请在 A1 单元格粘贴输入文本", vbExclamation
        Exit Sub
    End If
    
    ' 默认区号
    If prefix = "" Then prefix = "+1"
    
    ' 清理输入
    inputText = Replace(inputText, "h", " ")
    inputText = Trim(inputText)
    
    Dim items() As String
    items = Split(inputText, " ")
    
    Dim phoneList As Object
    Dim urlList As Object
    Set phoneList = CreateObject("System.Collections.ArrayList")
    Set urlList = CreateObject("System.Collections.ArrayList")
    
    Dim i As Integer, item As String
    For i = LBound(items) To UBound(items)
        item = Trim(items(i))
        If item <> "" Then
            If Left(item, 4) = "http" Then
                urlList.Add item
            ElseIf IsNumeric(item) And Len(item) = 10 Then
                phoneList.Add item
            End If
        End If
    Next i
    
    If phoneList.Count = 0 Then
        MsgBox "未找到有效的手机号", vbExclamation
        Exit Sub
    End If
    
    ' 生成结果
    ws.Range("B:E").ClearContents
    ws.Range("B1").Value = "编码"
    ws.Range("C1").Value = "内容"
    ws.Range("D1").Value = "分组"
    
    Dim row As Integer
    row = 2
    
    For i = 0 To phoneList.Count - 1
        Dim phone As String, url As String
        phone = phoneList(i)
        url = ""
        
        Dim j As Integer
        For j = 0 To urlList.Count - 1
            If InStr(urlList(j), phone) > 0 Then
                url = urlList(j)
                Exit For
            End If
        Next j
        
        If url = "" Then
            url = "http://card-manager-production-a1b5.up.railway.app/query?card=" & phone
        End If
        
        Dim content As String
        content = (i + 1) & "组手机号 " & phone & vbCrLf & _
                  "查看验证码 " & url & vbCrLf & _
                  "登陆流程:腾讯视频登录界面，把这个区号+86改为" & prefix & "(重点），输入手机号，获取验证码登陆"
        
        ws.Cells(row, 2).Value = i + 1
        ws.Cells(row, 3).Value = content
        ws.Cells(row, 4).Value = "手机号"
        row = row + 1
    Next i
    
    ws.Columns("B").AutoFit
    ws.Columns("C").ColumnWidth = 60
    ws.Columns("D").AutoFit
    
    MsgBox "生成完成！区号: " & prefix & "，共 " & phoneList.Count & " 条", vbInformation
End Sub