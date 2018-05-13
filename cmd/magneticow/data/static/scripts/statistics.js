"use strict";

var layout = {
    title: "New Discovered Torrents Per Day in the Past 30 Days",
    xaxis: {
        title: "Days",
        tickformat: "%d %B",
        domain: [0.1, 0.95]
    },
    yaxis: {
        title: "Torrents",
        titlefont: {color: '#1f77b4'},
        tickfont: {color: '#1f77b4'},
        side: "left",
        position: 0
    },
    yaxis2: {
        title: "Files",
        titlefont: {color: '#ff7f0e'},
        tickfont: {color: '#ff7f0e'},
        anchor: 'free',
        overlaying: 'y',
        side: 'left',
        position: 0.07
    },
    yaxis3: {
        title: "Bytes",
        titlefont: {color: '#2ca02c'},
        tickfont: {color: '#2ca02c'},
        anchor: 'x',
        overlaying: 'y',
        side: 'right',
        position: 0.95,
        tickformat: ".3s"
    }
}

window.onload = function() {
    Plotly.newPlot("torrentGraph", data, layout);
};