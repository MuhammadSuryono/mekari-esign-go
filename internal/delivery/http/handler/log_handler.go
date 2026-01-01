package handler

import (
	"github.com/gofiber/fiber/v2"

	"mekari-esign/internal/infrastructure/repository"
)

type LogHandler struct {
	logRepo repository.APILogRepository
}

func NewLogHandler(logRepo repository.APILogRepository) *LogHandler {
	return &LogHandler{logRepo: logRepo}
}

// LogViewer serves the HTML page for viewing logs
func (h *LogHandler) LogViewer(c *fiber.Ctx) error {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Log Viewer</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #1a1a2e; color: #eee; padding: 20px; }
        h1 { color: #00d4ff; margin-bottom: 20px; }
        .search-box { margin-bottom: 20px; display: flex; gap: 10px; flex-wrap: wrap; }
        input[type="text"] { padding: 12px 16px; font-size: 16px; border: 2px solid #00d4ff; border-radius: 8px; background: #16213e; color: #fff; width: 300px; }
        input[type="text"]:focus { outline: none; border-color: #00ff88; }
        button { padding: 12px 24px; font-size: 16px; background: #00d4ff; color: #000; border: none; border-radius: 8px; cursor: pointer; font-weight: bold; }
        button:hover { background: #00ff88; }
        .btn-secondary { background: #6c5ce7; color: #fff; }
        .btn-secondary:hover { background: #a29bfe; }
        table { width: 100%; border-collapse: collapse; background: #16213e; border-radius: 8px; overflow: hidden; margin-top: 10px; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #0f3460; }
        th { background: #0f3460; color: #00d4ff; font-weight: 600; position: sticky; top: 0; }
        tr:hover { background: #1f4068; }
        .status-success { color: #00ff88; font-weight: bold; }
        .status-error { color: #ff4757; font-weight: bold; }
        .endpoint { max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .body-cell { max-width: 100px; text-align: center; }
        .view-btn { background: #0f3460; color: #00d4ff; padding: 6px 12px; border-radius: 4px; cursor: pointer; border: 1px solid #00d4ff; font-size: 12px; }
        .view-btn:hover { background: #00d4ff; color: #000; }
        .loading { text-align: center; padding: 40px; color: #888; }
        .modal { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.8); z-index: 1000; }
        .modal-content { background: #16213e; margin: 5% auto; padding: 20px; border-radius: 12px; max-width: 80%; max-height: 80%; overflow: auto; }
        .modal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
        .modal-close { font-size: 28px; cursor: pointer; color: #ff4757; }
        .modal-close:hover { color: #ff6b81; }
        pre { background: #0f3460; padding: 15px; border-radius: 8px; overflow: auto; white-space: pre-wrap; word-wrap: break-word; font-size: 13px; max-height: 60vh; }
        .stats { background: #0f3460; padding: 15px; border-radius: 8px; margin-bottom: 20px; display: flex; gap: 30px; }
        .stat-item { text-align: center; }
        .stat-value { font-size: 24px; font-weight: bold; color: #00d4ff; }
        .stat-label { font-size: 12px; color: #888; }
        .table-container { max-height: 70vh; overflow: auto; }
    </style>
</head>
<body>
    <h1>🔍 API Log Viewer</h1>
    
    <div class="search-box">
        <input type="text" id="invoiceInput" placeholder="Enter Invoice Number..." onkeypress="if(event.key==='Enter')searchLogs()">
        <button onclick="searchLogs()">🔎 Search</button>
    </div>

    <div id="stats" class="stats" style="display:none;">
        <div class="stat-item">
            <div class="stat-value" id="totalCount">0</div>
            <div class="stat-label">Total Logs</div>
        </div>
        <div class="stat-item">
            <div class="stat-value status-success" id="successCount">0</div>
            <div class="stat-label">Success</div>
        </div>
        <div class="stat-item">
            <div class="stat-value status-error" id="errorCount">0</div>
            <div class="stat-label">Errors</div>
        </div>
    </div>

    <div id="tableContainer">
        <p class="loading">Enter an invoice number to search logs or click "Load All"...</p>
    </div>

    <div id="modal" class="modal" onclick="closeModal(event)">
        <div class="modal-content" onclick="event.stopPropagation()">
            <div class="modal-header">
                <h3 id="modalTitle">Details</h3>
                <span class="modal-close" onclick="closeModal()">&times;</span>
            </div>
            <pre id="modalBody"></pre>
        </div>
    </div>

    <script>
        let currentLogs = [];

        async function searchLogs() {
            const invoice = document.getElementById('invoiceInput').value.trim();
            if (!invoice) { alert('Please enter an invoice number'); return; }
            await fetchLogs('/api/v1/logs/search?invoice=' + encodeURIComponent(invoice));
        }

        async function loadAll() {
            await fetchLogs('/api/v1/logs?limit=50');
        }

        async function fetchLogs(url) {
            document.getElementById('tableContainer').innerHTML = '<p class="loading">Loading...</p>';
            document.getElementById('stats').style.display = 'none';
            try {
                const res = await fetch(url);
                const data = await res.json();
                if (data.success && data.data) {
                    currentLogs = data.data;
                    renderTable(data.data);
                    updateStats(data.data);
                } else {
                    document.getElementById('tableContainer').innerHTML = '<p class="loading">No logs found</p>';
                }
            } catch (err) {
                document.getElementById('tableContainer').innerHTML = '<p class="loading">Error: ' + err.message + '</p>';
            }
        }

        function updateStats(logs) {
            if (!logs || logs.length === 0) return;
            
            const success = logs.filter(l => l.status_code >= 200 && l.status_code < 300).length;
            const errors = logs.length - success;
            
            document.getElementById('totalCount').textContent = logs.length;
            document.getElementById('successCount').textContent = success;
            document.getElementById('errorCount').textContent = errors;
            document.getElementById('stats').style.display = 'flex';
        }

        function renderTable(logs) {
            if (!logs || logs.length === 0) {
                document.getElementById('tableContainer').innerHTML = '<p class="loading">No logs found</p>';
                return;
            }
            let html = '<div class="table-container"><table><thead><tr><th>ID</th><th>Invoice Number</th><th>Time</th><th>Method</th><th>Endpoint</th><th>Status</th><th>Duration</th><th>Email</th><th>Request</th><th>Response</th></tr></thead><tbody>';
            logs.forEach((log, idx) => {
                const statusClass = log.status_code >= 200 && log.status_code < 300 ? 'status-success' : 'status-error';
                const time = new Date(log.created_at).toLocaleString();
                html += '<tr>' +
                    '<td>' + log.id + '</td>' +
                    '<td>' + log.invoice_no + '</td>' +
                    '<td>' + time + '</td>' +
                    '<td><strong>' + log.method + '</strong></td>' +
                    '<td class="endpoint" title="' + escapeHtml(log.endpoint) + '">' + escapeHtml(log.endpoint) + '</td>' +
                    '<td class="' + statusClass + '">' + log.status_code + '</td>' +
                    '<td>' + log.duration_ms + 'ms</td>' +
                    '<td>' + (log.email || '-') + '</td>' +
                    '<td class="body-cell"><button class="view-btn" onclick="showBody(' + idx + ', \'request\')">View</button></td>' +
                    '<td class="body-cell"><button class="view-btn" onclick="showBody(' + idx + ', \'response\')">View</button></td>' +
                    '</tr>';
            });
            html += '</tbody></table></div>';
            document.getElementById('tableContainer').innerHTML = html;
        }

        function escapeHtml(str) {
            if (!str) return '';
            return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
        }

        function showBody(idx, type) {
            const log = currentLogs[idx];
            const title = type === 'request' ? 'Request Body' : 'Response Body';
            const body = type === 'request' ? log.request_body : log.response_body;
            
            document.getElementById('modalTitle').textContent = title + ' (ID: ' + log.id + ')';
            try {
                document.getElementById('modalBody').textContent = JSON.stringify(JSON.parse(body), null, 2);
            } catch {
                document.getElementById('modalBody').textContent = body || '(empty)';
            }
            document.getElementById('modal').style.display = 'block';
        }

        function closeModal(e) {
            if (!e || e.target.id === 'modal') {
                document.getElementById('modal').style.display = 'none';
            }
        }

        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') closeModal();
        });

		document.addEventListener('DOMContentLoaded', function () {
			loadAll();
		});
    </script>
</body>
</html>`
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

// GetLogs returns all logs with limit
func (h *LogHandler) GetLogs(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}

	logs, err := h.logRepo.FindAll(c.Context(), limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "data": logs})
}

// SearchLogs searches logs by invoice number
func (h *LogHandler) SearchLogs(c *fiber.Ctx) error {
	invoice := c.Query("invoice")
	if invoice == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invoice parameter required"})
	}

	logs, err := h.logRepo.FindByInvoice(c.Context(), invoice)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "data": logs})
}
