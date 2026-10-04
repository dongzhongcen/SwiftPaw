Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them with the values from ProjectInfo.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows to use this project.nsi manually
## from outside of Wails for debugging and development of the installer.
##
## For development first make a wails nsis build to populate the "wails_tools.nsh":
## > wails build --target windows/amd64 --nsis
## Then you can call makensis on this file with specifying the path to your binary:
## For a AMD64 only installer:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## For a ARM64 only installer:
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## For a installer with both architectures:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## The following information is taken from the ProjectInfo file, but they can be overwritten here.
####
## !define INFO_PROJECTNAME    "MyProject" # Default "{{.Name}}"
## !define INFO_COMPANYNAME    "MyCompany" # Default "{{.Info.CompanyName}}"
## !define INFO_PRODUCTNAME    "MyProduct" # Default "{{.Info.ProductName}}"
## !define INFO_PRODUCTVERSION "1.0.0"     # Default "{{.Info.ProductVersion}}"
## !define INFO_COPYRIGHT      "Copyright" # Default "{{.Info.Copyright}}"
###
## !define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
## !define UNINST_KEY_NAME     "UninstKeyInRegistry"  # Default "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
####
## !define REQUEST_EXECUTION_LEVEL "admin"            # Default "admin"  see also https://nsis.sourceforge.io/Docs/Chapter4.html
####
## Include the wails tools
####
!include "wails_tools.nsh"

####
## 极拍 SwiftPaw 自己加的部分：安装前检查旧版本（升级 / 重新安装 / 降级都会先问一下）
## 用到的 LogicLib、WordFunc、FileFunc 都是 NSIS 自带的头文件，不需要额外下载插件
####
!include "LogicLib.nsh"  # ${If} ... ${EndIf} 这类写法
!include "WordFunc.nsh"  # ${VersionCompare}：比较两个版本号
!include "FileFunc.nsh"  # ${GetParent}、${GetParameters}、${GetOptions}

# 主窗口的标题，要和 main.go 里 options.App 的 Title 一致，用来判断程序是不是正在运行
!define APP_WINDOW_TITLE "极拍 SwiftPaw"

Var OldVersion     # 已安装的版本号，比如 1.0.0；没装过时为空
Var OldInstDir     # 已安装的文件夹
Var OldUninstaller # 已安装版本的卸载程序（uninstall.exe）的完整路径
Var UpgradeOld     # 为 1 时表示用户同意替换旧版本，开始安装时要先卸载旧版本

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" #Include this to add a bitmap on the left side of the Welcome Page. Must be a size of 164x314
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # Adds a EULA page to the installer
!define MUI_PAGE_CUSTOMFUNCTION_PRE SkipDirectoryOnUpgrade # 升级时装回原来的文件夹，不显示选择文件夹的页面
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uinstalling page

!insertmacro MUI_LANGUAGE "English" # Set the Language of the installer

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
!ifdef WAILS_INSTALL_SCOPE
  !if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
  !else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
  !endif
!else
  InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!endif # Default installing folder ($PROGRAMFILES is Program Files folder).
ShowInstDetails show # This will always show the installation details.

# 把安装文件夹也写进“卸载程序”注册表项，下次升级时直接读 InstallLocation，不用再从 UninstallString 里拆。
# 写到哪个根键（HKLM / HKCU）和 wails.writeUninstaller 保持一致
!macro swiftpaw.writeInstallLocation
    SetRegView 64
    !ifdef WAILS_INSTALL_SCOPE
      !if "${WAILS_INSTALL_SCOPE}" == "user"
        WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
      !else
        WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
      !endif
    !else
        WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
    !endif
!macroend

Function .onInit
   !insertmacro wails.checkArchitecture
   Call FindOldInstall   # 1. 看看注册表里有没有已安装的版本
   Call CheckRunning     # 2. 程序正在运行时请用户先关掉
   Call AskReplaceOld    # 3. 有旧版本时问用户要不要替换
FunctionEnd

