class McpBinAT0123 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.23"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.23/mcp-bin-osx-universal"
    sha256 "842391984a64d4256a6d0bb3ee138bb653ef83e53fd148e216ae58401184ca32"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.23/mcp-bin-linux-arm64"
      sha256 "00294c0108e2947a718ca8875df38478f72768f9339d4e08e012f8f052bcfba8"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.23/mcp-bin-linux-amd64"
      sha256 "13cf55411535d768a8befc680299c38cbc7ce1bf90ae383f878cb660405b8a6b"
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
