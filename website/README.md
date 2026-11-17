# AzerMind — Official landing page

## Languages

The landing page is fully localized for **English (EN)**, **Português (PT)** and **Español (ES)**. A language switcher appears in the main navigation on desktop and mobile. The browser language is detected for the first visit (English is the fallback), and manual selections are saved locally. You may also use `?lang=en`, `?lang=pt` or `?lang=es` to override the initial language. The translation dictionary is maintained in `i18n.js`, including headings, descriptions, accessibility labels and release notices. Dates and file sizes are formatted for the selected locale. Release notes published by third parties on GitHub remain in their original language.


A complete, responsive, Portuguese-language landing page designed around the **actual AzerMind addon UI** and its leather, parchment, burgundy, ivory, and aged-bronze visual identity.

## Run locally

No bundler or package install required:

```sh
cd website
python3 -m http.server 8080
```

Open <http://localhost:8080>. You can also use `npx serve .`.

## Deploy

Publish the contents of this directory to GitHub Pages, Cloudflare Pages, Netlify, or any static host. The website has no server, cookies, analytics, or signup requirement.

For GitHub Pages in the same repository, copy this directory to `/docs` **only if your existing `/docs` can be moved** (the addon already uses it for documentation). Prefer a dedicated `/website` folder and a Pages deployment workflow instead; do not overwrite the addon documentation. The `website-pages.yml` workflow provided in this package deploys `website/` with GitHub Actions once the files are added at that location.

## Releases and downloads

The homepage automatically reads `https://api.github.com/repos/mateusmaaia/AzerMind-wow/releases?per_page=30`. Stable and Beta are based on GitHub's `prerelease` field. It attempts to identify real release packages by asset name; it **does not fake an installer** when an asset is missing. Before the first GitHub Release, it offers the source code ZIP and the addon's source folder. Changelog uses the actual release notes when published, with a static, repository-based 0.5.0 fallback.

Suggested naming for release attachments:
- `AzerMind-client-windows-vX.Y.Z.zip`
- `AzerMind-addon-wow-forever-vX.Y.Z.zip`

A real standalone executable is **not** built by this site. At present the bridge runs on Node.js and needs an authenticated, separately installed agent CLI. See the project README before advertising an `.exe`.

## Visual identity

`assets/addon-screenshot.webp` is derived from the **real user-supplied interface screenshot**, with only the export overlay “Screen Captured” removed. `assets/emblem.webp` and the leather/parchment material originate from that screenshot. Scenic backgrounds are derived from previously approved AzerMind design concepts, rather than being copied from third-party game art. Rebuild instructions are in `scripts/build-assets.py` (which requires the source images in the original creation workspace; the built website does not).

The PixelLab integration was unavailable in this chat. The landing page uses existing approved project visuals and local image preparation rather than claiming new PixelLab-generated assets.

## Accessibility and compatibility

Semantic sections, descriptive alt text, keyboard-operable navigation and screenshot dialog, disclosure FAQ, announcement states, contrast-aware components, and reduced-motion support. Desktop, tablet, and mobile responsive layouts. No tracking scripts or external JS libraries. Google Fonts are optional; system fallbacks are supplied.

## Repository and rights

AzerMind repository: <https://github.com/mateusmaaia/AzerMind-wow>

AzerMind is an independent community addon. It is not affiliated with Blizzard Entertainment. This website credits the upstream project and does not bundle third-party game assets.
### Hero banner illustration

The hero background `assets/hero-arcane-library.webp` is an optimized original arcane-library illustration with subtle code motifs. The readable text and UI remain native HTML, above layered CSS gradients rather than baked into the image. The focal point is adjusted for narrow screens. The original screenshot preview is unchanged.
