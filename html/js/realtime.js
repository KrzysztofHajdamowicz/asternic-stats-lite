/*
   Realtime tab: reloads the agent and queue status from auxstate_helper.php
   every few seconds. Polling pauses while the browser tab is hidden.
*/
(function () {
    'use strict';

    var box, url, interval, timer = null, busy = false;

    function schedule() {
        clearTimeout(timer);
        if (!document.hidden) {
            timer = setTimeout(refresh, interval);
        }
    }

    function refresh() {
        if (busy) {
            return;
        }
        busy = true;
        fetch(url, { cache: 'no-store', credentials: 'same-origin' })
            .then(function (response) {
                if (!response.ok) {
                    throw new Error('HTTP ' + response.status);
                }
                return response.text();
            })
            .then(function (html) {
                box.innerHTML = html;
                box.classList.remove('stale');
            })
            .catch(function () {
                // Keep the last good data on screen, but mark it as outdated.
                box.classList.add('stale');
            })
            .finally(function () {
                busy = false;
                schedule();
            });
    }

    // The "hide logged off agents" checkbox is part of the refreshed HTML.
    function onChange(event) {
        var input = event.target;
        if (!input.matches('[data-hide-loggedoff]')) {
            return;
        }
        var params = new URLSearchParams({ sesvar: 'hideloggedoff', value: input.checked ? 'true' : 'false' });
        fetch('set_sesvar.php?' + params, { cache: 'no-store', credentials: 'same-origin' })
            .finally(refresh);
    }

    function init() {
        box = document.getElementById('content_refresh');
        if (!box) {
            return;
        }
        url = box.dataset.url;
        interval = parseInt(box.dataset.interval, 10) || 5000;
        box.addEventListener('change', onChange);
        document.addEventListener('visibilitychange', function () {
            if (document.hidden) {
                clearTimeout(timer);
            } else {
                refresh();
            }
        });
        refresh();
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
