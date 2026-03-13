class McpBinAT0118 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.18"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.18/mcp-bin-osx-universal"
    sha256 "8ea120ac21b77c9df3fa26d29ceb3ca34499a15be7d2737e73d781b672988169"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.18/mcp-bin-linux-arm64"
      sha256 "c270827fb2f1f02d894331f237b075eb60bfa48db6d24b6c3d79ef578a9ed5b0"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.18/mcp-bin-linux-amd64"
      sha256 "689e3e84afaf3e4dc7aea4d120fc443a0b552190d27445000c95c509e4e4efd8"
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
