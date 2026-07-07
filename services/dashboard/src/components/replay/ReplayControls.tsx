"use client";

interface ReplayControlsProps {
  playing: boolean;
  speed: number;
  onPlayPause: () => void;
  onStep: () => void;
  onSpeedChange: (speed: number) => void;
  disabled?: boolean;
}

export function ReplayControls({ playing, speed, onPlayPause, onStep, onSpeedChange, disabled }: ReplayControlsProps) {
  return (
    <div className="flex items-center gap-3">
      <button
        onClick={onPlayPause}
        disabled={disabled}
        className="px-4 py-2 rounded-lg bg-[#00d4ff] text-black font-medium text-sm hover:bg-[#00d4ff]/80 transition-colors disabled:opacity-40"
      >
        {playing ? "Pause" : "Play"}
      </button>
      <button
        onClick={onStep}
        disabled={disabled || playing}
        className="px-4 py-2 rounded-lg bg-white/10 text-white font-medium text-sm hover:bg-white/20 transition-colors disabled:opacity-40"
      >
        Step →
      </button>
      <div className="flex items-center gap-2">
        <span className="text-xs text-gray-400">Speed:</span>
        {[1, 2, 5, 10].map((s) => (
          <button
            key={s}
            onClick={() => onSpeedChange(s)}
            className={`px-2 py-1 rounded text-xs font-medium transition-colors ${
              speed === s
                ? "bg-[#00d4ff] text-black"
                : "bg-white/10 text-gray-400 hover:text-white"
            }`}
          >
            {s}x
          </button>
        ))}
      </div>
    </div>
  );
}
