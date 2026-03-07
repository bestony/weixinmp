class Weixinmp < Formula
  desc "CLI toolbox for WeChat Official Account"
  homepage "https://github.com/bestony/weixinmp"
  license "GPL-3.0-only"
  version "0.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/bestony/weixinmp/releases/download/v#{version}/weixinmp_v#{version}_darwin_arm64.zip"
      sha256 "84ff0e940537df5649c36e1f7cad0df4e6267e9b7b406b1a9086f622293fd309"
    else
      url "https://github.com/bestony/weixinmp/releases/download/v#{version}/weixinmp_v#{version}_darwin_amd64.zip"
      sha256 "e6def372f56674e1394287b38299fe503096509ce4b406f3e74ac5d5d925879d"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/bestony/weixinmp/releases/download/v#{version}/weixinmp_v#{version}_linux_arm64.zip"
      sha256 "45def14737712530ee446b1b7dfc95805e737740811a93eefb40a597ab08012c"
    else
      url "https://github.com/bestony/weixinmp/releases/download/v#{version}/weixinmp_v#{version}_linux_amd64.zip"
      sha256 "8470475acd3cb7fd410938005e8da045eb4b5cc3792ca02a0122330636be0963"
    end
  end

  def install
    bin.install "weixinmp"
  end

  test do
    output = shell_output("#{bin}/weixinmp --version")
    assert_match "v#{version}", output
  end
end
