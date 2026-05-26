// ═══════════════════════════════════════════════════════════════════
// AI Tracker — Google Apps Script (Config & Adoption Tracking)
// ═══════════════════════════════════════════════════════════════════
// Foca em eventos de CONFIGURAÇÃO (não commits).
// Commits são rastreados via GitHub API posteriormente.
//
// Deploy:
//   1. Tools → Script Editor → colar este código
//   2. Deploy → New Deployment → Web App
//      - Execute as: Me
//      - Who has access: Anyone
//   3. Copiar URL e configurar: export AI_TRAILER_WEBHOOK_URL=<URL>
// ═══════════════════════════════════════════════════════════════════

var CONFIG = {
  AUTH_TOKEN: PropertiesService.getScriptProperties().getProperty('AUTH_TOKEN') || '',
  RATE_LIMIT_PER_IP: 60,
};

// Colunas da planilha
var COLUMNS = [
  "Timestamp",              // A
  "User Email",             // B: git config user.email
  "User Name",              // C: git config user.name
  "System User",            // D: $USER / whoami
  "Hostname",               // E
  "Platform",               // F: linux / darwin / win32 / wsl
  "Event",                  // G: install / configure / update / uninstall
  "Tools Detected",         // H: ferramentas auto-detectadas (lista)
  "Tools Configured",       // I: ferramentas selecionadas (lista)
  "Hook Installed",         // J: true/false
  "CLAUDE.md Updated",      // K: true/false (legacy)
  "Instruction Files",      // L: arquivos de instrução atualizados
  "Webhook Configured",     // M: true/false
  "Client Version",         // N
  "Extra",                  // O: JSON adicional
];

var rateLimitCache = CacheService.getScriptCache();

// ── doPost ──────────────────────────────────────────────────────────

function doPost(e) {
  try {
    // Auth check
    if (CONFIG.AUTH_TOKEN) {
      var auth = (e.parameter && e.parameter.auth) || '';
      if (e.postData && e.postData.headers) {
        var h = e.postData.headers['Authorization'] || e.postData.headers['authorization'] || '';
        if (h.startsWith('Bearer ')) auth = h.substring(7);
      }
      if (auth !== CONFIG.AUTH_TOKEN) {
        return error("Unauthorized", 401);
      }
    }

    // Rate limit
    var ip = (e && e.postData && e.postData.headers && e.postData.headers['X-Forwarded-For'] || '').split(',')[0].trim();
    if (ip) {
      var key = 'rl_' + ip;
      var count = parseInt(rateLimitCache.get(key) || '0');
      if (count >= CONFIG.RATE_LIMIT_PER_IP) return error("Rate limit exceeded", 429);
      rateLimitCache.put(key, String(count + 1), 60);
    }

    // Parse payload
    var data;
    if (e.postData && e.postData.contents) {
      data = JSON.parse(e.postData.contents);
    } else if (e.parameter && e.parameter.payload) {
      data = JSON.parse(e.parameter.payload);
    } else {
      return error("No payload", 400);
    }

    if (!data.event) return error("Missing 'event' field", 400);

    // Open sheet
    var sheet = SpreadsheetApp.getActiveSpreadsheet().getActiveSheet();

    // Setup header on first use
    if (sheet.getLastRow() === 0) {
      sheet.appendRow(COLUMNS);
      sheet.setFrozenRows(1);
      sheet.getRange(1, 1, 1, COLUMNS.length).setFontWeight("bold").setBackground("#1a1a2e").setFontColor("#ffffff");
    }

    // Append row
    sheet.appendRow([
      data.timestamp              || new Date().toISOString(),
      data.user_email             || "",
      data.user_name              || "",
      data.system_user            || "",
      data.hostname               || "",
      data.platform               || "",
      data.event                  || "",
      data.tools_detected         || "",
      data.tools_configured       || "",
      String(data.hook_installed  || false),
      String(data.claude_md_updated || data.instruction_files_updated ? 'true' : 'false'),
      data.instruction_files_updated || "",
      String(data.webhook_configured || false),
      data.client_version         || "",
      data.extra                  || "",
    ]);

    // Auto-resize periodically
    if (sheet.getLastRow() % 20 === 0) {
      for (var c = 1; c <= COLUMNS.length; c++) sheet.autoResizeColumn(c);
    }

    return ok({ status: "ok", row: sheet.getLastRow() });

  } catch (err) {
    return error("Error: " + err.toString(), 500);
  }
}

// ── doGet (health check) ────────────────────────────────────────────

function doGet(e) {
  var sheet = SpreadsheetApp.getActiveSpreadsheet().getActiveSheet();
  return ok({
    status: "ok",
    message: "AI Tracker — Config & Adoption webhook",
    total_rows: sheet.getLastRow(),
    auth_configured: !!CONFIG.AUTH_TOKEN,
    timestamp: new Date().toISOString(),
  });
}

// ── Helpers ─────────────────────────────────────────────────────────

function ok(data) {
  return ContentService.createTextOutput(JSON.stringify(data))
    .setMimeType(ContentService.MimeType.JSON);
}

function error(msg, code) {
  return ContentService.createTextOutput(JSON.stringify({ status: "error", message: msg }))
    .setMimeType(ContentService.MimeType.JSON);
}

// ── Setup ──────────────────────────────────────────────────────────

function setupSheet() {
  var sheet = SpreadsheetApp.getActiveSpreadsheet().getActiveSheet();
  sheet.clear();
  sheet.appendRow(COLUMNS);
  sheet.setFrozenRows(1);
  sheet.getRange(1, 1, 1, COLUMNS.length)
    .setFontWeight("bold")
    .setBackground("#1a1a2e")
    .setFontColor("#ffffff");
  sheet.getRange("A:A").setNumberFormat("yyyy-mm-dd hh:mm:ss");
  Logger.log("Setup complete. " + COLUMNS.length + " columns.");
}
