class McpBinAT021 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.2.1"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.2.1/mcp-bin-osx-universal"
    sha256 "8b71046cc9c66fbd22b7391ce77231321fa6fd7c536b122d81b3072e490a3666"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.2.1/mcp-bin-linux-arm64"
      sha256 "057651cbf7a0eefef17545eead878e6d0f6db95bdeb5780fe40e0abf0b7c687f"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.2.1/mcp-bin-linux-amd64"
      sha256 "46c96c05cb63b2b76e219e85f8b70e4c6855891d06b7994194940709db841d58"
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
