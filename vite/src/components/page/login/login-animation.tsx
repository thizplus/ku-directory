export function LoginAnimation() {
  return (
    <div className="flex h-full w-full flex-col items-center justify-center p-8 bg-primary overflow-hidden">
      {/* Multiple animated blobs */}
      <div className="absolute inset-0">
        <div className="absolute -top-20 -left-20 w-[500px] h-[500px] bg-primary-foreground/20 rounded-full filter blur-3xl animate-pulse" />
        <div
          className="absolute top-20 -right-32 w-[600px] h-[600px] bg-primary-foreground/15 rounded-full filter blur-3xl animate-pulse"
          style={{ animationDelay: '1s' }}
        />
        <div
          className="absolute -bottom-32 left-1/4 w-[550px] h-[550px] bg-primary-foreground/15 rounded-full filter blur-3xl animate-pulse"
          style={{ animationDelay: '2s' }}
        />
        <div
          className="absolute bottom-20 -right-20 w-[450px] h-[450px] bg-primary-foreground/20 rounded-full filter blur-3xl animate-pulse"
          style={{ animationDelay: '3s' }}
        />
      </div>

      {/* Content */}
      <div className="relative z-10 text-center">
        <div className="text-7xl mb-6">📸</div>
        <h2 className="text-4xl font-bold text-primary-foreground">KU DIRECTORY</h2>
        <p className="mt-4 text-xl text-primary-foreground/80">
          University Photo Directory System
        </p>
        <p className="mt-2 text-primary-foreground/60">
          ค้นหารูปภาพกิจกรรมด้วยชื่อหรือใบหน้า
        </p>
      </div>
    </div>
  );
}
