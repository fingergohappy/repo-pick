#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <version> <checksums-file> <tap-root>" >&2
  exit 2
fi

version="$1"
checksums_file="$2"
tap_root="$3"
formula_dir="${tap_root}/Formula"
formula_file="${formula_dir}/repo-pick.rb"
release_url="https://github.com/fingergohappy/repo-pick/releases/download/v#{version}"

mkdir -p "${formula_dir}"

# checksum_for 从 checksums.txt 中读取指定 release asset 的 sha256。
checksum_for() {
  local asset="$1"
  awk -v asset="${asset}" '$2 == asset { print $1 }' "${checksums_file}"
}

darwin_arm64_asset="repo-pick_${version}_darwin_arm64.tar.gz"
darwin_amd64_asset="repo-pick_${version}_darwin_amd64.tar.gz"
linux_arm64_asset="repo-pick_${version}_linux_arm64.tar.gz"
linux_amd64_asset="repo-pick_${version}_linux_amd64.tar.gz"

darwin_arm64_sha="$(checksum_for "${darwin_arm64_asset}")"
darwin_amd64_sha="$(checksum_for "${darwin_amd64_asset}")"
linux_arm64_sha="$(checksum_for "${linux_arm64_asset}")"
linux_amd64_sha="$(checksum_for "${linux_amd64_asset}")"

if [[ -z "${darwin_arm64_sha}" || -z "${darwin_amd64_sha}" || -z "${linux_arm64_sha}" || -z "${linux_amd64_sha}" ]]; then
  echo "checksums file is missing one or more repo-pick assets" >&2
  exit 1
fi

cat >"${formula_file}" <<RUBY
class RepoPick < Formula
  desc "TUI-only remote Git repository file and directory downloader"
  homepage "https://github.com/fingergohappy/repo-pick"
  version "${version}"

  on_macos do
    if Hardware::CPU.arm?
      url "${release_url}/${darwin_arm64_asset}"
      sha256 "${darwin_arm64_sha}"
    else
      url "${release_url}/${darwin_amd64_asset}"
      sha256 "${darwin_amd64_sha}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "${release_url}/${linux_arm64_asset}"
      sha256 "${linux_arm64_sha}"
    else
      url "${release_url}/${linux_amd64_asset}"
      sha256 "${linux_amd64_sha}"
    end
  end

  # install 安装 release tarball 中的 repo-pick 二进制。
  def install
    bin.install "repo-pick"
  end

  # test 验证二进制可以非交互输出版本信息。
  test do
    assert_match "repo-pick #{version}", shell_output("#{bin}/repo-pick --version")
  end
end
RUBY

echo "updated ${formula_file}"
