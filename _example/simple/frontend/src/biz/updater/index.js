/**
 * @file 更新检查模块
 */
function callGo(method, args) {
  var goCall = window && window["goCall"];
  if (typeof goCall === "function") {
    return goCall(method, args);
  }
  return Promise.reject(new Error("go bridge not available"));
}

export function createCheckUpdateButton($root, $output) {
  var $btnCheckUpdate = document.createElement("button");
  $btnCheckUpdate.innerHTML = "Check Update";
  $btnCheckUpdate.style.marginLeft = "10px";
  $btnCheckUpdate.onclick = function () {
    $btnCheckUpdate.disabled = true;
    $btnCheckUpdate.innerHTML = "Checking...";
    $output.textContent = "🔍 Checking for updates...";

    callGo("CheckUpdate", {})
      .then(function (data) {
        $btnCheckUpdate.disabled = false;
        $btnCheckUpdate.innerHTML = "Check Update";

        if (data.error) {
          if (
            data.error.includes("development mode") ||
            data.error.includes("disabled")
          ) {
            $output.textContent = "ℹ️  " + data.error;
          } else {
            $output.textContent = "❌ Error: " + data.error;
          }
        } else if (data.hasUpdate) {
          showUpdateAvailable(data, $root, $output);
        } else {
          $output.textContent =
            "✅ You are running the latest version (" +
            data.currentVersion +
            ")";
        }
      })
      .catch(function (e) {
        $btnCheckUpdate.disabled = false;
        $btnCheckUpdate.innerHTML = "Check Update";
        $output.textContent = "❌ Error: " + e;
      });
  };

  return $btnCheckUpdate;
}

