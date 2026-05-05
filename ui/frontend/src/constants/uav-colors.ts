export const UAV_COLORS = [
  "#ef4444",
  "#3b82f6",
  "#10b981",
  "#f59e0b",
  "#8b5cf6",
  "#ec4899",
  "#14b8a6",
  "#f97316",
];

export function getUavColor(idx: number) {
  return UAV_COLORS[idx % UAV_COLORS.length];
}
