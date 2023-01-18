<#
.SYNOPSIS
    Prepare the GoWSL repository so you can run the examples and tests.
    You will be prompted to install WSL and Ubuntu from the Microsoft store.
    Winget must be available.
.PARAMETER AcceptAll
    Enable this flag to automatically accept package and source agreements for
    these dependencies. This will make the install non-interactive.
.EXAMPLE
    .\prepare-repository.ps1

    Prepare this repo in your machine.
.EXAMPLE
    powershell -ExecutionPolicy Bypass .\prepare-repository.ps1

    You may have script execution disallowed. This command will bypass it.
.EXAMPLE
    .\prepare-repository.ps1 -AcceptAll

    Prepare this repo non-interactively.
#>

param (
    [switch]$AcceptAll = $false
)

$acceptance = ""
if ( $AcceptAll ) {
    $acceptance = "--accept-package-agreements","--accept-source-agreements"
}

function Test-Winget {
    &{ winget --version } 2>&1 | Out-Null
    if ( $LASTEXITCODE -eq "0" ) { return $true }
    Write-Error "Winget is not installed. Please install it from https://learn.microsoft.com/en-us/windows/package-manager/winget/"
    return $false
}

# Gettting WSL
if ( ! $(Get-AppPackage | Where-Object Name -like 'MicrosoftCorporationII.WindowsSubsystemForLinux') ) {
    if (! $(Test-Winget) ) { Exit(1) }
    Write-Output "Installing WSL"
    winget install --name 'Windows Subsystem for Linux' --silent ${acceptance}
    if ( ! $? ) { Exit(1) }
}
Write-Output "WSL is installed"

# Gettting Ubuntu
if ( $(Get-AppPackage | Where-Object Name -like 'CanonicalGroupLimited.Ubuntu').Count -eq 0 ) {
    if (! $(Test-Winget) ) { Exit(1) }
    Write-Output "Installing Ubuntu"
    winget install --Id '9PDXGNCFSCZV' --silent ${acceptance}
    #                    ^~~~~~~~~~~~
    # If this looks fishy to you, you can veryify it with `winget search 9PDXGNCFSCZV`
    # or by checking out https://apps.microsoft.com/store/detail/9PDXGNCFSCZV
    if ( ! $? ) { Exit(1) }
}
Write-Output "Ubuntu is installed"

# Creating images directory
Write-Output "Creating images directory"

$images = ".\images"
$tarball = "${images}\jammy.tar.gz"
$sourceRootfs = "$((Get-AppPackage | Where-Object Name -like 'CanonicalGroupLimited.Ubuntu').InstallLocation)\install.tar.gz"

if ( ! (Test-Path "${images}") ) {
    New-Item -Path "${images}" -ItemType "directory"           | Out-Null
}
Remove-Item -Path "${images}\empty.tar.gz"                2>&1 | Out-Null
New-Item -Path "${images}\empty.tar.gz" -ItemType "file"       | Out-Null
Copy-Item -Path "${sourceRootfs}" -Destination "${tarball}"   

Write-Output "Done"