# FindOldInstall 从“卸载程序”注册表项里读出已安装的版本号和文件夹。
# Wails 的模板按安装方式写在 HKLM（给所有用户安装，默认）或 HKCU（只给当前用户安装）下，两个都看一下。
# 读到的结果放在 $OldVersion、$OldInstDir、$OldUninstaller 里；没装过时 $OldUninstaller 为空
Function FindOldInstall
    SetRegView 64 # Wails 写注册表时用的是 64 位视图，读的时候也要一样
    StrCpy $OldUninstaller ""

    ReadRegStr $R0 HKLM "${UNINST_KEY}" "UninstallString"
    ${If} $R0 != ""
        ReadRegStr $OldVersion HKLM "${UNINST_KEY}" "DisplayVersion"
        ReadRegStr $R1 HKLM "${UNINST_KEY}" "InstallLocation"
    ${Else}
        ReadRegStr $R0 HKCU "${UNINST_KEY}" "UninstallString"
        ReadRegStr $OldVersion HKCU "${UNINST_KEY}" "DisplayVersion"
        ReadRegStr $R1 HKCU "${UNINST_KEY}" "InstallLocation"
    ${EndIf}
    ${If} $R0 == ""
        Return # 没有安装过
    ${EndIf}

    ${If} $R1 != ""
        # 1.0.1 起会写 InstallLocation，直接用
        StrCpy $OldInstDir $R1
        StrCpy $OldUninstaller "$R1\uninstall.exe"
    ${Else}
        # 1.0.0 只写了 UninstallString，样子是 "C:\...\uninstall.exe"（带引号），去掉引号后取它所在的文件夹
        StrCpy $R2 ""   # 去掉引号后的结果
        StrLen $R3 $R0  # 字符串长度
        StrCpy $R4 0    # 当前位置
        ${While} $R4 < $R3
            StrCpy $R5 $R0 1 $R4 # 取第 $R4 个字符
            ${If} $R5 != '"'
                StrCpy $R2 "$R2$R5"
            ${EndIf}
            IntOp $R4 $R4 + 1
        ${EndWhile}
        StrCpy $OldUninstaller $R2
        ${GetParent} $R2 $OldInstDir
    ${EndIf}

    # 注册表里有记录，但卸载程序已经不在了（比如文件夹被手动删掉），就当作没装过
    ${IfNot} ${FileExists} "$OldUninstaller"
        StrCpy $OldUninstaller ""
    ${EndIf}
FunctionEnd

# CheckRunning 检查极拍 SwiftPaw 是不是正在运行，运行中的程序文件没法被替换。
# 两种办法：按窗口标题找主窗口；或者试着用写入方式打开旧的 SwiftPaw.exe（正在运行时 Windows 不允许）
Function CheckRunning
    ${Do}
        StrCpy $R1 0 # 为 1 表示正在运行
        FindWindow $R0 "" "${APP_WINDOW_TITLE}"
        ${If} $R0 != 0
            StrCpy $R1 1
        ${ElseIf} $OldUninstaller != ""
        ${AndIf} ${FileExists} "$OldInstDir\${PRODUCT_EXECUTABLE}"
            ClearErrors
            FileOpen $R2 "$OldInstDir\${PRODUCT_EXECUTABLE}" a # 只是打开再关上，不会改文件内容
            ${If} ${Errors}
                StrCpy $R1 1
            ${Else}
                FileClose $R2
            ${EndIf}
        ${EndIf}

        ${If} $R1 == 0
            Return # 没在运行，继续安装
        ${EndIf}
        ${If} ${Silent}
            SetErrorLevel 2 # 静默安装时没法提示，直接退出
            Quit
        ${EndIf}
        ${IfNot} ${Cmd} `MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION "极拍 SwiftPaw 正在运行。$\r$\n$\r$\n请先关闭它，然后点「重试」继续安装。" IDRETRY`
            Quit
        ${EndIf}
    ${Loop}
FunctionEnd

# AskReplaceOld 有旧版本时按版本号的新旧问用户：升级、重新安装相同版本、降级。
# 点“是”就记下来，真正的卸载在开始安装时才做（见 UninstallOld），中途退出安装程序不会动旧版本；点“否”直接退出
Function AskReplaceOld
    ${If} $OldUninstaller == ""
        Return # 没有旧版本，正常安装
    ${EndIf}
    ${If} $OldVersion == ""
        StrCpy $OldVersion "旧版本" # 注册表里没有版本号时，提示里显示“旧版本”
        StrCpy $R0 2
    ${Else}
        # $R0：0 表示版本相同，1 表示已安装的更新，2 表示已安装的更旧
        ${VersionCompare} "$OldVersion" "${INFO_PRODUCTVERSION}" $R0
    ${EndIf}

    ${IfNot} ${Silent} # 静默安装（/S）时不提示，直接替换
        ${If} $R0 == 0
            ${IfNot} ${Cmd} `MessageBox MB_YESNO|MB_ICONQUESTION "已安装相同版本，是否重新安装？$\r$\n$\r$\n你的歌单、收藏和插件设置会保留。" IDYES`
                Quit
            ${EndIf}
        ${ElseIf} $R0 == 1
            ${IfNot} ${Cmd} `MessageBox MB_YESNO|MB_ICONEXCLAMATION|MB_DEFBUTTON2 "已安装更新的版本 $OldVersion，继续会降级，是否继续？$\r$\n$\r$\n你的歌单、收藏和插件设置会保留。" IDYES`
                Quit
            ${EndIf}
        ${Else}
            ${IfNot} ${Cmd} `MessageBox MB_YESNO|MB_ICONQUESTION "检测到已安装 极拍 SwiftPaw $OldVersion，是否替换为 ${INFO_PRODUCTVERSION}？$\r$\n$\r$\n你的歌单、收藏和插件设置会保留。" IDYES`
                Quit
            ${EndIf}
        ${EndIf}
    ${EndIf}

    StrCpy $UpgradeOld 1
    StrCpy $INSTDIR $OldInstDir # 装回原来的文件夹
