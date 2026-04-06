class Musica < Formula
  desc "TUI music player for Navidrome and Jellyfin"
  homepage "https://github.com/codila125/musica"
  url "https://github.com/codila125/musica/archive/refs/tags/v0.0.0.tar.gz"
  sha256 "REPLACE_WITH_RELEASE_TARBALL_SHA256"
  license "NOASSERTION"

  depends_on "go" => :build
  depends_on "mpv"

  def install
    system "go", "build", "-trimpath", "-tags", "nocgo", "-o", libexec/"musica", "./cmd"

    env = {}
    if OS.mac?
      env["DYLD_FALLBACK_LIBRARY_PATH"] = "#{Formula["mpv"].opt_lib}:#{ENV["DYLD_FALLBACK_LIBRARY_PATH"]}"
    elsif OS.linux?
      env["LD_LIBRARY_PATH"] = "#{Formula["mpv"].opt_lib}:#{ENV["LD_LIBRARY_PATH"]}"
    end

    (bin/"musica").write_env_script libexec/"musica", env
  end

  test do
    assert_match "musica", shell_output("#{bin}/musica help")
  end
end
