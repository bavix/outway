import { useEffect, useRef } from 'preact/hooks';

interface ChartProps {
  /** Chart title */
  title?: string;
  /** Chart data */
  data?: {
    labels: string[];
    datasets: {
      label: string;
      data: number[];
      borderColor?: string;
      backgroundColor?: string;
      fill?: boolean;
    }[];
  };
  /** Chart type */
  type?: 'line' | 'bar' | 'doughnut';
  /** Chart height */
  height?: number;
  /** Loading state */
  loading?: boolean;
  /** Error state */
  error?: string;
  /** Additional CSS classes */
  className?: string;
  /** Axis labels */
  xLabel?: string;
  yLabel?: string;
}

export function Chart({ 
  title,
  data,
  type = 'line',
  height = 300,
  loading = false,
  error,
  className = '',
  xLabel,
  yLabel,
}: ChartProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    if (!canvasRef.current || !data) return;

    // Simple chart implementation without external dependencies
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const padding = 56;
    const chartWidth = canvas.width - padding * 2;
    const chartHeight = canvas.height - padding * 2;

    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    if (type === 'line') {
      drawLineChart(ctx, data, padding, chartWidth, chartHeight, xLabel, yLabel);
    } else if (type === 'bar') {
      drawBarChart(ctx, data, padding, chartWidth, chartHeight, xLabel, yLabel);
    } else if (type === 'doughnut') {
      drawDoughnutChart(ctx, data, canvas.width / 2, canvas.height / 2, Math.min(canvas.width, canvas.height) / 3);
    }
  }, [data, type]);

  const drawLineChart = (ctx: CanvasRenderingContext2D, data: any, padding: number, width: number, height: number, xLabel?: string, yLabel?: string) => {
    const dataset = data.datasets[0];
    if (!dataset) return;

    const maxValue = Math.max(...dataset.data);
    const minValue = Math.min(...dataset.data);
    const valueRange = maxValue - minValue || 1;

    // Draw grid lines
    ctx.strokeStyle = '#e5e7eb';
    ctx.lineWidth = 1;
    for (let i = 0; i <= 5; i++) {
      const y = padding + (height / 5) * i;
      ctx.beginPath();
      ctx.moveTo(padding, y);
      ctx.lineTo(padding + width, y);
      ctx.stroke();
    }

    // Axis labels and ticks
    ctx.fillStyle = '#6b7280';
    ctx.font = '12px system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial';
    // Y ticks
    ctx.textAlign = 'right';
    for (let i = 0; i <= 5; i++) {
      const y = padding + (height / 5) * i;
      const val = maxValue - (valueRange / 5) * i;
      const text = Number.isFinite(val) ? (val >= 100 ? val.toFixed(0) : val.toFixed(1)) : '0';
      ctx.fillText(text, padding - 8, y + 4);
    }
    if (yLabel) {
      ctx.save();
      ctx.translate(16, padding + height / 2);
      ctx.rotate(-Math.PI / 2);
      ctx.textAlign = 'center';
      ctx.fillText(yLabel, 0, 0);
      ctx.restore();
    }
    // X ticks
    ctx.textAlign = 'center';
    const n = dataset.data.length;
    const tickCount = Math.min(6, Math.max(2, n));
    const step = Math.max(1, Math.floor((n - 1) / (tickCount - 1)));
    for (let i = 0; i < n; i += step) {
      const x = padding + (width / (n - 1)) * i;
      const label = data.labels?.[i] || '';
      if (label) ctx.fillText(label, x, padding + height + 18);
    }
    if (xLabel) ctx.fillText(xLabel, padding + width / 2, padding + height + 34);

    // Draw line
    ctx.strokeStyle = '#3b82f6';
    ctx.lineWidth = 2;
    ctx.beginPath();

    dataset.data.forEach((value: number, index: number) => {
      const x = padding + (width / (dataset.data.length - 1)) * index;
      const y = padding + height - ((value - minValue) / valueRange) * height;
      
      if (index === 0) {
        ctx.moveTo(x, y);
      } else {
        ctx.lineTo(x, y);
      }
    });

    ctx.stroke();

    // Draw points
    ctx.fillStyle = '#3b82f6';
    dataset.data.forEach((value: number, index: number) => {
      const x = padding + (width / (dataset.data.length - 1)) * index;
      const y = padding + height - ((value - minValue) / valueRange) * height;
      
      ctx.beginPath();
      ctx.arc(x, y, 4, 0, 2 * Math.PI);
      ctx.fill();
    });
  };

  const drawBarChart = (ctx: CanvasRenderingContext2D, data: any, padding: number, width: number, height: number, xLabel?: string, yLabel?: string) => {
    const dataset = data.datasets[0];
    if (!dataset) return;

    const maxValue = Math.max(...dataset.data);
    const barWidth = width / dataset.data.length * 0.8;
    const barSpacing = width / dataset.data.length * 0.2;

    ctx.fillStyle = '#3b82f6';
    
    dataset.data.forEach((value: number, index: number) => {
      const barHeight = (value / maxValue) * height;
      const x = padding + index * (barWidth + barSpacing) + barSpacing / 2;
      const y = padding + height - barHeight;
      
      ctx.fillRect(x, y, barWidth, barHeight);
    });

    // Axis labels
    ctx.fillStyle = '#6b7280';
    ctx.font = '12px system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial';
    // Y ticks
    ctx.textAlign = 'right';
    for (let i = 0; i <= 5; i++) {
      const y = padding + (height / 5) * i;
      const val = maxValue - (maxValue / 5) * i;
      const text = Number.isFinite(val) ? (val >= 100 ? val.toFixed(0) : val.toFixed(1)) : '0';
      ctx.fillText(text, padding - 8, y + 4);
    }
    if (yLabel) {
      ctx.save();
      ctx.translate(16, padding + height / 2);
      ctx.rotate(-Math.PI / 2);
      ctx.textAlign = 'center';
      ctx.fillText(yLabel, 0, 0);
      ctx.restore();
    }
    // X ticks
    ctx.textAlign = 'center';
    const n = dataset.data.length;
    const tickCount = Math.min(6, Math.max(2, n));
    const step = Math.max(1, Math.floor((n - 1) / (tickCount - 1)));
    for (let i = 0; i < n; i += step) {
      const x = padding + i * (barWidth + barSpacing) + (barSpacing / 2) + (barWidth / 2);
      const label = data.labels?.[i] || '';
      if (label) ctx.fillText(label, x, padding + height + 18);
    }
    if (xLabel) ctx.fillText(xLabel, padding + width / 2, padding + height + 34);
  };

  const drawDoughnutChart = (ctx: CanvasRenderingContext2D, data: any, centerX: number, centerY: number, radius: number) => {
    const dataset = data.datasets[0];
    if (!dataset) return;

    const total = dataset.data.reduce((sum: number, value: number) => sum + value, 0);
    const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'];
    
    let currentAngle = 0;
    
    dataset.data.forEach((value: number, index: number) => {
      const sliceAngle = (value / total) * 2 * Math.PI;
      
      ctx.beginPath();
      ctx.arc(centerX, centerY, radius, currentAngle, currentAngle + sliceAngle);
      ctx.arc(centerX, centerY, radius * 0.6, currentAngle + sliceAngle, currentAngle, true);
      ctx.closePath();
      
      ctx.fillStyle = colors[index % colors.length] as string;
      ctx.fill();
      
      currentAngle += sliceAngle;
    });
  };

  if (loading) {
    return (
      <div className={`flex items-center justify-center ${className}`} style={{ height }}>
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-2"></div>
          <p className="text-sm text-gray-500 dark:text-gray-400">Loading chart...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`flex items-center justify-center ${className}`} style={{ height }}>
        <div className="text-center">
          <svg className="w-12 h-12 mx-auto mb-4 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.732-.833-2.5 0L4.268 16.5c-.77.833.192 2.5 1.732 2.5z" />
          </svg>
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className={`chart-container ${className}`}>
      {title && (
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
          {title}
        </h3>
      )}
      <div className="relative">
        <canvas
          ref={canvasRef}
          width={800}
          height={height}
          className="w-full h-auto"
        />
      </div>
    </div>
  );
}
