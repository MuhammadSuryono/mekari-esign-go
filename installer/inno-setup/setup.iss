; Mekari E-Sign Service Installer
; Inno Setup Script
; Download Inno Setup from: https://jrsoftware.org/isinfo.php

#define MyAppName "Mekari E-Sign Service"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "MyBoost"
#define MyAppURL "https://github.com/muhammadsuryono/mekari-esign-go"
#define MyAppExeName "mekari-esign.exe"

[Setup]
; Unique identifier for this application
AppId={{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}/releases
DefaultDirName={autopf}\MekariEsign
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
; Output settings
OutputDir=..\..\dist
OutputBaseFilename=MekariEsignSetup-{#MyAppVersion}
; Compression settings
Compression=lzma2/ultra64
SolidCompression=yes
; UI settings
WizardStyle=modern
WizardImageFile=compiler:WizModernImage.bmp
WizardSmallImageFile=compiler:WizModernSmallImage.bmp
; Require admin privileges for service installation
PrivilegesRequired=admin
PrivilegesRequiredOverridesAllowed=dialog
; 64-bit installation
ArchitecturesInstallIn64BitMode=x64
ArchitecturesAllowed=x64
; Enable logging
SetupLogging=yes
; Uninstall settings
UninstallDisplayIcon={app}\{#MyAppExeName}
UninstallDisplayName={#MyAppName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Types]
Name: "full"; Description: "Full installation (recommended)"
Name: "compact"; Description: "Compact installation (service only)"
Name: "custom"; Description: "Custom installation"; Flags: iscustom

[Components]
Name: "main"; Description: "Mekari E-Sign Service"; Types: full compact custom; Flags: fixed
Name: "redis"; Description: "Redis Server (embedded)"; Types: full
Name: "postgres"; Description: "PostgreSQL Server (embedded)"; Types: full
Name: "tools"; Description: "Management Tools"; Types: full

[Files]
; Main application
Source: "..\..\bin\windows\mekari-esign.exe"; DestDir: "{app}"; Flags: ignoreversion; Components: main

; Configuration file (only copy if doesn't exist)
Source: "..\..\config.example.yml"; DestDir: "{app}"; DestName: "config.yml"; Flags: onlyifdoesntexist; Components: main

; Redis Portable
Source: "..\..\embedded\redis-win\*"; DestDir: "{app}\redis"; Flags: ignoreversion recursesubdirs createallsubdirs; Components: redis

; PostgreSQL Portable  
Source: "..\..\embedded\pgsql-portable\*"; DestDir: "{app}\pgsql"; Flags: ignoreversion recursesubdirs createallsubdirs; Components: postgres

; NSSM (Non-Sucking Service Manager)
Source: "..\..\tools\nssm.exe"; DestDir: "{app}\tools"; Flags: ignoreversion; Components: redis postgres

; Helper scripts
Source: "..\..\installer\scripts\*"; DestDir: "{app}\scripts"; Flags: ignoreversion; Components: main

[Dirs]
; Create data directories
Name: "{app}\data"; Components: main
Name: "{app}\data\postgres"; Components: postgres
Name: "{app}\data\redis"; Components: redis
Name: "{app}\logs"; Components: main
Name: "{app}\documents"; Components: main
Name: "{app}\documents\ready"; Components: main
Name: "{app}\documents\progress"; Components: main
Name: "{app}\documents\finish"; Components: main
Name: "{app}\.backup"; Components: main

[Icons]
; Start menu shortcuts
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Parameters: "-debug"; WorkingDir: "{app}"
Name: "{group}\Configuration"; Filename: "notepad.exe"; Parameters: """{app}\config.yml"""
Name: "{group}\Logs Folder"; Filename: "{app}\logs"
Name: "{group}\Start Service"; Filename: "{app}\{#MyAppExeName}"; Parameters: "-start"; Flags: runminimized
Name: "{group}\Stop Service"; Filename: "{app}\{#MyAppExeName}"; Parameters: "-stop"; Flags: runminimized
Name: "{group}\Check for Updates"; Filename: "{app}\{#MyAppExeName}"; Parameters: "-update"
Name: "{group}\Uninstall {#MyAppName}"; Filename: "{uninstallexe}"

[Run]
; Post-installation tasks

; Initialize PostgreSQL database (if component selected)
Filename: "{app}\scripts\init-postgres.bat"; Parameters: """{app}"""; StatusMsg: "Initializing PostgreSQL database..."; Flags: runhidden waituntilterminated; Components: postgres

; Install Redis service (if component selected)
Filename: "{app}\scripts\install-redis-service.bat"; Parameters: """{app}"""; StatusMsg: "Installing Redis service..."; Flags: runhidden waituntilterminated; Components: redis

; Install PostgreSQL service (if component selected)
Filename: "{app}\scripts\install-postgres-service.bat"; Parameters: """{app}"""; StatusMsg: "Installing PostgreSQL service..."; Flags: runhidden waituntilterminated; Components: postgres

; Start Redis service
Filename: "net.exe"; Parameters: "start MekariRedis"; StatusMsg: "Starting Redis..."; Flags: runhidden waituntilterminated; Components: redis

; Start PostgreSQL service
Filename: "net.exe"; Parameters: "start MekariPostgres"; StatusMsg: "Starting PostgreSQL..."; Flags: runhidden waituntilterminated; Components: postgres

; Wait for databases to start
Filename: "timeout.exe"; Parameters: "/t 5 /nobreak"; StatusMsg: "Waiting for databases..."; Flags: runhidden waituntilterminated; Components: postgres

; Create database
Filename: "{app}\scripts\create-database.bat"; Parameters: """{app}"""; StatusMsg: "Creating application database..."; Flags: runhidden waituntilterminated; Components: postgres

; Install main service
Filename: "{app}\{#MyAppExeName}"; Parameters: "-install"; StatusMsg: "Installing Mekari E-Sign service..."; Flags: runhidden waituntilterminated; Components: main

; Schedule auto-update task
Filename: "schtasks.exe"; Parameters: "/create /tn ""MekariEsignUpdater"" /tr """"""{app}\{#MyAppExeName}"" -update"" /sc daily /st 03:00 /f"; StatusMsg: "Setting up auto-update schedule..."; Flags: runhidden waituntilterminated; Components: main

; Open configuration file for editing
Filename: "notepad.exe"; Parameters: """{app}\config.yml"""; Description: "Edit configuration file"; Flags: postinstall nowait skipifsilent unchecked

; Start service
Filename: "{app}\{#MyAppExeName}"; Parameters: "-start"; Description: "Start service now"; Flags: postinstall nowait skipifsilent

[UninstallRun]
; Stop and uninstall services in reverse order
Filename: "{app}\{#MyAppExeName}"; Parameters: "-stop"; Flags: runhidden waituntilterminated
Filename: "{app}\{#MyAppExeName}"; Parameters: "-uninstall"; Flags: runhidden waituntilterminated
Filename: "net.exe"; Parameters: "stop MekariPostgres"; Flags: runhidden waituntilterminated
Filename: "net.exe"; Parameters: "stop MekariRedis"; Flags: runhidden waituntilterminated
Filename: "{app}\tools\nssm.exe"; Parameters: "remove MekariPostgres confirm"; Flags: runhidden waituntilterminated
Filename: "{app}\tools\nssm.exe"; Parameters: "remove MekariRedis confirm"; Flags: runhidden waituntilterminated
; Remove scheduled task
Filename: "schtasks.exe"; Parameters: "/delete /tn ""MekariEsignUpdater"" /f"; Flags: runhidden waituntilterminated

[UninstallDelete]
Type: filesandordirs; Name: "{app}\logs"
Type: filesandordirs; Name: "{app}\.backup"
Type: dirifempty; Name: "{app}\data"

[Code]
var
  ConfigPage: TInputQueryWizardPage;
  PortPage: TInputQueryWizardPage;

procedure InitializeWizard;
begin
  // Database configuration page
  ConfigPage := CreateInputQueryPage(wpSelectComponents,
    'Database Configuration',
    'Configure database settings',
    'Please enter the database configuration. You can change these later in config.yml');
  ConfigPage.Add('PostgreSQL Password:', False);
  ConfigPage.Add('Redis Password (optional):', True);
  ConfigPage.Values[0] := 'postgres123';
  ConfigPage.Values[1] := '';

  // Port configuration page
  PortPage := CreateInputQueryPage(ConfigPage.ID,
    'Service Configuration',
    'Configure service port',
    'Enter the port number for the E-Sign service:');
  PortPage.Add('HTTP Port:', False);
  PortPage.Values[0] := '8080';
end;

function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := False;
  // Skip database config if postgres not selected
  if PageID = ConfigPage.ID then
    Result := not WizardIsComponentSelected('postgres');
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  ConfigFile: String;
  Lines: TArrayOfString;
  I: Integer;
  AppPort: String;
  PgPassword: String;
begin
  if CurStep = ssPostInstall then
  begin
    // Update configuration file with user-provided values
    ConfigFile := ExpandConstant('{app}\config.yml');
    
    if LoadStringsFromFile(ConfigFile, Lines) then
    begin
      AppPort := PortPage.Values[0];
      if WizardIsComponentSelected('postgres') then
        PgPassword := ConfigPage.Values[0]
      else
        PgPassword := '';
      
      // Update values in config
      for I := 0 to GetArrayLength(Lines) - 1 do
      begin
        // Update port
        if (AppPort <> '') and (Pos('port: 8080', Lines[I]) > 0) then
          Lines[I] := '  port: ' + AppPort;
        
        // Update PostgreSQL password  
        if (PgPassword <> '') and (Pos('password: "your_password"', Lines[I]) > 0) then
          Lines[I] := '  password: "' + PgPassword + '"';
      end;
      
      // Save back to file
      SaveStringsToFile(ConfigFile, Lines, False);
    end;
  end;
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
begin
  Result := '';
  // Stop existing services before upgrading
  Exec(ExpandConstant('{sys}\net.exe'), 'stop MekariEsign', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec(ExpandConstant('{sys}\net.exe'), 'stop MekariRedis', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec(ExpandConstant('{sys}\net.exe'), 'stop MekariPostgres', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  // Wait a bit for services to stop
  Sleep(2000);
end;

function InitializeSetup(): Boolean;
begin
  Result := True;
  // Check if running as admin
  if not IsAdmin then
  begin
    MsgBox('This installer requires administrator privileges. Please run as Administrator.', mbError, MB_OK);
    Result := False;
  end;
end;

