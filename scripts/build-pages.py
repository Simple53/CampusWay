"""Build static GitHub Pages interactive preview bundle into dist/pages/."""
from pathlib import Path
import argparse
import shutil

REPO = Path(__file__).resolve().parents[1]
STATIC = REPO / 'gui' / 'static'
PREVIEW = REPO / 'scripts' / 'preview'


def build_pages(output_dir: Path | None = None) -> Path:
    out = (output_dir or (REPO / 'dist' / 'pages')).resolve()
    if out.exists():
        shutil.rmtree(out)
    out.mkdir(parents=True, exist_ok=True)

    # 1. Copy production UI static files into dist/pages/app/ and rewrite root paths
    app_dir = out / 'app'
    shutil.copytree(STATIC, app_dir)

    app_index = app_dir / 'index.html'
    app_html = app_index.read_text(encoding='utf-8-sig')
    app_html = (
        app_html
        .replace('href="/icons/route.svg"', 'href="./icons/route.svg"')
        .replace('href="/style.css"', 'href="./style.css"')
        .replace('href="/themes.css"', 'href="./themes.css"')
        .replace('<script type="module" src="/app.js"></script>', '<script type="module" src="../preview/inject.mjs"></script><script type="module" src="./app.js"></script>')
    )
    app_index.write_text(app_html, encoding='utf-8')

    ui_js = app_dir / 'core' / 'ui.js'
    ui_code = ui_js.read_text(encoding='utf-8-sig').replace('/icons.svg#', './icons.svg#')
    ui_js.write_text(ui_code, encoding='utf-8')

    # 2. Copy preview runtime assets into dist/pages/preview/
    preview_dir = out / 'preview'
    preview_dir.mkdir(parents=True, exist_ok=True)
    for name in ('workbench.css', 'workbench.mjs', 'inject.mjs', 'mock.mjs'):
        content = (PREVIEW / name).read_bytes()
        if name == 'workbench.mjs':
            text = content.decode('utf-8-sig').replace("'/app?", "'./app/?")
            content = text.encode('utf-8')
        (preview_dir / name).write_bytes(content)

    # 3. Copy favicon for workbench root
    icons_dir = out / 'icons'
    icons_dir.mkdir(parents=True, exist_ok=True)
    shutil.copy2(STATIC / 'icons' / 'route.svg', icons_dir / 'route.svg')

    # 4. Generate static workbench index.html with relative paths for subpath hosting
    wb_html = (PREVIEW / 'index.html').read_text(encoding='utf-8-sig')
    wb_html = (
        wb_html
        .replace('href="/"', 'href="./"')
        .replace('href="/icons/route.svg"', 'href="./icons/route.svg"')
        .replace('href="/preview/workbench.css"', 'href="./preview/workbench.css"')
        .replace('src="/app#home"', 'src="./app/#home"')
        .replace('href="/app"', 'href="./app/"')
        .replace('src="/preview/workbench.mjs"', 'src="./preview/workbench.mjs"')
    )
    (out / 'index.html').write_text(wb_html, encoding='utf-8')

    # 5. Disable Jekyll processing on GitHub Pages
    (out / '.nojekyll').write_text('', encoding='utf-8')
    return out


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Build static RouteDesk preview site for GitHub Pages.')
    parser.add_argument('--output', type=Path, default=None)
    args = parser.parse_args()
    target = build_pages(args.output)
    print(f'Built static Pages preview at {target}')
