class McpBinAT0111 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.11"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.11/mcp-bin-osx-universal"
    sha256 "8779c3d7edb18a6c2893093d8161bc022d0f42d9c117da3fa67b9ab425fd6fcd"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.11/mcp-bin-linux-arm64"
      sha256 "5e8aaaea8b503e47d31084fdabd169f3815fb2558d9f0cba6327d175273a3d86"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.11/mcp-bin-linux-amd64"
      sha256 "fe744321c1b3f38074358fd5db6d5f2623b43b7b531fb513ce0edeb7653d912b"
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
