class McpBinAT0115 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.15"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.15/mcp-bin-osx-universal"
    sha256 "247c91d6cca13961b5a13cdc7611dd3fbda5a56e058ab6483c38db7eb2838199"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.15/mcp-bin-linux-arm64"
      sha256 "3a16d9bfcb39bce24b47e1f0ad6c0370ec6afa4cb38921cef8e7b81fd1fface5"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.15/mcp-bin-linux-amd64"
      sha256 "604668043e213175ae54a70d7903a8d5b1d7fe7227f80202292a10ef40400ee9"
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
