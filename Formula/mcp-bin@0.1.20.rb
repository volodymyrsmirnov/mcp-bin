class McpBinAT0120 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.20"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.20/mcp-bin-osx-universal"
    sha256 "1494017aa4f7625407819152ca7c25965fcf4f81e1d23403a7aff7ecc27bcec8"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.20/mcp-bin-linux-arm64"
      sha256 "d2b04a635c167dd0d17808bd3d65e1a5b1c50a3ff7177450be3320d430b21cca"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.20/mcp-bin-linux-amd64"
      sha256 "b13630458916bdfc12f3ef1b17f3fbff3edac6cfd782fdc6319046715dcee81b"
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
