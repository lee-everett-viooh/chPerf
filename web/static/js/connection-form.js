(function() {
	const form = document.getElementById('connection-form');
	if (!form) return;
	const testBtn = document.getElementById('test-btn');
	const resultEl = document.getElementById('test-result');
	if (!testBtn || !resultEl) return;

	testBtn.addEventListener('click', async function() {
		const formData = new FormData(form);
		resultEl.classList.remove('hidden', 'alert-success', 'alert-error');
		resultEl.textContent = 'Testing...';
		try {
			const r = await fetch('/connections/test', {
				method: 'POST',
				body: formData
			});
			const data = await r.json();
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
})();
