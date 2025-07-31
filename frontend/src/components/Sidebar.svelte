<script>
  const icons = [
    'cloud.svg',
    'server-stack.svg',
    'computer-desktop.svg',
    'router-network.svg',
    'shield-check.svg'
  ];
</script>

<aside class="sidebar">
  {#each icons as icon}
    <img src={'/icons/' + icon} alt={icon} />
  {/each}
</aside>

<style>
.sidebar {
  width: 60px;
  background: white;
  padding-top: 1rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  border-right: 1px solid #e5e7eb;
  box-shadow: 2px 0 4px rgba(0,0,0,0.1);
}

.sidebar img {
  width: 24px;
  height: 24px;
  margin: 0.75rem 0;
}
</style>

