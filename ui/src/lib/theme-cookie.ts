export type GhosttyTheme = Partial<{
  background: string;
  foreground: string;
  cursor: string;
  cursorAccent: string;
  selectionBackground: string;
  selectionForeground: string;
  black: string;
  red: string;
  green: string;
  yellow: string;
  blue: string;
  magenta: string;
  cyan: string;
  white: string;
  brightBlack: string;
  brightRed: string;
  brightGreen: string;
  brightYellow: string;
  brightBlue: string;
  brightMagenta: string;
  brightCyan: string;
  brightWhite: string;
}>;

export const themeCookiePrefix = "serveterm-";
export const themeCookieName = `${themeCookiePrefix}theme`;

const oneYearSeconds = 60 * 60 * 24 * 365;
const directThemeKeyMap: Record<string, keyof GhosttyTheme> = {
  background: "background",
  foreground: "foreground",
  "cursor-color": "cursor",
  "cursor-text": "cursorAccent",
  "selection-background": "selectionBackground",
  "selection-foreground": "selectionForeground",
};
const paletteThemeKeyMap: Array<keyof GhosttyTheme> = [
  "black",
  "red",
  "green",
  "yellow",
  "blue",
  "magenta",
  "cyan",
  "white",
  "brightBlack",
  "brightRed",
  "brightGreen",
  "brightYellow",
  "brightBlue",
  "brightMagenta",
  "brightCyan",
  "brightWhite",
];

type CookieTarget = {
  cookie: string;
};

export function readThemeCookie(cookieSource: string = document.cookie): string {
  const key = `${themeCookieName}=`;
  for (const part of cookieSource.split(";")) {
    const token = part.trim();
    if (!token.startsWith(key)) {
      continue;
    }
    const value = token.slice(key.length);
    try {
      return decodeURIComponent(value);
    } catch {
      return "";
    }
  }
  return "";
}

export function writeThemeCookie(rawThemeInput: string, target: CookieTarget = document): void {
  const normalized = rawThemeInput.trim();
  if (normalized === "") {
    target.cookie = `${themeCookieName}=; Path=/; Max-Age=0; SameSite=Lax`;
    return;
  }
  target.cookie = `${themeCookieName}=${encodeURIComponent(normalized)}; Path=/; Max-Age=${oneYearSeconds}; SameSite=Lax`;
}

function trimConfigValue(value: string): string {
  const trimmed = value.trim();
  if (
    (trimmed.startsWith('"') && trimmed.endsWith('"')) ||
    (trimmed.startsWith("'") && trimmed.endsWith("'"))
  ) {
    return trimmed.slice(1, -1).trim();
  }
  return trimmed;
}

function normalizeColorValue(value: string): string {
  if (/^[0-9a-fA-F]{6}$/.test(value)) {
    return `#${value}`;
  }
  return value;
}

function assignThemeColor(theme: GhosttyTheme, key: keyof GhosttyTheme, rawValue: string): void {
  const value = normalizeColorValue(trimConfigValue(rawValue));
  if (value === "") {
    return;
  }
  theme[key] = value;
}

export function parseThemeInput(rawThemeInput: string): GhosttyTheme | undefined {
  if (rawThemeInput.trim() === "") {
    return undefined;
  }

  const theme: GhosttyTheme = {};
  for (const line of rawThemeInput.split(/\r?\n/)) {
    const trimmedLine = line.trim();
    if (trimmedLine === "" || trimmedLine.startsWith("#")) {
      continue;
    }

    const separatorIndex = trimmedLine.indexOf("=");
    if (separatorIndex <= 0) {
      continue;
    }

    const configKey = trimmedLine.slice(0, separatorIndex).trim().toLowerCase();
    const configValue = trimmedLine.slice(separatorIndex + 1).trim();

    if (configKey === "palette") {
      const paletteEntry = trimConfigValue(configValue);
      const paletteMatch = paletteEntry.match(/^(\d+)\s*=\s*(.+)$/);
      if (!paletteMatch) {
        continue;
      }
      const paletteIndex = Number(paletteMatch[1]);
      const themeKey = paletteThemeKeyMap[paletteIndex];
      if (!themeKey) {
        continue;
      }
      assignThemeColor(theme, themeKey, paletteMatch[2]);
      continue;
    }

    const themeKey = directThemeKeyMap[configKey];
    if (!themeKey) {
      continue;
    }
    assignThemeColor(theme, themeKey, configValue);
  }

  return Object.keys(theme).length > 0 ? theme : undefined;
}

export function isThemeInputValid(rawThemeInput: string): boolean {
  return rawThemeInput.trim() === "" || parseThemeInput(rawThemeInput) !== undefined;
}
