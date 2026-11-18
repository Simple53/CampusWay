!include "MUI2.nsh"
Name "RouteDesk"
OutFile "RouteDesk-Setup.exe"
InstallDir "$LOCALAPPDATA\Programs\RouteDesk"
RequestExecutionLevel user
!define MUI_ICON "assets\campusway.ico"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"
Section
SetOutPath "$INSTDIR"
File "..\dist\routedesk\RouteDesk.exe"
File "..\LICENSE"
File /oname=ICON-LICENSE "..\gui\static\icons\LICENSE"
WriteUninstaller "$INSTDIR\Uninstall.exe"
CreateShortcut "$SMPROGRAMS\RouteDesk.lnk" "$INSTDIR\RouteDesk.exe"
WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RouteDesk" "DisplayName" "RouteDesk"
WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RouteDesk" "UninstallString" '$"$INSTDIR\Uninstall.exe$"'
SectionEnd
Section "Uninstall"
Delete "$SMPROGRAMS\RouteDesk.lnk"
Delete "$INSTDIR\RouteDesk.exe"
Delete "$INSTDIR\LICENSE"
Delete "$INSTDIR\ICON-LICENSE"
Delete "$INSTDIR\Uninstall.exe"
RMDir "$INSTDIR"
DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\RouteDesk"
SectionEnd
