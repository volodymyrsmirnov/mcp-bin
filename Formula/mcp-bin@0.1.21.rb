class McpBinAT0121 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.21"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.21/mcp-bin-osx-universal"
    sha256 "e4ce7342eeddbdfe187c310e0769e9f358ba232965c0208ca51af631cda55ca5"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.21/mcp-bin-linux-arm64"
      sha256 "30835e8bee16e12ff90d30274daf81d7cf483a5aa312aa37863054b2c9432d54"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.21/mcp-bin-linux-amd64"
      sha256 "a1fb78e575d9575c8bbec813fc4d075731b50c0a8fb93f90a250f319271cd29b"
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
