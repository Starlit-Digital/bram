param([Parameter(Mandatory=$true)][string]$BuildDll, [switch]$Apply)
$ErrorActionPreference='Stop'
if($env:COMPUTERNAME -ne 'WIN11'){throw 'WIN11 only'}
# Creates an exact hash supplement. Applying requires Chris's explicit approval.
$hash=(Get-FileHash -LiteralPath $BuildDll -Algorithm SHA256).Hash
$work=Join-Path $env:LOCALAPPDATA ("Bram/policies/"+$hash)
$scan=Join-Path $work files
New-Item -ItemType Directory -Force -Path $scan|Out-Null
Copy-Item -LiteralPath $BuildDll -Destination (Join-Path $scan BramWindows.dll) -Force
$xml=Join-Path $work BramWindows.xml
New-CIPolicy -FilePath $xml -Level Hash -ScanPath $scan -UserPEs -MultiplePolicyFormat | Out-Null
Set-CIPolicyIdInfo -FilePath $xml -PolicyName ('Bram test build '+$hash.Substring(0,12)) -SupplementsBasePolicyID '{0283ac0f-fff1-49ae-ada1-8a933130cad6}' -ResetPolicyID | Out-Null
Set-RuleOption -FilePath $xml -Option 3 -Delete
[xml]$policy=Get-Content $xml
# No signer, directory, wildcard or package trust is added.
foreach($allow in $policy.SiPolicy.FileRules.Allow){if(-not $allow.Hash){throw 'Expected hash only allow rules'}}
$cip=Join-Path $work BramWindows.cip
ConvertFrom-CIPolicy -XmlFilePath $xml -BinaryFilePath $cip | Out-Null
if($Apply){
 & "$env:windir/System32/CiTool.exe" --update-policy $cip -json
 if($LASTEXITCODE -ne 0){throw 'Policy installation failed'}
}
[ordered]@{sha256=$hash;policyId=[string]$policy.SiPolicy.PolicyID;xml=$xml;binary=$cip;applied=[bool]$Apply}|ConvertTo-Json
