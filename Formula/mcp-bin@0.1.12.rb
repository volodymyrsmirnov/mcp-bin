class McpBinAT0112 < Formula
  desc "Turn MCP server tools into CLI commands"
  homepage "https://github.com/volodymyrsmirnov/mcp-bin"
  version "0.1.12"
  license "MIT"

  on_macos do
    url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.12/mcp-bin-osx-universal"
    sha256 "6036cda02e700153e813bddb60c4a51cfbec4b79172392594638e1204e9cfade"
  end

  on_linux do
    on_arm do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.12/mcp-bin-linux-arm64"
      sha256 "40ff96b1547d96ecc16a148d6b1614c80fed41ac014cb756fdb6af93ab89250e"
    end
    on_intel do
      url "https://github.com/volodymyrsmirnov/mcp-bin/releases/download/v0.1.12/mcp-bin-linux-amd64"
      sha256 "40c829d54b8ca262cba36f5018bfe66d1170bc6bcc02f72e03deb20c05824cfa"
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
