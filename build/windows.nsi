!include "MUI2.nsh"
Name "CampusWay"
OutFile "CampusWay-Setup.exe"
InstallDir "$LOCALAPPDATA\Programs\CampusWay"
RequestExecutionLevel user
!define MUI_ICON "assets\campusway.ico"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"
Section
SetOutPath "$INSTDIR"
File "..\dist\campusway\CampusWay.exe"
File "..\LICENSE"
File /oname=ICON-LICENSE "..\gui\static\icons\LICENSE"
WriteUninstaller "$INSTDIR\Uninstall.exe"
CreateShortcut "$SMPROGRAMS\CampusWay.lnk" "$INSTDIR\CampusWay.exe"
WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\CampusWay" "DisplayName" "CampusWay"
WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\CampusWay" "UninstallString" '$"$INSTDIR\Uninstall.exe$"'
SectionEnd
Section "Uninstall"
Delete "$SMPROGRAMS\CampusWay.lnk"
Delete "$INSTDIR\CampusWay.exe"
Delete "$INSTDIR\LICENSE"
Delete "$INSTDIR\ICON-LICENSE"
Delete "$INSTDIR\Uninstall.exe"
RMDir "$INSTDIR"
DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\CampusWay"
SectionEnd
