(function () {
    var AIRHORN_URL = 'https://airhorn.shywim.fr';
    var MESSAGE_TWITTER = 'This Discord bot makes airhorn sounds ayy lmao';
    var HASHTAGS_TWITTER = 'ReadyForHorning';
    var FB_SHARE_URL = "http://www.facebook.com/sharer.php?u=" + AIRHORN_URL;
    var TWITTER_SHARE_URL = "https://twitter.com/share?text=" + MESSAGE_TWITTER + "&url=" + AIRHORN_URL + "&hashtags=" + HASHTAGS_TWITTER;
    var MODAL_ID = 'modal';
    var video = document.getElementById('video');
    var audio = document.getElementById('audio');
    var count = document.getElementById('count');
    var playsCount = document.getElementById('plays-count');
    var usersCount = document.getElementById('users-count');
    var guildsCount = document.getElementById('guilds-count');
    var channelsCount = document.getElementById('channels-count');
    var statsToggler = document.getElementById('stats-toggler');
    var statsBtn = document.getElementById('stats-btn');
    var statsPanel = document.getElementById('stats-panel');
    var statsPanelClose = document.getElementById('stats-panel-close');
    var statsPanelOpen = document.getElementById('stats-panel-open');
    var shareFb = document.getElementById('share-fb');
    var shareTwitter = document.getElementById('share-twitter');
    // javascript is enabled
    statsPanel.classList.remove('is-hidden');
    statsBtn.classList.remove('is-hidden');
    count.innerText = '0';
    var statsTogglerState = false;
    var stats = {
        channels: 0,
        count: 0,
        guilds: 0,
        users: 0,
    };
    // video handler
    video.addEventListener('ended', function () {
        video.currentTime = 0;
    }, false);
    video.onclick = function () {
        video.play();
        audio.play();
    };
    // bot stats
    var removeCountBig = function () {
        count.classList.remove('count-big');
    };
    if (EventSource != null) {
        var es = new EventSource('/events');
        es.onmessage = function (msg) {
            var data = JSON.parse(msg.data);
            if (stats.count !== data.total) {
                count.classList.add('count-big');
                setTimeout(removeCountBig, 400);
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
    // panel handler
    statsToggler.onclick = function () {
        var remove;
        var add;
        if (statsTogglerState === false) {
            remove = '-reverse';
            add = '';
        }
        else {
            remove = '';
            add = '-reverse';
        }
        statsTogglerState = !statsTogglerState;
        statsPanelClose.classList.remove("three" + remove);
        statsPanelClose.classList.add("three" + add);
        statsPanelOpen.classList.remove("two" + remove);
        statsPanelOpen.classList.add("two" + add);
        statsPanel.classList.remove("one" + remove);
        statsPanel.classList.add("one" + add);
    };
    // social share
    var openShare = function (url) {
        window.open(url, '', 'height=500, width=500');
    };
    shareFb.onclick = function () { return openShare(FB_SHARE_URL); };
    shareTwitter.onclick = function () { return openShare(TWITTER_SHARE_URL); };
    // play video if just logged in
    if (URLSearchParams != null) {
        var params = new URLSearchParams(location.search.slice(1));
        var keyToSuccess = params.get('key_to_success');
        if (keyToSuccess === '1') {
            setTimeout(function () {
                video.play();
                audio.play();
            }, 1000);
        }
    }
    // --- MANAGE ---
    (function () {
        var manageState = {
            extendedRow: -1,
            guilds: {
                airhorn: [],
                boring: [],
            },
        };
        var mainSection = document.getElementsByClassName('main-section')[0];
        var manageNotification = document.getElementById('manage-notification');
        var manageSection = document.getElementById('manage-section');
        var manageButton = document.getElementById('manage-btn');
        var homeButton = document.getElementById('back-home');
        var airhornTable = document.getElementById('airhorn-guilds');
        var boringTable = document.getElementById('boring-guilds');
        var editSoundModal = document.getElementById('edit-sound-modal');
        var editSoundModalForm = document.getElementById('edit-sound-form');
        var editSoundModalFile = document.getElementById('edit-sound-file');
        var editSoundModalCancel = editSoundModal.getElementsByTagName('button')[0];
        var circleContainer = document.getElementsByClassName('circle-container')[0];
        var circleReveal = document.getElementsByClassName('circle-reveal')[0];
        /**
         * Switch between home and manage section
         * @param show
         */
        var showHideManage = function (show) {
            if (show === true) {
                if (window.location.hash !== '#manage') {
                    history.pushState(null, document.title, '#manage');
                }
                mainSection.classList.add('is-hidden');
                manageSection.classList.remove('is-hidden');
            }
            else {
                if (window.location.hash !== '') {
                    history.pushState(null, document.title, window.location.pathname + window.location.search);
                }
                mainSection.classList.remove('is-hidden');
                manageSection.classList.add('is-hidden');
            }
        };
        /**
         * Display the form for editing a sound details or creating a new sound
         * @param guildId
         * @param sound
         */
        var editSound = function (guildId, sound) {
            var nameKey = 'name';
            var commandKey = 'command';
            var weightKey = 'weight';
            var fileKey = 'file';
            if (sound != null) {
                editSoundModalForm.action = "/manage/" + guildId + "/" + sound.id;
                editSoundModalForm.elements[nameKey].value = sound.name;
                editSoundModalForm.elements[commandKey].value = sound.command;
                editSoundModalForm.elements[weightKey].value = sound.weight;
                editSoundModalForm.elements[fileKey].required = false;
                editSoundModalFile.classList.add('is-hidden');
                editSoundModalForm.onsubmit = function () { return postForm(guildId, sound.id); };
            }
            else {
                editSoundModalForm.action = "/manage/" + guildId + "/new";
                editSoundModalForm.elements[nameKey].value = '';
                editSoundModalForm.elements[commandKey].value = '';
                editSoundModalForm.elements[weightKey].value = '';
                editSoundModalForm.elements[fileKey].required = true;
                editSoundModalFile.classList.remove('is-hidden');
                editSoundModalForm.onsubmit = function () { return postForm(guildId); };
            }
            editSoundModal.classList.add('is-active');
        };
        /**
         * Send form data to the server
         * @param guildId guild's id
         * @param soundId optional soundId, if null it creates a new sound
         */
        var postForm = function (guildId, soundId) {
            var formData = new FormData(editSoundModalForm);
            var url;
            var method;
            // if the sound is null it is a new sound
            if (soundId == null) {
                url = "/manage/" + guildId + "/new";
                method = 'POST';
            }
            else {
                url = "/manage/" + guildId + "/" + soundId;
                method = 'PUT';
            }
            fetch(url, {
                body: formData,
                credentials: 'same-origin',
                method: method,
            })
                .then(function (response) {
                if (response.ok === true) {
                    editSoundModal.classList.remove('is-active');
                    return fetchGuilds()
                        .then(updateLayout);
                }
                // TODO: handle errors
            });
            return false;
        };
        /**
         * Send a request to delete the given sound
         * @param guildId guild's id
         * @param soundId sound to delete
         */
        var deleteSound = function (guildId, soundId) {
            return fetch("/manage/" + guildId + "/" + soundId, {
                credentials: 'same-origin',
                method: 'DELETE',
            })
                .then(function (response) {
                if (response.ok === true) {
                    return fetchGuilds()
                        .then(updateLayout);
                }
                // TODO: handle errors
            });
        };
        var fetchGuilds = function () {
            return fetch('/me/guilds', {
                credentials: 'same-origin',
            })
                .then(function (response) {
                if (response.ok) {
                    return response.json();
                }
                if (response.status === 401) {
                    manageNotification.classList.remove('is-hidden');
                    return Promise.reject('unauthorized');
                }
            })
                .then(function (json) { return manageState.guilds = json; });
            // TODO: handle errors
        };
        // javascript and this module are enabled
        var guildsRequest = fetchGuilds();
        editSoundModalCancel.onclick = function () { return editSoundModal.classList.remove('is-active'); };
        manageButton.classList.remove('disabled');
        manageButton.onclick = function () {
            // TODO: redirect to login if not logged in
            showHideManage(true);
        };
        homeButton.onclick = function () {
            showHideManage(false);
        };
        // show manage section if #manage
        if (window.location.hash === '#manage') {
            // TODO: don't do it if not logged in
            setTimeout(function () { return showHideManage(true); }, 1000);
        }
        var airhornTb = airhornTable.getElementsByTagName('tbody')[0];
        var boringTb = boringTable.getElementsByTagName('tbody')[0];
        var rowClick = function (guild, i) {
            if (manageState.extendedRow === i) {
                // juste close if the same row was clicked
                circleReveal.classList.remove('expand');
                circleReveal.style.width = '0';
                circleReveal.style.height = '0';
                setTimeout(function () {
                    airhornTb.rows.item(i).classList.remove('selected');
                    airhornTb.deleteRow(i + 1);
                }, 330);
                manageState.extendedRow = -1;
                return;
            }
            var oldRow = manageState.extendedRow;
            manageState.extendedRow = i;
            var detailsRow = airhornTb.insertRow(i + 1);
            var cell = detailsRow.insertCell(0);
            cell.colSpan = 4;
            if (guild.sounds.length === 0) {
                // display a nice message
                var messageDiv = document.createElement('div');
                var message = document.createElement('p');
                message.classList.add('with-button');
                message.textContent = 'There is no custom sound yet, ';
                var newButton = document.createElement('button');
                newButton.classList.add('button', 'is-primary');
                newButton.textContent = 'Add One';
                newButton.onclick = function () { return editSound(guild.id); };
                message.appendChild(newButton);
                messageDiv.appendChild(message);
                cell.appendChild(messageDiv);
                return;
            }
            var button = document.createElement('button');
            button.classList.add('button', 'is-primary');
            button.textContent = 'New Sound';
            button.onclick = function () { return editSound(guild.id); };
            // display a table with the guild's sounds
            var collectionTable = document.createElement('table');
            collectionTable.classList.add('table');
            var collectionTh = collectionTable.createTHead();
            var collectionTb = collectionTable.createTBody();
            // create table headers
            var headerRow = collectionTh.insertRow(0);
            var nameHead = headerRow.insertCell(0);
            var commandHead = headerRow.insertCell(1);
            var weightHead = headerRow.insertCell(2);
            var editHead = headerRow.insertCell(3);
            var deleteHead = headerRow.insertCell(4);
            nameHead.innerText = 'Name';
            nameHead.style.minWidth = '160px';
            commandHead.innerText = 'Command';
            commandHead.style.minWidth = '80px';
            weightHead.innerText = 'Weight';
            weightHead.style.minWidth = '40px';
            editHead.style.width = '80px';
            deleteHead.style.width = '80px';
            guild.sounds.forEach(function (sound, j) {
                var soundRow = collectionTb.insertRow(j);
                var nameCell = soundRow.insertCell(0);
                var commandCell = soundRow.insertCell(1);
                var weightCell = soundRow.insertCell(2);
                var editCell = soundRow.insertCell(3);
                var deleteCell = soundRow.insertCell(4);
                nameCell.innerText = sound.name;
                commandCell.innerHTML = "<b>" + sound.command + "</b>";
                weightCell.innerText = sound.weight.toString();
                var editElt = document.createElement('button');
                editElt.classList.add('button', 'is-primary');
                editElt.textContent = 'Edit';
                editElt.onclick = function () { return editSound(guild.id, sound); };
                editCell.appendChild(editElt);
                var deleteElt = document.createElement('a');
                deleteElt.classList.add('button', 'is-danger');
                deleteElt.textContent = 'Delete';
                deleteElt.onclick = function () { return deleteSound(guild.id, sound.id); };
                deleteCell.appendChild(deleteElt);
            });
            cell.appendChild(button);
            circleReveal.classList.remove('expand');
            circleReveal.style.width = '0';
            circleReveal.style.height = '0';
            setTimeout(function () {
                if (oldRow > -1) {
                    // delete row if a details row exists
                    airhornTb.deleteRow(oldRow + 1);
                }
                // remove hover effect from bulma
                airhornTb.rows.item(i).classList.add('selected');
                airhornTb.rows.item(i + 1).classList.add('selected');
                cell.appendChild(collectionTable);
                setTimeout(function () {
                    // reenable bulma's hover on previously selected row
                    if (oldRow > -1) {
                        airhornTb.rows.item(oldRow).classList.remove('selected');
                    }
                    // set the circle reveal animation size
                    var cellPos = cell.getBoundingClientRect();
                    circleContainer.style.top = cellPos.top - 100 + "px";
                    circleContainer.style.left = cellPos.left + "px";
                    var cellHeight = cell.offsetHeight + 100;
                    circleContainer.style.height = cellHeight + "px";
                    circleContainer.style.width = cell.offsetWidth + "px";
                    var circleSize = 2 * (cell.offsetWidth > cellHeight ? cell.offsetWidth : cellHeight);
                    circleReveal.classList.add('expand');
                    circleReveal.style.width = circleSize + "px";
                    circleReveal.style.height = circleSize + "px";
                }, 330);
            }, 330);
        };
        var updateLayout = function () {
            // erase all rows
            airhornTb.innerHTML = '';
            boringTb.innerHTML = '';
            manageState.extendedRow = -1;
            manageState.guilds.airhorn.forEach(function (guild, i) {
                var row = airhornTb.insertRow(airhornTb.rows.length);
                var logoCell = row.insertCell(0);
                var nameCell = row.insertCell(1);
                var soundsCell = row.insertCell(2);
                var playsCell = row.insertCell(3);
                row.onclick = function () { return rowClick(guild, i); };
                var logo = document.createElement('img');
                logo.src = guild.icon;
                logo.classList.add('image', 'is-64x64', 'guild-logo');
                logoCell.appendChild(logo);
                var name = document.createElement('b');
                name.textContent = guild.name;
                nameCell.appendChild(name);
                var sounds = document.createTextNode(guild.sounds.length + " / 20");
                soundsCell.appendChild(sounds);
                var plays = document.createTextNode(guild.plays);
                playsCell.appendChild(plays);
            });
            manageState.guilds.boring.forEach(function (guild) {
                var row = boringTb.insertRow(boringTb.rows.length);
                var logoCell = row.insertCell(0);
                var nameCell = row.insertCell(1);
                var loginCell = row.insertCell(2);
                var logo = document.createElement('img');
                logo.src = guild.icon;
                logo.classList.add('image', 'is-64x64', 'guild-logo');
                logoCell.appendChild(logo);
                var name = document.createElement('b');
                name.textContent = guild.name;
                nameCell.appendChild(name);
                var login = document.createElement('a');
                login.classList.add('button', 'is-primary');
                login.href = "/login?guild_id=" + guild.id;
                login.textContent = 'Login >';
                loginCell.appendChild(login);
                row.classList.add('boring-guild');
            });
        };
        // finally display data when the request is finished
        guildsRequest.then(updateLayout);
    })();
    function ListItem() {
        var view = document.createElement('div');
        view.className = 'list-item';
        return view;
    }
    function GuildItem() {
        var view = ListItem();
    }
})();
