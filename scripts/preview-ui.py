"""Loopback-only preview. Mock assets stay outside the embedded production frontend."""
from http.server import ThreadingHTTPServer, SimpleHTTPRequestHandler
from pathlib import Path
from urllib.parse import urlsplit, unquote
import argparse
import mimetypes

REPO = Path(__file__).resolve().parents[1]
STATIC = REPO / 'gui' / 'static'
PREVIEW = REPO / 'scripts' / 'preview'

class Handler(SimpleHTTPRequestHandler):
    def do_GET(self):
        path = unquote(urlsplit(self.path).path)
        if path in ('/', '/index.html'):
            target = PREVIEW / 'index.html'
        elif path in ('/app', '/app/', '/app/index.html'):
            page = (STATIC / 'index.html').read_text(encoding='utf-8-sig')
            page = page.replace('<script type="module" src="/app.js">', '<script type="module" src="/preview/inject.mjs"></script><script type="module" src="/app.js">')
            self.send_data(page.encode(), 'text/html; charset=utf-8')
            return
        else:
            root = PREVIEW if path.startswith('/preview/') else STATIC
            relative = path[len('/preview/'):] if root == PREVIEW else path.lstrip('/')
            target = (root / relative).resolve()
            if not target.is_relative_to(root.resolve()):
                self.send_error(403)
                return
        if not target.is_file():
            self.send_error(404)
            return
        kind = 'text/javascript' if target.suffix in ('.js', '.mjs') else mimetypes.guess_type(str(target))[0] or 'application/octet-stream'
        self.send_data(target.read_bytes(), kind)

    def send_data(self, data, kind):
        self.send_response(200)
        self.send_header('Content-Type', kind)
        self.send_header('Cache-Control', 'no-store')
        self.send_header('Content-Length', str(len(data)))
        self.end_headers()
        self.wfile.write(data)

class PreviewServer(ThreadingHTTPServer):
    allow_reuse_address = False

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, default=4174)
    args = parser.parse_args()
    server = PreviewServer(('127.0.0.1', args.port), Handler)
    print(f'RouteDesk local preview: http://127.0.0.1:{args.port}', flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        server.server_close()
