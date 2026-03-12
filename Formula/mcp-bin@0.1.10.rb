class McpBinAT0110 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.10"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.10/mcp-bin-osx-universal"
    sha256 "13f477007fc9697d879469ad1d5b471f7b743fee6465bbdb7e710a5fa35c00d7"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.10/mcp-bin-linux-arm64"
      sha256 "faba521290132e10d72cab624ed85c64a2d5d208cfcd6100709e39fa8394c246"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.10/mcp-bin-linux-amd64"
      sha256 "6afd6095349e19568d416453157cd92014acf37acb4ddada816e2b7aaf8541dd"
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
