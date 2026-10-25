/** Position labels in SVG canvas coordinates, independent of zoom and pan. */
export function routeMidpoint(points = []) {
  if (!points.length) return { x: 0, y: 0 };
  const lengths = points.slice(1).map((point, index) =>
    Math.hypot(point.x - points[index].x, point.y - points[index].y),
  );
  let remaining = lengths.reduce((sum, length) => sum + length, 0) / 2;
  for (let index = 0; index < lengths.length; index += 1) {
    const length = lengths[index];
    if (length === 0) continue;
    if (remaining <= length) {
      const start = points[index];
      const end = points[index + 1];
      const fraction = remaining / length;
      return {
        x: start.x + (end.x - start.x) * fraction,
        y: start.y + (end.y - start.y) * fraction,
      };
    }
    remaining -= length;
  }
  // A one-point route, or a route consisting entirely of repeated points.
  return { x: points[0].x, y: points[0].y };
}

export function flowLabelPosition(points, offset) {
  const midpoint = routeMidpoint(points);
  return {
    x: midpoint.x + (offset?.x ?? 0),
    y: midpoint.y + (offset?.y ?? 0),
  };
}

export function labelOffsetForDrag(beforeOffset, startPoint, currentPoint) {
  const rounded = (value) => {
    const clamped = Math.max(-10000, Math.min(10000, value));
    // Avoid serializing a negative zero after a very small movement.
    return Math.round(clamped * 10) / 10 || 0;
  };
  return {
    x: rounded((beforeOffset?.x ?? 0) + currentPoint.x - startPoint.x),
    y: rounded((beforeOffset?.y ?? 0) + currentPoint.y - startPoint.y),
  };
}
