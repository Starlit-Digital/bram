param([Parameter(Mandatory=$true)][string]$Bundle, [Parameter(Mandatory=$true)][string]$SourceBundle, [switch]$ApproveBuildHash)
$ErrorActionPreference='Stop'
if ($env:COMPUTERNAME -ne 'WIN11') { throw 'This installation is qualified for WIN11 only.' }
$base=Join-Path $env:LOCALAPPDATA 'Bram'
$stamp=Get-Date -Format 'yyyyMMddHHmmss'
$release=Join-Path $base "releases/$stamp"
New-Item -ItemType Directory -Force -Path $release | Out-Null
Expand-Archive -LiteralPath $Bundle -DestinationPath $release
$exe=Join-Path $release 'BramWindows.exe'
if(-not(Test-Path $exe)){throw 'Missing app executable'}
$source=Join-Path $env:USERPROFILE 'BramWindows/source'
if(Test-Path $source){Move-Item $source ($source+'.backup.'+$stamp)}
New-Item -ItemType Directory -Force -Path $source | Out-Null
Expand-Archive -LiteralPath $SourceBundle -DestinationPath $source
# Limit inbound app traffic to the Private profile and the local subnet.
foreach($name in @('Bram Windows TCP','Bram Windows mDNS')){
 $old=Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue
 if($old){$old | Remove-NetFirewallRule}
}
New-NetFirewallRule -DisplayName 'Bram Windows TCP' -Direction Inbound -Action Allow -Program $exe -Protocol TCP -LocalPort 47831 -Profile Private -RemoteAddress LocalSubnet | Out-Null
New-NetFirewallRule -DisplayName 'Bram Windows mDNS' -Direction Inbound -Action Allow -Program $exe -Protocol UDP -LocalPort 5353 -Profile Private -RemoteAddress LocalSubnet | Out-Null
$user=[Security.Principal.WindowsIdentity]::GetCurrent().Name
$action=New-ScheduledTaskAction -Execute $exe -Argument 'tray' -WorkingDirectory $release
$trigger=New-ScheduledTaskTrigger -AtLogOn -User $user
$principal=New-ScheduledTaskPrincipal -UserId $user -LogonType Interactive -RunLevel Limited
$settings=New-ScheduledTaskSettingsSet -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
$previous=Get-ScheduledTask -TaskName 'Bram Windows' -ErrorAction SilentlyContinue
if($previous){Export-ScheduledTask -TaskName 'Bram Windows' | Set-Content (Join-Path $base ('task.backup.'+$stamp+'.xml')); Stop-ScheduledTask -TaskName 'Bram Windows'}
Register-ScheduledTask -TaskName 'Bram Windows' -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Description 'Bram tray relay at sign in for cs' -Force | Out-Null
$receipt=[ordered]@{installed=[DateTime]::UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ");machine=$env:COMPUTERNAME;exe=$exe;source=$source;sha256=(Get-FileHash $exe -Algorithm SHA256).Hash;assemblySha256=(Get-FileHash (Join-Path $release 'BramWindows.dll') -Algorithm SHA256).Hash;task='Bram Windows';framework='self contained .NET 10';firewallProfile='Private';remoteScope='LocalSubnet'}
$receipt|ConvertTo-Json | Set-Content -Encoding UTF8 (Join-Path $base 'install-info.json')
if($ApproveBuildHash){ & (Join-Path $source 'approve-build.ps1') -BuildDll (Join-Path $release 'BramWindows.dll') -Apply }
Start-ScheduledTask -TaskName 'Bram Windows' 
$receipt|ConvertTo-Json
