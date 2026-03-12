class McpBinAT019 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.9"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.9/mcp-bin-osx-universal"
    sha256 "64d008c81d7f273ffc6275dd86e5b45b123ae7a6989c16eb2041c449e9b6af5d"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.9/mcp-bin-linux-arm64"
      sha256 "4299fba2c99207cbf82eca4a6afc99e7fedc45a4e83cdd070d871cf93eb0e230"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.9/mcp-bin-linux-amd64"
      sha256 "19e7821741db7e5bd2fd1476f65cf83859ed74ba8f666789799bde8f141dbfbc"
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
