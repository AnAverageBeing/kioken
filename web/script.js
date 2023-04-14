const numTotalConn = document.getElementById("numTotalConn");
const numConnPerSec = document.getElementById("numConnPerSec");
const numActiveConn = document.getElementById("numActiveConn");
const numIpPerSec = document.getElementById("numIpPerSec");

const chartCanvas = document.getElementById("chart");
const chartCtx = chartCanvas.getContext("2d");

let limit = 120;

// Custom plugin to draw value tags on the chart
Chart.plugins.register({
    afterDatasetsDraw: function(chart, easing) {
      // Loop through each dataset
      chart.data.datasets.forEach(function(dataset, datasetIndex) {
        const meta = chart.getDatasetMeta(datasetIndex);
        if (!meta.hidden) {
          // Loop through each point in the dataset
          meta.data.forEach(function(point, index) {
            // Only draw a tag every 5 seconds
            if (index % 5 === 0) {
              const currentValue = dataset.data[index];
              const text = currentValue.toFixed(2);
              // Draw the tag as a small box with the current value
              chartCtx.fillStyle = dataset.borderColor;
              chartCtx.fillRect(point._model.x - 15, point._model.y - 15, 30, 30);
              chartCtx.textAlign = 'center';
              chartCtx.font = '12px Arial';
              chartCtx.fillText(text, point._model.x, point._model.y + 4);
            }
          });
        }
      });
    }
  });

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
                pointRadius: 0,
                pointHoverRadius: 0
            },
            {
                label: 'Active Connections',
                data: [],
                backgroundColor: 'rgba(54, 162, 235, 0.2)',
                borderColor: 'rgba(54, 162, 235, 1)',
                borderWidth: 2,
                fill: false,
                pointRadius: 0,
                pointHoverRadius: 0
            },
            {
                label: 'IPs Per Second',
                data: [],
                backgroundColor: 'rgba(255, 206, 86, 0.2)',
                borderColor: 'rgba(255, 206, 86, 1)',
                borderWidth: 2,
                fill: false,
                pointRadius: 0,
                pointHoverRadius: 0
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

ws.onmessage = function(event) {
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
