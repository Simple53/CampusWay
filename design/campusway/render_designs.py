from pathlib import Path
from html import escape
import urllib.request
import xml.etree.ElementTree as ET

# 设计交付使用 SVG，覆盖 Product Design 默认位图探索流程；不触碰现有 VPN 界面。
# 所有连接状态都是构图示例，文案只呈现产品内容，设计解释留在注释。
ROOT = Path(__file__).resolve().parent
ASSETS = ROOT / 'icons'
NAMES = ['route', 'plus', 'settings-2', 'activity', 'arrow-right',
         'chevron-down', 'check', 'search', 'more-horizontal', 'shield',
         'server', 'circle-help']
SVG = '{http://www.w3.org/2000/svg}'
ET.register_namespace('', 'http://www.w3.org/2000/svg')


def fetch_icons():
    ASSETS.mkdir(exist_ok=True)
    base = 'https://raw.githubusercontent.com/lucide-icons/lucide/main/'
    aliases = {'more-horizontal': 'ellipsis', 'circle-help': 'circle-question-mark'}
    for name, relative in [(n + '.svg', 'icons/' + aliases.get(n, n) + '.svg') for n in NAMES] + [('LICENSE', 'LICENSE')]:
        target = ASSETS / name
        if not target.exists():
            with urllib.request.urlopen(base + relative, timeout=25) as response:
                target.write_bytes(response.read())


def definitions():
    result = []
    for name in NAMES:
        root = ET.parse(ASSETS / (name + '.svg')).getroot()
        shapes = ''.join(ET.tostring(child, encoding='unicode') for child in root)
        result.append(f'<symbol id="{name}" viewBox="0 0 24 24">{shapes}</symbol>')
    return '<defs>' + ''.join(result) + '</defs>'


CSS = '''
:root{color-scheme:light dark;--bg:#ffffff;--side:#f6f6f6;--fill:#f0f0f0;--line:#e3e3e3;--ink:#191919;--muted:#707070;--action:#202020;--on:#ffffff}
@media(prefers-color-scheme:dark){:root{--bg:#202020;--side:#181818;--fill:#2c2c2c;--line:#3b3b3b;--ink:#ededed;--muted:#aaaaaa;--action:#ededed;--on:#202020}}
.bg{fill:var(--bg)}.side{fill:var(--side)}.fill{fill:var(--fill)}
.line{stroke:var(--line);stroke-width:1}.border{stroke:var(--line);stroke-width:1;fill:none}
text{fill:var(--ink);font:14px 'Segoe UI','Microsoft YaHei',sans-serif}
.muted{fill:var(--muted)}.small{font-size:12px}.title{font-size:28px;font-weight:600;letter-spacing:-.6px}
.heading{font-size:18px;font-weight:600}.brand{font-size:19px;font-weight:600}
.mono{font-family:'Cascadia Code',Consolas,monospace;font-size:13px}
.icon{fill:none;stroke:currentColor;stroke-width:1.65;stroke-linecap:round;stroke-linejoin:round;color:var(--ink)}
.primary{fill:var(--action)}.on{fill:var(--on);font-weight:600}
'''


def text(x, y, value, cls='', anchor='start'):
    return f'<text x="{x}" y="{y}" class="{cls}" text-anchor="{anchor}">{escape(value)}</text>'


def rect(x, y, w, h, cls='fill', r=8):
    return f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="{r}" class="{cls}"/>'


def line(x, y, x2, y2):
    return f'<path d="M{x} {y}H{x2}" class="line"/>' if y == y2 else f'<path d="M{x} {y}L{x2} {y2}" class="line"/>'


def icon(name, x, y, size=20):
    return f'<use href="#{name}" x="{x}" y="{y}" width="{size}" height="{size}" class="icon"/>'


def button(x, y, w, label, primary=False, h=42):
    return rect(x, y, w, h, 'primary' if primary else 'fill') + text(x+w/2, y+h/2+5, label, 'on' if primary else '', 'middle')


def brand(x, y):
    return icon('route', x, y-20, 25) + text(x+36, y, '校径', 'brand')


def shell_desktop(active='线路'):
    s = rect(0, 0, 1120, 760, 'bg', 0) + rect(0, 0, 208, 760, 'side', 0)
    s += line(208, 0, 208, 760) + line(208, 54, 1120, 54)
    s += brand(24, 41) + text(245, 33, active, 'muted small')
    s += '<path d="M1008 27h12M1053 21h12v12h-12zM1093 21l12 12m0-12l-12 12" class="icon"/>'
    for label, glyph, y in [('线路', 'route', 104), ('活动', 'activity', 152)]:
        if active == label:
            s += rect(12, y-25, 184, 40)
        s += icon(glyph, 26, y-15) + text(60, y, label)
    s += icon('settings-2', 26, 671) + text(60, 687, '设置')
    s += icon('circle-help', 26, 715) + text(60, 731, '关于校径')
    return s


def mobile_header(title='校径'):
    return rect(0, 0, 390, 844, 'bg', 0) + (brand(24, 47) if title == '校径' else text(24, 47, title, 'brand')) + icon('plus', 342, 27, 24)


