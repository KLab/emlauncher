"""PHP 版と Go 版の同じ画面を撮影して並べる (画面の配置が変わっていないかの目視確認用).

Playwright のコンテナで実行する. 両方とも同じ DB を見ている状態で使う.

    docker run --rm --network host -v $PWD:/e2e -w /e2e \
        mcr.microsoft.com/playwright/python:v1.48.0-jammy \
        python3 screenshots.py --mail owner@example.com --password owner-password --app 1 --package 8

結果は .work/screenshots/index.html (左が PHP 版, 右が Go 版).
"""
import argparse
import html
import os

from playwright.sync_api import sync_playwright

TARGETS = {"php": "http://localhost:18081", "go": "http://localhost:18082"}


def pages(app, pkg):
    return [
        ("login", "/login", False),
        ("top", "/", True),
        ("app", f"/app?id={app}&pf=all", True),
        ("app_comment", f"/app/comment?id={app}", True),
        ("app_preferences", f"/app/preferences?id={app}", True),
        ("app_upload", f"/app/upload?id={app}", True),
        ("app_new", "/app/new", True),
        ("package", f"/package?id={pkg}", True),
        ("package_edit", f"/package/edit?id={pkg}", True),
        ("package_delete_confirm", f"/package/delete_confirm?id={pkg}", True),
        ("myapps_installed", "/myapps/installed", True),
        ("myapps_own", "/myapps/own", True),
        ("guestpass_list", "/guestpass/list", True),
        ("doc_api", "/doc/api", True),
        ("doc_api_upload", "/doc/api_upload", True),
    ]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--mail", required=True)
    ap.add_argument("--password", required=True)
    ap.add_argument("--app", type=int, required=True)
    ap.add_argument("--package", type=int, required=True)
    ap.add_argument("--out", default=".work/screenshots")
    args = ap.parse_args()
    os.makedirs(args.out, exist_ok=True)

    shots = pages(args.app, args.package)
    with sync_playwright() as p:
        browser = p.chromium.launch()
        for name, base in TARGETS.items():
            for width, label in [(1280, "pc"), (390, "sp")]:
                ctx = browser.new_context(viewport={"width": width, "height": 900})
                page = ctx.new_page()
                logged_in = False
                for shot, path, login in shots:
                    if login and not logged_in:
                        page.goto(base + "/login")
                        page.fill("#email", args.mail)
                        page.fill("#password", args.password)
                        page.click("input[type=submit]")
                        page.wait_for_load_state("networkidle")
                        logged_in = True
                    page.goto(base + path)
                    page.wait_for_load_state("networkidle")
                    page.screenshot(path=f"{args.out}/{shot}-{label}-{name}.png", full_page=True)
                ctx.close()
        browser.close()

    rows = []
    for shot, path, _ in shots:
        for label in ["pc", "sp"]:
            rows.append(
                f'<h2>{html.escape(path)} ({label})</h2><div class="pair">'
                f'<figure><figcaption>PHP</figcaption><img src="{shot}-{label}-php.png"></figure>'
                f'<figure><figcaption>Go</figcaption><img src="{shot}-{label}-go.png"></figure></div>'
            )
    with open(f"{args.out}/index.html", "w", encoding="utf-8") as f:
        f.write(
            '<!doctype html><meta charset="utf-8"><title>EMLauncher PHP / Go</title>'
            "<style>body{font-family:sans-serif;margin:16px}.pair{display:flex;gap:16px;align-items:flex-start}"
            "figure{margin:0;flex:1;border:1px solid #ccc}figcaption{background:#eee;padding:4px}"
            "img{max-width:100%}</style>" + "".join(rows)
        )


if __name__ == "__main__":
    main()
