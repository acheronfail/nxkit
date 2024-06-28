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

  complete(name: string) {
    if (!this.timers[name]) {
      throw new Error(`No timer started with name: ${name}`);
    }

    const { timings } = this.timers[name];
    const median = timings.sort((a, b) => a - b)[Math.floor(timings.length / 2)];
    console.log(
      `${name.padStart(this.longestLength, ' ')}: ${median.toFixed(4).padStart(9, ' ')} ms (${timings.length} total)`,
    );

    delete this.timers[name];
  }

  completeAll() {
    Object.keys(this.timers)
      .sort()
      .forEach((name) => this.complete(name));
  }
}

export default new Timers();
