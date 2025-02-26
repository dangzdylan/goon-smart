<script lang="ts">
  import { onMount } from "svelte";

  let players = {};
  let bullets = [];
  let canvas;

  let ws = new WebSocket("ws://10.41.68.22:8080/ws"); // Use Mac's IP

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    players = data.players;
    bullets = data.bullets;
  };

  function move(x: number, y: number) {
    ws.send(JSON.stringify({ type: "move", x, y }));
  }

  function shoot(event: MouseEvent) {
    const rect = canvas.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;
    ws.send(JSON.stringify({ type: "shoot", x, y }));
  }

  onMount(() => {
    const ctx = canvas.getContext("2d");

    function render() {
      ctx.clearRect(0, 0, 640, 480);
      ctx.fillStyle = "black";

      // Draw players
      Object.values(players).forEach((p: any) => {
        ctx.beginPath();
        ctx.arc(p.x, p.y, 30, 0, Math.PI * 2);
        ctx.fill();
      });

      // Draw bullets
      ctx.fillStyle = "yellow";
      bullets.forEach((b: any) => {
        ctx.beginPath();
        ctx.arc(b.x, b.y, 5, 0, Math.PI * 2);
        ctx.fill();
      });

      requestAnimationFrame(render);
    }
    render();
  });
</script>

<canvas bind:this={canvas} width="640" height="480" on:click={shoot}></canvas>

<!-- Movement Controls -->
<button on:click={() => move(-1, 0)}>←</button>
<button on:click={() => move(1, 0)}>→</button>
<button on:click={() => move(0, -1)}>↑</button>
<button on:click={() => move(0, 1)}>↓</button>
