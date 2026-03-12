class McpBinAT0114 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.14"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.14/mcp-bin-osx-universal"
    sha256 "6c78e5126464cfa25104079e4d3edd466a8aeb4b781c9aa15c72ebd3a88de015"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.14/mcp-bin-linux-arm64"
      sha256 "581f8c7266c700cd350df27a661ba9a15e11c1774111688ca317f66cdfcb676b"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.14/mcp-bin-linux-amd64"
      sha256 "5f7df957353da1b07047479addaa607c88d7adce54481635477d6f789b5d304a"
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
