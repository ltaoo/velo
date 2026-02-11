$ErrorActionPreference = 'Stop'
$root = Get-Location
$installer = Join-Path $env:TEMP 'MicrosoftEdgeWebview2Setup.exe'
Invoke-WebRequest -Uri 'https://go.microsoft.com/fwlink/?linkid=2124701' -OutFile $installer
Start-Process -FilePath $installer -ArgumentList '/silent','/install' -Wait -NoNewWindow
Remove-Item $installer -Force
$zip = Join-Path $env:TEMP 'Microsoft.Web.WebView2.zip'
Invoke-WebRequest -Uri 'https://www.nuget.org/api/v2/package/Microsoft.Web.WebView2' -OutFile $zip
$dest = Join-Path $env:TEMP 'webview2_pkg'
if (Test-Path $dest) { Remove-Item $dest -Recurse -Force }
Expand-Archive -Path $zip -DestinationPath $dest -Force
$dllCandidates = @(
  (Join-Path $dest 'build/native/x64/WebView2Loader.dll'),
  (Join-Path $dest 'build/native/arm64/WebView2Loader.dll')
)
$dll = $dllCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (!$dll) { throw 'WebView2Loader.dll not found in extracted package' }
Copy-Item $dll -Destination (Join-Path $root 'WebView2Loader.dll') -Force
$env:CGO_ENABLED = 1
$exe = Join-Path $root 'wx_video_download.exe'
go build -o $exe
& $exe