FunctionEnd

# SkipDirectoryOnUpgrade 是选择文件夹页面的“显示前”回调：升级时调用 Abort 跳过这个页面
Function SkipDirectoryOnUpgrade
    ${If} $UpgradeOld == 1
        Abort
    ${EndIf}
FunctionEnd

# UninstallOld 在复制新文件之前，静默运行旧版本的卸载程序。
# /S 表示静默卸载；_?=文件夹 让卸载程序直接在原地运行，并且等它运行完才返回（NSIS 的常用写法）。
# /UPGRADE 告诉 1.0.1 起的卸载程序这是升级，不要删 WebView2 的缓存（1.0.0 的卸载程序会忽略这个参数）。
# 用户数据（歌单、收藏、插件和插件设置、config.json、window.json）在 %AppData%\SwiftPaw，不在安装文件夹里，卸载程序不会删
Function UninstallOld
    ${If} $UpgradeOld != 1
        Return
    ${EndIf}
    DetailPrint "正在卸载旧版本 $OldVersion ..."
    ClearErrors
    ExecWait '"$OldUninstaller" /S /UPGRADE _?=$OldInstDir' $R0
    ${If} ${Errors}
    ${OrIf} $R0 != 0
        ${IfNot} ${Cmd} `MessageBox MB_OKCANCEL|MB_ICONEXCLAMATION "旧版本没能自动卸载（返回值：$R0）。$\r$\n$\r$\n点「确定」直接覆盖安装，点「取消」停止安装。" IDOK`
            Abort "已取消安装" # 安装过程中要用 Abort 停止，Quit 只适合在 .onInit 里用
        ${EndIf}
    ${EndIf}
    # 用 _?= 运行时，卸载程序删不掉它自己，这里帮它删掉；新的卸载程序稍后会重新写入
    Delete "$OldUninstaller"
FunctionEnd

Section
    !insertmacro wails.setShellContext

    Call UninstallOld # 用户同意替换旧版本时，先静默卸载旧版本

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    !insertmacro wails.files

    # 许可证文件放在 SwiftPaw.exe 旁边：本项目的 MIT 许可证、第三方开源软件的许可证、界面字体的许可证。
    # 路径相对于这个 .nsi 文件所在的 build/windows/installer 文件夹
    File "..\..\..\LICENSE"
    File "..\..\..\THIRD_PARTY_NOTICES.md"
    File "..\..\..\frontend\src\assets\fonts\OFL.txt"

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    !insertmacro wails.writeUninstaller
    !insertmacro swiftpaw.writeInstallLocation
SectionEnd


Section "uninstall"
    !insertmacro wails.setShellContext

    # 新版本安装程序升级时会带上 /UPGRADE 参数：这时保留 WebView2 的缓存，只有真正卸载时才删
    ${GetParameters} $R0
    ClearErrors
    ${GetOptions} $R0 "/UPGRADE" $R1
    ${If} ${Errors} # 没有 /UPGRADE，是用户自己卸载
        RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath
    ${EndIf}

    # 注意：用户数据在 %AppData%\SwiftPaw（歌单数据库、插件、插件设置、config.json、window.json），
    # 不在安装文件夹里，下面只删安装文件夹，不会删用户数据。
    # 万一有人把程序装进了数据文件夹（里面有 swiftpaw.db），就只删程序本身，不整个删掉
    ${If} ${FileExists} "$INSTDIR\swiftpaw.db"
        Delete "$INSTDIR\${PRODUCT_EXECUTABLE}"
        Delete "$INSTDIR\LICENSE"
        Delete "$INSTDIR\THIRD_PARTY_NOTICES.md"
        Delete "$INSTDIR\OFL.txt"
    ${Else}
        RMDir /r $INSTDIR
    ${EndIf}

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
