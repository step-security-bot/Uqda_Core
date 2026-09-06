#!/bin/sh

# This script generates an MSI file for UQDA for a given architecture. It
# needs to run on Windows within MSYS2 and Go 1.21 or later must be installed on
# the system and within the PATH. This is ran currently by GitHub Actions (see
# the workflows in the repository).
#
# Author: Neil Alexander <neilalexander@users.noreply.github.com>

# Get arch from command line if given
PKGARCH=$1
if [ "${PKGARCH}" == "" ];
then
  echo "tell me the architecture: x86, x64 or arm64"
  exit 1
fi

# Download the wix tools!
dotnet tool install --global wix --version 5.0.0

# Build UQDA unless the release workflow supplied already-built, Authenticode-
# signed binaries. Rebuilding here would silently strip those signatures from
# the files embedded in the MSI.
if [ "${UQDA_USE_PREBUILT:-0}" != "1" ]; then
  [ "${PKGARCH}" == "x64" ] && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 ./build
  [ "${PKGARCH}" == "x86" ] && GOOS=windows GOARCH=386 CGO_ENABLED=0 ./build
  [ "${PKGARCH}" == "arm64" ] && GOOS=windows GOARCH=arm64 CGO_ENABLED=0 ./build
fi

if [ ! -s uqda.exe ] || [ ! -s uqdactl.exe ]; then
  echo "uqda.exe and uqdactl.exe must exist before packaging" >&2
  exit 1
fi

# Create the postinstall script
cat > updateconfig.bat << EOF
@echo off
setlocal
set "UQDA_CONFIG_DIR=%ProgramData%\\UQDA"
if not exist "%UQDA_CONFIG_DIR%" (
  mkdir "%UQDA_CONFIG_DIR%" || exit /b 1
)
if not exist "%UQDA_CONFIG_DIR%\\uqda.conf" (
  if exist "%~dp0uqda.exe" (
    "%~dp0uqda.exe" -genconf > "%UQDA_CONFIG_DIR%\\uqda.conf" || exit /b 1
  )
)
exit /b 0
EOF

# Work out metadata for the package info
PKGNAME=$(sh contrib/semver/name.sh)
PKGVERSION=$(sh contrib/msi/msversion.sh --bare)
PKGSEMVER=$(sh contrib/semver/version.sh --bare)
PKGVERSIONMS=$(echo $PKGVERSION | tr - .)
([ "${PKGARCH}" == "x64" ] || [ "${PKGARCH}" == "arm64" ]) && \
  PKGGUID="77757838-1a23-40a5-a720-c3b43e0260cc" PKGINSTFOLDER="ProgramFiles64Folder" || \
  PKGGUID="54a3294e-a441-4322-aefb-3bb40dd022bb" PKGINSTFOLDER="ProgramFilesFolder"

# Download the Wintun driver
if [ ! -d wintun ];
then
  curl -o wintun.zip https://www.wintun.net/builds/wintun-0.14.1.zip
  if [ `sha256sum wintun.zip | cut -f 1 -d " "` != "07c256185d6ee3652e09fa55c0b673e2624b565e02c4b9091c79ca7d2f24ef51" ];
  then
    echo "wintun package didn't match expected checksum"
    exit 1
  fi
  unzip wintun.zip
fi
if [ $PKGARCH = "x64" ]; then
  PKGWINTUNDLL=wintun/bin/amd64/wintun.dll
elif [ $PKGARCH = "x86" ]; then
  PKGWINTUNDLL=wintun/bin/x86/wintun.dll
elif [ $PKGARCH = "arm64" ]; then
  PKGWINTUNDLL=wintun/bin/arm64/wintun.dll
else
  echo "wasn't sure which architecture to get wintun for"
  exit 1
fi

if [ "$PKGNAME" != "uqda" ]; then
  PKGDISPLAYNAME="UQDA Network (${PKGNAME} branch)"
else
  PKGDISPLAYNAME="UQDA Network"
fi

