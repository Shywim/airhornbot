(() => {
  const AIRHORN_URL = 'https://airhorn.shywim.fr'
  const MESSAGE_TWITTER = 'This Discord bot makes airhorn sounds ayy lmao'
  const HASHTAGS_TWITTER = 'ReadyForHorning'
  const FB_SHARE_URL = `http://www.facebook.com/sharer.php?u=${AIRHORN_URL}`
  const TWITTER_SHARE_URL =
    `https://twitter.com/share?text=${MESSAGE_TWITTER}&url=${AIRHORN_URL}&hashtags=${HASHTAGS_TWITTER}`
  const MODAL_ID = 'modal'

  const video: HTMLVideoElement = document.getElementById('video') as HTMLVideoElement
  const audio: HTMLAudioElement = document.getElementById('audio') as HTMLAudioElement
  const count: HTMLElement = document.getElementById('count')
  const playsCount: HTMLElement = document.getElementById('plays-count')
  const usersCount: HTMLElement = document.getElementById('users-count')
  const guildsCount: HTMLElement = document.getElementById('guilds-count')
  const channelsCount: HTMLElement = document.getElementById('channels-count')
  const statsToggler: HTMLElement = document.getElementById('stats-toggler')
  const statsBtn: HTMLElement = document.getElementById('stats-btn')
  const statsPanel: HTMLElement = document.getElementById('stats-panel')
  const statsPanelClose: HTMLElement = document.getElementById('stats-panel-close')
  const statsPanelOpen: HTMLElement = document.getElementById('stats-panel-open')
  const shareFb: HTMLElement = document.getElementById('share-fb')
  const shareTwitter: HTMLElement = document.getElementById('share-twitter')

  // javascript is enabled
  statsPanel.classList.remove('is-hidden')
  statsBtn.classList.remove('is-hidden')
  count.innerText = '0'

  let statsTogglerState = false
  const stats = {
    channels: 0,
    count: 0,
    guilds: 0,
    users: 0,
  }

  // video handler
  video.addEventListener('ended', () => {
    video.currentTime = 0
  }, false)
  video.onclick = () => {
    video.play()
    audio.play()
  }

  // bot stats
  const removeCountBig = () => {
    count.classList.remove('count-big')
  }
  if (EventSource != null) {
    const es: EventSource = new EventSource('/events')
    es.onmessage = (msg) => {
      const data = JSON.parse(msg.data)

      if (stats.count !== data.total) {
        count.classList.add('count-big')
        setTimeout(removeCountBig, 400)
      }

      stats.count = data.total || 0
      stats.users = data.unique_users || 0
      stats.guilds = data.unique_guilds || 0
      stats.channels = data.unique_channels || 0

      count.textContent = stats.count.toString()
      playsCount.textContent = stats.count.toString()
      usersCount.textContent = stats.users.toString()
      guildsCount.textContent = stats.guilds.toString()
      channelsCount.textContent = stats.channels.toString()
    }
  }

  // panel handler
  statsToggler.onclick = () => {
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
  const openShare = (url) => {
    window.open(url, '', 'height=500, width=500')
  }
  shareFb.onclick = () => openShare(FB_SHARE_URL)
  shareTwitter.onclick = () => openShare(TWITTER_SHARE_URL)

  // play video if just logged in
  if (URLSearchParams != null) {
    const params = new URLSearchParams(location.search.slice(1))
    const keyToSuccess = params.get('key_to_success')
    if (keyToSuccess === '1') {
      setTimeout(() => {
        video.play()
        audio.play()
      }, 1000)
    }
  }

  // --- MANAGE ---
  (() => {
    const manageState = {
      extendedRow: -1,
      guilds: {
        airhorn: [],
        boring: [],
      },
    }
    const mainSection: Element = document.getElementsByClassName('main-section')[0]
    const manageNotification: HTMLElement = document.getElementById('manage-notification')
    const manageSection: HTMLElement = document.getElementById('manage-section')
    const manageButton: HTMLElement = document.getElementById('manage-btn')
    const homeButton: HTMLElement = document.getElementById('back-home')
    const airhornTable: HTMLElement = document.getElementById('airhorn-guilds')
    const boringTable: HTMLElement = document.getElementById('boring-guilds')
    const editSoundModal: HTMLElement = document.getElementById('edit-sound-modal')
    const editSoundModalForm: HTMLFormElement = document.getElementById('edit-sound-form') as HTMLFormElement
    const editSoundModalFile: HTMLElement = document.getElementById('edit-sound-file')
    const editSoundModalCancel: HTMLElement = editSoundModal.getElementsByTagName('button')[0]

    /**
     * Switch between home and manage section
     * @param show
     */
    const showHideManage = (show: boolean) => {
      if (show === true) {
        if (window.location.hash !== '#manage') {
          history.pushState(null, document.title, '#manage')
        }
        mainSection.classList.add('is-hidden')
        manageSection.classList.remove('is-hidden')
      } else {
        if (window.location.hash !== '') {
          history.pushState(null, document.title, window.location.pathname + window.location.search)
        }
        mainSection.classList.remove('is-hidden')
        manageSection.classList.add('is-hidden')
      }
    }

    /**
     * Display the form for editing a sound details or creating a new sound
     * @param guildId
     * @param sound
     */
    const editSound = (guildId: string, sound?: Sound) => {
      const nameKey = 'name'
      const commandKey = 'command'
      const weightKey = 'weight'
      const fileKey = 'file'

      if (sound != null) {
        editSoundModalForm.action = `/manage/${guildId}/${sound.id}`
        editSoundModalForm.elements[nameKey].value = sound.name
        editSoundModalForm.elements[commandKey].value = sound.command
        editSoundModalForm.elements[weightKey].value = sound.weight
        editSoundModalForm.elements[fileKey].required = false
        editSoundModalFile.classList.add('is-hidden')
        editSoundModalForm.onsubmit = () => postForm(guildId, sound.id)
      } else {
        editSoundModalForm.action = `/manage/${guildId}/new`
        editSoundModalForm.elements[nameKey].value = ''
        editSoundModalForm.elements[commandKey].value = ''
        editSoundModalForm.elements[weightKey].value = ''
        editSoundModalForm.elements[fileKey].required = true
        editSoundModalFile.classList.remove('is-hidden')
        editSoundModalForm.onsubmit = () => postForm(guildId)
      }
      editSoundModal.classList.add('is-active')
    }

    /**
     * Send form data to the server
     * @param guildId guild's id
     * @param soundId optional soundId, if null it creates a new sound
     */
    const postForm = (guildId: string, soundId?: string) => {
      const formData: FormData = new FormData(editSoundModalForm)
      let url
      let method
      // if the sound is null it is a new sound
      if (soundId == null) {
        url = `/manage/${guildId}/new`
        method = 'POST'
      } else {
        url = `/manage/${guildId}/${soundId}`
        method = 'PUT'
      }

      fetch(url, {
        body: formData,
        credentials: 'same-origin',
        method,
      })
      .then((response) => {
        if (response.ok === true) {
          editSoundModal.classList.remove('is-active')
          return fetchGuilds()
            .then(updateLayout)
        }
        // TODO: handle errors
      })

      return false
    }

    /**
     * Send a request to delete the given sound
     * @param guildId guild's id
     * @param soundId sound to delete
     */
    const deleteSound = (guildId: string, soundId: string) => {
      return fetch(`/manage/${guildId}/${soundId}`, {
        credentials: 'same-origin',
        method: 'DELETE',
      })
      .then((response) => {
        if (response.ok === true) {
          return fetchGuilds()
            .then(updateLayout)
        }
        // TODO: handle errors
      })
    }

    const fetchGuilds = () => {
      return fetch('/me/guilds', {
        credentials: 'same-origin',
      })
      .then((response) => {
        if (response.ok) {
          return response.json()
        }

        if (response.status === 401) {
          manageNotification.classList.remove('is-hidden')
          return Promise.reject('unauthorized')
        }
      })
      .then((json: GuildsResponse) => manageState.guilds = json)
      // TODO: handle errors
    }

    // javascript and this module are enabled
    const guildsRequest = fetchGuilds()
    editSoundModalCancel.onclick = () => editSoundModal.classList.remove('is-active')
    manageButton.classList.remove('disabled')

    manageButton.onclick = () => {
      // TODO: redirect to login if not logged in
      showHideManage(true)
    }

    homeButton.onclick = () => {
      showHideManage(false)
    }

    // show manage section if #manage
    if (window.location.hash === '#manage') {
      // TODO: don't do it if not logged in
      setTimeout(() => showHideManage(true), 1000)
    }

    const airhornTb: HTMLTableSectionElement = airhornTable.getElementsByTagName('tbody')[0]
    const boringTb: HTMLTableSectionElement = boringTable.getElementsByTagName('tbody')[0]

    const rowClick = (guild: Guild, i) => {
      if (manageState.extendedRow > -1) {
        // delete row if a details row exists
        airhornTb.deleteRow(i + 1)
      }
      if (manageState.extendedRow === i) {
        // juste close if the same row was clicked
        manageState.extendedRow = -1
        return
      }
      manageState.extendedRow = i

      const detailsRow: HTMLTableRowElement = airhornTb.insertRow(i + 1)
      const cell: HTMLTableCellElement = detailsRow.insertCell(0)
      cell.colSpan = 4

      if (guild.sounds.length === 0) {
        // display a nice message
        const messageDiv: HTMLElement = document.createElement('div')
        const message: HTMLElement = document.createElement('p')
        message.classList.add('with-button')
        message.textContent = 'There is no custom sound yet, '
        const newButton: HTMLElement = document.createElement('button')
        newButton.classList.add('button', 'is-primary')
        newButton.textContent = 'Add One'
        newButton.onclick = () => editSound(guild.id)

        message.appendChild(newButton)
        messageDiv.appendChild(message)
        cell.appendChild(messageDiv)
        return
      }

      const button: HTMLElement = document.createElement('button')
      button.classList.add('button', 'is-primary')
      button.textContent = 'New Sound'
      button.onclick = () => editSound(guild.id)

      // display a table with the guild's sounds
      const collectionTable: HTMLTableElement = document.createElement('table')
      collectionTable.classList.add('table')
      const collectionTh: HTMLTableSectionElement = collectionTable.createTHead()
      const collectionTb: HTMLTableSectionElement = collectionTable.createTBody()

      // create table headers
      const headerRow: HTMLTableRowElement = collectionTh.insertRow(0)
      const nameHead: HTMLTableHeaderCellElement = headerRow.insertCell(0)
      const commandHead: HTMLTableHeaderCellElement = headerRow.insertCell(1)
      const weightHead: HTMLTableHeaderCellElement = headerRow.insertCell(2)
      const editHead: HTMLTableHeaderCellElement = headerRow.insertCell(3)
      const deleteHead: HTMLTableHeaderCellElement = headerRow.insertCell(4)
      nameHead.innerText = 'Name'
      nameHead.style.minWidth = '160px'
      commandHead.innerText = 'Command'
      commandHead.style.minWidth = '80px'
      weightHead.innerText = 'Weight'
      weightHead.style.minWidth = '40px'
      editHead.style.width = '80px'
      deleteHead.style.width = '80px'

      guild.sounds.forEach((sound, j) => {
        const soundRow: HTMLTableRowElement = collectionTb.insertRow(j)
        const nameCell: HTMLTableCellElement = soundRow.insertCell(0)
        const commandCell: HTMLTableCellElement = soundRow.insertCell(1)
        const weightCell: HTMLTableCellElement = soundRow.insertCell(2)
        const editCell: HTMLTableCellElement = soundRow.insertCell(3)
        const deleteCell: HTMLTableCellElement = soundRow.insertCell(4)

        nameCell.innerText = sound.name
        commandCell.innerHTML = `<b>${sound.command}</b>`
        weightCell.innerText = sound.weight.toString()

        const editElt = document.createElement('button')
        editElt.classList.add('button', 'is-primary')
        editElt.textContent = 'Edit'
        editElt.onclick = () => editSound(guild.id, sound)
        editCell.appendChild(editElt)

        const deleteElt = document.createElement('a')
        deleteElt.classList.add('button', 'is-danger')
        deleteElt.textContent = 'Delete'
        deleteElt.onclick = () => deleteSound(guild.id, sound.id)
        deleteCell.appendChild(deleteElt)
      })

      cell.appendChild(button)
      cell.appendChild(collectionTable)
    }

    const updateLayout = () => {
      // erase all rows
      airhornTb.innerHTML = ''
      boringTb.innerHTML = ''
      manageState.extendedRow = -1

      manageState.guilds.airhorn.forEach((guild, i) => {
        const row: HTMLTableRowElement = airhornTb.insertRow(airhornTb.rows.length)
        const logoCell: HTMLTableCellElement = row.insertCell(0)
        const nameCell: HTMLTableCellElement = row.insertCell(1)
        const soundsCell: HTMLTableCellElement = row.insertCell(2)
        const playsCell: HTMLTableCellElement = row.insertCell(3)

        row.onclick = () => rowClick(guild, i)

        const logo = document.createElement('img')
        logo.src = guild.icon
        logo.classList.add('image', 'is-64x64', 'guild-logo')
        logoCell.appendChild(logo)

        const name = document.createElement('b')
        name.textContent = guild.name
        nameCell.appendChild(name)

        const sounds = document.createTextNode(`${guild.sounds.length} / 20`)
        soundsCell.appendChild(sounds)

        const plays = document.createTextNode(guild.plays)
        playsCell.appendChild(plays)
      })
      manageState.guilds.boring.forEach((guild) => {
        const row: HTMLTableRowElement = boringTb.insertRow(boringTb.rows.length)
        const logoCell: HTMLTableCellElement = row.insertCell(0)
        const nameCell: HTMLTableCellElement = row.insertCell(1)
        const loginCell: HTMLTableCellElement = row.insertCell(2)

        const logo = document.createElement('img')
        logo.src = guild.icon
        logo.classList.add('image', 'is-64x64', 'guild-logo')
        logoCell.appendChild(logo)

        const name = document.createElement('b')
        name.textContent = guild.name
        nameCell.appendChild(name)

        const login = document.createElement('a')
        login.classList.add('button', 'is-primary')
        login.href = `/login?guild_id=${guild.id}`
        login.textContent = 'Login >'
        loginCell.appendChild(login)

        row.classList.add('boring-guild')
      })
    }

    // finally display data when the request is finished
    guildsRequest.then(updateLayout)
  })()
})()
