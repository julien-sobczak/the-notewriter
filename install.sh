#!/bin/sh
# Copyright The NoteWriter authors. All rights reserved.
# TODO(everyone): Keep this script simple and easily auditable.

set -e

if ! command -v tar >/dev/null; then
	echo "Error: tar is required to install The NoteWriter." 1>&2
	exit 1
fi

if [ "$OS" = "Windows_NT" ]; then
	platform="windows"
	arch="amd64"
else
	case $(uname -s) in
	"Darwin")
		platform="darwin"
		case $(uname -m) in
		"arm64") arch="arm64" ;;
		*) arch="amd64" ;;
		esac
		;;
	"Linux")
		platform="linux"
		arch="amd64"
		;;
	*) 
		echo "Error: Unsupported platform $(uname -s)" 1>&2
		exit 1
		;;
	esac
fi

if [ $# -eq 0 ]; then
	nt_uri="https://github.com/julien-sobczak/the-notewriter/releases/latest/download/nt-${platform}-${arch}.tar.gz"
else
	nt_uri="https://github.com/julien-sobczak/the-notewriter/releases/download/${1}/nt-${platform}-${arch}.tar.gz"
fi

nt_install="${NT_INSTALL:-$HOME/.nt}"
bin_dir="$nt_install/bin"

if [ ! -d "$bin_dir" ]; then
	mkdir -p "$bin_dir"
fi

echo "Downloading The NoteWriter from $nt_uri..."
curl --fail --location --progress-bar --output "$bin_dir/nt.tar.gz" "$nt_uri"

echo "Extracting archive..."
tar -xzf "$bin_dir/nt.tar.gz" -C "$bin_dir"

# Make all binaries executable
chmod +x "$bin_dir"/nt* 2>/dev/null || true

rm "$bin_dir/nt.tar.gz"

echo ""
echo "The NoteWriter was installed successfully to $bin_dir"
echo ""

if command -v nt >/dev/null; then
	echo "Run 'nt --help' to get started"
else
	case $SHELL in
	/bin/zsh) shell_profile=".zshrc" ;;
	*) shell_profile=".bashrc" ;;
	esac
	echo "Manually add the directory to your \$HOME/$shell_profile (or similar)"
	echo "  export NT_INSTALL=\"$nt_install\""
	echo "  export PATH=\"\$NT_INSTALL/bin:\$PATH\""
	echo ""
	echo "Run '$bin_dir/nt --help' to get started"
fi
