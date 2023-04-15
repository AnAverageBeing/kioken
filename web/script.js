// Get DOM elements
const numTotalConn = document.getElementById("numTotalConn");
const numConnPerSec = document.getElementById("numConnPerSec");
const numActiveConn = document.getElementById("numActiveConn");
const numIpPerSec = document.getElementById("numIpPerSec");
const chartCanvas = document.getElementById("chart");

// Initialize variables
let limit = 120;
const chartCtx = chartCanvas.getContext("2d");

// Create chart
const chart = new Chart(chartCtx, {
  type: "line",
  data: {
    labels: [],
    datasets: [
      {
        label: "Connections Per Second",
        data: [],
        backgroundColor: "rgba(255, 99, 132, 0.2)",
        borderColor: "rgba(255, 99, 132, 1)",
        borderWidth: 4,
        fill: true,
        pointRadius: 0,
        pointHoverRadius: 1,
      },
      {
        label: "Active Connections",
        data: [],
        backgroundColor: "rgba(54, 162, 235, 0.2)",
        borderColor: "rgba(54, 162, 235, 1)",
        borderWidth: 4,
        fill: true,
        pointRadius: 0,
        pointHoverRadius: 1,
      },
      {
        label: "IPs Per Second",
        data: [],
        backgroundColor: "rgba(255, 206, 86, 0.2)",
        borderColor: "rgba(255, 206, 86, 1)",
        borderWidth: 4,
        fill: true,
        pointRadius: 0,
        pointHoverRadius: 1,
      },
    ],
  },
  options: {
    responsive: true,
    scales: {
      xAxes: [{ display: false }],
      yAxes: [{ display: true, ticks: { beginAtZero: true } }],
    },
  },
});

// Connect to WebSocket
const ws = new WebSocket("ws://" + window.location.host + "/ws");

// Handle WebSocket messages
ws.onmessage = function (event) {
  const data = JSON.parse(event.data);

  // Update text content of elements
  numTotalConn.innerText = data.numTotalConn;
  numConnPerSec.innerText = data.numConnPerSec;
  numActiveConn.innerText = data.numActiveConn;
  numIpPerSec.innerText = data.numIpPerSec;

  // Update chart data
  const timestamp = new Date().toLocaleTimeString();
  chart.data.labels.push(timestamp);
  chart.data.datasets[0].data.push(data.numConnPerSec);
  chart.data.datasets[1].data.push(data.numActiveConn);
  chart.data.datasets[2].data.push(data.numIpPerSec);

  // Remove oldest data points if limit is reached
  if (chart.data.labels.length > limit) {
    chart.data.labels.splice(0, 1);
    chart.data.datasets[0].data.splice(0, 1);
    chart.data.datasets[1].data.splice(0, 1);
    chart.data.datasets[2].data.splice(0, 1);
  }

  // Update chart
  chart.update();
};
