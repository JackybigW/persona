#!/bin/bash
# Generate the changed permission screens on Xcode 26.3's iOS 18.6 simulator for #13286.
# The artifact contains direct PNG writes from the existing UI tests, their logs and xcresult bundles.
set -euo pipefail
source "$(dirname "$0")/common.sh"

[ "$(uname -s)" = Darwin ] || { printf 'permission-shots: FAIL requires macOS with Xcode\n' >&2; exit 1; }
export DEVELOPER_DIR="${DEVELOPER_DIR:-/Applications/Xcode_26.3.app/Contents/Developer}"
case "$(xcodebuild -version)" in 'Xcode 26.3'*$'\n'*) ;; *) printf 'permission-shots: FAIL select Xcode 26.3\n' >&2; exit 1 ;; esac
OUTPUT="${CI_APPLE_DIAGNOSTICS:?CI_APPLE_DIAGNOSTICS is required}"
[ ! -e "$OUTPUT" ] || { printf 'permission-shots: FAIL output already exists: %s\n' "$OUTPUT" >&2; exit 2; }
mkdir -p "$OUTPUT"/{logs,png,results}
printf 'status=incomplete; the run did not reach its final verdict\n' > "$OUTPUT/summary.txt"
WORK=$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/persona-permission-shots.XXXXXX")
SOURCE="$WORK/source"
DERIVED="$WORK/DerivedData"
SPM="${CI_SPM_CACHE:-$HOME/Library/Caches/persona-ci/spm}"
mkdir -p "$SOURCE" "$SPM"
SIM=""
GENERAL_PID=""
MODEL_PID=""
FAILED=()

cleanup() {
  local code=$?
  trap - EXIT
  for pid in "$GENERAL_PID" "$MODEL_PID"; do
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; fi
  done
  if [ -n "$SIM" ]; then
    xcrun simctl shutdown "$SIM" >/dev/null 2>&1 || true
    xcrun simctl delete "$SIM" >/dev/null 2>&1 || true
  fi
  printf 'permission-shots: %s (exit %s); evidence: %s\n' "$([ "$code" = 0 ] && printf PASS || printf FAIL)" "$code" "$OUTPUT"
  exit "$code"
}
trap cleanup EXIT

# Snapshot the exact checkout, then generate only synthetic unsigned configuration there.
git ls-files --cached --others --exclude-standard -z -- . ':(exclude).ci-cache/**' > "$WORK/inputs"
ci_run 'Snapshot permission screenshot source' 120 rsync -a --from0 --files-from="$WORK/inputs" "$CI_ROOT/" "$SOURCE/"
export PERSONA_CONFIG="$WORK/config.json"
python3 -I - "$PERSONA_CONFIG" <<'PY'
import json, sys
with open(sys.argv[1], "w") as output:
    json.dump({"app": {"bundle_id": "org.example.persona.ci", "team_id": ""}}, output)
PY
export PERSONA_META_WEARABLES=NO
export PERSONA_LEAN_DEV_BUILD=YES
# The Meta-disabled project must not retain the thirteenth Meta package pin (#11063).
ci_run 'Stage resolved packages for Meta configuration' 60 python3 "$CI_ROOT/scripts/ci/apple_resolved.py" \
  --tracked "$CI_ROOT/apple/Packages/PersonaSSH/Package.resolved" \
  --staged "$SOURCE/apple/Packages/PersonaSSH/Package.resolved" \
  --meta "$PERSONA_META_WEARABLES" --evidence "$OUTPUT/resolved-variant.txt"
ci_run 'Build PersonaCore simulator slice' 1800 "$SOURCE/native/app-core/scripts/build-apple.sh" --simulator
ci_run 'Generate synthetic Apple config' 60 python3 "$SOURCE/scripts/persona_config.py" xcconfig
ci_run 'Generate iPhone UI-test project' 120 xcodegen generate --spec "$SOURCE/apple/project.yml" --project "$SOURCE/apple"

