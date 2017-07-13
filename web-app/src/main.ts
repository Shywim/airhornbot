(function () {
  const AIRHORN_URL = 'https://airhorn.shywim.fr'
  const MESSAGE_TWITTER = 'This Discord bot makes airhorn sounds ayy lmao'
  const HASHTAGS_TWITTER = 'ReadyForHorning'
  const FB_SHARE_URL = `http://www.facebook.com/sharer.php?u=${AIRHORN_URL}`
  const TWITTER_SHARE_URL = `https://twitter.com/share?text=${MESSAGE_TWITTER}&url=${AIRHORN_URL}&hashtags=${HASHTAGS_TWITTER}`

  const video: HTMLVideoElement = <HTMLVideoElement>document.getElementById('video')
  const audio: HTMLAudioElement = <HTMLAudioElement>document.getElementById('audio')
  const count: HTMLElement = document.getElementById('count')
  const playsCount: HTMLElement = document.getElementById('plays-count')
  const usersCount: HTMLElement = document.getElementById('users-count')
  const guildsCount: HTMLElement = document.getElementById('guilds-count')
  const channelsCount: HTMLElement = document.getElementById('channels-count')
  const statsToggler: HTMLElement = document.getElementById('stats-toggler')
  const statsPanel: HTMLElement = document.getElementById('stats-panel')
  const statsPanelClose: HTMLElement = document.getElementById('stats-panel-close')
  const statsPanelOpen: HTMLElement = document.getElementById('stats-panel-open')
  const shareFb: HTMLElement = document.getElementById('share-fb')
  const shareTwitter: HTMLElement = document.getElementById('share-twitter')

  let statsTogglerState = false
  const stats = {
    count: 0,
    users: 0,
    guilds: 0,
    channels: 0,
  }

  // video handler
  video.onclick = function () {
    video.play()
    audio.play()
  }

  // bot stats 
  const removeCountBig = function(){
    count.classList.remove('count-big')
  }
  if (EventSource != null) {
    const es: EventSource = new EventSource('/events')
    es.onmessage = function (msg) {
      const data = JSON.parse(msg.data)

      if (stats.count !== data.total){
        count.classList.add('count-big')
        setTimeout(removeCountBig, 400)
      }

      stats.count = data.total || 0
      stats.users = data.unique_users || 0
      stats.guilds = data.unique_guilds || 0
      stats.channels = data.unique_channels || 0

      count.innerText = stats.count.toString()
      playsCount.innerText = stats.count.toString()
      usersCount.innerText = stats.users.toString()
      guildsCount.innerText = stats.guilds.toString()
      channelsCount.innerText = stats.channels.toString()
    }
  }

  // panel handler
  statsToggler.onclick = function () {
    let remove
    let add
    if (statsTogglerState === false) {
      remove = '-reverse'
      add = ''
    } else {
      remove = ''
      add = '-reverse'
    }
    statsTogglerState = !statsTogglerState
    statsPanelClose.classList.remove(`three${remove}`)
    statsPanelClose.classList.add(`three${add}`)
    statsPanelOpen.classList.remove(`two${remove}`)
    statsPanelOpen.classList.add(`two${add}`)
    statsPanel.classList.remove(`one${remove}`)
    statsPanel.classList.add(`one${add}`)
  }

  // social share
  const openShare = function(url){
    window.open(url, '', 'height=500, width=500')
  }
  shareFb.onclick = () => openShare(FB_SHARE_URL)
  shareTwitter.onclick = () => openShare(TWITTER_SHARE_URL)

  // play video if just logged in
  if (URLSearchParams != null) {
    const params = new URLSearchParams(location.search.slice(1))
    const keyToSuccess = params.get('key_to_success')
    if (keyToSuccess === '1') {
      setTimeout(function () {
        video.play()
        audio.play()
      }, 1000)
    }
  }
})()