(function () {
  const script = document.currentScript;
  const runId = script.getAttribute("data-run-id");
  const startedAt = new Date(script.getAttribute("data-started-at"));

  const elapsedEl = document.getElementById("elapsed-time");
  const executedEl = document.getElementById("executed-count");
  const errorEl = document.getElementById("error-count");

  function padZero(n) {
    return n < 10 ? "0" + n : "" + n;
  }

  function updateClock() {
    const diff = Math.floor((Date.now() - startedAt.getTime()) / 1000);
    const h = Math.floor(diff / 3600);
    const m = Math.floor((diff % 3600) / 60);
    const s = diff % 60;
    elapsedEl.textContent = padZero(h) + ":" + padZero(m) + ":" + padZero(s);
  }

  let clockInterval = setInterval(updateClock, 1000);
  updateClock();

  function poll() {
    fetch("/api/runs/" + runId + "/status")
      .then(function (r) { return r.json(); })
      .then(function (data) {
        executedEl.textContent = (data.executed_count || 0).toLocaleString();
        errorEl.textContent = (data.error_count || 0).toLocaleString();

        if (data.status === "completed" || data.status === "failed") {
          clearInterval(clockInterval);
          clearInterval(pollInterval);
          window.location.reload();
        }
      })
      .catch(function () {});
  }

  let pollInterval = setInterval(poll, 1000);
  poll();
})();
