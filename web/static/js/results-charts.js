(function() {
	var dataEl = document.getElementById('chart-data');
	if (!dataEl || !dataEl.textContent) return;
	var data = JSON.parse(dataEl.textContent);

	// Timeseries charts (one per query)
	var timeseries = data.timeseries || [];
	for (var i = 0; i < timeseries.length; i++) {
		var ts = timeseries[i];
		var canvas = document.getElementById('ts-chart-' + i);
		canvas.parentElement.style.height = '500px';
		if (!canvas || !ts.points || ts.points.length === 0) continue;

		var pts = ts.points;
		new Chart(canvas.getContext('2d'), {
			type: 'line',
			data: {
				labels: pts.map(function(p) { return p.t; }),
				datasets: [{
					label: 'Latency (ms)',
					data: pts.map(function(p) { return p.y; }),
					borderColor: 'rgba(59, 130, 246, 1)',
					backgroundColor: 'rgba(59, 130, 246, 0.1)',
					fill: true,
					tension: 0.1
				}]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				scales: {
					x: {
						title: { display: true, text: 'Time' }
					},
					y: {
						beginAtZero: true,
						title: { display: true, text: 'Latency (ms)' }
					}
				}
			}
		});
	}
})();
