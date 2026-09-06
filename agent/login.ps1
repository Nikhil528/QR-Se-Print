param([string]$DefaultServer='https://bvv-djql.onrender.com',[string]$DefaultShop='')
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$form=New-Object Windows.Forms.Form
$form.Text='QR Se Print - Setup';$form.Size=New-Object Drawing.Size(540,330);$form.StartPosition='CenterScreen';$form.MaximizeBox=$false
$h=New-Object Windows.Forms.Label;$h.Text='QR Se Print me aapka swagat hai';$h.Font=New-Object Drawing.Font('Segoe UI',18,[Drawing.FontStyle]::Bold);$h.Location=New-Object Drawing.Point(24,24);$h.AutoSize=$true;$form.Controls.Add($h)
$sub=New-Object Windows.Forms.Label;$sub.Text='Shuru karne ke liye apni Shop ID aur Password daalo';$sub.Location=New-Object Drawing.Point(26,62);$sub.AutoSize=$true;$form.Controls.Add($sub)
function field($text,$y,$value,$pass){$l=New-Object Windows.Forms.Label;$l.Text=$text;$l.Location=New-Object Drawing.Point(26,$y);$l.AutoSize=$true;$l.Font=New-Object Drawing.Font('Segoe UI',9,[Drawing.FontStyle]::Bold);$form.Controls.Add($l);$t=New-Object Windows.Forms.TextBox;$t.Location=New-Object Drawing.Point(26,($y+22));$t.Size=New-Object Drawing.Size(475,28);$t.Text=$value;if($pass){$t.UseSystemPasswordChar=$true};$form.Controls.Add($t);return $t}
$shop=field 'Shop ID' 92 $DefaultShop $false;$pw=field 'Password' 150 '' $true
$info=New-Object Windows.Forms.Label;$info.Text='Server: '+$DefaultServer;$info.Location=New-Object Drawing.Point(26,210);$info.AutoSize=$true;$info.ForeColor=[Drawing.Color]::Gray;$form.Controls.Add($info)
$ok=New-Object Windows.Forms.Button;$ok.Text='Shuru karo';$ok.Location=New-Object Drawing.Point(285,245);$ok.Size=New-Object Drawing.Size(110,38);$ok.DialogResult=[Windows.Forms.DialogResult]::OK;$form.AcceptButton=$ok;$form.Controls.Add($ok)
$cancel=New-Object Windows.Forms.Button;$cancel.Text='Cancel';$cancel.Location=New-Object Drawing.Point(405,245);$cancel.Size=New-Object Drawing.Size(95,38);$cancel.DialogResult=[Windows.Forms.DialogResult]::Cancel;$form.CancelButton=$cancel;$form.Controls.Add($cancel)
$r=$form.ShowDialog();if($r -eq [Windows.Forms.DialogResult]::OK){'SERVER='+$DefaultServer; 'SHOP='+$shop.Text.Trim(); 'PASSWORD='+$pw.Text}
