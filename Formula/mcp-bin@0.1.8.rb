class McpBinAT018 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.8"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.8/mcp-bin-osx-universal"
    sha256 "f74cc7217d3631071669ea8938138323e3a00696be4ce9752ae6f5c24458bb20"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.8/mcp-bin-linux-arm64"
      sha256 "0c675ba0fb4188bbc5a938e5e407273573cc009acf9a263bb2ab025a53ed49ac"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.8/mcp-bin-linux-amd64"
      sha256 "83d6be6596613cbd0239d64f4d5555447a41f396a6ca632c2069cb55178976dc"
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
