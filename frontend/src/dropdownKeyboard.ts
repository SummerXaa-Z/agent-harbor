export type DropdownKeyAction = "close" | "ignore" | "move" | "open" | "select";

export function dropdownKeyAction(key: string, open: boolean): DropdownKeyAction {
  if (key === "Escape") return "close";
  if (key === "Enter" || key === " ") return open ? "select" : "open";
  if (key === "ArrowDown" || key === "ArrowUp" || key === "Home" || key === "End") {
    return open ? "move" : "open";
  }
  return "ignore";
}

export function nextDropdownActiveIndex(currentIndex: number, optionCount: number, key: string): number {
  if (optionCount <= 0) return -1;
  if (key === "Home") return 0;
  if (key === "End") return optionCount - 1;
  if (key === "ArrowUp") {
    return currentIndex <= 0 ? optionCount - 1 : currentIndex - 1;
  }
  if (key === "ArrowDown") {
    return currentIndex < 0 || currentIndex >= optionCount - 1 ? 0 : currentIndex + 1;
  }
  return currentIndex < 0 ? 0 : currentIndex;
}
