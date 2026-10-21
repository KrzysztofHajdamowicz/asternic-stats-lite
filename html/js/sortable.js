/*
   Click a column header of a <table class="sortable"> to sort its rows;
   click again to reverse. Works for tables added later too (realtime tab).

   Understands plain numbers, "12.5 %", "443 secs" and m:ss durations.
*/
(function () {
    'use strict';

    function sortKey(text) {
        text = text.trim();
        var duration = text.match(/^(\d+):(\d{2})\b/);
        if (duration) {
            return parseInt(duration[1], 10) * 60 + parseInt(duration[2], 10);
        }
        var number = text.match(/^[+-]?\d+(\.\d+)?/);
        if (number) {
            return parseFloat(number[0]);
        }
        return text.toLowerCase();
    }

    function compare(a, b) {
        if (typeof a === 'number' && typeof b === 'number') {
            return a - b;
        }
        return String(a).localeCompare(String(b));
    }

    function sortTable(th) {
        var table = th.closest('table');
        var body = table.tBodies[0];
        if (!body) {
            return;
        }
        var column = Array.prototype.indexOf.call(th.parentNode.children, th);
        var ascending = th.getAttribute('aria-sort') !== 'ascending';

        table.querySelectorAll('th[aria-sort]').forEach(function (other) {
            other.removeAttribute('aria-sort');
        });
        th.setAttribute('aria-sort', ascending ? 'ascending' : 'descending');

        var rows = Array.from(body.rows).map(function (row) {
            var cell = row.cells[column];
            return { row: row, key: sortKey(cell ? cell.textContent : '') };
        });
        rows.sort(function (a, b) {
            var result = compare(a.key, b.key);
            return ascending ? result : -result;
        });
        rows.forEach(function (item) {
            body.appendChild(item.row);
        });
    }

    document.addEventListener('click', function (event) {
        var th = event.target.closest('table.sortable thead th');
        if (th) {
            sortTable(th);
        }
    });
})();
