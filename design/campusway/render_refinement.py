from pathlib import Path
from render_designs import rect, line, icon, text, button, brand, mobile_nav, write_svg, validate

# 用户选定方案 3 的首次登录与方案 1 的多线路管理；这是一套流程的不同状态。
# 参考附件中的网格和底部偏好面板，只借鉴布局，不继承彩色、emoji、节点或测速数据。
# 静态 SVG 尚不执行登录、切换或持久化；实现阶段以 architecture.go 中的契约接入。
ROOT = Path(__file__).resolve().parent / 'refined'
PROFILES = [('校园入口', '南京师范大学', 'EasyConnect'),
            ('实验室', '研究网络', 'EasyConnect'),
            ('联合校园', '协作网络', 'aTrust'),
            ('备用入口', '南京师范大学', 'EasyConnect')]


def titlebar(w):
    return brand(22, 34) + line(0, 52, w, 52) + f'<path d="M{w-76} 26h12M{w-34} 20l12 12m0-12l-12 12" class="icon"/>'


def field(x, y, w, label, value, select=False):
    s = text(x, y, label, 'muted small') + rect(x, y+10, w, 42, 'side')
    s += rect(x, y+10, w, 42, 'border') + text(x+12, y+37, value, 'muted')
    return s + (icon('chevron-down', x+w-30, y+21, 18) if select else '')


def login(mobile=False):
    w, h = (390, 844) if mobile else (460, 600)
    s = rect(0, 0, w, h, 'bg', 0)
    s += brand(24, 42) if mobile else titlebar(w)
    x, form_w = (24, 342) if mobile else (48, 364)
    top = 124 if mobile else 111
    s += text(x, top, '连接校园网络', 'title') + text(x, top+30, '添加第一条线路，使用校园账号登录。', 'muted small')
    y = top+74
    s += field(x, y, form_w, '服务器', 'vpn.njnu.edu.cn')
    s += field(x, y+74, form_w, '协议', 'EasyConnect', True)
    s += field(x, y+148, form_w, '学工号', '输入校园账号')
    s += field(x, y+222, form_w, '密码', '输入密码')
    s += rect(x, y+299, 16, 16, 'border', 4) + text(x+26, y+312, '记住密码', 'small')
    s += button(x, y+337, form_w, '登录并连接', True, 46)
    if mobile:
        s += text(x, 747, '连接时将请求系统 VPN 授权。', 'muted small')
    return s


def desktop_shell():
    s = rect(0, 0, 900, 620, 'bg', 0) + rect(0, 0, 156, 620, 'side', 0)
    s += titlebar(900) + line(156, 52, 156, 620)
    s += rect(10, 75, 136, 40) + icon('route', 22, 86) + text(54, 102, '线路')
    s += icon('activity', 22, 136) + text(54, 152, '活动')
    s += icon('settings-2', 22, 563) + text(54, 579, '设置')
    return s


def view_toggle(x, y, grid=False):
    s = rect(x, y, 124, 34, 'side', 6)
    s += rect(x+(64 if grid else 3), y+3, 57, 28, 'fill', 4)
    s += text(x+32, y+22, '表格', '' if not grid else 'muted small', 'middle')
    s += text(x+94, y+22, '方块', '' if grid else 'muted small', 'middle')
    return s


