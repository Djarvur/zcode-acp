[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$AppServer,

    [Parameter(Mandatory = $true)]
    [string]$Binary,

    [string]$LogRoot = "logs",

    [int]$TimeoutSeconds = 30
)

$ErrorActionPreference = "Stop"

function Get-DescendantProcessIds {
    param(
        [int[]]$RootProcessIds,
        [object[]]$Processes
    )

    $byParent = @{}
    foreach ($entry in $Processes) {
        if (-not $byParent.ContainsKey($entry.ParentProcessId)) {
            $byParent[$entry.ParentProcessId] = New-Object System.Collections.ArrayList
        }
        [void]$byParent[$entry.ParentProcessId].Add($entry.ProcessId)
    }

    $queue = New-Object System.Collections.Queue
    foreach ($processId in $RootProcessIds) {
        $queue.Enqueue([int]$processId)
    }

    $seen = @{}
    while ($queue.Count -gt 0) {
        $processId = [int]$queue.Dequeue()
        if ($seen.ContainsKey($processId)) {
            continue
        }
        $seen[$processId] = $true
        if ($byParent.ContainsKey($processId)) {
            foreach ($childId in $byParent[$processId]) {
                $queue.Enqueue([int]$childId)
            }
        }
    }

    return @($seen.Keys)
}

$appServer = (Resolve-Path -LiteralPath $AppServer).Path
$binary = (Resolve-Path -LiteralPath $Binary).Path
$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$logDir = Join-Path $LogRoot $stamp
New-Item -ItemType Directory -Path $logDir -Force | Out-Null

$startInfo = New-Object System.Diagnostics.ProcessStartInfo
$startInfo.FileName = $binary
$startInfo.WorkingDirectory = Split-Path -Parent $binary
$startInfo.UseShellExecute = $false
$startInfo.CreateNoWindow = $true
$startInfo.RedirectStandardInput = $true
$startInfo.RedirectStandardOutput = $true
$startInfo.RedirectStandardError = $true
$startInfo.Environment["ZCODE_ACP_APP_SERVER"] = $appServer
if ($env:ZCODE_ACP_API_KEY) {
    $startInfo.Environment["ZCODE_ACP_API_KEY"] = $env:ZCODE_ACP_API_KEY
}

$process = New-Object System.Diagnostics.Process
$process.StartInfo = $startInfo
$stdoutTask = $null
$stderrTask = $null
$responseLine = ""
$started = $false
$preExistingPids = @()
$trackedPids = @()

try {
    $preExistingPids = @(Get-CimInstance Win32_Process | Select-Object -ExpandProperty ProcessId)
    if (-not $process.Start()) {
        throw "Failed to start $binary"
    }
    $started = $true

    $stdoutTask = $process.StandardOutput.ReadLineAsync()
    $stderrTask = $process.StandardError.ReadToEndAsync()

    $request = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1}}'
    $process.StandardInput.WriteLine($request)
    $process.StandardInput.Flush()

    $delay = [System.Threading.Tasks.Task]::Delay([TimeSpan]::FromSeconds($TimeoutSeconds))
    $winner = [System.Threading.Tasks.Task]::WhenAny($stdoutTask, $delay).GetAwaiter().GetResult()
    if ($winner -ne $stdoutTask) {
        throw "Timed out waiting for the ACP initialize response"
    }

    $responseLine = $stdoutTask.GetAwaiter().GetResult()
    if ([string]::IsNullOrWhiteSpace($responseLine)) {
        throw "ACP server closed stdout before returning initialize"
    }

    $response = $responseLine | ConvertFrom-Json
    if ($null -ne $response.error) {
        throw "ACP initialize returned an error: $($response.error | ConvertTo-Json -Compress)"
    }
    if ([int]$response.id -ne 1) {
        throw "ACP response id = $($response.id), want 1"
    }
    if ([int]$response.result.protocolVersion -ne 1) {
        throw "ACP protocolVersion = $($response.result.protocolVersion), want 1"
    }
    if ($response.result.agentInfo.name -ne "zcode-acp") {
        throw "ACP agentInfo.name = $($response.result.agentInfo.name), want zcode-acp"
    }

    $liveProcesses = @(Get-CimInstance Win32_Process)
    $backendRoots = @($liveProcesses | Where-Object {
        $_.ExecutablePath -eq $appServer -and $preExistingPids -notcontains $_.ProcessId
    } | Select-Object -ExpandProperty ProcessId)
    if ($backendRoots.Count -eq 0) {
        throw "Could not identify the newly started backend process for $appServer"
    }
    $trackedPids = @(Get-DescendantProcessIds -RootProcessIds $backendRoots -Processes $liveProcesses)

    $process.StandardInput.Close()
    if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
        throw "ACP server did not shut down within $TimeoutSeconds seconds"
    }
    if ($process.ExitCode -ne 0) {
        throw "ACP server exit code = $($process.ExitCode), want 0"
    }

    $stderr = $stderrTask.GetAwaiter().GetResult()
    Set-Content -LiteralPath (Join-Path $logDir "stdout.jsonl") -Value $responseLine -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $logDir "stderr.log") -Value $stderr -Encoding UTF8

    if ($stderr -match "(?m)level=ERROR|exiting after failure") {
        throw "ACP stderr contains an error-level log; see $logDir\stderr.log"
    }

    $orphans = @()
    $deadline = (Get-Date).AddSeconds(5)
    do {
        $remainingProcesses = @(Get-CimInstance Win32_Process)
        $orphans = @($remainingProcesses | Where-Object { $trackedPids -contains $_.ProcessId })
        if ($orphans.Count -eq 0) {
            break
        }
        Start-Sleep -Milliseconds 100
    } while ((Get-Date) -lt $deadline)
    if ($orphans.Count -gt 0) {
        $details = $orphans | Select-Object ProcessId, ParentProcessId, ExecutablePath, CommandLine | Format-List | Out-String
        throw "Tracked backend processes remain:`n$details"
    }

    $summary = "PASS`ninitialize: protocolVersion=1 agent=zcode-acp`nexitCode=0`nlogDir=$logDir"
    Set-Content -LiteralPath (Join-Path $logDir "summary.txt") -Value $summary -Encoding UTF8
    Write-Host $summary
}
catch {
    $failure = $_ | Out-String
    if ($stderrTask -and $stderrTask.IsCompleted) {
        $failure += "`n--- stderr ---`n" + $stderrTask.GetAwaiter().GetResult()
    }
    Set-Content -LiteralPath (Join-Path $logDir "failure.txt") -Value $failure -Encoding UTF8
    throw
}
finally {
    if ($started -and -not $process.HasExited) {
        & taskkill.exe /PID $process.Id /T /F *> $null
    }
}
