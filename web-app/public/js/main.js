(function () {
    var AIRHORN_URL = 'https://airhorn.shywim.fr';
    var MESSAGE_TWITTER = 'This Discord bot makes airhorn sounds ayy lmao';
    var HASHTAGS_TWITTER = 'ReadyForHorning';
    var FB_SHARE_URL = "http://www.facebook.com/sharer.php?u=" + AIRHORN_URL;
    var TWITTER_SHARE_URL = "https://twitter.com/share?text=" + MESSAGE_TWITTER + "&url=" + AIRHORN_URL + "&hashtags=" + HASHTAGS_TWITTER;
    var video = document.getElementById('video');
    var audio = document.getElementById('audio');
    var count = document.getElementById('count');
    var playsCount = document.getElementById('plays-count');
    var usersCount = document.getElementById('users-count');
    var guildsCount = document.getElementById('guilds-count');
    var channelsCount = document.getElementById('channels-count');
    var statsToggler = document.getElementById('stats-toggler');
    var statsPanel = document.getElementById('stats-panel');
    var statsPanelClose = document.getElementById('stats-panel-close');
    var statsPanelOpen = document.getElementById('stats-panel-open');
    var shareFb = document.getElementById('share-fb');
    var shareTwitter = document.getElementById('share-twitter');
    var statsTogglerState = false;
    var stats = {
        count: 0,
        users: 0,
        guilds: 0,
        channels: 0,
    };
    // video handler
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
            count.innerText = stats.count.toString();
            playsCount.innerText = stats.count.toString();
            usersCount.innerText = stats.users.toString();
            guildsCount.innerText = stats.guilds.toString();
            channelsCount.innerText = stats.channels.toString();
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
})();
