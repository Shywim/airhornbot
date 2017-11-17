(() => {
  const AIRHORN_URL = `${window.location.protocol}//${window.location.host}`;
  const MESSAGE_TWITTER = 'This Discord bot makes airhorn sounds ayy lmao';
  const FB_SHARE_URL = `https://www.facebook.com/sharer.php?u=${AIRHORN_URL}`;
  const TWITTER_SHARE_URL = `https://twitter.com/share?text=${MESSAGE_TWITTER}&url=${AIRHORN_URL}`;
  const count = document.getElementById('count');
  const playsCount = document.getElementById('plays-count');
  const usersCount = document.getElementById('users-count');
  const guildsCount = document.getElementById('guilds-count');
  const channelsCount = document.getElementById('channels-count');
  const statsToggler = document.getElementById('stats-toggler');
  const statsBtn = document.getElementById('stats-btn');
  const statsPanel = document.getElementById('stats-panel');
  const statsPanelClose = document.getElementById('stats-panel-close');
  const statsPanelOpen = document.getElementById('stats-panel-open');
  const shareFb = document.getElementById('share-fb');
  const shareTwitter = document.getElementById('share-twitter');

  statsPanel.classList.remove('is-hidden');
  statsBtn.classList.remove('is-hidden');
  count.innerText = '0';

  let statsTogglerState = false;
  const stats = {
    channels: 0,
    count: 0,
    guilds: 0,
    users: 0
  };

  if (window.EventSource != null) {
    const es = new EventSource(`${AIRHORN_URL}/events`);
    es.onmessage = (msg) => {
      const data = JSON.parse(msg.data);

      if (stats.count !== data.total) {
        count.classList.add('count-big');
        setTimeout(() => count.classList.remove('count-big'), 400);
      }

      stats.count = data.total || 0;
      stats.users = data.unique_users || 0;
      stats.guilds = data.unique_guilds || 0;
      stats.channels = data.unique_channels || 0;

      count.textContent = stats.count.toString();
      playsCount.textContent = stats.count.toString();
      usersCount.textContent = stats.users.toString();
      guildsCount.textContent = stats.guilds.toString();
      channelsCount.textContent = stats.channels.toString();
    };
  }

  // stats panel handler
  statsToggler.onclick = () => {
    let remove;
    let add;
    if (statsTogglerState === false) {
      remove = '-reverse';
      add = '';
    } else {
      remove = '';
      add = '-reverse';
    }
    statsTogglerState = !statsTogglerState;
    statsPanelClose.classList.remove(`three${remove}`);
    statsPanelClose.classList.add(`three${add}`);
    statsPanelOpen.classList.remove(`two${remove}`);
    statsPanelOpen.classList.add(`two${add}`);
    statsPanel.classList.remove(`one${remove}`);
    statsPanel.classList.add(`one${add}`);
  };

  // social share
  const openShare = (url) => {
    window.open(url, '', 'height=500, width=500');
    return false;
  };
  shareFb.onclick = () => openShare(FB_SHARE_URL);
  shareTwitter.onclick = () => openShare(TWITTER_SHARE_URL);
})();
