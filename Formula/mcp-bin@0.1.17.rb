class McpBinAT0117 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.17"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.17/mcp-bin-osx-universal"
    sha256 "0ffebfe9f136b1a185becf2d22bafc9dbae4398414d4d4efb763f66ac83f0316"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.17/mcp-bin-linux-arm64"
      sha256 "bd5f0d97c72be47a65ad6f807c1946ea9a65f45bc9475ec119563d02b2cc8179"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.17/mcp-bin-linux-amd64"
      sha256 "25954f64f55737ab489663af6754ab68a7801e2ec196b5673a11d60159f9b797"
    end
  end

  def install
    binary = Dir["mcp-bin-*"].first
    bin.install binary => "mcp-bin"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/mcp-bin --version")
  end
end
