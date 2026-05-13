function ApplicationView() {
  return View(
    {
      class: "reader-device",
    },
    [
      View(
        {
          class: "screen-container",
        },
        [
          View(
            {
              class: "header velo-drag",
            },
            [
              View(
                {
                  class: "book-title",
                },
                ["侠客行"],
              ),
              View(
                {
                  style: {
                    display: "flex",
                    "align-items": "center",
                    gap: "12px",
                  },
                },
                [
                  Button(
                    {
                      class: "menu-btn",
                    },
                    ["打开文件"],
                  ),
                  FilePicker({ class: "file-input", accept: ".txt" }),
                  View(
                    {
                      class: "battery",
                    },
                    [
                      View(
                        {
                          class: "battery-icon",
                        },
                        [
                          View({
                            class: "battery-level",
                          }),
                        ],
                      ),
                    ],
                  ),
                ],
              ),
            ],
          ),
          View(
            {
              class: "content-area",
            },
            [
              View(
                {
                  class: "chapter-block",
                },
                [
                  View(
                    {
                      class: "chapter-title",
                    },
                    ["Title"],
                  ),
                ],
              ),
            ],
          ),
        ],
      ),
    ],
  );
}

// function render($root) {
//   // --- DOM structure ---
//   var device = document.createElement("div");
//   device.className = "reader-device";

//   var screen = document.createElement("div");
//   screen.className = "screen-container";

//   var loadingOverlay = document.createElement("div");
//   loadingOverlay.className = "loading-overlay";
//   loadingOverlay.innerHTML = '<span class="loading-text">加载中...</span>';

//   var header = document.createElement("div");
//   header.className = "header velo-drag";

//   var bookTitle = document.createElement("span");
//   bookTitle.className = "book-title";
//   bookTitle.textContent = "加载中...";

//   var headerRight = document.createElement("div");
//   headerRight.style.cssText = "display:flex;align-items:center;gap:12px;";

//   var openFileBtn = document.createElement("button");
//   openFileBtn.className = "menu-btn";
//   openFileBtn.textContent = "打开文件";

//   var fileInput = document.createElement("input");
//   fileInput.type = "file";
//   fileInput.className = "file-input";
//   fileInput.accept = ".txt";

//   var battery = document.createElement("div");
//   battery.className = "battery";
//   battery.innerHTML =
//     '<div class="battery-icon"><div class="battery-level"></div></div>';

//   headerRight.appendChild(openFileBtn);
//   headerRight.appendChild(fileInput);
//   headerRight.appendChild(battery);

//   header.appendChild(bookTitle);
//   header.appendChild(headerRight);

//   var contentArea = document.createElement("div");
//   contentArea.className = "content-area";

//   var pagesContainer = document.createElement("div");
//   contentArea.appendChild(pagesContainer);

//   screen.appendChild(loadingOverlay);
//   screen.appendChild(header);
//   screen.appendChild(contentArea);
//   device.appendChild(screen);
//   $root.appendChild(device);

//   // --- Logic ---
//   function showLoading(show) {
//     if (show) {
//       loadingOverlay.classList.add("active");
//     } else {
//       loadingOverlay.classList.remove("active");
//     }
//   }

// }

function parseBookContent(text) {
  var lines = text
    .split("\n")
    .map(function (line) {
      return line.trim();
    })
    .filter(function (line) {
      return line.length > 0;
    });
  var chapters = [];
  var currentChapter = { title: "内容", paragraphs: [] };

  var chapterPattern =
    /^={10,}\s*第([一二三四五六七八九十百千万\d]+)[回卷部章节]\s*(.*?)\s*={10,}$/;

  for (var i = 0; i < lines.length; i++) {
    var line = lines[i];
    var match = line.match(chapterPattern);
    if (match) {
      var chapterNum = match[1];
      var chapterTitle = match[2].trim();
      var fullTitle = "第" + chapterNum + "章 " + chapterTitle;

      if (currentChapter.paragraphs.length > 0) {
        chapters.push(currentChapter);
      }
      currentChapter = { title: fullTitle, paragraphs: [] };
    } else {
      currentChapter.paragraphs.push(line);
    }
  }

  if (currentChapter.paragraphs.length > 0) {
    chapters.push(currentChapter);
  }

  if (chapters.length === 0) {
    chapters.push({ title: "内容", paragraphs: lines });
  }

  return chapters;
}

function formatContent(text) {
  var paragraphs = text.split("\n\n").filter(function (p) {
    return p.trim();
  });
  return paragraphs
    .map(function (p) {
      return "<p>" + p.replace(/\n/g, "<br>") + "</p>";
    })
    .join("");
}

function loadBook(content, title) {
  showLoading(true);

  setTimeout(function () {
    var chapters = parseBookContent(content);

    bookTitle.textContent = title;
    pagesContainer.innerHTML = "";

    chapters.forEach(function (chapter) {
      var chapterDiv = document.createElement("div");
      chapterDiv.className = "chapter-block";

      var html = '<h2 class="chapter-title">' + chapter.title + "</h2>";
      html += formatContent(chapter.paragraphs.join("\n\n"));

      chapterDiv.innerHTML = html;
      pagesContainer.appendChild(chapterDiv);
    });

    showLoading(false);
  }, 100);
}

function loadFromFile(file) {
  var reader = new FileReader();
  reader.onload = function (e) {
    var fileName = file.name.replace(".txt", "");
    loadBook(e.target.result, fileName);
  };
  reader.readAsText(file);
}

function loadDefaultBook() {
  showLoading(true);
  fetch("全集_格式化/金庸-侠客行txt全本精校版.txt")
    .then(function (response) {
      if (!response.ok) throw new Error("Failed to load");
      return response.text();
    })
    .then(function (text) {
      loadBook(text, "侠客行");
    })
    .catch(function (error) {
      console.error("Failed to load default book:", error);
      bookTitle.textContent = "点击打开文件";
      showLoading(false);
      pagesContainer.innerHTML =
        '<div class="chapter-block"><p style="text-align:center;color:var(--ink-secondary);font-size:14px;margin-top:100px;">请点击右上角「打开文件」按钮选择 txt 文件</p></div>';
    });
}

// openFileBtn.addEventListener("click", function () {
//   fileInput.click();
// });

// fileInput.addEventListener("change", function (e) {
//   if (e.target.files.length > 0) {
//     loadFromFile(e.target.files[0]);
//     e.target.value = "";
//   }
// });

//   loadDefaultBook();

document.addEventListener("DOMContentLoaded", function () {
  var $root = document.getElementById("root");
  if (!$root) {
    console.error("[Render] Root element not found");
    return;
  }
  // render($root);
  Timeless.DOM.render(ApplicationView(), $root);
});
