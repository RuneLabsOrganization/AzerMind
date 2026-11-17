/* AzerMind landing page: native JS, no third-party runtime, no account tracking. */
(() => {
  'use strict';
  const REPO = 'mateusmaaia/AzerMind-wow';
  const GITHUB = `https://github.com/${REPO}`;
  const API = `https://api.github.com/repos/${REPO}/releases?per_page=30`;
  const fallback = {
    client: `${GITHUB}/archive/refs/heads/main.zip`,
    addon: `${GITHUB}/tree/main/addon`,
    changes: `${GITHUB}/blob/main/CHANGELOG.md`,
  };
  const $ = id => document.getElementById(id);
  const client = $('client-download');
  const addon = $('addon-download');
  const notice = $('release-notice');
  const switchers = Array.from(document.querySelectorAll('[data-channel]'));
  let releases = [];
  let fetchStatus = 'loading';
  let activeChannel = 'stable';

  const datePT = value => {
    if (!value) return '';
    const d = new Date(value);
    return isNaN(d.getTime()) ? '' : new Intl.DateTimeFormat('pt-BR', {day:'numeric',month:'long',year:'numeric',timeZone:'UTC'}).format(d);
  };
  const bytes = n => {
    if (!Number.isFinite(n) || n <= 0) return '';
    return n < 1048576 ? `${Math.round(n/1024)} KB` : `${(n/1048576).toFixed(1).replace('.',',')} MB`;
  };
  const releaseAsset = (release, type) => {
    const files = Array.isArray(release.assets) ? release.assets : [];
    const eligible = files.filter(a => a.browser_download_url && /\.(zip|7z|exe|msi|dmg|tar\.gz)$/i.test(a.name || ''));
    const patterns = type === 'addon'
      ? [/addon|interface|forever|wow[_-]?ui|data[_-]?module/i]
      : [/client|bridge|installer|setup|windows|win64|desktop/i];
    // Never label an arbitrary release archive as a standalone client installer.
    return eligible.find(a => patterns.some(re => re.test(a.name || ''))) || null;
  };
  const setPackage = (element, labelID, subtitleID, asset, sourceURL, sourceLabel, sourceSubtitle) => {
    const name = $(labelID);
    const sub = $(subtitleID);
    element.href = asset ? asset.browser_download_url : sourceURL;
    name.textContent = asset ? asset.name : sourceLabel;
    sub.textContent = asset ? `Arquivo de release · ${bytes(asset.size)}` : sourceSubtitle;
    element.querySelector('.package-arrow').textContent = asset ? '↓' : '↗';
  };
  const releaseList = channel => releases.filter(r => !r.draft && Boolean(r.prerelease) === (channel === 'prerelease'));

  function renderDownloads() {
    const release = releaseList(activeChannel)[0];
    const beta = activeChannel === 'prerelease';
    const tag = $('release-tag');
    if (release) {
      const ver = release.tag_name || release.name || 'Versão publicada';
      const clientAsset = releaseAsset(release, 'client');
      const addonAsset = releaseAsset(release, 'addon');
      $('release-subtitle').textContent = `${ver} · ${datePT(release.published_at) || 'Versão publicada'}`;
      $('hero-version').textContent = `${ver} · ${beta ? 'Beta' : 'Estável'}`;
      tag.textContent = beta ? 'Beta' : 'Estável';
      tag.className = 'tag tag-live';
      setPackage(client, 'client-label', 'client-subtitle', clientAsset,
        release.html_url || `${GITHUB}/releases`, 'AzerMind — ver arquivos da release', 'Nenhum instalador de cliente identificado');
      setPackage(addon, 'addon-label', 'addon-subtitle', addonAsset,
        release.html_url || `${GITHUB}/releases`, 'Addon — ver arquivos da release', 'Nenhum pacote de addon identificado');
      notice.textContent = clientAsset && addonAsset ? `Pacotes oficiais disponíveis em ${ver}.` : `Release ${ver} publicada. Alguns pacotes não foram identificados automaticamente; confira os arquivos oficiais.`;
    } else {
      tag.textContent = 'Código-fonte';
      tag.className = 'tag tag-source';
      $('release-subtitle').textContent = 'Versão no repositório · v0.5.0';
      $('hero-version').textContent = 'Código-fonte disponível';
      setPackage(client, 'client-label','client-subtitle',null,fallback.client,'AzerMind — código-fonte','Projeto completo · ZIP · Requer Node.js');
      setPackage(addon, 'addon-label','addon-subtitle',null,fallback.addon,'Addon — WoW Forever','Módulos incluídos no repositório');
      if (fetchStatus === 'error') notice.textContent = 'Não foi possível consultar as releases agora. Confira os downloads diretamente no GitHub.';
      else if (beta) notice.textContent = 'Nenhuma release beta publicada neste canal. Você pode acessar o código-fonte ou conferir as versões no GitHub.';
      else notice.textContent = 'Ainda não há pacotes de release publicados. O código-fonte está disponível no GitHub.';
    }
  }

  function renderChangelog() {
    const latest = releases.filter(r => !r.draft)[0];
    if (!latest) return;
    const ver = latest.tag_name || latest.name || 'Versão publicada';
    $('changelog-version').textContent = ver;
    $('changelog-date').textContent = datePT(latest.published_at) || '';
    $('changelog-source').textContent = 'GitHub Releases';
    $('changelog-link').href = latest.html_url || fallback.changes;
    const notes = (latest.body || '').split('\n').map(s => s.trim())
      .filter(s => /^[-*]\s+/.test(s) && !/<!--|https?:\/\//.test(s))
      .map(s => s.replace(/^[-*]\s+/, '').replace(/[*_`#]/g,'').replace(/\[[^\]]*\]\([^)]*\)/g, ''))
      .filter(Boolean).slice(0,5);
    if (!notes.length) return;
    const ul = $('changelog-list');
    ul.replaceChildren(...notes.map(line => { const li = document.createElement('li'); li.textContent = line; return li; }));
  }

  async function fetchReleases() {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 7500);
    try {
      const response = await fetch(API,{headers:{Accept:'application/vnd.github+json'},signal:controller.signal});
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const data = await response.json();
      if (!Array.isArray(data)) throw new Error('Invalid release response');
      releases = data.filter(r => !r.draft).sort((a,b) => new Date(b.published_at || b.created_at || 0) - new Date(a.published_at || a.created_at || 0));
      fetchStatus = 'ok';
    } catch (_error) {
      fetchStatus = 'error';
    } finally {
      clearTimeout(timer);
      renderDownloads();
      renderChangelog();
    }
  }
  switchers.forEach(button => button.addEventListener('click',() => {
    activeChannel = button.dataset.channel;
    switchers.forEach(b => {const on = b === button; b.classList.toggle('active',on); b.setAttribute('aria-pressed',String(on));});
    renderDownloads();
  }));
  fetchReleases();

  // User interaction: mobile nav with matching aria-expanded.
  const toggle = $('mobile-toggle');
  const menu = $('site-menu');
  if (toggle && menu) {
    const close = () => {menu.classList.remove('open');toggle.setAttribute('aria-expanded','false');};
    toggle.addEventListener('click',() => {const isOpen=menu.classList.toggle('open');toggle.setAttribute('aria-expanded',String(isOpen));});
    menu.querySelectorAll('a').forEach(a => a.addEventListener('click',close));
    document.addEventListener('keydown',e => {if(e.key==='Escape') close();});
    document.addEventListener('click',e => {if(!menu.contains(e.target) && !toggle.contains(e.target)) close();});
  }

  // Readable full-size preview of the genuine AzerMind interface.
  const dialog = $('preview-dialog');
  if (dialog && typeof dialog.showModal === 'function') {
    $('open-preview').addEventListener('click', () => dialog.showModal());
    dialog.addEventListener('click', e => { if (e.target === dialog) dialog.close(); });
  }

  // Instantly copy the exact, visible setup instructions. Clipboard fallback for local files.
  const copy = $('copy-commands');
  copy.addEventListener('click',async () => {
    const value = $('install-commands').innerText.trim();
    try {
      if(navigator.clipboard && window.isSecureContext) await navigator.clipboard.writeText(value);
      else {
        const helper=document.createElement('textarea');helper.value=value;helper.style.position='fixed';helper.style.opacity='0';document.body.append(helper);helper.select();
        const ok=document.execCommand('copy');helper.remove();if(!ok) throw new Error('Clipboard denied');
      }
      copy.textContent='✓ Copiado!';
      setTimeout(()=>{copy.textContent='⧉ Copiar comandos';},1800);
    } catch (_) {
      copy.textContent='Selecione e copie abaixo';
    }
  });
})();