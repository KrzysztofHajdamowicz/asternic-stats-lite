/*
   Copyright 2026 Krzysztof Hajdamowicz
   Part of Asternic Call Center Stats Lite, GPL-3.0-or-later (see LICENSE).

   Draws every <canvas data-chart='{...}'> on the page with Chart.js.
   The JSON comes from chart_bar() in misc.php:
     { title, labels: [...], datasets: [{label, data: [...]}, ...], stacked }

   Colours are read from CSS custom properties (--chart-1, --chart-2,
   --chart-text, --chart-grid) so the charts follow the page theme.
*/
(function () {
    'use strict';

    function cssVar(name, fallback) {
        var value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
        return value || fallback;
    }

    function theme() {
        return {
            series: [cssVar('--chart-1', '#ff6600'), cssVar('--chart-2', '#538353')],
            text: cssVar('--chart-text', '#333333'),
            grid: cssVar('--chart-grid', 'rgba(0, 0, 0, 0.1)')
        };
    }

    function build(canvas, colors) {
        var config = JSON.parse(canvas.dataset.chart);
        var datasets = config.datasets.map(function (dataset, i) {
            return {
                label: dataset.label,
                data: dataset.data,
                backgroundColor: colors.series[i % colors.series.length],
                borderRadius: 2
            };
        });
        var axis = {
            stacked: config.stacked,
            grid: { color: colors.grid },
            ticks: { color: colors.text }
        };
        return new Chart(canvas, {
            type: 'bar',
            data: { labels: config.labels, datasets: datasets },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: { mode: 'index', intersect: false },
                scales: {
                    x: Object.assign({}, axis, { grid: { display: false } }),
                    y: Object.assign({}, axis, { beginAtZero: true, ticks: { color: colors.text, precision: 0 } })
                },
                plugins: {
                    title: { display: true, text: config.title, color: colors.text, font: { size: 14 } },
                    legend: { display: datasets.length > 1, labels: { color: colors.text } }
                }
            }
        });
    }

    function drawAll() {
        var colors = theme();
        document.querySelectorAll('canvas[data-chart]').forEach(function (canvas) {
            var old = Chart.getChart(canvas);
            if (old) {
                old.destroy();
            }
            build(canvas, colors);
        });
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', drawAll);
    } else {
        drawAll();
    }

    // Redraw with the other palette when the system switches light/dark mode.
    if (window.matchMedia) {
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', drawAll);
    }
})();
