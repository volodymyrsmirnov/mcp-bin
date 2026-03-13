class McpBinAT0119 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.19"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.19/mcp-bin-osx-universal"
    sha256 "db4c4a8e9b2b1f0ba913ebf981f2adf400d3bb9b52c2d9dafa0256796a36e69f"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.19/mcp-bin-linux-arm64"
      sha256 "4aa87c59a0f0532478045bd6d55a01870058fa7506dce8c714f4d655dc224134"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.19/mcp-bin-linux-amd64"
      sha256 "6fab2ce2618b2a99f4af0a877392c52e52b763dcd1f92ba4c4252c862d1290b9"
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
