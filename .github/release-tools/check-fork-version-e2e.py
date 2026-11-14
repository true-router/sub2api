#!/usr/bin/env python3
"""Exercise the fork release contract through real Git, CLI, and application builds."""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import tarfile
import zipfile

ROOT = Path(__file__).resolve().parents[2]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    output = Path(args.output).resolve()
    checks = []
    with tempfile.TemporaryDirectory(prefix='sub2api-version-e2e-') as temp:
        work = Path(temp)
        fixture = work / 'repo'
        fixture.mkdir()
        for name in ('.goreleaser.yaml', '.goreleaser.simple.yaml', 'Dockerfile.goreleaser',
                     'deploy/docker-entrypoint.sh', 'backend/Makefile', 'deploy/Makefile',
                     '.github/release-tools/release_matrix.py',
                     'backend/scripts/resolve-version.sh', 'backend/cmd/server/VERSION',
                     'backend/cmd/server/FORK_VERSION'):
            dest = fixture / name
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(ROOT / name, dest)
        shutil.copytree(ROOT / 'backend/resources', fixture / 'backend/resources')
        # Keep fixture versions independent of the next real release number.
        (fixture / 'backend/cmd/server/FORK_VERSION').write_text('0.2.25\n')
        env = {**os.environ, 'GITHUB_OUTPUT': str(work / 'outputs'),
               'GITHUB_REPOSITORY_OWNER': 'true-router', 'GITHUB_REPOSITORY': 'true-router/sub2api', 'GIT_CONFIG_NOSYSTEM': '1',
               'GIT_CONFIG_GLOBAL': os.devnull}

        def run(command, cwd=fixture, ok=True):
            result = subprocess.run(command, cwd=cwd, env=env, text=True,
                                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            if (result.returncode == 0) != ok:
                raise RuntimeError(f'{command}: unexpected exit {result.returncode}\n{result.stdout}')
            return result.stdout.strip()

        def check(name, condition):
            if not condition:
                raise AssertionError(name)
            checks.append(name)

        run(['git', 'init', '-b', 'main'])
        run(['git', 'config', 'user.name', 'Version E2E'])
        run(['git', 'config', 'user.email', 'version-e2e@example.invalid'])
        run(['git', 'add', '.'])
        run(['git', 'commit', '-m', 'fixture'])
        version_file = fixture / 'backend/cmd/server/FORK_VERSION'
        upstream_file = fixture / 'backend/cmd/server/VERSION'
        version = version_file.read_text().strip()
        original_upstream = upstream_file.read_bytes()
        resolver = ['sh', 'backend/scripts/resolve-version.sh']
        planner = ['python3', '.github/release-tools/release_matrix.py', 'plan']
        check('untagged build reads fork version', run(resolver) == version)
        for makefile in ('Makefile', '../deploy/Makefile'):
            run(['make', '-f', makefile, 'check-version'], cwd=fixture / 'backend')
            run(['make', '-f', makefile, 'check-version', 'VERSION=99.0.0'], cwd=fixture / 'backend', ok=False)
            checks.append(f'{makefile} rejects explicit version override')
        run(['git', 'tag', 'v99.0.0'])
        check('source build ignores upstream tag', run(resolver) == version)
        check('publication rejects upstream tag', 'tr-v' in run(planner + ['--ref', 'v99.0.0'], ok=False))
        run(['git', 'tag', 'tr-v' + version])
        check('matching fork tag resolves', run(resolver) == version)
        for simple in (False, True):
            Path(env['GITHUB_OUTPUT']).write_text('')
            run(planner + ['--ref', 'tr-v' + version] + (['--simple'] if simple else []))
            values = dict(line.split('=', 1) for line in Path(env['GITHUB_OUTPUT']).read_text().splitlines())
            check(f'release plan binds version and tag (simple={simple})',
                  values['version'] == version and values['tag'] == 'tr-v' + version
                  and len(json.loads(values['matrix'])['include']) == (1 if simple else 5))
        check('release planning preserves upstream version', upstream_file.read_bytes() == original_upstream)
        check('release planning preserves fork version', version_file.read_text().strip() == version)
        run(['git', 'tag', '-d', 'tr-v' + version])
        run(['git', 'tag', 'tr-v99.0.0'])
        run(resolver, ok=False)
        run(['make', 'check-version'], cwd=fixture / 'backend', ok=False)
        run(planner + ['--ref', 'tr-v99.0.0'], ok=False)
        checks.append('mismatched release tag rejected by source and release builds')
        run(['git', 'tag', '-d', 'tr-v99.0.0'])
        run(['git', 'checkout', '-b', 'upstream'])
        upstream_file.write_text('99.0.0\n')
        run(['git', 'add', '.'])
        run(['git', 'commit', '-m', 'upstream version bump'])
        run(['git', 'checkout', 'main'])
        version_file.write_text('0.2.26\n')
        run(['git', 'add', '.'])
        run(['git', 'commit', '-m', 'independent fork bump'])
        run(['git', 'merge', '--no-edit', 'upstream'])
        check('upstream merge preserves both independent versions',
              upstream_file.read_text().strip() == '99.0.0' and run(resolver) == '0.2.26')
        run(['git', 'tag', 'tr-v0.2.26'])
        run(['git', 'commit', '--allow-empty', '-m', 'new source commit'])
        run(planner + ['--ref', 'tr-v0.2.26'], ok=False)
        checks.append('publication rejects tag pointing at another commit')
        Path(env['GITHUB_OUTPUT']).write_text('')
        run(planner + ['--ref', 'main', '--dry-run'])
        values = dict(line.split('=', 1) for line in Path(env['GITHUB_OUTPUT']).read_text().splitlines())
        check('dry run uses independent fork version', values['version'] == '0.2.26')
        release_sha = run(['git', 'rev-parse', 'HEAD'])
        tool = ['python3', '.github/release-tools/release_matrix.py']
        # Exercise the real collection, matrix verification, and publication CLI.
        # Archive payloads are fixtures; this does not claim cross-platform builds.
        for simple in (False, True):
            Path(env['GITHUB_OUTPUT']).write_text('')
            run(planner + ['--ref', 'main', '--dry-run'] + (['--simple'] if simple else []))
            values = dict(line.split('=', 1) for line in Path(env['GITHUB_OUTPUT']).read_text().splitlines())
            targets = json.loads(values['matrix'])['include']
            input_dir = 'release-input-' + str(simple)
            (fixture / 'dist').mkdir(exist_ok=True)
            for target in targets:
                suffix = 'zip' if target['goos'] == 'windows' else 'tar.gz'
                name = f"sub2api_0.2.26_{target['goos']}_{target['goarch']}.{suffix}"
                archive = fixture / 'dist' / name
                payload = b'isolated archive payload'
                if suffix == 'zip':
                    with zipfile.ZipFile(archive, 'w') as stream:
                        stream.writestr('sub2api.exe', payload)
                else:
                    with tarfile.open(archive, 'w:gz') as stream:
                        member = tarfile.TarInfo('sub2api')
                        member.size, member.mode = len(payload), 0o755
                        stream.addfile(member, io.BytesIO(payload))
                digest = hashlib.sha256(archive.read_bytes()).hexdigest()
                (fixture / 'dist/checksums.txt').write_text(f'{digest}  {name}\n')
                run(tool + ['collect', '--version', '0.2.26', '--sha', release_sha,
                            '--goos', target['goos'], '--goarch', target['goarch'], '--output', input_dir])
            destination = 'publication-' + str(simple)
            arguments = ['--version', '0.2.26', '--sha', release_sha, '--input', input_dir]
            run(tool + ['contexts'] + arguments + ['--output', 'contexts-' + str(simple)] + (['--simple'] if simple else []))
            run(tool + ['publication'] + arguments + ['--output', destination] + (['--simple'] if simple else []))
            metadata = json.loads((fixture / destination / 'metadata.json').read_text())
            check(f'publication preserves numeric assets and real fork tag (simple={simple})',
                  metadata['tag'] == 'tr-v0.2.26' and metadata['sha'] == release_sha
                  and len(metadata['sha256']) == len(targets)
                  and all(name.startswith('sub2api_0.2.26_') for name in metadata['sha256']))
            checksums = (fixture / destination / 'checksums.txt').read_text().splitlines()
            check(f'publication checksums match collected archives (simple={simple})',
                  all(hashlib.sha256((fixture / input_dir / line.split()[1]).read_bytes()).hexdigest()
                      == line.split()[0] for line in checksums))
            archive = next((fixture / input_dir).glob('sub2api_*'))
            archive.write_bytes(b'corruption')
            run(tool + ['publication'] + arguments + ['--output', destination] + (['--simple'] if simple else []), ok=False)
            checks.append(f'corrupted publication rejected (simple={simple})')
        # Run the real update HTTP flow as a subprocess with only local upstream fixtures.
        update_output = work / 'update.json'
        run(['go', 'run', './e2e/fork-version', '--output', str(update_output)], cwd=ROOT / 'backend')
        update = json.loads(update_output.read_text())
        check('real update HTTP flow passed with cleanup', update['result'] == 'PASS' and update['fixture_services_removed'])
        shutil.rmtree(fixture / '.git')
        check('source archive resolves without Git metadata', run(resolver) == '0.2.26')
        binary = work / 'sub2api'
        run(['go', 'build', '-trimpath', '-o', str(binary), './cmd/server'], cwd=ROOT / 'backend')
        expected = (ROOT / 'backend/cmd/server/FORK_VERSION').read_text().strip()
        binary_version = run([str(binary), '--version'])
        check('real application embeds fork version', f'Sub2API {expected} ' in binary_version)
        binary_hash = hashlib.sha256(binary.read_bytes()).hexdigest()
    evidence = {'result': 'PASS', 'checks': checks, 'binary_version': binary_version,
                'binary_sha256': binary_hash, 'fixture_services_removed': True, 'update_http': update,
                'boundary': 'Real Git, release CLI, checksum verification, application binary and update HTTP flow. Packaging payloads and GitHub responses are local fixtures; no CI publication, registry push or production deployment.',
                'source_sha256': {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest()
                                  for p in [ROOT / 'backend/cmd/server/main.go',
                                            ROOT / 'backend/cmd/server/FORK_VERSION',
                                            ROOT / 'backend/scripts/resolve-version.sh',
                                            ROOT / '.github/release-tools/release_matrix.py',
                                            ROOT / '.github/release-tools/check-fork-version-e2e.py',
                                            ROOT / 'AGENTS.md', ROOT / 'backend/Makefile',
                                            ROOT / 'deploy/Makefile', ROOT / 'Dockerfile',
                                            ROOT / 'deploy/Dockerfile', ROOT / 'deploy/install.sh',
                                            ROOT / 'backend/internal/service/update_service.go',
                                            ROOT / 'backend/e2e/fork-version/main.go',
                                            ROOT / 'frontend/src/components/common/VersionBadge.vue',
                                            ROOT / '.github/workflows/release.yml',
                                            ROOT / '.github/workflows/backend-ci.yml',
                                            ROOT / '.github/workflows/production-rollout.yml',
                                            ROOT / '.github/scripts/notify-rollout.py']}}
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(evidence, indent=2) + '\n')
    print(f'PASS: {len(checks)} checks; evidence: {output}')


if __name__ == '__main__':
    main()
