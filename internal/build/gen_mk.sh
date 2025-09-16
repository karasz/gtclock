#!/bin/sh
# shellcheck disable=SC1007,SC3043 # empty assignments and local usage

set -eu

INDEX="$1"

PROJECTS="$(cut -d':' -f1 "$INDEX")"
COMMANDS="tidy get build test coverage race up"

TAB=$(printf "\t")

escape_dir() {
	echo "$1" | sed -e 's|/|\\/|g' -e 's|\.|\\.|g'
}

expand() {
	local prefix="$1" suffix="$2"
	local x= out=
	shift 2

	for x; do
		out="${out:+$out }${prefix}$x${suffix}"
	done

	echo "$out"
}

prefixed() {
	local prefix="${1:+$1-}"
	shift
	expand "$prefix" "" "$@"
}

suffixed() {
	local suffix="${1:+-$1}"
	shift
	expand "" "$suffix" "$@"
}

# packed remove excess whitespace from lines of commands
packed() {
	sed -e 's/^[ \t]\+//' -e 's/[ \t]\+$//' -e '/^$/d;' -e '/^#/d';
}

# packet_oneline converts a multiline script into packed single-line equivalent
packed_oneline() {
	packed | tr '\n' ';' | sed -e 's|;$||' -e 's|then;|then |g' -e 's|;[ \t]*|; |g'
}

gen_revive_exclude() {
	local self="$1"
	local dirs= d=

	dirs="$(cut -d: -f2 "$INDEX" | grep -v '^.$' || true)"
	if [ "." != "$self" ]; then
		dirs=$(echo "$dirs" | sed -n -e "s;^$self/\(.*\)$;\1;p")
	fi

	for d in $dirs; do
		printf -- "-exclude ./%s/... " "$d"
	done
}

gen_var_name() {
	local x=
	for x; do
		echo "$x" | tr 'a-z-' 'A-Z_'
	done
}

# generate files lists
#
gen_files_lists() {
	local name= dir= mod= deps=
	local files= files_cmd=
	local filter= out_pat=

	cat <<EOT
GO_FILES = \$(shell find * \\
	-type d -name node_modules -prune -o \\
	-type f -name '*.go' -print )

EOT

	# shellcheck disable=2094 # false positive - INDEX is only read.
	while IFS=: read -r name dir mod deps; do
		files=GO_FILES_$(gen_var_name "$name")
		filter="-e '/^\.$/d;'"
		[ "$dir" = "." ] || filter="$filter -e '/^$(escape_dir "$dir")$/d;'"
		out_pat="$(cut -d: -f2 "$INDEX" | eval "sed $filter -e 's|$|/%|'" | tr '\n' ' ' | sed -e 's| \+$||')"

		if [ "$dir" = "." ]; then
			# root
			files_cmd="\$(GO_FILES)"
			files_cmd="\$(filter-out $out_pat, $files_cmd)"
		else
			files_cmd="\$(filter $dir/%, \$(GO_FILES))"
			files_cmd="\$(filter-out $out_pat, $files_cmd)"
			files_cmd="\$(patsubst $dir/%,%,$files_cmd)"
		fi

		cat <<-EOT
		$files$TAB=$TAB$files_cmd
		EOT
	done < "$INDEX" | column -t -s "$TAB"
}

gen_make_targets() {
	local cmd="$1" name="$2" dir="$3" mod="$4" deps="$5"
	local call= callu=
	local depsx=
	local sequential=

	# default calls
	case "$cmd" in
	tidy)
		# unconditional
		callu="\$(GO) mod tidy"

		# go vet and revive only if there are .go files
		#
		call="$(cat <<-EOT | packed
		\$(GO) vet ./...
		\$(GOLANGCI_LINT) run
		\$(REVIVE) \$(REVIVE_RUN_ARGS) ./...
		EOT
		)"

		depsx="fmt"
		;;
	up)
		call="\$(GO) get -u \$(GOUP_FLAGS) \$(GOUP_PACKAGES)
\$(GO) mod tidy"
		;;
	test)
		call="\$(GO) $cmd \$(GOTEST_FLAGS) ./..."
		;;
	coverage)
		call="\$(TOOLSDIR)/make_coverage.sh \"$name\" \".\" \"\$(COVERAGE_DIR)\""
		depsx="\$(COVERAGE_DIR)"
		;;
	race)
		call="CGO_ENABLED=1 \$(GO) test -race \$(GOTEST_FLAGS) ./..."
		;;
	*)
		call="\$(GO) $cmd -v ./..."
		;;
	esac

	case "$cmd" in
	build|test|coverage|race)
		sequential=true ;;
	*)
		sequential=false ;;
	esac

	# cd $dir
	if [ "." = "$dir" ]; then
		# root
		cd=
	else
		cd="cd '$dir'; "
	fi

	case "$cmd" in
	build)
		# special build flags for cmd/* or main.go at root
		#
		call="$(cat <<-EOL | packed_oneline
		set -e
		MOD="\$\$(\$(GO) list -f '{{.ImportPath}}' ./...)"
		if echo "\$\$MOD" | grep -q -e '.*/cmd/[^/]\+\$\$'; then
			\$(GO_BUILD_CMD) ./...
		elif [ -f "main.go" ]; then
			\$(GO_BUILD_CMD) .
		elif [ -n "\$\$MOD" ]; then
			\$(GO_BUILD) ./...
		fi
		EOL
		)"
		;;
	tidy)
		# exclude submodules when running revive
		#
		exclude=$(gen_revive_exclude "$dir")
		if [ -n "$exclude" ]; then
			call=$(echo "$call" | sed -e "s;\(REVIVE)\);\1 $exclude;")
		fi
		;;
	esac


	if ! $sequential; then
		deps=
	fi

	files=GO_FILES_$(gen_var_name "$name")
	# shellcheck disable=SC2086 # word splitting of deps intended
	cat <<EOT

