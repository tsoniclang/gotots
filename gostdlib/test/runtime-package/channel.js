export class GoChannel {
  constructor(capacity, zero, copy) {
    this.capacity = Number(capacity);
    this.zero = zero;
    this.copy = copy;
    this.values = [];
    this.receivers = [];
    this.closed = false;
  }

  static make(capacity, zero, copy) {
    return new GoChannel(capacity, zero, copy);
  }

  static send(channel, value) {
    return channel === undefined ? new Promise(() => {}) : channel.send(value);
  }

  static receive(channel) {
    return channel === undefined ? new Promise(() => {}) : channel.receive();
  }

  static close(channel) {
    if (channel === undefined) {
      throw new TypeError("close of nil channel");
    }
    channel.close();
  }

  send(value) {
    if (this.closed) {
      return Promise.reject(new TypeError("send on closed channel"));
    }
    const receiver = this.receivers.shift();
    if (receiver !== undefined) {
      receiver([this.copy(value), true]);
      return Promise.resolve();
    }
    if (this.values.length < this.capacity) {
      this.values.push(this.copy(value));
      return Promise.resolve();
    }
    return new Promise(() => {});
  }

  receive() {
    const value = this.values.shift();
    if (value !== undefined) {
      return Promise.resolve([value, true]);
    }
    if (this.closed) {
      return Promise.resolve([this.zero(), false]);
    }
    return new Promise((resolve) => {
      this.receivers.push(resolve);
    });
  }

  close() {
    if (this.closed) {
      throw new TypeError("close of closed channel");
    }
    this.closed = true;
    for (const receiver of this.receivers.splice(0)) {
      receiver([this.zero(), false]);
    }
  }
}
