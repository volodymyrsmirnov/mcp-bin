class McpBinAT0113 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.13"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.13/mcp-bin-osx-universal"
    sha256 "98a4a04818ac30d2e51e4f4488ea9afe127e24d210093e7e0a0a730b45779919"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.13/mcp-bin-linux-arm64"
      sha256 "8c0d4195d95d5b5817167463e5b718ee90e238e8381f0d9353ec8fa8d26f2cb3"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.13/mcp-bin-linux-amd64"
      sha256 "f17424dd75d039bb27d4047b3de1e4d65f3b86ed3838f84adc865aaebf0fb2f0"
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
