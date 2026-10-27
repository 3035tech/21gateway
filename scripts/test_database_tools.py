"""Exercise safety/control flow with fake CLIs, not PostgreSQL or age encryption."""
import json
import os
from pathlib import Path
import stat
import subprocess
import tempfile
import unittest

SCRIPTS = Path(__file__).resolve().parent

class DatabaseToolsTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        tools = self.root / 'bin'
        tools.mkdir()
        self.log = self.root / 'calls.jsonl'
        identity = self.root / 'identity'
        identity.write_text('test fixture, not a real key')
        self.env = {**os.environ, 'PATH': f'{tools}:{os.environ["PATH"]}', 'BACKUP_RECIPIENT': 'test-recipient', 'BACKUP_IDENTITY': str(identity), 'FAKE_LOG': str(self.log)}
        docker = tools / 'docker'
        docker.write_text('''#!/usr/bin/env python3
import json,os,sys
args=sys.argv[1:]
script=args[args.index('-c')+1]
action=next(x for x in ['pg_dump','createdb','pg_restore','psql','dropdb'] if x in script)
with open(os.environ['FAKE_LOG'],'a') as f:f.write(json.dumps([action,args[-1]])+'\\n')
if action=='pg_dump':
 if os.environ.get('FAKE_DUMP_FAIL'):sys.exit(1)
 sys.stdout.buffer.write(b'fixture dump')
elif action=='createdb' and os.environ.get('FAKE_CREATE_FAIL'):sys.exit(1)
elif action=='pg_restore':
 sys.stdin.buffer.read()
 if os.environ.get('FAKE_RESTORE_FAIL'):sys.exit(1)
elif action=='psql':print('t')
''')
        age = tools / 'age'
        age.write_text('''#!/usr/bin/env python3
import pathlib,sys
if '-d' in sys.argv:sys.stdout.buffer.write(pathlib.Path(sys.argv[-1]).read_bytes())
else:sys.stdout.buffer.write(sys.stdin.buffer.read())
''')
        for path in [docker, age]: path.chmod(0o700)

    def run_script(self, name, argument, **env):
        return subprocess.run(['bash', str(SCRIPTS / name), str(argument)], env={**self.env, **env}, capture_output=True, text=True)

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def test_backup_is_atomic_and_private(self):
        dest = self.root / 'backups'
        result = self.run_script('database-backup.sh', dest)
        self.assertEqual(result.returncode, 0, result.stderr)
        files = list(dest.iterdir())
        self.assertEqual(len(files), 1)
        self.assertEqual(stat.S_IMODE(files[0].stat().st_mode), 0o600)
        self.assertEqual(files[0].read_bytes(), b'fixture dump')
        result = self.run_script('database-backup.sh', dest, FAKE_DUMP_FAIL='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(list(dest.iterdir()), files)

    def test_restore_uses_owned_scratch_and_cleans_failed_restore(self):
        archive = self.root / 'backup.dump.age'
        archive.write_bytes(b'fixture dump')
        for failed in [False, True]:
            if self.log.exists(): self.log.unlink()
            result = self.run_script('database-restore-check.sh', archive, FAKE_RESTORE_FAIL='1' if failed else '')
            self.assertEqual(result.returncode == 0, not failed, result.stderr)
            calls = self.calls()
            created = [database for action, database in calls if action == 'createdb']
            dropped = [database for action, database in calls if action == 'dropdb']
            self.assertEqual(created, dropped)
            self.assertEqual(len(created), 1)
            self.assertTrue(created[0].startswith('restore_21mobile_'))

    def test_failed_create_never_drops_a_database(self):
        archive = self.root / 'backup.dump.age'
        archive.write_bytes(b'fixture dump')
        result = self.run_script('database-restore-check.sh', archive, FAKE_CREATE_FAIL='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(action == 'dropdb' for action, _ in self.calls()))

if __name__ == '__main__': unittest.main()
