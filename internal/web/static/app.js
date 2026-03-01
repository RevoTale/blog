(() => {
  const copyResetDelayMs = 2000;
  const copyTimers = new WeakMap();

  const fallbackCopyText = text => {
    const area = document.createElement("textarea");
    area.value = text;
    area.setAttribute("readonly", "");
    area.style.position = "fixed";
    area.style.left = "-9999px";
    area.style.opacity = "0";
    document.body.appendChild(area);
    area.focus();
    area.select();
    area.setSelectionRange(0, area.value.length);

    try {
      return document.execCommand("copy");
    } catch (_) {
      return false;
    } finally {
      document.body.removeChild(area);
    }
  };

  const copyText = async text => {
    if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
      try {
        await navigator.clipboard.writeText(text);
        return true;
      } catch (_) {
        return fallbackCopyText(text);
      }
    }

    return fallbackCopyText(text);
  };

  const setCopyButtonState = (button, state) => {
    const label =
      state === "copied"
        ? button.dataset.copiedLabel || "copied"
        : button.dataset.copyLabel || "copy";
    const labelNode = button.querySelector(".code-copy-button-label");
    if (labelNode) {
      labelNode.textContent = label;
    }
    button.dataset.copyState = state;
  };

  document.addEventListener("click", async event => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }

    const button = target.closest(".code-copy-button");
    if (!(button instanceof HTMLButtonElement)) {
      return;
    }

    const codeBlock = button.closest(".code-block");
    if (!(codeBlock instanceof HTMLElement)) {
      return;
    }

    const source = codeBlock.querySelector(".code-copy-source");
    if (!(source instanceof HTMLTextAreaElement)) {
      return;
    }

    const copied = await copyText(source.value);
    if (!copied) {
      return;
    }

    setCopyButtonState(button, "copied");
    const previousTimer = copyTimers.get(button);
    if (typeof previousTimer === "number") {
      window.clearTimeout(previousTimer);
    }

    const timeoutID = window.setTimeout(() => {
      setCopyButtonState(button, "idle");
      copyTimers.delete(button);
    }, copyResetDelayMs);
    copyTimers.set(button, timeoutID);
  });

  document.addEventListener("htmx:afterSettle", event => {
    const detail = event && event.detail;
    const target = detail && detail.target;
    if (!(target instanceof HTMLElement)) {
      return;
    }
    if (target.id !== "notes-content") {
      return;
    }

    window.scrollTo({ top: 0, left: 0, behavior: "smooth" });
  });
})();
