update-deps:
	nix develop .#depShell -c "gomod2nix"
