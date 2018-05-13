"use strict";

var layout = {
    title: "New Discovered Torrents Per Day in the Past 30 Days",
    xaxis: {
        title: "Days",
        tickformat: "%d %B"
    },
    yaxis: {
        title: "Torrents",
        titlefont: {color: '#1f77b4'},
        tickfont: {color: '#1f77b4'}
    },
    yaxis2: {
        title: "Files",
        titlefont: {color: '#ff7f0e'},
        tickfont: {color: '#ff7f0e'},
        anchor: 'free',
        overlaying: 'y',
        side: 'left',
        position: 0.15
    },
    yaxis3: {
        title: "Bytes",
        titlefont: {color: '#d62728'},
        tickfont: {color: '#d62728'},
        anchor: 'x',
        overlaying: 'y',
        side: 'right',
        position: 0.85,
        tickformat: ".3s"
    }
}

window.onload = function() {
    Plotly.newPlot("torrentGraph", data, layout);
};