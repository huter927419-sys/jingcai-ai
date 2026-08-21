type IconProps = { size?: number; className?: string };

function Svg({ size = 16, className, children }: IconProps & { children: React.ReactNode }) {
  return (
    <svg
      className={className}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      {children}
    </svg>
  );
}

export function IconLogo({ size = 22 }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden>
      <rect width="24" height="24" rx="3" fill="#eef9f5" stroke="#0b9668" strokeWidth="1.2" />
      <circle cx="12" cy="12" r="6.2" fill="none" stroke="#0b9668" strokeWidth="1.4" />
      <path d="M12 6.2v11.6M6.2 12h11.6" stroke="#087a55" strokeWidth="1.4" />
      <path d="M8.2 8.2c2.4 1.6 5.2 1.6 7.6 0M8.2 15.8c2.4-1.6 5.2-1.6 7.6 0" stroke="#3b82f6" strokeWidth="1.2" />
    </svg>
  );
}

export const IconBall = (p: IconProps) => (
  <Svg {...p}>
    <circle cx="12" cy="12" r="9" />
    <path d="M12 3c2 3 2 15 0 18M3 12c3-2 15-2 18 0" />
    <path d="M7.2 6.2c3 1.4 6.6 1.4 9.6 0M7.2 17.8c3-1.4 6.6-1.4 9.6 0" />
  </Svg>
);

export const IconClock = (p: IconProps) => (
  <Svg {...p}>
    <circle cx="12" cy="12" r="9" />
    <path d="M12 7v5l3 2" />
  </Svg>
);

export const IconChart = (p: IconProps) => (
  <Svg {...p}>
    <path d="M4 19h16" />
    <path d="M7 16V9M12 16V6M17 16v-4" />
  </Svg>
);

export const IconScale = (p: IconProps) => (
  <Svg {...p}>
    <path d="M12 4v3" />
    <path d="M5 10h14" />
    <path d="M7 10l-3 6h6l-3-6M17 10l-3 6h6l-3-6" />
    <path d="M12 7v13" />
  </Svg>
);

export const IconGauge = (p: IconProps) => (
  <Svg {...p}>
    <path d="M5.5 17a8 8 0 1 1 13 0" />
    <path d="M12 13l4-3" />
    <circle cx="12" cy="13" r="1.2" fill="currentColor" stroke="none" />
  </Svg>
);

export const IconGoals = (p: IconProps) => (
  <Svg {...p}>
    <rect x="3" y="6" width="18" height="12" rx="2" />
    <path d="M8 6v12M16 6v12M3 12h18" />
  </Svg>
);

export const IconScore = (p: IconProps) => (
  <Svg {...p}>
    <rect x="3" y="5" width="18" height="14" rx="2" />
    <path d="M8 9v6M12 9v6M16 9v6" />
  </Svg>
);

export const IconBack = (p: IconProps) => (
  <Svg {...p}>
    <path d="M15 5l-7 7 7 7" />
  </Svg>
);

export const IconTalk = (p: IconProps) => (
  <Svg {...p}>
    <path d="M5 6h14v9H8l-3 3V6z" />
  </Svg>
);

export const IconGrid = (p: IconProps) => (
  <Svg {...p}>
    <rect x="4" y="4" width="7" height="7" />
    <rect x="13" y="4" width="7" height="7" />
    <rect x="4" y="13" width="7" height="7" />
    <rect x="13" y="13" width="7" height="7" />
  </Svg>
);

export const IconShield = (p: IconProps) => (
  <Svg {...p}>
    <path d="M12 3l8 3v6c0 5-3.4 8.4-8 9.5C7.4 20.4 4 17 4 12V6l8-3z" />
  </Svg>
);

export const IconPulse = (p: IconProps) => (
  <Svg {...p}>
    <path d="M3 12h4l2.2-5 4.1 10 2.2-5H21" />
  </Svg>
);

export const IconSpark = (p: IconProps) => (
  <Svg {...p}>
    <path d="M12 3l1.4 4.1L17.5 8.5l-4.1 1.4L12 14l-1.4-4.1-4.1-1.4 4.1-1.4L12 3z" />
    <path d="M18.5 14.5l.7 2.1 2.1.7-2.1.7-.7 2.1-.7-2.1-2.1-.7 2.1-.7.7-2.1z" />
  </Svg>
);