def mobile_nav():
    s = rect(0, 760, 390, 84, 'bg', 0) + line(0, 760, 390, 760)
    for x, label, glyph in [(65, '线路', 'route'), (195, '活动', 'activity'), (325, '设置', 'settings-2')]:
        if label == '线路':
            s += rect(x-30, 769, 60, 34, 'fill', 16)
        s += icon(glyph, x-11, 775, 22) + text(x, 824, label, 'small', 'middle')
    return s


PROFILES = [('校园入口', '南京师范大学', 'EasyConnect'),
            ('实验室', '研究网络', 'EasyConnect'),
            ('联合校园', '协作网络', 'aTrust')]


def profile_row(x, y, w, index, selected=False):
    name, campus, protocol = PROFILES[index]
    s = rect(x, y, w, 76, 'fill', 8) if selected else ''
    s += icon('server', x+16, y+26, 21) + text(x+50, y+30, name)
    s += text(x+50, y+52, campus + ' / ' + protocol, 'muted small')
    s += icon('check' if selected else 'more-horizontal', x+w-36, y+27, 18)
    return s


def desktop_one():
    s = shell_desktop() + line(510, 54, 510, 760)
    s += text(236, 102, '我的线路', 'heading') + icon('plus', 463, 84, 23)
    s += rect(232, 123, 254, 38) + icon('search', 244, 133, 18) + text(273, 148, '搜索线路', 'muted small')
    for i in range(3):
        s += profile_row(220, 183+i*88, 278, i, i == 0)
    s += text(236, 714, '3 条线路', 'muted small')
    s += text(558, 109, '南京师范大学', 'muted') + icon('more-horizontal', 1050, 93)
    s += text(558, 157, '校园入口', 'title')
    s += rect(558, 178, 86, 28, 'fill', 14) + text(601, 197, '未连接', 'small', 'middle')
    for y, label, value in [(264, '网关', 'vpn.njnu.edu.cn'), (319, '接入协议', 'EasyConnect'), (374, '访问范围', '校园资源'), (429, '接入模式', '本地代理')]:
        s += text(558, y, label, 'muted') + text(1058, y, value, 'mono' if label == '网关' else '', 'end') + line(558, y+20, 1058, y+20)
    s += button(558, 493, 500, '连接此线路', True, 46)
    s += text(558, 571, '连接后可访问学校授权的网络资源。', 'muted small')
    s += text(558, 696, '账户与认证', 'muted small') + text(1058, 696, '管理', '', 'end')
    return s


def mobile_one():
    s = mobile_header() + text(24, 109, '我的线路', 'title') + text(24, 140, '选择一条线路，连接校园网络。', 'muted')
    for i in range(3):
        s += profile_row(16, 169+i*88, 358, i, i == 0)
    s += line(24, 454, 366, 454) + text(24, 488, '当前选择', 'muted small')
    s += text(24, 521, '校园入口', 'heading') + text(366, 521, '未连接', 'muted small', 'end')
    s += text(24, 553, 'vpn.njnu.edu.cn', 'mono muted')
    s += text(24, 595, '访问范围', 'muted') + text(366, 595, '校园资源', '', 'end')
    s += button(24, 654, 342, '连接此线路', True, 50)
    return s + mobile_nav()


def desktop_two():
    s = shell_desktop() + text(248, 118, '校园网络', 'title') + button(924, 84, 150, '添加线路')
    s += text(248, 155, '选择适合当前访问需求的线路。', 'muted')
    s += rect(248, 192, 826, 174, 'side', 10) + icon('shield', 272, 219, 26)
    s += text(312, 239, '校园入口', 'heading') + text(312, 267, '南京师范大学 / EasyConnect', 'muted')
    s += text(272, 329, '未连接', 'muted') + button(872, 298, 178, '连接', True)
    s += text(248, 421, '已保存的线路', 'heading') + icon('search', 1048, 403)
    for x, label in [(264, '名称'), (534, '协议'), (720, '范围')]:
        s += text(x, 464, label, 'muted small')
    s += line(248, 481, 1074, 481)
    for i, (name, campus, protocol) in enumerate(PROFILES):
        y = 516+i*64
        s += text(264, y, name) + text(534, y, protocol) + text(720, y, '校园资源', 'muted')
        s += text(984, y, '已选择' if i == 0 else '选择', 'small', 'end') + icon('more-horizontal', 1038, y-15, 20)
        s += line(248, y+24, 1074, y+24)
    return s


def mobile_two():
    s = mobile_header() + text(24, 112, '校园网络', 'title')
    s += rect(24, 145, 342, 264, 'side', 12) + icon('shield', 44, 168, 30)
    s += text(44, 230, '校园入口', 'heading') + text(44, 260, '南京师范大学', 'muted')
    s += text(44, 292, 'EasyConnect / 校园资源', 'muted small')
    s += text(44, 332, '未连接', 'small') + button(44, 348, 302, '连接', True, 44)
    s += text(24, 456, '已保存的线路', 'heading')
    for i in range(3):
        y = 478+i*76
        s += text(24, y+23, PROFILES[i][0]) + text(24, y+46, PROFILES[i][2], 'muted small')
        s += icon('check' if i == 0 else 'arrow-right', 340, y+20, 20) + line(24, y+65, 366, y+65)
    return s + mobile_nav()


