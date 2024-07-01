interface Timer {
  timings: number[];
}

class Timers {
  timers: Record<string, Timer> = {};
  longestLength = 0;

  start(name: string): () => void {
    if (!this.timers[name]) {
      this.timers[name] = { timings: [] };
    }

    this.longestLength = Math.max(this.longestLength, name.length);

    const start = performance.now();
    return () => this.timers[name].timings.push(performance.now() - start);
  }

  formatTimestamp(millis: number): string {
    const fmt = (n: number, u: string) => `${n.toFixed(4).padStart(9, ' ')} ${u}`;

    if (millis > 1) {
      return fmt(millis, 'ms');
    }

    const pos = Math.floor(Math.abs(Math.log10(millis)));
    if (pos <= 3) {
      return fmt(millis * 1_000, 'us');
    }

    return fmt(millis * 1_000_000, 'ns');
  }

  complete(name: string) {
    if (!this.timers[name]) {
      throw new Error(`No timer started with name: ${name}`);
    }

    const { timings } = this.timers[name];
    const median = timings.sort((a, b) => a - b)[Math.floor(timings.length / 2)];

    const label = name.padStart(this.longestLength, ' ');
    console.log(`${label}: ${this.formatTimestamp(median)} (${timings.length} total)`);

    delete this.timers[name];
  }

  completeAll() {
    Object.keys(this.timers)
      .sort()
      .forEach((name) => this.complete(name));
  }
}

export default new Timers();
