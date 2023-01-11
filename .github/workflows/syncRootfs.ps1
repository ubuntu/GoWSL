<#
.SYNOPSIS
    Sync your WSL tarball to cloud-images.ubuntu.com/wsl
.DESCRIPTION
    Compare checksums with the remote tarball and download it if the checksums don't match.
.PARAMETER Path
    The path to your local copy of the rootfs tarball.
.PARAMETER CodeName
    The codename of the release you want to sync it to (e.g: jammy, kinetic, lunar).
.EXAMPLE
    PS> .\syncRootfs.ps1 -Path $HOME\Downloads\jammy.tar.gz -CodeName jammy

    This downloads the latest Ubuntu 22.04 LTS image into your downloads folder. If you run it
    again, the rootfs will not be downloaded unless the remote is updated or your local file is
    damaged.
.NOTES
    Author: Edu Gómez Escandell
#>

# Download latest LTS rootfs
param(
    [Parameter(Mandatory)][string]$Path,
    [Parameter(Mandatory)][string]$CodeName
)

$rootfsDir = $(Split-Path -Path "${Path}")
$tarball = $(Split-Path -Path "${Path}" -Leaf)
$tmpDir = "C:\Temp\syncRootfs-${CodeName}"

# This function validates a rootfs against its checksum
function Test-Rootfs {
    param ([Parameter(Mandatory)][string]$Path)

    $sha256file = "${tmpDir}\${CodeName}.SHA256SUMS.txt"

    if ( ! $(Test-Path -Path "${Path}")) {
        # local rootfs does not exist
        return $false
    }

    Invoke-WebRequest                                                               `
        -Uri "https://cloud-images.ubuntu.com/wsl/${CodeName}/current/SHA256SUMS"   `
        -OutFile "${sha256file}"
    
    if ( ! $? ) {
        Write-Warning "failed to download remote checksum"
        return $false
    }

    $remoteSha256 = $(Select-String -Path "${sha256file}"                                       `
                                    -Pattern "ubuntu-${CodeName}-wsl-amd64-wsl.rootfs.tar.gz"   `
    ) -replace ".*:([0-9a-f]+)\s+ubuntu-${CodeName}-wsl-amd64-wsl.rootfs.tar.gz", '$1'
    if ( ! $? ) {
        Write-Warning "failed to parse remote checksum"
        return $false
    }

    $localSha256 = $(Get-FileHash -Path "${Path}" -Algorithm "Sha256").Hash
    if ( ! $? ) {
        Write-Warning "failed to get local checksum"
        return $false
    }

    # Case-insensitive comparison
    if ( "${localSha256}" -ne "${remoteSha256}" ) {
        return $false
    }

    return $true
}

# Creating temporary and final directories
if ( ! $(Test-Path -Path "${tmpDir}") ) {
    New-Item -Path "${tmpDir}" -ItemType "directory" | Out-Null
}

if ( ! $(Test-Path -Path "${rootfsDir}") ) {
    New-Item -Path "${rootfsDir}" -ItemType "directory" | Out-Null
}

# Testing if a cached rootfs exists
if ( $(Test-Rootfs -Path "${rootfsDir}\${tarball}") ) {
    Write-Output "Cache hit"
    Remove-Item -Force -Recurse -Path "${tmpDir}" 2>&1 | Out-Null
    Exit(0)
}

# Missing or outdated cache: downloading rootfs
Invoke-WebRequest                                                                                                   `
    -Uri "https://cloud-images.ubuntu.com/wsl/${codeName}/current/ubuntu-${codeName}-wsl-amd64-wsl.rootfs.tar.gz"   `
    -OutFile "${tmpDir}\${CodeName}.tar.gz"

if ( ! $(Test-Rootfs -Path "${tmpDir}\${tarball}") ) {
    Remove-Item -Force -Recurse -Path "${tmpDir}" 2>&1 | Out-Null
    Write-Error "failed checksum validation after download"
    Exit(1)
}

Move-Item -Force "${tmpDir}\${tarball}" "${rootfsDir}\${tarball}"
Remove-Item -Force -Recurse -Path "${tmpDir}" 2>&1 | Out-Null
