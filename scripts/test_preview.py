"""Read-only HTTP checks against a temporary loopback preview server."""
import importlib.util
from pathlib import Path
import threading
import unittest
import urllib.request
import urllib.error
spec=importlib.util.spec_from_file_location('preview',Path(__file__).with_name('preview-ui.py'))
preview=importlib.util.module_from_spec(spec)
spec.loader.exec_module(preview)
class PreviewHTTPTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.server=preview.PreviewServer(('127.0.0.1',0),preview.Handler)
  cls.thread=threading.Thread(target=cls.server.serve_forever,daemon=True);cls.thread.start()
  cls.base='http://127.0.0.1:'+str(cls.server.server_address[1])
 @classmethod
 def tearDownClass(cls):cls.server.shutdown();cls.server.server_close();cls.thread.join()
 def test_entry_and_modules(self):
  for path in ['/','/app','/app/','/preview/mock.mjs','/core/themes.js','/themes.css','/icons/pencil.svg']:
   with urllib.request.urlopen(self.base+path) as response:self.assertEqual(response.status,200)
 def test_preview_bridge_stays_outside_production(self):
  page=urllib.request.urlopen(self.base+'/app').read().decode()
  self.assertIn('/preview/inject.mjs',page)
  self.assertNotIn('/preview/',(preview.STATIC/'index.html').read_text(encoding='utf-8'))
 def test_no_parent_or_source_access(self):
  for path in ['/../go.mod','/preview/../../go.mod','/gui/server.go']:
   with self.assertRaises(urllib.error.HTTPError) as caught:urllib.request.urlopen(self.base+path)
   self.assertIn(caught.exception.code,[403,404])
if __name__=='__main__':unittest.main()
