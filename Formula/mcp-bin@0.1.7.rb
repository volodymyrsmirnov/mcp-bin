class McpBinAT017 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.7"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.7/mcp-bin-darwin-arm64"
      sha256 "afa4b6360aab9564b84ed2523c61259f52e2535e44aa5bc7b60a6761c61743a5"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.7/mcp-bin-darwin-amd64"
      sha256 "31b88d31a784042a668ae24cc29a13cc4d905e925462dea3d86db088ca6187d0"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.7/mcp-bin-linux-arm64"
      sha256 "8b02b76473d83edb6ece3f9d7dd5ee5d948bfaa90274e04a86be52c6e800398a"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.7/mcp-bin-linux-amd64"
      sha256 "97b7faa7ac33058de0a66bad29717236513861a9f5a288a28e99adca85e3a4ee"
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