# Use the existing compatibility lane's iOS 18.6 runtime with Xcode 26.3.
RUNTIMES="$WORK/runtimes-before.json"
xcrun simctl list runtimes -j > "$RUNTIMES"
if ! RUNTIME=$(python3 "$CI_ROOT/scripts/ci/apple_runtime.py" "$RUNTIMES" --platform iOS --version 18.6); then
  ci_run 'Download iOS 18.6 simulator runtime' 1800 xcodebuild -downloadPlatform iOS -buildVersion 18.6
  RUNTIMES="$WORK/runtimes-after.json"
  xcrun simctl list runtimes -j > "$RUNTIMES"
  RUNTIME=$(python3 "$CI_ROOT/scripts/ci/apple_runtime.py" "$RUNTIMES" --platform iOS --version 18.6)
fi
printf 'permission-shots: runtime %s\n' "$RUNTIME"
# On a hosted runner, xcodebuild owns the first boot. Directly waiting in sim-boot
# hit its 15-minute bootstatus deadline in run 45123938855 (docs/lessons/testing.md).
ci_run 'Create stock permission screenshot simulator' 100 bash -c \
  '"$1" --create-only --profile stock --runtime "$3" --device-type "iPhone 16 Pro" "$2" > "$4"' \
  -- "$SOURCE/scripts/app/sim-boot.sh" "Persona Permission Shots ${GITHUB_RUN_ID:-local}" "$RUNTIME" "$OUTPUT/simulator.log"
SIM=$(python3 -I -c 'import re,sys; value=open(sys.argv[1]).read().strip(); assert re.fullmatch(r"[0-9A-Fa-f-]{36}", value), value; print(value)' "$OUTPUT/simulator.log")

PROJECT="$SOURCE/apple/Persona.xcodeproj"
XCODE=(xcodebuild -project "$PROJECT" -scheme Persona -configuration Debug
  -destination "platform=iOS Simulator,id=$SIM" -destination-timeout 120 -derivedDataPath "$DERIVED" -parallel-testing-enabled NO
  -clonedSourcePackagesDirPath "$SPM/SourcePackages"
  CODE_SIGNING_ALLOWED=YES CODE_SIGN_IDENTITY=- CODE_SIGN_STYLE=Manual DEVELOPMENT_TEAM= PROVISIONING_PROFILE_SPECIFIER=)
# Match apple.sh's minimal resolve invocation after staging the Meta-disabled variant above.
ci_run 'Resolve permission UI-test packages' 900 xcodebuild -project "$PROJECT" -configuration Debug \
  -clonedSourcePackagesDirPath "$SPM/SourcePackages" -scheme Persona -resolvePackageDependencies
ci_run 'Build permission UI tests' 2400 "${XCODE[@]}" -disableAutomaticPackageResolution build-for-testing

free_port() {
  python3 -I -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()'
}
GENERAL_PORT=$(free_port)
MODEL_PORT=$(free_port)
MOCK_LEADERS=1 MOCK_DIRECTION_SESSIONS=1 PORT="$GENERAL_PORT" \
  python3 "$SOURCE/scripts/mock/mock_hub.py" > "$OUTPUT/logs/mock-general.log" 2>&1 &
GENERAL_PID=$!
PORT="$MODEL_PORT" python3 "$SOURCE/apple/PersonaUITests/Fixtures/lark_agent_hub.py" > "$OUTPUT/logs/mock-model.log" 2>&1 &
MODEL_PID=$!
GENERAL_HUB="http://127.0.0.1:$GENERAL_PORT"
MODEL_HUB="http://127.0.0.1:$MODEL_PORT"
READY=0
for _ in $(seq 1 80); do
  if curl -sf --max-time 2 -H 'Authorization: Bearer mock' "$GENERAL_HUB/api/v1/assistant" >/dev/null \
      && curl -sf --max-time 2 "$MODEL_HUB/mock/lark-agent" >/dev/null; then READY=1; break; fi
  sleep 0.25
done
[ "$READY" = 1 ] || { printf 'permission-shots: FAIL mock fixtures did not start\n' >&2; exit 1; }

