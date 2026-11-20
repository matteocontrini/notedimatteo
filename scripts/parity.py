"""Compare SvelteKit (A) and Go (B) responses for a list of paths.

Usage: python3 -I parity.py <base A> <base B> <paths file> <report file>
"""

import concurrent.futures
import html.parser
import json
import re
import sys
import urllib.error
import urllib.request
import urllib.parse
import xml.etree.ElementTree as ET

base_a, base_b, paths_file, report_file = sys.argv[1:5]
paths = [line.rstrip("\n") for line in open(paths_file, encoding="utf8") if line.strip()]


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


opener = urllib.request.build_opener(NoRedirect)


def fetch(base, path):
    try:
        resp = opener.open(base + path, timeout=60)
        return resp.status, resp.headers, resp.read()
    except urllib.error.HTTPError as e:
        return e.code, e.headers, e.read()


class Extract(html.parser.HTMLParser):
    """Collects visible text, link hrefs, and head metadata."""

    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.text, self.hrefs, self.meta, self.jsonld = [], [], {}, []
        self.skip = 0
        self.in_title = False
        self.title = ""
        self.in_jsonld = False

    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        if tag == "script":
            if a.get("type") == "application/ld+json":
                self.in_jsonld = True
                self.jsonld.append("")
            else:
                self.skip += 1
        elif tag in ("style", "template"):
            self.skip += 1
        elif tag == "title":
            self.in_title = True
        elif tag == "meta" and ("name" in a or "property" in a):
            self.meta[a.get("name") or a.get("property")] = a.get("content")
        elif tag == "link" and a.get("rel") in ("canonical", "alternate", "icon"):
            self.meta["link:" + a["rel"]] = a.get("href") if a["rel"] != "icon" else "(icon)"
        elif tag == "a":
            self.hrefs.append(a.get("href"))
        elif tag in ("input",) and a.get("name") == "q":
            self.meta["search-value"] = a.get("value")

    def handle_endtag(self, tag):
        if tag == "script":
            if self.in_jsonld:
                self.in_jsonld = False
            elif self.skip:
                self.skip -= 1
        elif tag in ("style", "template") and self.skip:
            self.skip -= 1
        elif tag == "title":
            self.in_title = False

    def handle_data(self, data):
        if self.in_jsonld:
            self.jsonld[-1] += data
        elif self.in_title:
            self.title += data
        elif not self.skip:
            self.text.append(data)


def absolute(page_url, href):
    """Resolves a link like a browser, ignoring how it's encoded."""
    if href is None:
        return None
    url = urllib.parse.urljoin(page_url, href)
    parts = urllib.parse.urlsplit(url)
    if parts.netloc.startswith("localhost:"):
        url = urllib.parse.urlunsplit(("", "", parts.path, parts.query, parts.fragment))
    return urllib.parse.unquote(url)


def sort_tag_ties(text):
    """The sidebar orders tags with equal counts by name in Go (a known difference)."""
    m = re.search(r" Tag A-Z (.*?) Segui ", text)
    if not m:
        return text
    items = re.findall(r"#\S+ \(\d+\)", m.group(1))
    return text[: m.start(1)] + " ".join(sorted(items)) + text[m.end(1):]


def summarize_html(body, page_url):
    p = Extract()
    p.feed(body.decode("utf8", "replace"))
    text = re.sub(r"\s+", " ", " ".join(p.text)).strip()
    meta = {k: (absolute(page_url, v) if k in ("link:canonical", "link:alternate") else v) for k, v in p.meta.items()}
    return {
        "title": p.title.strip(),
        "meta": meta,
        "jsonld": [json.loads(j) for j in p.jsonld],
        "hrefs": sorted(str(absolute(page_url, h)) for h in p.hrefs),
        "text": sort_tag_ties(text),
    }


def first_xml_difference(a, b, where="rss"):
    if a[:3] != b[:3]:
        return f"{where}: {a[:3]!r:.300} != {b[:3]!r:.300}"
    if len(a[3]) != len(b[3]):
        return f"{where}: {len(a[3])} children != {len(b[3])}"
    for i, (x, y) in enumerate(zip(a[3], b[3])):
        d = first_xml_difference(x, y, f"{where}/{x[0]}[{i}]")
        if d:
            return d
    return None


def summarize_xml(body):
    root = ET.fromstring(body)

    def walk(el):
        text = (el.text or "").strip()
        if el.tag.endswith("}encoded"):
            # HTML in CDATA: the Go renderer escapes some characters differently
            import html as h
            text = h.unescape(text)
        return [el.tag, sorted(el.attrib.items()), text, [walk(c) for c in el]]

    return walk(root)


def compare(path):
    sa, ha, ba = fetch(base_a, path)
    sb, hb, bb = fetch(base_b, path)
    issues = []
    if sa != sb:
        issues.append(f"status {sa} != {sb}")
    la = absolute(base_a + path, ha.get("Location")) or ""
    lb = absolute(base_b + path, hb.get("Location")) or ""
    if la != lb:
        issues.append(f"location {la!r} != {lb!r}")
    ctype = (ha.get("Content-Type") or "").split(";")[0]
    if sa == sb and (sa < 300 or sa >= 400) and ctype != (hb.get("Content-Type") or "").split(";")[0]:
        issues.append(f"content type {ha.get('Content-Type')} != {hb.get('Content-Type')}")
    if sa == sb and sa not in (301, 302, 307, 308):
        if ctype == "text/html":
            a, b = summarize_html(ba, base_a + path), summarize_html(bb, base_b + path)
            if a["meta"] != b["meta"]:
                keys = sorted(set(a["meta"]) | set(b["meta"]))
                diff = {k: (a["meta"].get(k), b["meta"].get(k)) for k in keys if a["meta"].get(k) != b["meta"].get(k)}
                issues.append(f"meta: {json.dumps(diff, ensure_ascii=False)[:400]}")
            for key in ("title", "jsonld", "hrefs"):
                if a[key] != b[key]:
                    issues.append(f"{key}: {json.dumps(a[key], ensure_ascii=False)[:300]} != {json.dumps(b[key], ensure_ascii=False)[:300]}")
            if a["text"] != b["text"]:
                if a["text"].replace(" ", "") == b["text"].replace(" ", ""):
                    issues.append("text differs only in whitespace")
                else:
                    i = next((k for k, (x, y) in enumerate(zip(a["text"], b["text"])) if x != y), min(len(a["text"]), len(b["text"])))
                    issues.append(f"text at {i}: {a['text'][max(0, i - 60):i + 60]!r} != {b['text'][max(0, i - 60):i + 60]!r}")
        elif ctype == "application/xml":
            d = first_xml_difference(summarize_xml(ba), summarize_xml(bb))
            if d:
                issues.append("feed: " + d)
            for header in ("Cache-Control",):
                if ha.get(header) != hb.get(header):
                    issues.append(f"{header} {ha.get(header)} != {hb.get(header)}")
    return path, sa, sb, issues


with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
    results = list(pool.map(compare, paths))

failed = [r for r in results if r[3]]
with open(report_file, "w", encoding="utf8") as out:
    for path, sa, sb, issues in failed:
        out.write(f"{path} [{sa}/{sb}]\n")
        for issue in issues:
            out.write(f"    {issue}\n")

statuses = {}
for _, sa, _, _ in results:
    statuses[sa] = statuses.get(sa, 0) + 1
print(f"{len(results)} paths, {len(failed)} with differences; SvelteKit statuses: {dict(sorted(statuses.items()))}")
