class McpBinAT020 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.2.0"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.2.0/mcp-bin-osx-universal"
    sha256 "e65a7c131df6ec15bcc79b88a7c07c0253840f268ec3245338f69389bf49da5f"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.2.0/mcp-bin-linux-arm64"
      sha256 "e04b299a739378b2bf02d468069e1798bdc19db72aee4abb893a4d7a6af8a91e"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.2.0/mcp-bin-linux-amd64"
      sha256 "790d1701aa1561343d8236281616725b4916f80b729dffbcacaa4ef48fb79adc"
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
