/*
   Report form on the home page: queue/agent pickers, date shortcuts and
   the checks done before the form is sent.
*/
(function () {
    'use strict';

    function moveOptions(from, to, all) {
        Array.from(from.options).forEach(function (option) {
            if (all || option.selected) {
                option.selected = false;
                to.appendChild(option);
            }
        });
    }

    function initPicker(picker) {
        var from = picker.querySelector('[data-from]');
        var to = picker.querySelector('[data-to]');
        picker.addEventListener('click', function (event) {
            var button = event.target.closest('[data-move]');
            if (!button) {
                return;
            }
            switch (button.dataset.move) {
                case 'add': moveOptions(from, to, false); break;
                case 'remove': moveOptions(to, from, false); break;
                case 'add-all': moveOptions(from, to, true); break;
                case 'remove-all': moveOptions(to, from, true); break;
            }
        });
        from.addEventListener('dblclick', function () { moveOptions(from, to, false); });
        to.addEventListener('dblclick', function () { moveOptions(to, from, false); });
    }

    function init() {
        var form = document.getElementById('report-form');
        if (!form) {
            return;
        }
        var dateStart = document.getElementById('date-start');
        var dateEnd = document.getElementById('date-end');

        form.querySelectorAll('[data-picker]').forEach(initPicker);

        form.querySelectorAll('[data-start]').forEach(function (button) {
            button.addEventListener('click', function () {
                dateStart.value = button.dataset.start;
                dateEnd.value = button.dataset.end;
            });
        });

        form.addEventListener('submit', function (event) {
            var queues = form.elements['List_Queue[]'];
            var agents = form.elements['List_Agent[]'];
            var error = null;

            if (queues.options.length === 0) {
                error = form.dataset.msgQueue;
            } else if (agents.options.length === 0) {
                error = form.dataset.msgAgent;
            } else if (!dateStart.value || !dateEnd.value || dateEnd.value < dateStart.value) {
                error = form.dataset.msgDate;
            }
            if (error) {
                event.preventDefault();
                alert(error);
                return;
            }

            // Everything in the "selected" lists is sent, highlighted or not.
            Array.from(queues.options).forEach(function (option) { option.selected = true; });
            Array.from(agents.options).forEach(function (option) { option.selected = true; });
            form.elements.start.value = dateStart.value + ' 00:00:00';
            form.elements.end.value = dateEnd.value + ' 23:59:59';
        });
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
