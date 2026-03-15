class McpBinAT0122 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.22"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.22/mcp-bin-osx-universal"
    sha256 "841162efd340751c9e0c229268b145d0d7e3418d4512f3333e41de232ebc937d"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.22/mcp-bin-linux-arm64"
      sha256 "7ce939762c4228398893351f8216c5d33a1d8cdfa6fbc3a61019af96d18307e9"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.22/mcp-bin-linux-amd64"
      sha256 "73821e361eb00889caa771300c720786527128c243908c0ba1d5ce6bbe52f7ba"
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