# Generate the wix.xml file
cat > wix.xml << EOF
<?xml version="1.0" encoding="windows-1252"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
  <Product
    Name="${PKGDISPLAYNAME}"
    Id="*"
    UpgradeCode="${PKGGUID}"
    Language="1033"
    Codepage="1252"
    Version="${PKGVERSIONMS}"
    Manufacturer="github.com/Uqda">

    <Package
      Id="*"
      Keywords="Installer"
      Description="UQDA Network Installer"
      Comments="UQDA Network standalone router for Windows."
      Manufacturer="github.com/Uqda"
      InstallerVersion="500"
      InstallScope="perMachine"
      Languages="1033"
      Compressed="yes"
      SummaryCodepage="1252" />

    <MajorUpgrade
      AllowDowngrades="yes" />

    <Media
      Id="1"
      Cabinet="Media.cab"
      EmbedCab="yes"
      CompressionLevel="high" />

    <Directory Id="TARGETDIR" Name="SourceDir">
      <Directory Id="${PKGINSTFOLDER}" Name="PFiles">
        <Directory Id="UQDAInstallFolder" Name="UQDA">

          <Component Id="MainExecutable" Guid="c2119231-2aa3-4962-867a-9759c87beb24">
            <File
              Id="UQDA"
              Name="uqda.exe"
              DiskId="1"
              Source="uqda.exe"
              KeyPath="yes" />

            <File
              Id="Wintun"
              Name="wintun.dll"
              DiskId="1"
              Source="${PKGWINTUNDLL}" />

            <Environment
              Id="UQDAPath"
              Name="PATH"
              Value="[UQDAInstallFolder]"
              Action="set"
              Part="last"
              Permanent="no"
              System="yes" />

            <ServiceInstall
              Id="ServiceInstaller"
              Account="LocalSystem"
              Description="UQDA Network router process"
              DisplayName="UQDA Service"
              ErrorControl="normal"
              LoadOrderGroup="NetworkProvider"
              Name="UQDA"
              Start="auto"
              Type="ownProcess"
              Arguments='-useconffile "[CommonAppDataFolder]UQDA\\uqda.conf" -logto "[CommonAppDataFolder]UQDA\\uqda.log"'
              Vital="yes" />

            <ServiceControl
              Id="ServiceControl"
              Name="uqda"
              Start="install"
              Stop="both"
              Remove="uninstall" />
          </Component>

          <Component Id="CtrlExecutable" Guid="a916b730-974d-42a1-b687-d9d504cbb86a">
            <File
              Id="UQDActl"
              Name="uqdactl.exe"
              DiskId="1"
              Source="uqdactl.exe"
              KeyPath="yes"/>
          </Component>

          <Component Id="ConfigScript" Guid="64a3733b-c98a-4732-85f3-20cd7da1a785">
            <File
              Id="Configbat"
              Name="updateconfig.bat"
              DiskId="1"
              Source="updateconfig.bat"
              KeyPath="yes"/>
          </Component>
        </Directory>
      </Directory>
    </Directory>

    <Feature Id="UQDAFeature" Title="UQDA" Level="1">
      <ComponentRef Id="MainExecutable" />
      <ComponentRef Id="CtrlExecutable" />
      <ComponentRef Id="ConfigScript" />
    </Feature>

    <CustomAction
      Id="UpdateGenerateConfig"
      Directory="UQDAInstallFolder"
      ExeCommand="cmd.exe /c updateconfig.bat"
      Execute="deferred"
      Return="check"
      Impersonate="yes" />

    <InstallExecuteSequence>
      <Custom
        Action="UpdateGenerateConfig"
        Before="StartServices">
          NOT Installed AND NOT REMOVE
      </Custom>
    </InstallExecuteSequence>

  </Product>
</Wix>
EOF

# Generate the MSI
CANDLEFLAGS="-nologo"
LIGHTFLAGS="-nologo -spdb -sice:ICE71 -sice:ICE61"
candle $CANDLEFLAGS -out ${PKGNAME}-${PKGVERSION}-${PKGARCH}.wixobj -arch ${PKGARCH} wix.xml && \
light $LIGHTFLAGS -ext WixUtilExtension.dll -out ${PKGNAME}-${PKGSEMVER}-${PKGARCH}.msi ${PKGNAME}-${PKGVERSION}-${PKGARCH}.wixobj
