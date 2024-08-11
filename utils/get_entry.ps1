param(
    [Parameter()]
    [String]$kppath,
    [String]$dbpath,
    [String]$dbpwd,
    [String]$title=""
)

try {
    [System.Reflection.Assembly]::LoadFrom($kppath) | Out-Null
    [KeePass.Program]::CommonInitialize()
 
}
catch {
    Write-Output "LOAD_KEEPASS_ERR"
    exit 1
}

$ioc = [KeePassLib.Serialization.IOConnectionInfo]::FromPath($dbpath)

$ck = New-Object KeePassLib.Keys.CompositeKey
$kp = New-Object KeePassLib.Keys.KcpPassword @($dbpwd)
$ck.AddUserKey($kp)

$pd = New-Object KeePassLib.PwDatabase

try {
    $pd.Open($ioc, $ck, $null)
    if ($title -eq "") {
        Write-Output "CON_DB_SUCC"
        exit 1
    }
}
catch {
    Write-Output "DB_VALID_ERR"
    exit 1
}

$sp = New-Object KeePassLib.SearchParameters
$sp.SearchString = $title

$pl = New-Object KeePassLib.Collections.PwObjectList[KeePassLib.PwEntry]
$pd.RootGroup.SearchEntries($sp, $pl)

try {
    Write-Output $pl.GetAt(0).Strings.ReadSafe([KeePassLib.PwDefs]::UrlField)
    Write-Output $pl.GetAt(0).Strings.ReadSafe([KeePassLib.PwDefs]::UserNameField)
    Write-Output $pl.GetAt(0).Strings.ReadSafe([KeePassLib.PwDefs]::PasswordField)
}
catch {
    Write-Output "GET_ENTRY_ERR"
}

$pd.Close()

[KeePass.Program]::CommonTerminate()