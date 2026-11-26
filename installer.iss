; Instalator Wydarzenia Giełdowe (Inno Setup 6). Budowany w GitHub Actions: build-exe.yml
#ifndef MyVersion
  #define MyVersion "0.1"
#endif

[Setup]
AppId={{6F1C2A9E-4B7D-4E3A-9C55-1D2E8F0A7B31}
AppName=Wydarzenia Giełdowe
AppVersion={#MyVersion}
AppVerName=Wydarzenia Giełdowe {#MyVersion}
AppPublisher=PanRodla
AppCopyright=made by PanRodla · all rights reserved
DefaultDirName={localappdata}\Programs\WydarzeniaGieldowe
DisableProgramGroupPage=yes
DisableDirPage=yes
PrivilegesRequired=lowest
OutputDir=dist
OutputBaseFilename=WydarzeniaGieldowe-Setup
SetupIconFile=icon.ico
UninstallDisplayIcon={app}\WydarzeniaGieldowe.exe
UninstallDisplayName=Wydarzenia Giełdowe
AppMutex=PanRodla.WydarzeniaGieldowe
CloseApplications=yes
RestartApplications=no
WizardStyle=modern
Compression=lzma2
SolidCompression=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "pl"; MessagesFile: "compiler:Languages\Polish.isl"

[Tasks]
Name: "autostart"; Description: "Uruchamiaj razem z Windows (w tle, ikona przy zegarze)"
Name: "desktopicon"; Description: "Skrót na pulpicie"; Flags: unchecked

[Files]
Source: "WydarzeniaGieldowe.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
; AppUserModelID = ten sam identyfikator co w powiadomieniach programu: Windows łączy je ze skrótem
Name: "{autoprograms}\Wydarzenia Giełdowe"; Filename: "{app}\WydarzeniaGieldowe.exe"; AppUserModelID: "PanRodla.WydarzeniaGieldowe"
Name: "{autodesktop}\Wydarzenia Giełdowe"; Filename: "{app}\WydarzeniaGieldowe.exe"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "WydarzeniaGieldowe"; ValueData: """{app}\WydarzeniaGieldowe.exe"" --min"; Tasks: autostart; Flags: uninsdeletevalue
Root: HKCU; Subkey: "Software\Classes\AppUserModelId\PanRodla.WydarzeniaGieldowe"; Flags: uninsdeletekey dontcreatekey

[Run]
Filename: "{app}\WydarzeniaGieldowe.exe"; Description: "Uruchom Wydarzenia Giełdowe"; Flags: nowait postinstall skipifsilent
