/* Cheque bounce section 138 case clock — local UI.
   Talks only to this binary over same-origin fetch. No other network use. */
(function () {
  "use strict";

  /* ---------------- theme ---------------- */

  var root = document.documentElement;

  function storedTheme() {
    try { return localStorage.getItem("omsme-theme"); } catch (e) { return null; }
  }

  function currentTheme() {
    var set = root.getAttribute("data-theme");
    if (set) return set;
    return (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches) ? "dark" : "light";
  }

  function paintThemer(button) {
    if (!button) return;
    var dark = currentTheme() === "dark";
    button.textContent = dark ? "☀ Light" : "☾ Dark";
    button.setAttribute("aria-label", dark ? "Switch to light theme" : "Switch to dark theme");
  }

  function wireTheme() {
    var button = document.querySelector("[data-themer]");
    if (!button) return;
    paintThemer(button);
    button.addEventListener("click", function () {
      var next = currentTheme() === "dark" ? "light" : "dark";
      root.setAttribute("data-theme", next);
      try { localStorage.setItem("omsme-theme", next); } catch (e) { /* private window */ }
      paintThemer(button);
    });
    if (window.matchMedia) {
      window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", function () {
        if (!storedTheme()) paintThemer(button);
      });
    }
  }

  /* ---------------- helpers ---------------- */

  function $(id) { return document.getElementById(id); }

  var rupees = new Intl.NumberFormat("en-IN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

  function inr(value) {
    var n = Number(value);
    if (!isFinite(n)) n = 0;
    return "₹ " + rupees.format(n);
  }

  function number(value) {
    var n = parseFloat(value);
    return isFinite(n) ? n : 0;
  }

  function integer(value) {
    var n = parseInt(value, 10);
    return isFinite(n) ? n : 0;
  }

  var MONTHS = ["January", "February", "March", "April", "May", "June",
    "July", "August", "September", "October", "November", "December"];

  /* Dates arrive as YYYY-MM-DD and are formatted as plain text. They are never
     put through the Date constructor, which would drag a timezone into a
     calculation that has no business having one. */
  function pretty(iso) {
    if (!iso) return "—";
    var parts = String(iso).split("-");
    if (parts.length !== 3) return iso;
    var month = MONTHS[integer(parts[1]) - 1];
    if (!month) return iso;
    return parts[2] + " " + month + " " + parts[0];
  }

  function daysPhrase(days) {
    if (days < -1) return (-days) + " days ago";
    if (days === -1) return "yesterday";
    if (days === 0) return "today";
    if (days === 1) return "tomorrow";
    return "in " + days + " days";
  }

  function el(tag, text, className) {
    var node = document.createElement(tag);
    if (text !== undefined && text !== null) node.textContent = text;
    if (className) node.className = className;
    return node;
  }

  function line(text, className) {
    var div = document.createElement("div");
    div.appendChild(el("small", text, className));
    return div;
  }

  var messageTimer = null;

  function showMessage(text, isError) {
    var box = $("message");
    box.textContent = text;
    box.hidden = false;
    if (messageTimer) window.clearTimeout(messageTimer);
    if (!isError) {
      messageTimer = window.setTimeout(function () { box.hidden = true; }, 6000);
    }
  }

  function fail(err) {
    showMessage(err && err.message ? err.message : "Something went wrong.", true);
  }

  function api(path, options) {
    return fetch(path, options).then(function (res) {
      return res.text().then(function (raw) {
        var body = {};
        if (raw) {
          try { body = JSON.parse(raw); } catch (e) { body = {}; }
        }
        if (!res.ok) {
          throw new Error(body.error || ("The tool refused that (" + res.status + ")."));
        }
        return body;
      });
    });
  }

  function jsonPost(path, payload, method) {
    return api(path, {
      method: method || "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload)
    });
  }

  /* ---------------- state ---------------- */

  var state = null;
  var editingId = null;
  var selectedId = null;
  var settingsLoaded = false;

  /* The case form: every field is an input id, a case field, and whether it is
     a date. One list drives reading the form, filling it and clearing it. */
  var FIELDS = [
    ["ref", "ref", false],
    ["amount", "amount", false],
    ["chequeNo", "chequeNo", false],
    ["chequeDate", "chequeDate", true],
    ["drawerName", "drawerName", false],
    ["drawerAddress", "drawerAddress", false],
    ["payeeName", "payeeName", false],
    ["payeeAddress", "payeeAddress", false],
    ["consideration", "consideration", false],
    ["draweeBank", "draweeBank", false],
    ["draweeBranch", "draweeBranch", false],
    ["collectingBank", "collectingBank", false],
    ["collectingBranch", "collectingBranch", false],
    ["presentedDate", "presentedDate", true],
    ["memoDate", "memoDate", true],
    ["memoReceivedDate", "memoReceivedDate", true],
    ["memoReason", "memoReason", false],
    ["noticeSentDate", "noticeSentDate", true],
    ["noticeServedDate", "noticeServedDate", true],
    ["noticeMode", "noticeMode", false],
    ["paidDate", "paidDate", true],
    ["complaintFiledDate", "complaintFiledDate", true],
    ["courtName", "courtName", false],
    ["caseNo", "caseNo", false],
    ["notes", "notes", false]
  ];

  var PERIOD_FIELDS = [
    "noticeDays", "paymentDays", "complaintMonths", "chequeValidityMonths",
    "statutoryPresentationMonths", "deemedServiceDays", "civilSuitYears"
  ];

  var DOCS = [
    ["notice", "Demand notice"],
    ["schedule", "Deadline schedule"],
    ["particulars", "Complaint particulars"],
    ["checklist", "File checklist"]
  ];

  function asOfQuery() {
    var value = $("asOf").value;
    return value ? "?asOf=" + encodeURIComponent(value) : "";
  }

  function load() {
    return api("/api/state" + asOfQuery()).then(apply).catch(fail);
  }

  function apply(data) {
    state = data;
    if (!$("asOf").value && data.today) $("asOf").value = data.today;
    if (!settingsLoaded) {
      fillSettings(data);
      settingsLoaded = true;
    }
    paintPeriodState();
    render();
  }

  function fillSettings(data) {
    $("advocate").value = data.office.advocate || "";
    $("firm").value = data.office.firm || "";
    $("officeAddress").value = data.office.address || "";
    $("phone").value = data.office.phone || "";
    $("email").value = data.office.email || "";
    $("enrolment").value = data.office.enrolment || "";
    PERIOD_FIELDS.forEach(function (key) { $(key).value = data.periods[key]; });

    var list = $("periodLines");
    list.textContent = "";
    (data.periodLines || []).forEach(function (text) { list.appendChild(el("li", text)); });
    if (data.disclaimer) $("disclaimer").textContent = data.disclaimer;
  }

  function paintPeriodState() {
    var statutory = true;
    PERIOD_FIELDS.forEach(function (key) {
      if (integer($(key).value) !== state.statutoryPeriods[key]) statutory = false;
    });
    $("periodState").textContent = statutory
      ? "These are the figures as enacted."
      : "One or more figure has been changed from the Act. Every date in the docket, and every document printed, uses the figures above.";
  }

  /* ---------------- the docket summary ---------------- */

  function renderSummary() {
    var s = state.summary;
    $("sumCases").textContent = s.cases;
    $("sumUrgent").textContent = s.urgent;
    $("sumBarred").textContent = s.barred;
    $("sumClosed").textContent = s.closed;

    if (!s.nextDue) {
      $("nextDue").textContent = "—";
      $("nextSub").textContent = s.cases
        ? "Nothing live on the limitation clock."
        : "No cases on record yet. Add the first one below.";
      return;
    }
    $("nextDue").textContent = pretty(s.nextDue);
    $("nextSub").textContent = s.nextRef + " · " + daysPhrase(s.nextDays) + " · " + s.nextWhat;
  }

  /* ---------------- the case list ---------------- */

  function render() {
    renderSummary();

    var rows = $("caseRows");
    rows.textContent = "";
    $("emptyNote").hidden = state.cases.length > 0;

    if (selectedId && !byId(selectedId)) selectedId = null;
    if (!selectedId && state.cases.length) selectedId = state.cases[0].id;

    state.cases.forEach(function (item) {
      rows.appendChild(caseRow(item));
    });
    renderCascade();
  }

  function byId(id) {
    for (var i = 0; i < state.cases.length; i++) {
      if (state.cases[i].id === id) return state.cases[i];
    }
    return null;
  }

  function caseRow(item) {
    var tr = document.createElement("tr");
    var cascade = item.cascade;

    var file = document.createElement("td");
    file.appendChild(el("div", item.ref || "(no reference)"));
    file.appendChild(line(item.drawerName + " · cheque " + item.chequeNo));
    file.appendChild(line(inr(item.amount) + " · memo " + pretty(item.memoDate)));
    tr.appendChild(file);

    var stage = document.createElement("td");
    stage.appendChild(el("div", cascade.stageLabel || "—"));
    if (cascade.fatal && cascade.fatal.length) {
      stage.appendChild(line(cascade.fatal.length + " claim-ending problem(s)", "neg"));
    } else if (cascade.warnings && cascade.warnings.length) {
      stage.appendChild(line(cascade.warnings.length + " warning(s)"));
    }
    tr.appendChild(stage);

    var next = document.createElement("td");
    if (cascade.nextDue) {
      next.appendChild(el("div", pretty(cascade.nextDue)));
      next.appendChild(line(daysPhrase(cascade.nextDaysLeft)));
    } else {
      next.appendChild(el("div", "—"));
    }
    tr.appendChild(next);

    var actions = document.createElement("td");
    actions.appendChild(button("Open", function () {
      selectedId = item.id;
      render();
      $("cascadeCard").scrollIntoView({ block: "nearest" });
    }));
    actions.appendChild(button("Edit", function () { startEdit(item); }));
    actions.appendChild(button("Delete", function () { remove(item); }));
    tr.appendChild(actions);
    return tr;
  }

  function button(label, onClick) {
    var b = el("button", label, "btn ghost");
    b.type = "button";
    b.addEventListener("click", onClick);
    return b;
  }

  /* ---------------- the cascade panel ---------------- */

  function renderCascade() {
    var item = selectedId ? byId(selectedId) : null;
    $("cascadeEmpty").hidden = !!item;
    $("cascadeBody").hidden = !item;
    if (!item) return;

    var cascade = item.cascade;
    $("cascadeTitle").textContent = "The cascade — " + (item.ref || "(no reference)");
    $("caseStage").textContent = cascade.stageLabel || "—";
    $("caseNextDue").textContent = cascade.nextDue ? pretty(cascade.nextDue) : "—";
    $("caseNextAction").textContent = cascade.nextAction || "—";

    var rows = $("cascadeRows");
    rows.textContent = "";
    (cascade.steps || []).forEach(function (step) {
      rows.appendChild(stepRow(step, cascade));
    });

    var flags = $("caseFlags");
    flags.textContent = "";
    (cascade.fatal || []).forEach(function (text) {
      var p = el("p", text, "note");
      flags.appendChild(p);
    });
    (cascade.warnings || []).forEach(function (text) {
      flags.appendChild(el("p", text, "assumptions"));
    });

    var docs = $("caseDocs");
    docs.textContent = "";
    DOCS.forEach(function (pair) {
      docs.appendChild(button(pair[1], function () { openDoc(item, pair[0], pair[1]); }));
    });
    var ics = el("a", "Add to my calendar (.ics)", "btn ghost");
    ics.href = "/api/cases/" + encodeURIComponent(item.id) + "/calendar.ics" + asOfQuery();
    ics.setAttribute("download", "");
    docs.appendChild(ics);
  }

  function stepRow(step, cascade) {
    var tr = document.createElement("tr");

    var what = document.createElement("td");
    what.appendChild(el("div", step.name));
    what.appendChild(line(step.section + " · " + step.actor + " · " + step.state));
    what.appendChild(line(step.rule));
    if (step.from) what.appendChild(line("counted from " + pretty(step.from) + " (" + step.fromWhat + ")"));
    if (step.done) what.appendChild(line("done on " + pretty(step.done)));
    if (step.note) what.appendChild(line(step.note));
    tr.appendChild(what);

    var when = document.createElement("td");
    if (step.due) {
      when.appendChild(el("div", pretty(step.due)));
      when.appendChild(line(daysPhrase(step.daysLeft)));
      if (step.weekend) when.appendChild(line("falls on a weekend"));
      if (step.outer) when.appendChild(line("outer limit " + pretty(step.outer)));
      if (step.key === "complaint" && cascade.fileBy && cascade.fileBy !== step.due) {
        when.appendChild(line("work to " + pretty(cascade.fileBy)));
      }
    } else {
      when.appendChild(el("div", "—"));
    }
    tr.appendChild(when);
    return tr;
  }

  /* ---------------- documents ---------------- */

  function openDoc(item, kind, label) {
    jsonPost("/api/documents", { kind: kind, caseId: item.id, asOf: $("asOf").value })
      .then(function (doc) {
        $("docTitle").textContent = doc.title || label;
        $("docText").textContent = doc.text;
        $("docCard").hidden = false;
        $("docCard").scrollIntoView({ block: "start" });
      })
      .catch(fail);
  }

  function wireDocCard() {
    $("printDoc").addEventListener("click", function () { window.print(); });
    $("closeDoc").addEventListener("click", function () { $("docCard").hidden = true; });
    $("copyDoc").addEventListener("click", function () {
      var text = $("docText").textContent;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(function () {
          showMessage("Copied. Nothing has been sent.");
        }).catch(function () {
          showMessage("Could not copy — select the text and copy it yourself.", true);
        });
        return;
      }
      showMessage("Select the text and copy it yourself.", true);
    });
  }

  /* ---------------- the case form ---------------- */

  function readForm() {
    var payload = {};
    FIELDS.forEach(function (field) {
      var value = $(field[0]).value;
      if (field[1] === "amount") {
        payload.amount = number(value);
        return;
      }
      payload[field[1]] = field[2] ? (value || "") : value.trim();
    });
    payload.id = editingId || "";
    return payload;
  }

  function fillForm(item) {
    FIELDS.forEach(function (field) {
      var value = item[field[1]];
      $(field[0]).value = (value === undefined || value === null) ? "" : value;
    });
  }

  function startEdit(item) {
    editingId = item.id;
    fillForm(item);
    $("formTitle").textContent = "Edit " + (item.ref || "this case");
    $("save").textContent = "Save the case";
    $("cancelEdit").hidden = false;
    $("formCard").scrollIntoView({ block: "start" });
  }

  function stopEdit() {
    editingId = null;
    $("caseForm").reset();
    $("formTitle").textContent = "Add a dishonoured cheque";
    $("save").textContent = "Add the case";
    $("cancelEdit").hidden = true;
  }

  function submitCase(event) {
    event.preventDefault();
    var payload = readForm();
    var path = editingId ? "/api/cases/" + encodeURIComponent(editingId) : "/api/cases";
    jsonPost(path + asOfQuery(), payload, editingId ? "PUT" : "POST")
      .then(function (data) {
        if (!editingId && data.cases.length) {
          selectedId = data.cases[data.cases.length - 1].id;
        }
        showMessage(editingId ? "Saved." : "Added. The cascade is on the right.");
        stopEdit();
        apply(data);
      })
      .catch(fail);
  }

  function remove(item) {
    var label = item.ref || "this case";
    if (!window.confirm("Delete " + label + " from your docket? This cannot be undone.")) return;
    api("/api/cases/" + encodeURIComponent(item.id) + asOfQuery(), { method: "DELETE" })
      .then(function (data) {
        if (editingId === item.id) stopEdit();
        if (selectedId === item.id) selectedId = null;
        showMessage("Deleted.");
        apply(data);
      })
      .catch(fail);
  }

  /* ---------------- settings ---------------- */

  function submitSettings(event) {
    event.preventDefault();
    var periods = {};
    PERIOD_FIELDS.forEach(function (key) { periods[key] = integer($(key).value); });
    jsonPost("/api/settings" + asOfQuery(), {
      office: {
        advocate: $("advocate").value,
        firm: $("firm").value,
        address: $("officeAddress").value,
        phone: $("phone").value,
        email: $("email").value,
        enrolment: $("enrolment").value
      },
      periods: periods
    }).then(function (data) {
      showMessage("Saved. Every date in the docket has been recomputed.");
      fillSettings(data);
      apply(data);
    }).catch(fail);
  }

  function resetPeriods() {
    PERIOD_FIELDS.forEach(function (key) { $(key).value = state.statutoryPeriods[key]; });
    paintPeriodState();
    showMessage("Reset to the statutory figures. Save to apply them.");
  }

  /* ---------------- start ---------------- */

  wireTheme();
  wireDocCard();
  $("caseForm").addEventListener("submit", submitCase);
  $("settingsForm").addEventListener("submit", submitSettings);
  $("cancelEdit").addEventListener("click", stopEdit);
  $("clearForm").addEventListener("click", stopEdit);
  $("resetPeriods").addEventListener("click", resetPeriods);
  $("asOf").addEventListener("change", function () { load(); });
  PERIOD_FIELDS.forEach(function (key) {
    $(key).addEventListener("input", function () { if (state) paintPeriodState(); });
  });

  load();
})();
