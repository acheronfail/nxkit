import net from 'node:net';
import { app } from 'electron';
import { exec } from '@vscode/sudo-prompt';
import xpipe from 'xpipe';

const args = process.argv.slice(2).join(' ');
const match = /--disk-pipe(?:=| +)(\S+)/.exec(args);
if (match) {
  const [, pipe] = match;
  if (!pipe) {
    throw new Error(`Failed to extract pipe path from: ${args}`);
  }

  setupClient(pipe);
} else {
  setupServer();
}

function setupClient(pipe) {
  // TODO: create a worker process here, and receive commands over the socket

  const client = net.connect(pipe, () => console.log('client connected'));
  client.on('data', (data) => {
    console.log({ data: data.toString() });
    client.write('World!');
    client.end();
  });

  client.on('end', () => {
    console.log('client ended');
    app.exit();
  });
}

function setupServer() {
  // TODO: should I re-use the same main script, or create a different one for this? :thinking:
  const myself = process.argv
    .slice(0, 2)
    .map((s) => `"${s}"`)
    .join(' ');

  const server = net.createServer((socket) => {
    console.log('connect');

    socket.on('data', (data) => console.log({ data: data.toString() }));
    socket.on('end', () => {
      console.log('end');
      server.close();
    });

    socket.write('Hello!');
  });

  // TODO: if the pipe already exists, create a new one at a different location? (which would allow multi-instance apps, etc)
  const pipePath = xpipe.eq('/tmp/nxkit.explorer.worker');

  server.listen(pipePath, () => {
    console.log(`Server listening at ${pipePath}`);
    console.log('Spawning sudo process...');

    // DANGER: backslashes can exec code if they exist in the env's key-value pairs
    // for example: `exec('echo $FOO', { env: { FOO: "`whoami`" } }, (e, out, err) => console.log(out))`
    // TODO: use icon here
    exec(`${myself} --disk-pipe="${pipePath}"`, { name: 'NXKit' }, (err, stdout, stderr) => {
      console.log('sudo:err', err);
      console.log('sudo:stdout', stdout);
      console.log('sudo:stderr', stderr);
      app.exit();
    });
  });

  server.on('close', () => console.log('server closed'));
}
