(() => {
    const form = document.querySelector('.upload-form');
    if (!form) return;
    const input = form.querySelector('.upload-file');
    const dashboard = document.querySelector('.jobs-dashboard');
    const summary = document.querySelector('.jobs-summary');
    const list = document.querySelector('.jobs-list');
    const cards = new Map();
    let uploading = false;
    const selected = [];
    const previews = [];
    let refreshing = false;
    let timer;
    let pollSeconds = Number(dashboard.dataset.pollSeconds);

    const element = (tag, className, text) => {
        const node = document.createElement(tag);
        node.className = className;
        if (text !== undefined) node.textContent = text;
        return node;
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
        if (selected.length) {
            for (const entry of selected) {
                const job = jobs.find(job => job.id === entry.id);
                if (!job) continue;
                entry.card.dataset.state = job.state;
                const labels = {queued: 'Queued', processing: 'Processing', succeeded: 'Completed', failed: 'Failed'};
                entry.status.textContent = labels[job.state] || job.state;
                if (job.state === 'processing') entry.progress.removeAttribute('value');
                else entry.progress.value = job.state === 'succeeded' ? 100 : 0;
                entry.detail.textContent = job.state === 'failed'
                    ? job.error || 'Processing failed.'
                    : job.state === 'processing' ? 'Validating and processing…'
                    : job.state === 'queued' ? 'Waiting for a worker…' : 'Ready';
                if (job.results.length && !entry.completed) {
                    entry.completed = true;
                    const media = job.results[0];
                    const kind = (media.mime || entry.file.type).split('/')[0];
                    if (kind === 'image' || kind === 'video') {
                        const preview = element(kind === 'image' ? 'img' : 'video', 'upload-preview');
                        preview.src = media.url;
                        if (kind === 'video') {
                            preview.controls = true;
                            preview.preload = 'metadata';
                            preview.playsInline = true;
                        } else {
                            preview.alt = entry.file.name;
                            preview.decoding = 'async';
                        }
                        entry.preview.replaceChildren(preview);
                    }
                    const result = renderJob(job);
                    entry.result.append(...result.querySelectorAll('.job-results, .job-details'));
                }
            }
            summary.textContent = 'Reload the page to start another upload.';
            return;
        }
        dashboard.hidden = !jobs.some(job => job.state === 'queued' || job.state === 'processing');
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
    const uploadFile = (entry, fields) => new Promise(resolve => {
        const data = new FormData();
        for (const [name, value] of fields) {
            if (!(value instanceof File)) data.append(name, value);
        }
        data.append(input.name, entry.file);
        const xhr = new XMLHttpRequest();
        entry.status.textContent = 'Uploading';
        entry.detail.textContent = 'Sending file…';
        xhr.open('POST', '/upload/jobs/multipart/');
        xhr.setRequestHeader('Accept', 'application/json');
        xhr.upload.addEventListener('progress', event => {
            if (event.lengthComputable) {
                const percent = Math.round(event.loaded / event.total * 100);
                entry.progress.value = percent;
                entry.detail.textContent = `Uploading: ${percent}%`;
            } else {
                entry.progress.removeAttribute('value');
            }
        });
        xhr.upload.addEventListener('load', () => {
            entry.progress.removeAttribute('value');
            entry.detail.textContent = 'Saving to the queue…';
        });
        const failed = message => {
            entry.card.dataset.state = 'failed';
            entry.status.textContent = 'Failed';
            entry.progress.value = 0;
            entry.detail.textContent = message;
        };
        xhr.addEventListener('load', () => {
            let result;
            try { result = JSON.parse(xhr.responseText); } catch (_) {}
            if (xhr.status !== 202 || !result?.id) {
                failed(result?.error?.message || `Upload failed (HTTP ${xhr.status}).`);
            } else {
                entry.id = result.id;
                entry.status.textContent = 'Queued';
                entry.progress.value = 0;
                entry.detail.textContent = 'Waiting for a worker…';
                refresh();
            }
        });
        const disconnected = () => failed('Connection interrupted. Reload to check the shared queue before retrying; the server may have received this file.');
        xhr.addEventListener('error', disconnected);
        xhr.addEventListener('abort', disconnected);
        xhr.addEventListener('loadend', resolve);
        xhr.send(data);
    });
    const startUpload = async files => {
        if (uploading || !files.length) return;
        uploading = true;
        const fields = new FormData(form);
        document.querySelector('.upload').hidden = true;
        dashboard.hidden = false;
        dashboard.querySelector('h2').textContent = 'Your upload';
        dashboard.querySelector('.jobs-description').hidden = true;
        summary.textContent = 'Reload the page to start another upload.';
        list.replaceChildren();
        cards.clear();
        for (const file of files) {
            const card = element('article', 'block job-card');
            const heading = element('div', 'job-heading');
            heading.append(element('h3', 'job-files', file.name));
            const status = element('span', 'job-state', 'Waiting');
            heading.append(status);
            card.append(heading);
            const previewContainer = element('div', 'file-preview');
            card.append(previewContainer);
            const kind = file.type.split('/')[0];
            if (kind === 'image') {
                const preview = element('img', 'upload-preview');
                const url = URL.createObjectURL(file);
                previews.push(url);
                preview.src = url;
                preview.alt = file.name;
                preview.loading = 'lazy';
                preview.decoding = 'async';
                preview.addEventListener('error', () => {
                    preview.replaceWith(element('p', 'job-detail', 'Preview unavailable in this browser.'));
                }, {once: true});
                previewContainer.append(preview);
            } else if (kind === 'video') {
                const placeholder = element('div', 'upload-video-placeholder');
                const icon = element('span', 'upload-video-icon', '▶');
                icon.setAttribute('aria-hidden', 'true');
                placeholder.append(icon, element('span', '', 'Video'));
                previewContainer.append(placeholder);
            } else {
                previewContainer.append(element('p', 'job-detail', 'Preview unavailable in this browser.'));
            }
            const progress = element('progress', 'job-progress');
            progress.max = 100;
            progress.value = 0;
            progress.setAttribute('aria-label', `Upload progress for ${file.name}`);
            const detail = element('p', 'job-detail', 'Waiting to upload…');
            detail.setAttribute('role', 'status');
            const result = element('div', 'upload-result');
            card.append(progress, detail, result);
            list.append(card);
            selected.push({file, card, status, progress, detail, result, preview: previewContainer});
        }
        for (const entry of selected) await uploadFile(entry, fields);
    };
    form.addEventListener('submit', event => {
        event.preventDefault();
        if (input.files.length) startUpload(Array.from(input.files));
        else input.click();
    });
    const dropZone = document.querySelector('.upload');
    dropZone.addEventListener('dragover', event => {
        event.preventDefault();
        event.dataTransfer.dropEffect = 'copy';
    });
    dropZone.addEventListener('drop', event => {
        event.preventDefault();
        startUpload(Array.from(event.dataTransfer.files));
    });
    window.addEventListener('dragover', event => event.preventDefault());
    window.addEventListener('drop', event => event.preventDefault());
    window.addEventListener('pagehide', event => {
        if (!event.persisted) previews.forEach(url => URL.revokeObjectURL(url));
    });
    input.addEventListener('change', () => startUpload(Array.from(input.files)));
    refresh();
})();
