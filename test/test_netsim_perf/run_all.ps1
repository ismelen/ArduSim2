$intervals = @(1, 5, 10, 15, 20)
$configFile = "resources\netsim\config.json"
$logFile = "run_all_results.log"
Clear-Content $logFile -ErrorAction SilentlyContinue

foreach ($val in $intervals) {
    Write-Host "Running for flush_interval_ms = $val"
    Add-Content $logFile "========== FLUSH INTERVAL = $val ms =========="
    
    $content = Get-Content $configFile
    $content = $content -replace '"flush_interval_ms": \d+', "`"flush_interval_ms`": $val"
    Set-Content $configFile $content

    docker-compose up --build --abort-on-container-exit > "compose_$val.log" 2>&1
    
    # Extract results
    $results = Select-String -Path "compose_$val.log" -Pattern "tester-1   \| \d+" | Select-Object -ExpandProperty Line
    Add-Content $logFile $results
    Add-Content $logFile "`n"
}
Write-Host "All done!"