def desktop_three():
    s = shell_desktop() + text(248, 110, '连接', 'heading') + button(928, 82, 144, '管理线路')
    s += icon('route', 634, 176, 48) + text(660, 263, '校园入口', 'title', 'middle')
    s += text(660, 296, '南京师范大学', 'muted', 'middle')
    s += rect(599, 320, 122, 30, 'fill', 15) + text(660, 340, '未连接', 'small', 'middle')
    s += text(420, 401, '线路', 'muted small') + rect(420, 419, 480, 54, 'side', 8)
    s += text(438, 452, '校园入口 / EasyConnect') + icon('chevron-down', 864, 435, 20)
    s += text(420, 512, '访问范围', 'muted') + text(900, 512, '校园资源', '', 'end')
    s += line(420, 532, 900, 532) + button(420, 561, 480, '连接', True, 48)
    s += icon('plus', 604, 643, 18) + text(633, 658, '添加线路')
    return s


def mobile_three():
    s = mobile_header() + icon('route', 171, 125, 48) + text(195, 225, '校园入口', 'title', 'middle')
    s += text(195, 258, '南京师范大学', 'muted', 'middle')
    s += rect(152, 282, 86, 28, 'fill', 14) + text(195, 301, '未连接', 'small', 'middle')
    s += text(24, 370, '线路', 'muted small') + rect(24, 389, 342, 56, 'side', 8)
    s += text(40, 422, '校园入口 / EasyConnect') + icon('chevron-down', 330, 407)
    s += text(24, 492, '访问范围', 'muted') + text(366, 492, '校园资源', '', 'end')
    s += line(24, 514, 366, 514) + button(24, 556, 342, '连接', True, 50)
    s += icon('plus', 145, 644, 20) + text(175, 660, '添加线路')
    return s + mobile_nav()


def write_svg(path, width, height, body, theme='system'):
    forced = ''
    if theme == 'light':
        forced = ':root{--bg:#ffffff;--side:#f6f6f6;--fill:#f0f0f0;--line:#e3e3e3;--ink:#191919;--muted:#707070;--action:#202020;--on:#ffffff}'
    elif theme == 'dark':
        forced = ':root{--bg:#202020;--side:#181818;--fill:#2c2c2c;--line:#3b3b3b;--ink:#ededed;--muted:#aaaaaa;--action:#ededed;--on:#202020}'
    license_text = (ASSETS / 'LICENSE').read_text(encoding='utf-8').replace('--', '—')
    content = f'''<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}" role="img" aria-label="校径 CampusWay">
<!-- Static design only; no VPN connection is attempted. Default theme follows the viewing system.
Lucide SVG icon sources: https://github.com/lucide-icons/lucide/tree/main/icons
{license_text}
-->
<title>校径 CampusWay</title><style>{CSS}{forced}</style>{definitions()}{body}</svg>'''
    path.write_text(content, encoding='utf-8')


def validate(path):
    root = ET.parse(path).getroot()
    assert root.tag == SVG+'svg'
    ids = {e.attrib['id'] for e in root.iter() if 'id' in e.attrib}
    for node in root.iter():
        assert node.tag not in {SVG+'image', SVG+'script', SVG+'foreignObject'}
        if node.tag == SVG+'use':
            assert node.attrib['href'].removeprefix('#') in ids
        if node.tag == SVG+'text':
            value = ''.join(node.itertext())
            assert 'NNU' not in value and 'nnuconnect' not in value.lower()
            assert not any(0x1F000 <= ord(c) <= 0x1FAFF or 0x2600 <= ord(c) <= 0x27BF for c in value)
    assert 'prefers-color-scheme:dark' in path.read_text(encoding='utf-8')


def main():
    fetch_icons()
    for i, (desktop, mobile) in enumerate([(desktop_one, mobile_one), (desktop_two, mobile_two), (desktop_three, mobile_three)], 1):
        d, m = desktop(), mobile()
        for theme in ['system', 'light', 'dark']:
            folder = ROOT / theme
            folder.mkdir(exist_ok=True)
            for kind, w, h, body in [('desktop', 1120, 760, d), ('mobile', 390, 844, m)]:
                target = folder / f'{i}-{kind}.svg'
                write_svg(target, w, h, body, theme)
                validate(target)
            # 配套总览保留手机原生纵横比；两端独立 SVG 可按 1:1 查看。
            board = rect(0, 0, 1574, 884, 'side', 0) + f'<g transform="translate(20 20)">{d}</g><g transform="translate(1164 20)">{m}</g>'
            target = folder / f'{i}-pair.svg'
            write_svg(target, 1574, 884, board, theme)
            validate(target)
    print('Validated 27 SVG files: 3 directions, desktop/mobile/pair, system/light/dark.')


if __name__ == '__main__':
    main()
