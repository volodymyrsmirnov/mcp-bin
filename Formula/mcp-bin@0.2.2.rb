class McpBinAT022 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.2.2"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.2.2/mcp-bin-osx-universal"
    sha256 "7eaa0ce8b79adafdee6c09e5833d4837c5dadec644c84f447156ac516e0fd67f"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.2.2/mcp-bin-linux-arm64"
      sha256 "d65ef39580d7e77374e922ed351824d456a3bb1d0abb6a7fd07da238d49b2825"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.2.2/mcp-bin-linux-amd64"
      sha256 "768c173a36c8f29c30290ab90744246da680448c4d8acfb2b72c148649ba51ff"
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
