type PingColor = "bg-green-600" | "bg-red-600";

interface IProps {
  color: PingColor;
}

// https://tailwindcss.com/docs/animation#ping
export function Ping({ color }: IProps) {
  return (
    <span className="flex absolute h-3 w-3 -mt-1 -mr-1">
      <span
        className={`animate-ping absolute inline-flex h-full w-full rounded-full ${color} opacity-75`}
      ></span>
      <span
        className={`relative inline-flex rounded-full h-3 w-3 ${color}`}
      ></span>
    </span>
  );
}
