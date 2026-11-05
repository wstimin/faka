"""Exercise start.sh in a temporary directory; never modify real users or services."""
import pathlib
import shlex
import shutil
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[2]
bash = shutil.which('bash')
if not bash and pathlib.Path('C:/Program Files/Git/bin/bash.exe').is_file():
    bash = 'C:/Program Files/Git/bin/bash.exe'
if not bash:
    raise SystemExit('bash is required')
source = (root / 'start.sh').read_text(encoding='utf-8')
with tempfile.TemporaryDirectory(prefix='dujiao-start-') as temporary:
    base = pathlib.Path(temporary)
    package, app = base / 'package', base / 'app'
    package.mkdir()
    app.mkdir()
    token = app / '.install-token'
    token.write_text('private-installation-code')
    (package / 'dujiao-next').write_text('new executable')
    (app / 'dujiao-next').write_text('old executable')
    log = base / 'service-calls.log'
    script = source.replace('app_dir=/opt/dujiao-next-vmq', 'app_dir=' + shlex.quote(app.as_posix()))
    script = script.replace('/etc/systemd/system/dujiao-next-vmq.service', (base / 'service.unit').as_posix())
    mocks = '\n'.join([
        'id() { if [[ ${1:-} == -u ]]; then printf 0; fi; }',
        'chown() { :; }',
        'systemctl() { printf "%s\\n" "$*" >> ' + shlex.quote(log.as_posix()) + '; }',
    ])
    script = script.replace('set -euo pipefail', 'set -euo pipefail\n' + mocks, 1)
    path = package / 'start.sh'
    path.write_text(script, encoding='utf-8', newline='\n')
    for attempt in range(2):
        result = subprocess.run([bash, str(path)], capture_output=True, text=True, encoding='utf-8')
        assert result.returncode == 0, result.stdout + result.stderr
        assert (app / 'dujiao-next').read_text() == 'new executable'
        assert token.read_text() == 'private-installation-code'
        assert not list(app.glob('*.new'))
    calls = log.read_text()
    assert calls.count('restart dujiao-next-vmq.service') == 2, calls
    (app / 'db').mkdir()
    (app / 'db/config.yml').write_text('existing configuration')
    (package / 'dujiao-next').write_text('must not replace')
    result = subprocess.run([bash, str(path)], capture_output=True, text=True, encoding='utf-8')
    assert result.returncode != 0
    assert (app / 'dujiao-next').read_text() == 'new executable'
    assert (app / 'db/config.yml').read_text() == 'existing configuration'
    assert log.read_text() == calls
print('PASS: start and retry preserve authorization; configured sites cannot be overwritten')
