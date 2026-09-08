#!/bin/sh

# This script generates an MSI file for UQDA for a given architecture. It
# needs to run on Windows within MSYS2 and Go 1.21 or later must be installed on
# the system and within the PATH. This is ran currently by GitHub Actions (see
# the workflows in the repository).
#
# Author: Neil Alexander <neilalexander@users.noreply.github.com>

# Get arch from command line if given
set -eu
PKGARCH=${1:-}
case "$PKGARCH" in
  x64|x86|arm64) ;;
  *) echo "tell me the architecture: x86, x64 or arm64" >&2; exit 1 ;;
esac

# This authoring uses WiX 3. Installing WiX 5 does not provide candle/light.
command -v candle >/dev/null || { echo "WiX 3 candle is required" >&2; exit 1; }
command -v light >/dev/null || { echo "WiX 3 light is required" >&2; exit 1; }

# Build UQDA unless the release workflow supplied already-built, Authenticode-
# signed binaries. Rebuilding here would silently strip those signatures from
# the files embedded in the MSI.
if [ "${UQDA_USE_PREBUILT:-0}" != "1" ]; then
  case "$PKGARCH" in
    x64) GOOS=windows GOARCH=amd64 CGO_ENABLED=0 ./build ;;
    x86) GOOS=windows GOARCH=386 CGO_ENABLED=0 ./build ;;
    arm64) GOOS=windows GOARCH=arm64 CGO_ENABLED=0 ./build ;;
  esac
fi

if [ ! -s uqda.exe ] || [ ! -s uqdactl.exe ]; then
  echo "uqda.exe and uqdactl.exe must exist before packaging" >&2
  exit 1
fi

# Keep this script reviewable and testable independently of XML generation.
cp contrib/msi/updateconfig.bat updateconfig.bat

# Work out metadata for the package info
PKGNAME=$(sh contrib/semver/name.sh)
PKGVERSION=$(sh contrib/msi/msversion.sh --bare)
PKGSEMVER=$(sh contrib/semver/version.sh --bare)
PKGVERSIONMS=$(echo $PKGVERSION | tr - .)
# One UQDA-only product family, including architecture transitions. Never put
# upstream UpgradeCodes in our Upgrade table: that can remove another product.
PKGGUID="c68fc7f4-9642-47e6-a8d5-a75e339d4318"
case "$PKGARCH" in
  x64|arm64) PKGINSTFOLDER="ProgramFiles64Folder" ;;
  x86) PKGINSTFOLDER="ProgramFilesFolder" ;;
esac

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
      DowngradeErrorMessage="A newer UQDA version is already installed."
      Schedule="afterInstallInitialize" />

    <Property Id="UQDA_EXISTING_SERVICE">
      <RegistrySearch Id="ExistingUqdaService" Root="HKLM"
        Key="SYSTEM\CurrentControlSet\Services\UQDA" Name="ImagePath" Type="raw" />
    </Property>
    <Condition Message="An older or manually installed UQDA service exists. Back up ProgramData\UQDA, uninstall only UQDA using Installed apps, then install this package. Do not remove Yggdrasil or delete your configuration. See docs/windows-installation.md.">
      Installed OR NOT UQDA_EXISTING_SERVICE OR WIX_UPGRADE_DETECTED
    </Condition>

    <Media
      Id="1"
      Cabinet="Media.cab"
      EmbedCab="yes"
      CompressionLevel="high" />

    <Directory Id="TARGETDIR" Name="SourceDir">
      <Directory Id="${PKGINSTFOLDER}" Name="PFiles">
        <Directory Id="UQDAInstallFolder" Name="UQDA">

          <Component Id="MainExecutable" Guid="*">
            <File
              Id="UQDA"
              Name="uqda.exe"
              DiskId="1"
              Source="uqda.exe"
              KeyPath="yes" />

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
              Name="UQDA"
              Start="install"
              Stop="both"
              Remove="uninstall" />
          </Component>

          <Component Id="WintunLibrary" Guid="*">
            <File
              Id="Wintun"
              Name="wintun.dll"
              DiskId="1"
              Source="${PKGWINTUNDLL}"
              KeyPath="yes" />
          </Component>

          <Component Id="CtrlExecutable" Guid="*">
            <File
              Id="UQDActl"
              Name="uqdactl.exe"
              DiskId="1"
              Source="uqdactl.exe"
              KeyPath="yes"/>
          </Component>

          <Component Id="ConfigScript" Guid="*">
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
      <ComponentRef Id="WintunLibrary" />
      <ComponentRef Id="CtrlExecutable" />
      <ComponentRef Id="ConfigScript" />
    </Feature>

    <CustomAction
      Id="UpdateGenerateConfig"
      Directory="UQDAInstallFolder"
      ExeCommand='&quot;[SystemFolder]cmd.exe&quot; /d /c &quot;&quot;[UQDAInstallFolder]updateconfig.bat&quot; &quot;[CommonAppDataFolder]UQDA&quot;&quot;'
      Execute="deferred"
      Return="check"
      Impersonate="no" />

    <InstallExecuteSequence>
      <Custom
        Action="UpdateGenerateConfig"
        Before="StartServices">
          NOT REMOVE~="ALL"
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
