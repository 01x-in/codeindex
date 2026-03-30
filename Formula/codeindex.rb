class Codeindex < Formula
  desc "Persistent structural knowledge graph for codebases — MCP tools + CLI tree explorer"
  homepage "https://github.com/01x-in/codeindex"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/01x-in/codeindex/releases/download/v#{version}/codeindex_#{version}_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_ARM64_SHA256"
    else
      url "https://github.com/01x-in/codeindex/releases/download/v#{version}/codeindex_#{version}_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_AMD64_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/01x-in/codeindex/releases/download/v#{version}/codeindex_#{version}_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"
    else
      url "https://github.com/01x-in/codeindex/releases/download/v#{version}/codeindex_#{version}_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_AMD64_SHA256"
    end
  end

  def install
    bin.install "codeindex"
  end

  test do
    assert_match "codeindex", shell_output("#{bin}/codeindex version")
  end
end
