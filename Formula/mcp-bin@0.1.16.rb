class McpBinAT0116 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.16"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.16/mcp-bin-osx-universal"
    sha256 "92f89c1d818557a9079a0320a23f6acec5447025355ba2fc14c54e982cababc6"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.16/mcp-bin-linux-arm64"
      sha256 "9d1319ed77dd27ae121e9c0625791f6c9383e9f90a1f0a47b913952ab453d6fd"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.16/mcp-bin-linux-amd64"
      sha256 "2c3760e3db570ba1f5360a3a7ed01f17622074487e1f1d92512ca73375a37deb"
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
