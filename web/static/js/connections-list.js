(function() {
	document.querySelectorAll('.test-conn').forEach(function(btn) {
		btn.addEventListener('click', async function() {
			var id = btn.dataset.id;
			var resultEl = document.getElementById('test-result');
			resultEl.classList.remove('hidden', 'alert-success', 'alert-error');
			resultEl.textContent = 'Testing...';
			try {
				var r = await fetch('/connections/' + id + '/test', { method: 'POST' });
				var data = await r.json();
				resultEl.classList.remove('hidden');
				if (data.ok) {
					resultEl.classList.add('alert-success');
					resultEl.textContent = 'Connection successful!';
				} else {
					resultEl.classList.add('alert-error');
					resultEl.textContent = 'Connection failed: ' + (data.error || 'Unknown error');
				}
			} catch (e) {
				resultEl.classList.add('alert-error');
				resultEl.textContent = 'Request failed: ' + e.message;
			}
		});
	});
})();