export function showUpdateAvailable(data, $root, $output) {
  $output.innerHTML = "";

  var $updateInfo = document.createElement("div");
  $updateInfo.style.padding = "15px";
  $updateInfo.style.backgroundColor = "var(--BG-1)";
  $updateInfo.style.borderRadius = "8px";
  $updateInfo.style.border = "2px solid var(--GREEN)";

  var $title = document.createElement("div");
  $title.style.fontSize = "18px";
  $title.style.fontWeight = "bold";
  $title.style.marginBottom = "10px";
  $title.style.color = "var(--GREEN)";
  $title.textContent = "🎉 New Version Available!";

  var $version = document.createElement("div");
  $version.style.marginBottom = "5px";
  $version.textContent = "New Version: " + data.version;

  var $current = document.createElement("div");
  $current.style.marginBottom = "10px";
  $current.style.color = "var(--FG-1)";
  $current.textContent = "Current Version: " + data.currentVersion;

  var $notes = document.createElement("div");
  $notes.style.marginBottom = "15px";
  $notes.style.padding = "10px";
  $notes.style.backgroundColor = "var(--BG-2)";
  $notes.style.borderRadius = "4px";
  $notes.style.maxHeight = "150px";
  $notes.style.overflow = "auto";
  $notes.style.whiteSpace = "pre-wrap";
  $notes.textContent =
    "Release Notes:\n" + (data.releaseNotes || "No release notes available");

  var $actions = document.createElement("div");
  $actions.style.display = "flex";
  $actions.style.gap = "10px";
  $actions.style.marginTop = "15px";

  var $downloadBtn = document.createElement("button");
  $downloadBtn.textContent = "Download Update";
  $downloadBtn.style.flex = "1";
  $downloadBtn.style.padding = "10px";
  $downloadBtn.style.backgroundColor = "var(--GREEN)";
  $downloadBtn.style.color = "white";
  $downloadBtn.style.border = "none";
  $downloadBtn.style.borderRadius = "4px";
  $downloadBtn.style.cursor = "pointer";
  $downloadBtn.onclick = function () {
    $downloadBtn.disabled = true;
    $downloadBtn.textContent = "Downloading...";

    callGo("DownloadUpdate", { version: data.version })
      .then(function (result) {
        if (result.success) {
          $output.innerHTML = "";
          var $successMsg = document.createElement("div");
          $successMsg.style.padding = "15px";
          $successMsg.style.backgroundColor = "var(--BG-1)";
          $successMsg.style.borderRadius = "8px";
          $successMsg.style.border = "2px solid var(--GREEN)";

          var $title = document.createElement("div");
          $title.style.fontSize = "18px";
          $title.style.fontWeight = "bold";
          $title.style.marginBottom = "10px";
          $title.style.color = "var(--GREEN)";
          $title.textContent =
            "✅ " + (result.message || "Update downloaded successfully!");

          var $info = document.createElement("div");
          $info.style.marginBottom = "15px";
          $info.style.color = "var(--FG-1)";
          $info.innerHTML =
            "Update path: " +
            (result.updatePath || "N/A") +
            "<br><br>Click the button below to restart and apply the update.";

          var $restartBtn = document.createElement("button");
          $restartBtn.textContent = "Restart Now";
          $restartBtn.style.width = "100%";
          $restartBtn.style.padding = "12px";
          $restartBtn.style.backgroundColor = "var(--GREEN)";
          $restartBtn.style.color = "white";
          $restartBtn.style.border = "none";
          $restartBtn.style.borderRadius = "4px";
          $restartBtn.style.cursor = "pointer";
          $restartBtn.style.fontSize = "16px";
          $restartBtn.style.fontWeight = "bold";
          $restartBtn.onclick = function () {
            $restartBtn.disabled = true;
            $restartBtn.textContent = "Applying Update & Restarting...";

            var $logContainer = document.createElement("div");
            $logContainer.style.marginTop = "15px";
            $logContainer.style.padding = "10px";
            $logContainer.style.backgroundColor = "var(--BG-2)";
            $logContainer.style.borderRadius = "4px";
            $logContainer.style.border = "1px solid var(--FG-3)";
            $logContainer.style.maxHeight = "300px";
            $logContainer.style.overflow = "auto";

            var $logTitle = document.createElement("div");
            $logTitle.style.fontWeight = "bold";
            $logTitle.style.marginBottom = "10px";
            $logTitle.textContent = "📋 Update Progress Logs:";
            $logContainer.appendChild($logTitle);

            var $logContent = document.createElement("div");
            $logContent.style.fontFamily = "monospace";
            $logContent.style.fontSize = "12px";
            $logContent.style.whiteSpace = "pre-wrap";
            $logContent.textContent = "Starting update process...\n";
            $logContainer.appendChild($logContent);

            $restartBtn.parentNode.insertBefore($logContainer, $restartBtn.nextSibling);

            function fetchLogs() {
              callGo("GetUpdateLogs", {})
                .then(function (logResult) {
                  if (logResult.success && logResult.logs) {
                    var logText = "";
                    logResult.logs.forEach(function (entry) {
                      var timestamp = entry.time ? new Date(entry.time).toLocaleTimeString() : "";
                      var level = entry.level ? entry.level.toUpperCase() : "INFO";
                      var message = entry.message || "";
                      logText += "[" + timestamp + "] " + level + ": " + message + "\n";
                    });
                    $logContent.textContent = logText;
                    $logContent.scrollTop = $logContent.scrollHeight;
                  } else if (logResult.error) {
                    $logContent.textContent += "\n⚠️ Error fetching logs: " + logResult.error + "\n";
                  }
                })
                .catch(function (e) {
                  $logContent.textContent += "\n⚠️ Error fetching logs: " + e + "\n";
                });
            }

            callGo("RestartApp", {})
              .then(function (restartResult) {
                if (!restartResult.success) {
                  $restartBtn.disabled = false;
                  $restartBtn.textContent = "Restart Now";
                  $logContent.textContent += "\n❌ Failed to restart: " + (restartResult.error || "Unknown error") + "\n";
                  return;
                }
                $logContent.textContent += "\n✅ Restart command sent successfully\n";
              })
              .catch(function (e) {
                $restartBtn.disabled = false;
                $restartBtn.textContent = "Restart Now";
                $logContent.textContent += "\n❌ Error: " + e + "\n";
              });

            setTimeout(function () {
              clearInterval(logInterval);
              clearTimeout(logTimeout);

              if (document.visibilityState === "visible") {
                $logContent.textContent += "\n⚠️ App did not restart within 15 seconds, checking diagnostics...\n";

                callGo("RestartDiagnostics", {})
                  .then(function (diag) {
                    $restartBtn.disabled = false;
                    $restartBtn.textContent = "Restart Now";

                    var msg = "\n⚠️ Restart may have failed.\n";
                    if (diag && diag.success) {
                      msg += "- Exec: " + (diag.execPath || "N/A") + "\n";
                      msg += "- App Bundle: " + (diag.appBundle || "N/A") + "\n";
                      if (Array.isArray(diag.openArgs) && diag.openArgs.length) {
                        msg += "- open args: " + diag.openArgs.join(" ") + "\n";
                      }
                      if (Array.isArray(diag.notes) && diag.notes.length) {
                        msg += "- Notes: " + diag.notes.join("; ") + "\n";
                      }
                    } else {
                      msg += "- Error: " + (diag && diag.error ? diag.error : "Unknown error") + "\n";
                    }
                    $logContent.textContent += msg;
                  })
                  .catch(function (e) {
                    $restartBtn.disabled = false;
                    $restartBtn.textContent = "Restart Now";
                    $logContent.textContent += "\n❌ Diagnostics error: " + e + "\n";
                  });
              } else {
                $logContent.textContent += "\n✅ App appears to have restarted successfully!\n";
              }
            }, 15000);

            var logInterval;
            var logTimeout;
          };

          $successMsg.appendChild($title);
          $successMsg.appendChild($info);
          $successMsg.appendChild($restartBtn);
          $output.appendChild($successMsg);
        } else {
          $downloadBtn.disabled = false;
          $downloadBtn.textContent = "Download Update";
          alert(
            "❌ Failed to download update: " +
              (result.error || "Unknown error")
          );
        }
      })
      .catch(function (e) {
        $downloadBtn.disabled = false;
        $downloadBtn.textContent = "Download Update";
        alert("❌ Error: " + e);
      });
  };

  var $remindBtn = document.createElement("button");
  $remindBtn.textContent = "Remind Me Later";
  $remindBtn.style.flex = "1";
  $remindBtn.style.padding = "10px";
  $remindBtn.style.backgroundColor = "var(--BG-3)";
  $remindBtn.style.color = "var(--FG-0)";
  $remindBtn.style.border = "1px solid var(--FG-3)";
  $remindBtn.style.borderRadius = "4px";
  $remindBtn.style.cursor = "pointer";
  $remindBtn.onclick = function () {
    $remindBtn.disabled = true;
    $remindBtn.textContent = "Setting...";

    callGo("RemindLater", {})
      .then(function (result) {
        if (result.success) {
          $output.textContent =
            "⏰ " + (result.message || "You will be reminded later");
        } else {
          $output.textContent =
            "❌ Error: " + (result.error || "Unknown error");
        }
      })
      .catch(function (e) {
        $output.textContent = "❌ Error: " + e;
      });
  };

  var $skipBtn = document.createElement("button");
  $skipBtn.textContent = "Skip This Version";
  $skipBtn.style.flex = "1";
  $skipBtn.style.padding = "10px";
  $skipBtn.style.backgroundColor = "var(--BG-3)";
  $skipBtn.style.color = "var(--FG-1)";
  $skipBtn.style.border = "1px solid var(--FG-3)";
  $skipBtn.style.borderRadius = "4px";
  $skipBtn.style.cursor = "pointer";
  $skipBtn.onclick = function () {
    $skipBtn.disabled = true;
    $skipBtn.textContent = "Skipping...";

    callGo("SkipVersion", { version: data.version })
      .then(function (result) {
        if (result.success) {
          $output.textContent =
            "⏭️  Version " + data.version + " will be skipped";
        } else {
          $output.textContent =
            "❌ Failed to skip version: " + (result.error || "Unknown error");
        }
      })
      .catch(function (e) {
        $output.textContent = "❌ Error: " + e;
      });
  };

  $actions.appendChild($downloadBtn);
  $actions.appendChild($remindBtn);
  $actions.appendChild($skipBtn);

  $updateInfo.appendChild($title);
  $updateInfo.appendChild($version);
  $updateInfo.appendChild($current);
  $updateInfo.appendChild($notes);
  $updateInfo.appendChild($actions);

  $output.appendChild($updateInfo);
}
