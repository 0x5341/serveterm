import { describe, expect, test } from "vitest";
import {
  isThemeInputValid,
  parseThemeInput,
  readThemeCookie,
  themeCookieName,
  writeThemeCookie,
} from "./theme-cookie";

describe("theme-cookie", () => {
  test("cookie名はserveterm- prefixを持つ", () => {
    expect(themeCookieName.startsWith("serveterm-")).toBe(true);
  });

  test("theme cookieを読み書きできる", () => {
    const sink = { cookie: "" };
    const raw = "palette = 0=#111111\nbackground = #111111\nforeground = #eeeeee";

    writeThemeCookie(raw, sink);

    expect(sink.cookie).toContain(`${themeCookieName}=`);
    expect(readThemeCookie(sink.cookie)).toBe(raw);
  });

  test("有効なghostty theme fileをparseできる", () => {
    expect(
      parseThemeInput(
        [
          "# sample",
          "palette = 0=#101010",
          "palette = 1=#ff0000",
          "palette = 8=#808080",
          "background = #000000",
          "foreground = #ffffff",
          "cursor-color = #00ff00",
          "cursor-text = #111111",
          "selection-background = #222222",
          "selection-foreground = #dddddd",
        ].join("\n"),
      ),
    ).toEqual({
      black: "#101010",
      red: "#ff0000",
      brightBlack: "#808080",
      background: "#000000",
      foreground: "#ffffff",
      cursor: "#00ff00",
      cursorAccent: "#111111",
      selectionBackground: "#222222",
      selectionForeground: "#dddddd",
    });
  });

  test("theme要素を含まない入力は無効として扱う", () => {
    expect(parseThemeInput("theme = Catppuccin Frappe")).toBeUndefined();
    expect(isThemeInputValid("theme = Catppuccin Frappe")).toBe(false);
  });

  test("空文字は有効として扱う", () => {
    expect(isThemeInputValid("")).toBe(true);
  });
});
