"""Read-only HTTP checks against a temporary loopback preview server and static Pages builder."""
import importlib.util
from pathlib import Path
import tempfile
import threading
import unittest
import urllib.request
import urllib.error

spec = importlib.util.spec_from_file_location('preview', Path(__file__).with_name('preview-ui.py'))
preview = importlib.util.module_from_spec(spec)
spec.loader.exec_module(preview)

build_spec = importlib.util.spec_from_file_location('build_pages', Path(__file__).with_name('build-pages.py'))
build_pages_mod = importlib.util.module_from_spec(build_spec)
build_spec.loader.exec_module(build_pages_mod)


class PreviewHTTPTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = preview.PreviewServer(('127.0.0.1', 0), preview.Handler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.base = 'http://127.0.0.1:' + str(cls.server.server_address[1])

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def test_entry_and_modules(self):
        for path in ['/', '/app', '/app/', '/preview/mock.mjs', '/core/themes.js', '/themes.css', '/icons/pencil.svg']:
            with urllib.request.urlopen(self.base + path) as response:
                self.assertEqual(response.status, 200)

    def test_preview_bridge_stays_outside_production(self):
        page = urllib.request.urlopen(self.base + '/app').read().decode()
        self.assertIn('/preview/inject.mjs', page)
        self.assertNotIn('/preview/', (preview.STATIC / 'index.html').read_text(encoding='utf-8'))

    def test_no_parent_or_source_access(self):
        for path in ['/../go.mod', '/preview/../../go.mod', '/gui/server.go']:
            with self.assertRaises(urllib.error.HTTPError) as caught:
                urllib.request.urlopen(self.base + path)
            self.assertIn(caught.exception.code, [403, 404])

    def test_static_pages_build(self):
        with tempfile.TemporaryDirectory() as tmp:
            out = build_pages_mod.build_pages(Path(tmp) / 'pages')
            for rel in [
                '.nojekyll',
                'index.html',
                'icons/route.svg',
                'preview/workbench.css',
                'preview/workbench.mjs',
                'preview/inject.mjs',
                'preview/mock.mjs',
                'app/index.html',
                'app/app.js',
                'app/style.css',
                'app/themes.css',
                'app/icons.svg',
                'app/core/ui.js',
            ]:
                self.assertTrue((out / rel).is_file(), f'Missing {rel}')

            wb_html = (out / 'index.html').read_text(encoding='utf-8')
            self.assertIn('./preview/workbench.mjs', wb_html)
            self.assertIn('./app/#home', wb_html)
            self.assertNotIn('"/app"', wb_html)

            app_html = (out / 'app' / 'index.html').read_text(encoding='utf-8')
            self.assertIn('../preview/inject.mjs', app_html)
            self.assertIn('./app.js', app_html)
            self.assertNotIn('"/style.css"', app_html)

            ui_js = (out / 'app' / 'core' / 'ui.js').read_text(encoding='utf-8')
            self.assertIn('./icons.svg#', ui_js)
            self.assertNotIn('href="/icons.svg#', ui_js)

            wb_js = (out / 'preview' / 'workbench.mjs').read_text(encoding='utf-8')
            self.assertIn('./app/?', wb_js)


if __name__ == '__main__':
    unittest.main()
