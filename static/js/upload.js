(() => {
    const form = document.querySelector('.upload-form');
    if (!form) return;
    const input = form.querySelector('.upload-file');
    const uploadPanel = document.querySelector('.upload-status');
    const uploadMessage = document.querySelector('.upload-message');
    const transferProgress = document.querySelector('.upload-progress');
    const dashboard = document.querySelector('.jobs-dashboard');
    const summary = document.querySelector('.jobs-summary');
    const list = document.querySelector('.jobs-list');
    const cards = new Map();
    let uploading = false;
    let refreshing = false;
    let timer;
    let pollSeconds = Number(dashboard.dataset.pollSeconds);

    const element = (tag, className, text) => {
        const node = document.createElement(tag);
        node.className = className;
        if (text !== undefined) node.textContent = text;
        return node;
    };
    const showUpload = message => {
        uploadPanel.hidden = false;
        uploadMessage.textContent = message;
    };
    const setUploading = value => {
        uploading = value;
        for (const control of form.elements) control.disabled = value;
    };
    const renderJob = job => {
        const card = element('article', 'block job-card');
        card.dataset.jobId = job.id;
        card.dataset.state = job.state;
        const heading = element('div', 'job-heading');
        heading.append(element('h3', '', `${job.files} ${job.files === 1 ? 'file' : 'files'}`));
        const states = {queued: 'Queued', processing: 'Processing', succeeded: 'Completed', failed: 'Failed'};
        heading.append(element('span', 'job-state', states[job.state] || job.state));
        card.append(heading);
        if (job.source_names?.length) card.append(element('p', 'job-files', job.source_names.join(', ')));
        const date = new Date(job.created);
        const submitted = element('time', 'job-created', date.toLocaleString());
        submitted.dateTime = job.created;
        card.append(submitted);
        const completed = job.results.length;
        const progress = element('progress', 'job-progress');
        progress.max = job.files;
        progress.setAttribute('aria-label', `Job ${job.id}: ${completed} of ${job.files} files completed`);
        if (job.state !== 'processing') progress.value = completed;
        card.append(progress);
        let detail = `${completed} of ${job.files} files completed.`;
        if (job.state === 'queued') detail += ' Waiting for a worker.';
        if (job.state === 'processing') detail += ' Validating and processing the next file…';
        if (job.state === 'failed') detail += ` ${job.error || 'Processing failed.'}`;
        card.append(element('p', 'job-detail', detail));
        if (job.results.length) {
            const results = element('ul', 'job-results');
            for (const result of job.results) {
                let url;
                try { url = new URL(result.url, window.location.href); } catch (_) { continue; }
                if (!['http:', 'https:'].includes(url.protocol)) continue;
                const row = element('li', 'job-result');
                const link = element('a', 'job-url', result.url);
                link.href = url.href;
                link.target = '_blank';
                link.rel = 'noopener';
                row.append(link);
                const copy = element('button', 'button job-copy', 'Copy URL');
                copy.type = 'button';
                copy.addEventListener('click', async () => {
                    try {
                        await navigator.clipboard.writeText(result.url);
                        copy.textContent = 'Copied';
                    } catch (_) {
                        copy.textContent = 'Select the link to copy';
                    }
                });
                row.append(copy);
                results.append(row);
            }
            card.append(results);
            const details = element('a', 'job-details', 'Previews and embed codes →');
            details.href = '/meta/' + job.results.map(result => encodeURIComponent(result.filename)).join(',');
            card.append(details);
        }
        return card;
    };
    const renderJobs = jobs => {
        const present = new Set();
        let previous = null;
        for (const job of jobs) {
            present.add(job.id);
            const signature = JSON.stringify(job);
            let entry = cards.get(job.id);
            if (!entry || entry.signature !== signature) {
                const node = renderJob(job);
                if (entry) entry.node.replaceWith(node);
                entry = {signature, node};
                cards.set(job.id, entry);
            }
            const expected = previous ? previous.nextSibling : list.firstChild;
            if (entry.node !== expected) list.insertBefore(entry.node, expected);
            previous = entry.node;
        }
        for (const [id, entry] of cards) {
            if (!present.has(id)) {
                entry.node.remove();
                cards.delete(id);
            }
        }
        const count = state => jobs.filter(job => job.state === state).length;
        summary.textContent = jobs.length
            ? `${count('queued')} queued · ${count('processing')} processing · ${count('succeeded')} completed · ${count('failed')} failed`
            : 'No background uploads yet.';
    };
    const refresh = async () => {
        if (refreshing) return;
        clearTimeout(timer);
        refreshing = true;
        let retry = true;
        try {
            const response = await fetch('/jobs/', {headers: {Accept: 'application/json'}, cache: 'no-store'});
            if (response.status === 401) {
                summary.textContent = 'Sign in again and reload to view the shared queue.';
                retry = false;
                return;
            }
            if (!response.ok) throw new Error('Could not refresh jobs');
            const data = await response.json();
            pollSeconds = data.poll_after_seconds;
            renderJobs(data.jobs);
        } catch (_) {
            summary.textContent = 'Connection interrupted. Retrying; your jobs continue on the server.';
        } finally {
            refreshing = false;
            if (retry) timer = setTimeout(refresh, pollSeconds * 1000);
        }
    };
    form.addEventListener('submit', event => {
        event.preventDefault();
        if (uploading || !input.files.length) return;
        const data = new FormData(form);
        const count = input.files.length;
        const xhr = new XMLHttpRequest();
        setUploading(true);
        transferProgress.hidden = false;
        transferProgress.max = 100;
        transferProgress.value = 0;
        showUpload(`Uploading ${count} ${count === 1 ? 'file' : 'files'}…`);
        xhr.open('POST', '/upload/jobs/multipart/');
        xhr.setRequestHeader('Accept', 'application/json');
        xhr.upload.addEventListener('progress', event => {
            if (event.lengthComputable) {
                const percent = Math.round(event.loaded / event.total * 100);
                transferProgress.value = percent;
                showUpload(`Uploading ${count} ${count === 1 ? 'file' : 'files'}: ${percent}%`);
            } else {
                transferProgress.removeAttribute('value');
            }
        });
        xhr.upload.addEventListener('load', () => showUpload('Upload sent. Saving to the queue…'));
        xhr.addEventListener('load', () => {
            let result;
            try { result = JSON.parse(xhr.responseText); } catch (_) {}
            if (xhr.status !== 202 || !result?.id) {
                showUpload(result?.error?.message || `Upload failed (HTTP ${xhr.status}).`);
            } else {
                input.value = '';
                showUpload('Upload queued. You can add more files; results will appear below.');
            }
            refresh();
        });
        const disconnected = () => {
            showUpload('Upload connection interrupted. Check the shared queue before retrying; the server may have received it.');
            refresh();
        };
        xhr.addEventListener('error', disconnected);
        xhr.addEventListener('abort', disconnected);
        xhr.addEventListener('loadend', () => {
            transferProgress.hidden = true;
            setUploading(false);
        });
        xhr.send(data);
    });
    input.addEventListener('change', () => form.requestSubmit());
    refresh();
})();