run_case() {
  local id="$1" selection="$2" expected="$3"; shift 3
  printf '\n===== %s =====\n' "$id"
  if ci_run "$id" 900 env "$@" "${XCODE[@]}" -disableAutomaticPackageResolution -collect-test-diagnostics never \
      -resultBundlePath "$OUTPUT/results/$id.xcresult" -only-testing:"$selection" test-without-building \
      > "$OUTPUT/logs/$id.log" 2>&1; then
    if [ -s "$expected" ]; then
      printf 'PASS %s; screenshot %s\n' "$id" "${expected#"$OUTPUT/"}" | tee -a "$OUTPUT/summary.txt"
    else
      printf 'FAIL %s; test passed but screenshot is missing: %s\n' "$id" "$expected" | tee -a "$OUTPUT/summary.txt"
      FAILED+=("$id:missing-screenshot")
    fi
  else
    local code=$?
    printf 'FAIL %s (exit %s); log %s\n' "$id" "$code" "logs/$id.log" | tee -a "$OUTPUT/summary.txt"
    rg -n 'error:|XCTAssert|failed|Test Case' "$OUTPUT/logs/$id.log" | tail -n 25 || true
    FAILED+=("$id:exit-$code")
  fi
}

{
  printf 'source_sha=%s\n' "$(git rev-parse HEAD)"
  printf 'xcode=%s\n' "$(xcodebuild -version | tr '\n' ' ')"
  printf 'macos=%s\n' "$(sw_vers -productVersion)"
  printf 'simulator=%s\nruntime=%s\nsimulator_os=iOS 18.6\n' "$SIM" "$RUNTIME"
} > "$OUTPUT/environment.txt"

for LOOK in light dark; do
  for SIZE in regular xxxl ax5; do
    case "$SIZE" in
      regular) CONTENT_SIZE=UICTContentSizeCategoryL ;;
      xxxl) CONTENT_SIZE=UICTContentSizeCategoryXXXL ;;
      ax5) CONTENT_SIZE=UICTContentSizeCategoryAccessibilityXXXL ;;
    esac
    LOOK_ENV=(TEST_RUNNER_PERSONA_UI_SCHEME="$LOOK" TEST_RUNNER_PERSONA_UI_CONTENT_SIZE="$CONTENT_SIZE")
    ID="$LOOK-$SIZE"
    SHOTS="$OUTPUT/png/$ID"
    mkdir -p "$SHOTS"/{followup,direction,inbox}

    run_case "$ID-followup" \
      'PersonaUITests/AgentFollowUpUITests/testNewConversationPermissionFromAgentSettings' \
      "$SHOTS/followup/followup-iphone-$LOOK-$SIZE-default-permissions.png" \
      "${LOOK_ENV[@]}" \
      TEST_RUNNER_PERSONA_MOCK_HUB="$GENERAL_HUB" \
      TEST_RUNNER_PERSONA_SHOTS="$SHOTS/followup" \
      TEST_RUNNER_PERSONA_SHOTS_APPEARANCE="$LOOK-$SIZE" \
      TEST_RUNNER_PERSONA_SHOTS_DEVICE=iphone

    run_case "$ID-direction" \
      'PersonaUITests/DirectionScreensUITests/testDefaultPermissionsEntry' \
      "$SHOTS/direction/default-permissions-en-$LOOK-$SIZE.png" \
      "${LOOK_ENV[@]}" \
      TEST_RUNNER_PERSONA_DIRECTION_SHOTS=1 \
      TEST_RUNNER_PERSONA_LEADERS_HUB="$GENERAL_HUB" \
      TEST_RUNNER_SHOT_DIR="$SHOTS/direction" \
      TEST_RUNNER_SHOT_SUFFIX="$LOOK-$SIZE" \
      TEST_RUNNER_PERSONA_SHOT_LANG=en

    if [ "$LOOK" = light ]; then INBOX_TEST=testLightTwinPermissionSaveAndReset; else INBOX_TEST=testDarkTwinPermissionSaveAndReset; fi
    run_case "$ID-inbox" \
      "PersonaUITests/LarkAgentModelConfigurationTests/$INBOX_TEST" \
      "$SHOTS/inbox/$LOOK-twin-permission-full-access.png" \
      "${LOOK_ENV[@]}" \
      TEST_RUNNER_PERSONA_MOCK_HUB="$MODEL_HUB" \
      TEST_RUNNER_PERSONA_MODEL_SHOTS="$SHOTS/inbox"
  done
done

if [ "${#FAILED[@]}" -gt 0 ]; then
  printf 'permission-shots: FAIL %s\n' "${FAILED[*]}" | tee -a "$OUTPUT/summary.txt"
  exit 1
fi
printf 'permission-shots: PASS all 18 cases and screenshots\n' | tee -a "$OUTPUT/summary.txt"