$cmd-$name:${deps:+ $(prefixed "$cmd" $deps)}${depsx:+ | $depsx} ; \$(info \$(M) $cmd: $name)
EOT
	if [ -n "$callu" ]; then
		# unconditionally
		echo "$callu" | sed -e "/^$/d;" -e "s|^|\t\$(Q) $cd|"
	fi
	if [ -n "$call" ]; then
		# only if there are files
		echo "ifneq (\$($files),)"
		echo "$call" | sed -e "/^$/d;" -e "s|^|\t\$(Q) $cd|"
		echo "endif"
	fi
}

gen_files_lists

# shellcheck disable=SC2086 # word splitting intentional
for cmd in $COMMANDS; do
	# shellcheck disable=SC2086 # word splitting intentional
	all="$(prefixed "$cmd" $PROJECTS)"
	depsx=

	cat <<EOT

.PHONY: $cmd $all
$cmd: $all
EOT

	while IFS=: read -r name dir mod deps; do
		deps=$(echo "$deps" | tr ',' ' ')

		gen_make_targets "$cmd" "$name" "$dir" "$mod" "$deps"
	done < "$INDEX"
done

for x in $PROJECTS; do
	cat <<EOT

$x: $(suffixed "$x" get build tidy)
EOT
done

# Add coverage-related rules
cat <<'EOT'

$(COVERAGE_DIR):
	$Q mkdir -p $@

.PHONY: clean-coverage
clean-coverage: ; $(info $(M) cleaning coverage data…)
	$Q rm -rf $(COVERAGE_DIR)

# Merge all coverage profiles into a single file
$(COVERAGE_DIR)/coverage.out: | coverage ; $(info $(M) merging coverage profiles…)
	$Q $(TOOLSDIR)/merge_coverage.sh $(wildcard $(COVERAGE_DIR)/coverage_*.prof) > $@~
	$Q mv $@~ $@
EOT

# Add release-related rules for projects with main.go or cmd/ directories
gen_release_targets() {
	local name="$1" dir="$2" mod="$3"
	local has_main=false

	# Check if this module has a main.go or cmd/ directory
	if [ -f "$dir/main.go" ] || find "$dir" -type d -name cmd -print -quit | grep -q .; then
		has_main=true
	fi

	# Only generate release targets for modules that produce binaries
	if [ "$has_main" = true ]; then
		cat <<EOT

# Release targets for $name
.PHONY: release-$name release-clean-$name release-build-$name release-sign-$name release-checksums-$name
release-$name: release-clean-$name release-build-$name release-sign-$name release-checksums-$name

release-clean-$name: ; \$(info \$(M) cleaning release directory for $name…)
	\$Q rm -rf \$(RELEASE_DIR)/$name
	\$Q mkdir -p \$(RELEASE_DIR)/$name

release-build-$name: | release-clean-$name ; \$(info \$(M) building release binaries for $name…)
	\$Q cd "$dir" && for platform in \$(RELEASE_PLATFORMS); do \\
		GOOS=\$\${platform%/*}; \\
		GOARCH=\$\${platform#*/}; \\
		binary_name="\$(BINARY_NAME)"; \\
		[ -z "\$\$binary_name" ] && binary_name="\$(shell basename $mod)"; \\
		output="\$(RELEASE_DIR)/$name/\$\${binary_name}-\$\${GOOS}-\$\${GOARCH}"; \\
		[ "\$\$GOOS" = "windows" ] && output="\$\${output}.exe"; \\
		echo "Building \$\$output..."; \\
		GOOS=\$\$GOOS GOARCH=\$\$GOARCH \$(GO_BUILD) -o "\$\$output" .; \\
	done

release-sign-$name: | release-build-$name ; \$(info \$(M) signing release binaries for $name…)
ifeq (\$(GPG_SIGN),true)
ifneq (\$(GPG_KEY),)
	\$Q for binary in \$(RELEASE_DIR)/$name/*; do \\
		[ -f "\$\$binary" ] || continue; \\
		case "\$\$binary" in *.asc|*.txt) continue ;; esac; \\
		echo "Signing \$\$binary..."; \\
		gpg --default-key \$(GPG_KEY) --armor --detach-sign "\$\$binary"; \\
	done
else
	\$Q echo "Warning: GPG_KEY not found, skipping signing for $name"
endif
else
	\$Q echo "GPG signing disabled for $name"
endif

release-checksums-$name: | release-build-$name ; \$(info \$(M) generating checksums for $name…)
	\$Q cd \$(RELEASE_DIR)/$name && \\
		find . -type f \( ! -name "*.asc" ! -name "*.txt" \) -exec sha256sum {} + > checksums.txt && \\
		echo "Generated checksums.txt for $name"

EOT
	fi
}

# Generate release targets for each project
while IFS=: read -r name dir mod deps; do
	gen_release_targets "$name" "$dir" "$mod"
done < "$INDEX"

# Add main release target that builds all projects
release_projects=""
while IFS=: read -r name dir mod deps; do
	# Check if this module produces binaries
	if [ -f "$dir/main.go" ] || find "$dir" -type d -name cmd -print -quit | grep -q .; then
		release_projects="$release_projects release-$name"
	fi
done < "$INDEX"

if [ -n "$release_projects" ]; then
	cat <<EOT

# Main release target
.PHONY: release
release:$release_projects
EOT
fi
