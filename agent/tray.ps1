Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$base='http://127.0.0.1:17845'
$ni=New-Object System.Windows.Forms.NotifyIcon;$ni.Icon=[System.Drawing.SystemIcons]::Application;$ni.Text='QR Se Print — Print Agent';$ni.Visible=$true
$menu=New-Object System.Windows.Forms.ContextMenuStrip
function Add-Item($text,$action){$i=$menu.Items.Add($text);$i.Add_Click($action);return $i}
Add-Item 'Status: Running — waiting for jobs' {} | Out-Null
Add-Item 'Shop: loading...' {} | Out-Null
Add-Item 'Printer: loading...' {} | Out-Null
$menu.Items.Add('-')|Out-Null
Add-Item '⚡ Change Demo ID to Paid Shop' {Start-Process 'https://bvv-djql.onrender.com/'}|Out-Null
Add-Item '⚙ Settings' {try{Invoke-WebRequest "$base/open" -UseBasicParsing|Out-Null}catch{}}|Out-Null
Add-Item '↻ Reconnect to Server' {try{Invoke-WebRequest "$base/reconnect" -Method POST -UseBasicParsing|Out-Null}catch{}}|Out-Null
$counter=$false
$counterItem=Add-Item '🔔 Counter Approval: OFF' {}
$counterItem.Add_Click({$script:counter=-not $script:counter;$counterItem.Text=if($script:counter){'🔔 Counter Approval: ON'}else{'🔔 Counter Approval: OFF'};try{Invoke-WebRequest "$base/counter" -Method POST -ContentType 'application/json' -Body (@{enabled=$script:counter}|ConvertTo-Json) -UseBasicParsing|Out-Null}catch{}})
Add-Item '▤ View Logs' {try{Invoke-WebRequest "$base/logs" -UseBasicParsing|Out-Null}catch{}}|Out-Null
Add-Item '♙ Contact Admin' {Start-Process 'https://bvv-djql.onrender.com/'}|Out-Null
Add-Item '↑ Check for Update' {try{Invoke-WebRequest "$base/open" -UseBasicParsing|Out-Null}catch{}}|Out-Null
Add-Item '▣ Change Shop ID' {try{Invoke-WebRequest "$base/change-shop" -Method POST -UseBasicParsing|Out-Null}catch{}}|Out-Null
$menu.Items.Add('-')|Out-Null
Add-Item '✕ Exit' {$ni.Visible=$false;[System.Windows.Forms.Application]::Exit()}|Out-Null
$ni.ContextMenuStrip=$menu;$ni.Add_DoubleClick({try{Invoke-WebRequest "$base/open" -UseBasicParsing|Out-Null}catch{}});[Windows.Forms.Application]::Run()
