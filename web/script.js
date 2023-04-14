const numTotalConn = document.getElementById("numTotalConn");
const numConnPerSec = document.getElementById("numConnPerSec");
const numActiveConn = document.getElementById("numActiveConn");
const numIpPerSec = document.getElementById("numIpPerSec");

const chartCanvas = document.getElementById("chart");
const chartCtx = chartCanvas.getContext("2d");

let limit = 60;

const chart = new Chart(chartCtx, {
    type: 'line',
    data: {
        labels: [],
        datasets: [
            {
                label: 'Connections Per Second',
                data: [],
                backgroundColor: 'rgba(255, 99, 132, 0.2)',
                borderColor: 'rgba(255, 99, 132, 1)',
                borderWidth: 2,
                fill: false,
            },
            {
                label: 'Active Connections',
                data: [],
                backgroundColor: 'rgba(54, 162, 235, 0.2)',
                borderColor: 'rgba(54, 162, 235, 1)',
                borderWidth: 2,
                fill: false,
            },
            {
                label: 'IPs Per Second',
                data: [],
                backgroundColor: 'rgba(255, 206, 86, 0.2)',
                borderColor: 'rgba(255, 206, 86, 1)',
                borderWidth: 2,
                fill: false,
            }
        ]
    },
    options: {
        responsive: true,
        scales: {
            xAxes: [{
                display: false
            }],
            yAxes: [{
                display: true,
                ticks: {
                    beginAtZero: true
                }
            }]
        }
    }
});

const ws = new WebSocket("ws://" + window.location.host + "/ws");

ws.onmessage = function (event) {
    const data = JSON.parse(event.data);
    numTotalConn.innerText = data.numTotalConn;
    numConnPerSec.innerText = data.numConnPerSec;
    numActiveConn.innerText = data.numActiveConn;
    numIpPerSec.innerText = data.numIpPerSec;

    const timestamp = new Date().toLocaleTimeString();
    chart.data.labels.push(timestamp);
    chart.data.datasets[0].data.push(data.numConnPerSec);
    chart.data.datasets[1].data.push(data.numActiveConn);
    chart.data.datasets[2].data.push(data.numIpPerSec);

    if (chart.data.labels.length > limit) {
        chart.data.labels.splice(0, 1);
        chart.data.datasets[0].data.splice(0, 1);
        chart.data.datasets[1].data.splice(0, 1);
        chart.data.datasets[2].data.splice(0, 1);
    }

    chart.update();
};
