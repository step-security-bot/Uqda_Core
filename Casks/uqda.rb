cask "uqda" do
  version "0.1.10"
  sha256 "202c191d612f3499570afab6a9cb731cbdd9e3402d2abc28409bd11652006379"

  url "https://github.com/Uqda/Core/releases/download/v#{version}/install.sh"
  name "UQDA"
  desc "Encrypted, self-organizing IPv6 overlay network"
  homepage "https://github.com/Uqda/Core"

  livecheck do
    url "https://github.com/Uqda/Core/releases/latest"
    strategy :github_latest
  end

  container type: :naked

  installer script: {
    executable: "/bin/sh",
    args:       ["#{staged_path}/install.sh", "--version", "v#{version}"],
    sudo:       true,
  }

  uninstall script: {
    executable: "/bin/sh",
    args:       ["/usr/local/share/uqda/uninstall.sh"],
    sudo:       true,
  }

  caveats <<~EOS
    UQDA's macOS package is unsigned because the project does not use a paid
    Apple Developer account. The installer verifies the package against the
    release SHA256SUMS before invoking the macOS system installer.

    Check the running node with:
      sudo uqdactl getSelf

    Homebrew uninstall preserves the node identity for a future reinstall:
      brew uninstall --cask uqda/core/uqda

    For permanent removal, download uninstall.sh from the latest release and
    run: sudo sh uninstall.sh --purge
  EOS
end
