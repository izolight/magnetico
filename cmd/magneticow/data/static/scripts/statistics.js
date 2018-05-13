"use strict";


window.onload = function() {
    Plotly.newPlot("torrentGraph", torrentsData, {
        title: "New Discovered Torrents Per Day in the Past 30 Days",
        xaxis: {
            title: "Days",
            tickformat: "%d %B"
        },
        yaxis: {
            title: "Amount of Torrents Discovered"
        }
    });

    Plotly.newPlot("filesGraph", filesData, {
        title: "New Discovered Files Per Day in the Past 30 Days",
        xaxis: {
            title: "Days",
            tickformat: "%d %B"
        },
        yaxis: {
            title: "Amount of Files Discovered"
        }
    });

    Plotly.newPlot("sizeGraph", sizeData, {
        title: "New Discovered Bytes Per Day in the Past 30 Days",
        xaxis: {
            title: "Days",
            tickformat: "%d %B"
        },
        yaxis: {
            title: "Amount of Bytes Discovered"
        }
    });
};