def desktop(grid=False):
    s = desktop_shell() + text(180, 100, '我的线路', 'heading') + button(762, 73, 114, '添加线路', h=36)
    s += rect(180, 125, 260, 36, 'side') + icon('search', 192, 135, 16) + text(218, 148, '搜索线路', 'muted small')
    s += view_toggle(706, 126, grid) + icon('settings-2', 850, 134, 20)
    if grid:
        for i, (name, campus, protocol) in enumerate(PROFILES):
            x, y = 180+(i % 3)*236, 185+(i//3)*133
            s += rect(x, y, 224, 116, 'fill' if i == 0 else 'side', 9)
            if i == 0:
                s += f'<rect x="{x}" y="{y}" width="224" height="116" rx="9" fill="none" stroke="var(--ink)"/>'
            s += text(x+16, y+28, name) + icon('check' if i == 0 else 'more-horizontal', x+188, y+13, 19)
            s += text(x+16, y+53, campus, 'muted small') + text(x+16, y+84, protocol, 'small')
            s += text(x+208, y+103, '未检测', 'muted small', 'end')
    else:
        for x, value in [(194, '线路'), (410, '协议'), (565, '延迟'), (700, '状态')]:
            s += text(x, 200, value, 'muted small')
        s += line(180, 215, 876, 215)
        for i, (name, campus, protocol) in enumerate(PROFILES):
            y = 225+i*61
            if i == 0:
                s += rect(180, y-3, 696, 56)
            s += text(194, y+18, name) + text(194, y+40, campus, 'muted small')
            s += text(410, y+26, protocol, 'small') + text(565, y+26, '未检测', 'muted small')
            s += text(700, y+26, '已选择' if i == 0 else '未连接', 'small') + icon('more-horizontal', 842, y+10, 20)
            s += line(180, y+56, 876, y+56)
    s += line(180, 492, 876, 492) + text(180, 521, '校园入口', 'heading')
    s += text(180, 548, 'vpn.njnu.edu.cn / 校园资源', 'muted small')
    s += button(690, 511, 186, '连接此线路', True, 44)
    s += text(180, 589, '4 条线路', 'muted small') + text(876, 589, '未连接', 'muted small', 'end')
    return s


def mobile(grid=False):
    s = rect(0, 0, 390, 844, 'bg', 0) + brand(24, 42) + icon('plus', 342, 23, 24)
    s += text(24, 98, '我的线路', 'title') + icon('settings-2', 342, 80, 24)
    s += rect(24, 124, 204, 40, 'side') + icon('search', 36, 136, 18) + text(62, 150, '搜索线路', 'muted small')
    s += view_toggle(242, 127, grid)
    if grid:
        for i, (name, campus, protocol) in enumerate(PROFILES):
            x, y = 24+(i % 2)*177, 189+(i//2)*137
            s += rect(x, y, 165, 121, 'fill' if i == 0 else 'side', 9)
            if i == 0:
                s += f'<rect x="{x}" y="{y}" width="165" height="121" rx="9" fill="none" stroke="var(--ink)"/>'
            s += text(x+13, y+29, name) + icon('check' if i == 0 else 'more-horizontal', x+131, y+13, 19)
            s += text(x+13, y+53, campus, 'muted small') + text(x+13, y+82, protocol, 'small')
            s += text(x+13, y+105, '未检测', 'muted small')
    else:
        s += text(24, 202, '线路 / 协议', 'muted small') + text(330, 202, '延迟', 'muted small', 'end')
        s += line(24, 216, 366, 216)
        for i, (name, campus, protocol) in enumerate(PROFILES):
            y = 224+i*65
            if i == 0:
                s += rect(16, y, 358, 60)
            s += text(28, y+24, name) + text(28, y+47, protocol, 'muted small')
            s += text(327, y+34, '未检测', 'muted small', 'end')
            s += icon('check' if i == 0 else 'more-horizontal', 342, y+22, 18)
            s += line(24, y+63, 366, y+63)
    s += line(24, 554, 366, 554) + text(24, 588, '校园入口', 'heading')
    s += text(366, 588, '未连接', 'muted small', 'end') + text(24, 620, 'vpn.njnu.edu.cn', 'mono muted')
    s += button(24, 665, 342, '连接此线路', True, 50)
    return s + mobile_nav()


def segments(x, y, w, choices, selected):
    gap = 8
    cw = (w-gap*(len(choices)-1))/len(choices)
    return ''.join(button(x+i*(cw+gap), y, cw, choice, i == selected, 44) for i, choice in enumerate(choices))


def preferences(mobile_view=False):
    s = mobile(True) if mobile_view else desktop(True)
    w, h = (390, 844) if mobile_view else (900, 620)
    s += f'<rect width="{w}" height="{h}" fill="#000000" opacity=".35"/>'
    x, y, pw = (0, 286, 390) if mobile_view else (500, 71, 376)
    s += rect(x, y, pw, 558 if mobile_view else 523, 'bg', 18)
    if mobile_view:
        s += rect(171, y+12, 48, 4, 'fill', 2)
    s += text(x+24, y+53, '显示设置', 'heading')
    s += f'<path d="M{x+pw-44} {y+36}l14 14m0-14l-14 14" class="icon"/>'
    for dy, label, choices, selected in [(91, '视图', ['表格', '方块'], 1),
                                        (191, '排序', ['默认', '名称', '延迟'], 0),
                                        (291, '密度', ['宽松', '标准', '紧凑'], 2),
                                        (391, '主题', ['跟随系统', '浅色', '深色'], 0)]:
        s += text(x+24, y+dy, label, 'muted small')
        s += segments(x+24, y+dy+15, pw-48, choices, selected)
    return s


def main():
    # 用户已完成方向选择；交付一个合并方案的流程稿，不再生成三种竞选风格。
    frames = [('login-desktop', 460, 600, login()), ('login-mobile', 390, 844, login(True)),
              ('table-desktop', 900, 620, desktop()), ('table-mobile', 390, 844, mobile()),
              ('grid-desktop', 900, 620, desktop(True)), ('grid-mobile', 390, 844, mobile(True)),
              ('preferences-desktop', 900, 620, preferences()), ('preferences-mobile', 390, 844, preferences(True))]
    count = 0
    for theme in ['system', 'light', 'dark']:
        folder = ROOT / theme
        folder.mkdir(parents=True, exist_ok=True)
        for name, w, h, body in frames:
            path = folder / (name+'.svg')
            write_svg(path, w, h, body, theme)
            validate(path)
            count += 1
        for name, d, m in [('login', frames[0], frames[1]), ('table', frames[2], frames[3]),
                           ('grid', frames[4], frames[5]), ('preferences', frames[6], frames[7])]:
            w, h = d[1]+m[1]+60, 884
            board = rect(0, 0, w, h, 'side', 0)
            board += f'<g transform="translate(20 20)">{d[3]}</g><g transform="translate({d[1]+40} 20)">{m[3]}</g>'
            path = folder / (name+'-pair.svg')
            write_svg(path, w, h, board, theme)
            validate(path)
            count += 1
    print(f'Validated {count} refined SVGs; desktop login 460x600, manager 900x620, mobile 390x844.')


if __name__ == '__main__':
    main()
