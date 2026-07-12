import { createHighlighter } from "https://esm.sh/shiki@3.17.1";

const grammarUrl = new URL("../assets/languages/drun.tmLanguage.json", import.meta.url);
const ebnfGrammarUrl = new URL("../assets/languages/ebnf.tmLanguage.json", import.meta.url);
const languageAliases = {
  sh: "bash",
  shell: "bash",
  yml: "yaml",
  plaintext: "text",
  txt: "text",
};
const bundledLanguages = ["bash", "go", "json", "makefile", "text", "yaml"];

const highlighterPromise = Promise.all([
  fetch(grammarUrl).then((response) => {
    if (!response.ok) {
      throw new Error(`Unable to load the drun grammar: ${response.status}`);
    }
    return response.json();
  }),
  fetch(ebnfGrammarUrl).then((response) => {
    if (!response.ok) {
      throw new Error(`Unable to load the EBNF grammar: ${response.status}`);
    }
    return response.json();
  }),
]).then(([grammar, ebnfGrammar]) =>
  createHighlighter({
    langs: [
      { ...grammar, name: "drun" },
      { ...ebnfGrammar, name: "ebnf" },
      ...bundledLanguages,
    ],
    themes: ["github-dark"],
  }),
);

function createCopyButton(source, language) {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "code-copy";
  button.setAttribute("aria-label", `Copy ${language} code to clipboard`);
  button.innerHTML = `
    <svg aria-hidden="true" viewBox="0 0 24 24">
      <path d="M16 1H4a2 2 0 0 0-2 2v14h2V3h12V1Zm3 4H8a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2Zm0 16H8V7h11v14Z" />
    </svg>
    <span>Copy</span>
  `;

  button.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(source);
      button.classList.add("is-copied");
      button.querySelector("span").textContent = "Copied";
      window.setTimeout(() => {
        button.classList.remove("is-copied");
        button.querySelector("span").textContent = "Copy";
      }, 5000);
    } catch (error) {
      console.error(`Unable to copy ${language} code`, error);
    }
  });

  return button;
}

function getLanguage(block) {
  const languageClass = [...block.classList].find((name) => name.startsWith("language-"));
  const language = languageClass?.slice("language-".length);
  return languageAliases[language] ?? language;
}

async function highlightCode(root = document) {
  const blocks = root.querySelectorAll('pre[class*="language-"]:not(.mermaid):not([data-shiki])');
  if (!blocks.length) return;

  try {
    const highlighter = await highlighterPromise;

    for (const block of blocks) {
      const language = getLanguage(block);
      if (!language) continue;

      const source = block.querySelector("code")?.textContent ?? block.textContent ?? "";
      const wrapper = document.createElement("div");
      try {
        wrapper.innerHTML = highlighter.codeToHtml(source.replace(/\n$/, ""), {
          lang: language,
          theme: "github-dark",
        });
      } catch (error) {
        console.warn(`No Shiki language registered for ${language}`, error);
        continue;
      }

      const highlighted = wrapper.firstElementChild;
      if (!highlighted) continue;

      highlighted.classList.add(`language-${language}`);
      highlighted.dataset.shiki = "true";

      const container = document.createElement("div");
      container.className = "code-block";

      const toolbar = document.createElement("div");
      toolbar.className = "code-toolbar";
      toolbar.append(createCopyButton(source, language));

      container.append(toolbar, highlighted);
      block.replaceWith(container);
    }
  } catch (error) {
    console.error("Unable to initialize syntax highlighting", error);
  }
}

document$.subscribe(() => highlightCode());
