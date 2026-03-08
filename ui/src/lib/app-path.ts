const relativeBaseURL = "./";

function trimTrailingSlash(pathname: string): string {
  return pathname.replace(/\/+$/, "") || "/";
}

export function resolveAppBaseURL(currentHref: string, baseURL = relativeBaseURL): URL {
  return new URL(baseURL, currentHref);
}

export function resolveAppPath(currentHref: string, relativePath: string, baseURL = relativeBaseURL): string {
  return trimTrailingSlash(new URL(relativePath, resolveAppBaseURL(currentHref, baseURL)).pathname);
}

export function isCurrentAppPath(
  currentHref: string,
  relativePath: string,
  baseURL = relativeBaseURL,
): boolean {
  return trimTrailingSlash(new URL(currentHref).pathname) === resolveAppPath(currentHref, relativePath, baseURL);
}
